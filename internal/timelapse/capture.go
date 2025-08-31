package timelapse

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"senrigan/internal/camera"
)

// Capture は統合タイムラプスキャプチャを管理する
type Capture struct {
	frameBuffer  []CombinedFrame      // 結合フレーム保存
	outputDir    string               // 動画出力先
	currentVideo string               // 現在の動画ファイル
	lastUpdate   time.Time            // 最後の動画更新時刻
	config       Config               // 設定
	videoSources []camera.VideoSource // 全ての映像ソース

	// 制御用
	stopCh chan struct{}
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// フレーム結合・動画生成用
	frameComposer  *FrameComposer
	videoGenerator *VideoGenerator
}

// NewCapture は新しいCapture を作成する
func NewCapture(outputDir string, config Config, videoSources []camera.VideoSource) *Capture {
	return &Capture{
		frameBuffer:    make([]CombinedFrame, 0, config.MaxFrameBuffer),
		outputDir:      outputDir,
		config:         config,
		videoSources:   videoSources,
		stopCh:         make(chan struct{}),
		frameComposer:  NewFrameComposer(config.Resolution.Width, config.Resolution.Height, config.Quality),
		videoGenerator: NewVideoGenerator(),
	}
}

// Start はタイムラプスキャプチャを開始する
func (tc *Capture) Start(ctx context.Context) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// 出力ディレクトリを作成
	if err := os.MkdirAll(tc.outputDir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリの作成に失敗: %w", err)
	}

	// フレーム撮影を開始
	tc.wg.Add(1)
	go tc.captureFrames(ctx)

	// 動画更新スケジューラーを開始
	tc.wg.Add(1)
	go tc.videoUpdateScheduler(ctx)

	log.Printf("統合タイムラプスキャプチャを開始 (%d個の映像ソース)", len(tc.videoSources))
	return nil
}

// Stop はタイムラプスキャプチャを停止する
func (tc *Capture) Stop(ctx context.Context) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	close(tc.stopCh)

	// ワーカーゴルーチンの終了を短いタイムアウトで待機
	done := make(chan struct{})
	go func() {
		tc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 正常にワーカーが終了した場合のみログを記録（最終動画更新はスキップ）
		if len(tc.frameBuffer) > 0 {
			log.Printf("シャットダウン時に %d フレームのバッファが残りました。", len(tc.frameBuffer))
		}
	case <-time.After(3 * time.Second):
		log.Printf("ワーカーゴルーチンの停止がタイムアウトしました。強制終了します。")
	case <-ctx.Done():
		log.Printf("コンテキストがキャンセルされました。停止処理を中断します。")
	}

	log.Println("統合タイムラプスキャプチャを停止")
	return nil
}

// captureFrames はフレームを定期的にキャプチャする
func (tc *Capture) captureFrames(ctx context.Context) {
	defer tc.wg.Done()

	ticker := time.NewTicker(tc.config.CaptureInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tc.stopCh:
			return
		case <-ticker.C:
			if err := tc.captureFrame(ctx); err != nil {
				log.Printf("結合フレームキャプチャエラー: %v", err)
			}
		}
	}
}

// captureFrame は1つの結合フレームをキャプチャしてバッファに追加する
func (tc *Capture) captureFrame(ctx context.Context) error {
	// 全映像ソースから結合フレームを作成
	combinedFrame, err := tc.frameComposer.ComposeFrames(ctx, tc.videoSources)
	if err != nil {
		return fmt.Errorf("フレーム結合に失敗: %w", err)
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// フレームをバッファに追加
	tc.frameBuffer = append(tc.frameBuffer, combinedFrame)

	// バッファサイズ制限をチェック
	if len(tc.frameBuffer) > tc.config.MaxFrameBuffer {
		// 古いフレームを削除（FIFO）
		tc.frameBuffer = tc.frameBuffer[1:]
	}

	return nil
}

// videoUpdateScheduler は動画更新のスケジューリングを行う
func (tc *Capture) videoUpdateScheduler(ctx context.Context) {
	defer tc.wg.Done()

	ticker := time.NewTicker(tc.config.UpdateInterval)
	defer ticker.Stop()

	// 日次ローテーションのためのタイマー
	nextMidnight := tc.getNextMidnight()
	midnightTimer := time.NewTimer(time.Until(nextMidnight))
	defer midnightTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tc.stopCh:
			return
		case <-ticker.C:
			if err := tc.updateVideo(); err != nil {
				log.Printf("動画更新エラー: %v", err)
			}
		case <-midnightTimer.C:
			// 日次ローテーション
			if err := tc.rotateVideo(); err != nil {
				log.Printf("動画ローテーションエラー: %v", err)
			}
			// 次の日の0時にタイマーをリセット
			nextMidnight = tc.getNextMidnight()
			midnightTimer.Reset(time.Until(nextMidnight))
		}
	}
}

