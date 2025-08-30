package camera

import (
	"context"
	"fmt"
	"sync"
)

// X11ScreenSource は画面キャプチャの VideoSource 実装
type X11ScreenSource struct {
	BaseVideoSource

	// X11キャプチャ用
	capturer *X11Capturer

	// 制御用
	stopCh chan struct{}
	wg     sync.WaitGroup

	// ストリーミング用の内部チャンネル
	internalFrameChan chan []byte
	internalErrorChan chan error
}


// Start は画面キャプチャを開始する
func (s *X11ScreenSource) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == StatusActive {
		return nil // 既に開始済み
	}

	// キャプチャの利用可能性をチェック
	if !s.capturer.IsDeviceAvailable(ctx) {
		s.status = StatusError
		return fmt.Errorf("X11画面キャプチャが利用できません")
	}

	// テストキャプチャを実行
	if err := s.capturer.TestCapture(ctx); err != nil {
		s.status = StatusError
		return fmt.Errorf("画面キャプチャのテストに失敗: %w", err)
	}

	// ストリーミングを開始
	go s.capturer.StartStream(ctx, s.internalFrameChan, s.internalErrorChan)

	// フレーム転送ゴルーチンを開始
	s.wg.Add(1)
	go s.forwardFrames()

	s.status = StatusActive
	return nil
}

// Stop は画面キャプチャを停止する
func (s *X11ScreenSource) Stop(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == StatusInactive {
		return nil // 既に停止済み
	}

	// 停止シグナルを送信
	close(s.stopCh)

	// ゴルーチンの終了を待機
	s.wg.Wait()

	// 新しいstopChを作成（再開可能にするため）
	s.stopCh = make(chan struct{})

	s.status = StatusInactive
	return nil
}

// IsAvailable は画面キャプチャが利用可能かチェックする
func (s *X11ScreenSource) IsAvailable(ctx context.Context) bool {
	return s.capturer.IsDeviceAvailable(ctx)
}

// ApplySettings は設定を適用する
func (s *X11ScreenSource) ApplySettings(ctx context.Context, settings VideoSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 新しい設定でキャプチャを再作成
	s.capturer = NewX11Capturer(":0.0", settings.Width, settings.Height, settings.FrameRate)

	// 内部設定を更新
	s.settings = settings

	// アクティブな場合は再開始が必要
	if s.status == StatusActive {
		// 停止してから再開始
		s.status = StatusInactive
		close(s.stopCh)
		s.wg.Wait()
		s.stopCh = make(chan struct{})

		// 再開始
		go s.capturer.StartStream(ctx, s.internalFrameChan, s.internalErrorChan)
		s.wg.Add(1)
		go s.forwardFrames()
		s.status = StatusActive
	}

	return nil
}


// forwardFrames はキャプチャからフレームを転送する
func (s *X11ScreenSource) forwardFrames() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopCh:
			return

		case frame, ok := <-s.internalFrameChan:
			if !ok {
				// チャンネルがクローズされた
				return
			}

			// フレームを転送
			select {
			case s.frameChan <- frame:
			case <-s.stopCh:
				return
			default:
				// チャンネルがフルの場合は古いフレームを破棄
				select {
				case <-s.frameChan:
				default:
				}
				select {
				case s.frameChan <- frame:
				case <-s.stopCh:
					return
				}
			}

		case err, ok := <-s.internalErrorChan:
			if !ok {
				// チャンネルがクローズされた
				return
			}

			// エラーを転送
			select {
			case s.errorChan <- err:
			case <-s.stopCh:
				return
			default:
				// エラーチャンネルがフルの場合は古いエラーを破棄
				select {
				case <-s.errorChan:
				default:
				}
				select {
				case s.errorChan <- err:
				case <-s.stopCh:
					return
				}
			}
		}
	}
}

// NewX11ScreenSourceFromConfig は設定からX11ScreenSourceを作成する
func NewX11ScreenSourceFromConfig(config SourceConfig) (VideoSource, error) {
	// デフォルト設定
	width := 1920
	height := 1080
	fps := 10

	// 設定が指定されている場合は使用
	if config.Settings.Width > 0 {
		width = config.Settings.Width
	}
	if config.Settings.Height > 0 {
		height = config.Settings.Height
	}
	if config.Settings.FrameRate > 0 {
		fps = config.Settings.FrameRate
	}

	// VideoSourceInfo を設定
	info := VideoSourceInfo{
		ID:          generateCameraID(),
		Name:        "画面キャプチャ",
		Type:        SourceTypeX11Screen,
		Driver:      "x11grab",
		Description: "X11 Screen Capture",
		Device:      "x11:screen",
	}

	// VideoCapabilities を設定
	capabilities := VideoCapabilities{
		SupportedResolutions: []Resolution{
			{Width: 800, Height: 600},
			{Width: 1280, Height: 720},
			{Width: 1920, Height: 1080},
		},
		SupportedFrameRates: []int{5, 10, 15, 30},
		SupportedFormats:    []string{"MJPEG"},
	}

	// VideoSettings を設定
	settings := VideoSettings{
		Width:      width,
		Height:     height,
		FrameRate:  fps,
		Format:     "MJPEG",
		Quality:    3,
		Properties: make(map[string]interface{}),
	}

	source := &X11ScreenSource{
		BaseVideoSource: BaseVideoSource{
			info:         info,
			capabilities: capabilities,
			settings:     settings,
			frameChan:    make(chan []byte, 10),
			errorChan:    make(chan error, 5),
			status:       StatusInactive,
		},
		capturer:          NewX11Capturer(":0.0", width, height, fps),
		stopCh:            make(chan struct{}),
		internalFrameChan: make(chan []byte, 10),
		internalErrorChan: make(chan error, 5),
	}

	return source, nil
}
