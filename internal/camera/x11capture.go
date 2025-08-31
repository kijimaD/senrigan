package camera

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// X11Capturer はX11画面キャプチャを行う
type X11Capturer struct {
	display string
	width   int
	height  int
	fps     int
}

// NewX11Capturer は新しいX11Capturerを作成する
func NewX11Capturer(display string, width, height, fps int) *X11Capturer {
	return &X11Capturer{
		display: display,
		width:   width,
		height:  height,
		fps:     fps,
	}
}

// IsDeviceAvailable はX11ディスプレイが利用可能かチェックする
func (c *X11Capturer) IsDeviceAvailable(ctx context.Context) bool {
	// xdpyinfoコマンドでX11ディスプレイの利用可能性をチェック
	cmd := exec.CommandContext(ctx, "xdpyinfo", "-display", c.display)
	err := cmd.Run()
	return err == nil
}

// TestCapture はX11画面キャプチャのテストを実行する
func (c *X11Capturer) TestCapture(ctx context.Context) error {
	// タイムアウト付きでテストキャプチャ
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// ffmpegを使って1フレームをキャプチャしてテスト
	cmd := exec.CommandContext(testCtx,
		"ffmpeg",
		"-f", "x11grab",
		"-video_size", fmt.Sprintf("%dx%d", c.width, c.height),
		"-i", c.display,
		"-vframes", "1",
		"-f", "image2",
		"-c:v", "mjpeg",
		"-y", // 出力ファイルを上書き
		"/tmp/x11test.jpg",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("X11画面キャプチャのテストに失敗: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}

// StartStream はX11画面キャプチャのストリームを開始する
func (c *X11Capturer) StartStream(ctx context.Context, frameChan chan<- []byte, errorChan chan<- error) {
	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-f", "x11grab",
		"-video_size", fmt.Sprintf("%dx%d", c.width, c.height),
		"-r", strconv.Itoa(c.fps),
		"-i", c.display,
		"-vf", "format=yuv420p",
		"-f", "image2pipe",
		"-c:v", "mjpeg",
		"-q:v", "3",
		"-",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errorChan <- fmt.Errorf("stdoutパイプの作成に失敗: %w", err)
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		errorChan <- fmt.Errorf("stderrパイプの作成に失敗: %w", err)
		return
	}

	if err := cmd.Start(); err != nil {
		errorChan <- fmt.Errorf("ffmpegの起動に失敗: %w", err)
		return
	}

	// stderrを別goroutineで読み取り
	go func() {
		buf := make([]byte, 1024)
		for {
			_, err := stderr.Read(buf)
			if err != nil {
				break
			}
		}
	}()

	// JPEGフレームを読み取り
	go func() {
		defer func() {
			// コンテキストキャンセル時はプロセスを強制終了
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			_ = cmd.Wait()
		}()

		buffer := make([]byte, 1024*1024) // 1MBバッファ
		frameBuffer := bytes.Buffer{}

		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := stdout.Read(buffer)
				if err != nil {
					if err.Error() != "EOF" {
						errorChan <- fmt.Errorf("フレーム読み取りエラー: %w", err)
					}
					return
				}

				frameBuffer.Write(buffer[:n])

				// JPEGマーカーを探してフレームを分割
				data := frameBuffer.Bytes()
				for {
					// JPEGの開始マーカー（FF D8）を探す
					startIdx := bytes.Index(data, []byte{0xFF, 0xD8})
					if startIdx == -1 {
						break
					}

					// JPEGの終了マーカー（FF D9）を探す
					endIdx := bytes.Index(data[startIdx+2:], []byte{0xFF, 0xD9})
					if endIdx == -1 {
						// 完全なフレームがまだない
						if startIdx > 0 {
							// 不要なデータを削除
							frameBuffer.Reset()
							frameBuffer.Write(data[startIdx:])
						}
						break
					}

					// 完全なJPEGフレームを抽出
					endIdx += startIdx + 2 + 2 // マーカーのサイズを含める
					frame := make([]byte, endIdx)
					copy(frame, data[:endIdx])

					// フレームを送信
					select {
					case frameChan <- frame:
					case <-ctx.Done():
						return
					}

					// 処理済みデータを削除
					remaining := data[endIdx:]
					frameBuffer.Reset()
					if len(remaining) > 0 {
						frameBuffer.Write(remaining)
						data = frameBuffer.Bytes()
					} else {
						break
					}
				}
			}
		}
	}()
}
