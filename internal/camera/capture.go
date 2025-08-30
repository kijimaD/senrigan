package camera

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// V4L2Capturer はシェルコマンドを使ってV4L2デバイスから画像を取得する
type V4L2Capturer struct {
	devicePath string
	width      int
	height     int
	fps        int
}

// NewV4L2Capturer は新しいV4L2Capturerを作成する
func NewV4L2Capturer(devicePath string, width, height, fps int) *V4L2Capturer {
	return &V4L2Capturer{
		devicePath: devicePath,
		width:      width,
		height:     height,
		fps:        fps,
	}
}

// IsDeviceAvailable はV4L2デバイスが利用可能かチェックする
func (c *V4L2Capturer) IsDeviceAvailable(ctx context.Context) bool {
	// v4l2-ctlコマンドでデバイス情報を取得して確認
	cmd := exec.CommandContext(ctx, "v4l2-ctl", "--device", c.devicePath, "--info")
	err := cmd.Run()
	return err == nil
}

// GetDeviceInfo はデバイス情報を取得する
func (c *V4L2Capturer) GetDeviceInfo(ctx context.Context) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "v4l2-ctl", "--device", c.devicePath, "--info")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("デバイス情報の取得に失敗: %w", err)
	}

	info := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				info[key] = value
			}
		}
	}

	return info, nil
}

// CaptureFrame は1フレームをキャプチャしてJPEG画像として返す
func (c *V4L2Capturer) CaptureFrame(ctx context.Context) (image.Image, error) {
	// ffmpegを使って1フレームをキャプチャ
	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-f", "v4l2",
		"-video_size", fmt.Sprintf("%dx%d", c.width, c.height),
		"-i", c.devicePath,
		"-vframes", "1",
		"-f", "image2",
		"-c:v", "mjpeg",
		"-",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("フレームキャプチャに失敗: %w (stderr: %s)", err, stderr.String())
	}

	// JPEGデータを画像にデコード
	img, err := jpeg.Decode(&stdout)
	if err != nil {
		return nil, fmt.Errorf("JPEG画像のデコードに失敗: %w", err)
	}

	return img, nil
}

// CaptureFrameAsJPEG は1フレームをキャプチャしてJPEGバイト配列として返す
func (c *V4L2Capturer) CaptureFrameAsJPEG(ctx context.Context) ([]byte, error) {
	// ffmpegを使って1フレームをJPEGとしてキャプチャ
	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-f", "v4l2",
		"-video_size", fmt.Sprintf("%dx%d", c.width, c.height),
		"-i", c.devicePath,
		"-vframes", "1",
		"-f", "image2",
		"-c:v", "mjpeg",
		"-q:v", "2", // 高品質JPEG
		"-",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("JPEGフレームキャプチャに失敗: %w (stderr: %s)", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// StartStream は連続キャプチャ用のストリームを開始する
func (c *V4L2Capturer) StartStream(ctx context.Context, frameChan chan<- []byte, errorChan chan<- error) {
	// ffmpegを使って連続的にフレームをキャプチャ
	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-f", "v4l2",
		"-video_size", fmt.Sprintf("%dx%d", c.width, c.height),
		"-r", strconv.Itoa(c.fps),
		"-i", c.devicePath,
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
			// エラー情報をログに出力（実装は後で追加）
		}
	}()

	// JPEGフレームを読み取り
	go func() {
		defer func() {
			_ = cmd.Wait() // エラーは無視（コンテキストキャンセル時に発生するため）
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

// TestCapture はデバイステスト用の簡単なキャプチャ機能
func (c *V4L2Capturer) TestCapture(ctx context.Context) error {
	// タイムアウト付きでテストキャプチャ
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.CaptureFrame(testCtx)
	return err
}

// GetSupportedFormats はサポートされているフォーマット一覧を取得する
func (c *V4L2Capturer) GetSupportedFormats(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "v4l2-ctl", "--device", c.devicePath, "--list-formats-ext")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("フォーマット一覧の取得に失敗: %w", err)
	}

	var formats []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
			// フォーマット行を抽出
			formats = append(formats, line)
		}
	}

	return formats, nil
}

// SetControls はカメラのコントロール（明度、コントラストなど）を設定する
func (c *V4L2Capturer) SetControls(ctx context.Context, controls map[string]interface{}) error {
	for control, value := range controls {
		var strValue string
		switch v := value.(type) {
		case int:
			strValue = strconv.Itoa(v)
		case float64:
			strValue = strconv.FormatFloat(v, 'f', -1, 64)
		case string:
			strValue = v
		default:
			return fmt.Errorf("サポートされていない値の型: %T", value)
		}

		cmd := exec.CommandContext(ctx, "v4l2-ctl", "--device", c.devicePath, "--set-ctrl", fmt.Sprintf("%s=%s", control, strValue))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("コントロール %s の設定に失敗: %w", control, err)
		}
	}

	return nil
}