// updateVideo は現在のフレームバッファから動画を更新する
func (tc *Capture) updateVideo() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if len(tc.frameBuffer) == 0 {
		return nil // フレームがない場合はスキップ
	}

	// 現在の動画ファイル名を決定
	if tc.currentVideo == "" {
		tc.currentVideo = tc.generateVideoFilename(time.Now())
	}

	videoPath := filepath.Join(tc.outputDir, tc.currentVideo)

	// 動画を生成または延長
	if err := tc.videoGenerator.ExtendVideo(videoPath, tc.frameBuffer, tc.config); err != nil {
		return fmt.Errorf("動画の延長に失敗: %w", err)
	}

	// フレームバッファをクリア
	tc.frameBuffer = tc.frameBuffer[:0]
	tc.lastUpdate = time.Now()

	return nil
}

// rotateVideo は日次ローテーションを実行する
func (tc *Capture) rotateVideo() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// 最終更新を実行
	if len(tc.frameBuffer) > 0 {
		if err := tc.updateVideo(); err != nil {
			log.Printf("ローテーション前の最終更新に失敗: %v", err)
		}
	}

	// 新しい動画ファイル名を設定
	tc.currentVideo = tc.generateVideoFilename(time.Now())

	log.Printf("日次ローテーション実行: %s", tc.currentVideo)
	return nil
}

// generateVideoFilename は動画ファイル名を生成する
func (tc *Capture) generateVideoFilename(t time.Time) string {
	dateStr := t.Format("2006-01-02")
	return fmt.Sprintf("timelapse_%s.mp4", dateStr)
}

// getNextMidnight は次の0時の時刻を取得する
func (tc *Capture) getNextMidnight() time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	return next
}

// GetVideos はこのキャプチャの動画一覧を取得する
func (tc *Capture) GetVideos() ([]Video, error) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	var videos []Video

	// 出力ディレクトリの動画ファイルを走査
	entries, err := os.ReadDir(tc.outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return videos, nil // ディレクトリが存在しない場合は空のリストを返す
		}
		return nil, fmt.Errorf("ディレクトリの読み取りに失敗: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".mp4" {
			videoPath := filepath.Join(tc.outputDir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				log.Printf("ファイル情報の取得に失敗: %v", err)
				continue
			}

			video := Video{
				FilePath:    videoPath,
				FileSize:    info.Size(),
				Date:        info.ModTime(),
				Status:      tc.determineVideoStatus(entry.Name()),
				SourceCount: len(tc.videoSources),
			}

			videos = append(videos, video)
		}
	}

	return videos, nil
}

// determineVideoStatus は動画ファイルのステータスを判定する
func (tc *Capture) determineVideoStatus(filename string) Status {
	if tc.currentVideo == filename {
		return StatusRecording
	}
	return StatusCompleted
}

// UpdateConfig は設定を更新する
func (tc *Capture) UpdateConfig(config Config) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.config = config
	return nil
}

// GetConfig は現在の設定を取得する
func (tc *Capture) GetConfig() Config {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	return tc.config
}

// CaptureStatus はキャプチャの現在状態
type CaptureStatus struct {
	CurrentVideo    string
	FrameBufferSize int
	LastUpdate      time.Time
}

// GetStatus は現在の状態を取得する
func (tc *Capture) GetStatus() CaptureStatus {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	return CaptureStatus{
		CurrentVideo:    tc.currentVideo,
		FrameBufferSize: len(tc.frameBuffer),
		LastUpdate:      tc.lastUpdate,
	}
}
