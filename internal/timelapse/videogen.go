package timelapse

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// VideoGenerator は動画生成を担当する
type VideoGenerator struct {
	tempDir string // 一時ファイル用ディレクトリ
}

// NewVideoGenerator は新しいVideoGeneratorを作成する
func NewVideoGenerator() *VideoGenerator {
	return &VideoGenerator{
		tempDir: "/tmp/senrigan-timelapse",
	}
}

// ExtendVideo は既存の動画にフレームを追加して延長する
func (vg *VideoGenerator) ExtendVideo(videoPath string, frames []CombinedFrame, config Config) error {
	if len(frames) == 0 {
		return nil // フレームがない場合は何もしない
	}

	// 一時ディレクトリを作成
	sessionDir := filepath.Join(vg.tempDir, fmt.Sprintf("session_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("一時ディレクトリの作成に失敗: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(sessionDir) // cleanup中のエラーは無視
	}()

	// フレームを一時画像ファイルとして保存
	imageFiles, err := vg.saveFramesAsImages(sessionDir, frames)
	if err != nil {
		return fmt.Errorf("フレーム画像の保存に失敗: %w", err)
	}

	// 動画を生成または延長
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		// 新規動画作成
		return vg.createNewVideo(videoPath, imageFiles, config)
	}
	// 既存動画に追加
	return vg.appendToVideo(videoPath, imageFiles, config)
}

// saveFramesAsImages はフレームを一時画像ファイルとして保存する
func (vg *VideoGenerator) saveFramesAsImages(sessionDir string, frames []CombinedFrame) ([]string, error) {
	// 事前にスライスの容量を確保（prealloc）
	imageFiles := make([]string, 0, len(frames))

	for i, frame := range frames {
		if len(frame.ComposedData) == 0 {
			continue // 空のフレームはスキップ
		}

		filename := fmt.Sprintf("frame_%06d.jpg", i)
		filepath := filepath.Join(sessionDir, filename)

		if err := os.WriteFile(filepath, frame.ComposedData, 0644); err != nil {
			return nil, fmt.Errorf("フレーム画像の保存に失敗 (%s): %w", filename, err)
		}

		imageFiles = append(imageFiles, filepath)
	}

	return imageFiles, nil
}

// createNewVideo は新しい動画ファイルを作成する
func (vg *VideoGenerator) createNewVideo(videoPath string, imageFiles []string, config Config) error {
	if len(imageFiles) == 0 {
		return fmt.Errorf("画像ファイルがありません")
	}

	// 画像ファイルリストを作成
	listFile := filepath.Join(filepath.Dir(imageFiles[0]), "images.txt")
	if err := vg.createImageList(listFile, imageFiles); err != nil {
		return fmt.Errorf("画像リストの作成に失敗: %w", err)
	}

	// FFmpegで動画を作成
	cmd := exec.Command("ffmpeg",
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-r", "30", // 30fpsで出力
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", vg.qualityToCRF(config.Quality),
		"-pix_fmt", "yuv420p",
		"-y", // 上書き許可
		videoPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("新規動画作成に失敗: %w (output: %s)", err, string(output))
	}

	return nil
}

// appendToVideo は既存の動画にフレームを追加する
func (vg *VideoGenerator) appendToVideo(videoPath string, imageFiles []string, config Config) error {
	if len(imageFiles) == 0 {
		return fmt.Errorf("追加する画像ファイルがありません")
	}

	// 追加用の一時動画を作成
	tempVideoPath := videoPath + ".temp"
	if err := vg.createNewVideo(tempVideoPath, imageFiles, config); err != nil {
		return fmt.Errorf("一時動画の作成に失敗: %w", err)
	}
	defer func() {
		_ = os.Remove(tempVideoPath) // cleanup中のエラーは無視
	}()

	// 動画リストファイルを作成
	listFile := filepath.Join(filepath.Dir(videoPath), "concat_list.txt")
	listContent := fmt.Sprintf("file '%s'\nfile '%s'\n", videoPath, tempVideoPath)
	if err := os.WriteFile(listFile, []byte(listContent), 0644); err != nil {
		return fmt.Errorf("結合リストの作成に失敗: %w", err)
	}
	defer func() {
		_ = os.Remove(listFile) // cleanup中のエラーは無視
	}()

	// 結合した動画を出力
	outputPath := videoPath + ".new"
	cmd := exec.Command("ffmpeg",
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy", // 再エンコードなし
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("動画結合に失敗: %w (output: %s)", err, string(output))
	}

	// 元のファイルを置き換え
	if err := os.Rename(outputPath, videoPath); err != nil {
		return fmt.Errorf("ファイル置き換えに失敗: %w", err)
	}

	return nil
}

// createImageList は画像ファイルリストを作成する
func (vg *VideoGenerator) createImageList(listFile string, imageFiles []string) error {
	var content string
	for _, imageFile := range imageFiles {
		// 各フレームを0.033秒（30fpsの逆数）表示
		content += fmt.Sprintf("file '%s'\nduration 0.033\n", imageFile)
	}

	// 最後のフレームは追加の表示時間なし
	if len(imageFiles) > 0 {
		content += fmt.Sprintf("file '%s'\n", imageFiles[len(imageFiles)-1])
	}

	return os.WriteFile(listFile, []byte(content), 0644)
}

// qualityToCRF は品質設定をFFmpegのCRF値に変換する
func (vg *VideoGenerator) qualityToCRF(quality int) string {
	// 品質1(低) -> CRF28, 品質5(高) -> CRF18
	crf := 28.0 - float64(quality-1)*2.5
	if crf < 18 {
		crf = 18
	}
	if crf > 28 {
		crf = 28
	}
	return strconv.FormatFloat(crf, 'f', 1, 64)
}

// ValidateFFmpeg はFFmpegが利用可能かチェックする
func (vg *VideoGenerator) ValidateFFmpeg() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FFmpegが見つかりません。インストールしてください: %w", err)
	}

	return nil
}
