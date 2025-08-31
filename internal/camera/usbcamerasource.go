package camera

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// USBCameraSource はUSBカメラの VideoSource 実装
type USBCameraSource struct {
	BaseVideoSource

	// V4L2キャプチャ用
	capturer *V4L2Capturer

	// 制御用
	stopCh chan struct{}
	wg     sync.WaitGroup

	// ストリーミング用の内部チャンネル
	internalFrameChan chan []byte
	internalErrorChan chan error

	// 最新フレーム保持用（タイムラプス用）
	latestFrame []byte
	latestMutex sync.RWMutex
}

// NewDirectUSBCameraSource は新しいUSBCameraSourceを作成する（Service不使用）
func NewDirectUSBCameraSource(info VideoSourceInfo, capabilities VideoCapabilities, settings VideoSettings) VideoSource {
	// V4L2Capturerを直接作成
	capturer := NewV4L2Capturer(info.Device, settings.Width, settings.Height, settings.FrameRate)

	source := &USBCameraSource{
		BaseVideoSource: BaseVideoSource{
			info:         info,
			capabilities: capabilities,
			settings:     settings,
			frameChan:    make(chan []byte, 10),
			errorChan:    make(chan error, 5),
			status:       StatusInactive,
		},
		capturer:          capturer,
		stopCh:            make(chan struct{}),
		internalFrameChan: make(chan []byte, 10),
		internalErrorChan: make(chan error, 5),
	}

	return source
}

// Start はカメラを開始する
func (s *USBCameraSource) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == StatusActive {
		return nil // 既に開始済み
	}

	// デバイステストを実行
	if err := s.capturer.TestCapture(ctx); err != nil {
		s.status = StatusError
		return fmt.Errorf("カメラのテストキャプチャに失敗: %w", err)
	}

	// ストリーミングを開始
	go s.capturer.StartStream(ctx, s.internalFrameChan, s.internalErrorChan)

	// フレーム転送ゴルーチンを開始
	s.wg.Add(1)
	go s.forwardFrames()

	s.status = StatusActive
	return nil
}

// Stop はカメラを停止する
func (s *USBCameraSource) Stop(_ context.Context) error {
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

// IsAvailable はカメラが利用可能かチェックする
func (s *USBCameraSource) IsAvailable(ctx context.Context) bool {
	return s.capturer.IsDeviceAvailable(ctx)
}

// ApplySettings は設定を適用する
func (s *USBCameraSource) ApplySettings(ctx context.Context, settings VideoSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 新しい設定でキャプチャを再作成
	s.capturer = NewV4L2Capturer(s.info.Device, settings.Width, settings.Height, settings.FrameRate)

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
func (s *USBCameraSource) forwardFrames() {
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

			// 最新フレームを保存（タイムラプス用）
			s.latestMutex.Lock()
			s.latestFrame = make([]byte, len(frame))
			copy(s.latestFrame, frame)
			s.latestMutex.Unlock()

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

		case <-time.After(100 * time.Millisecond):
			// 定期的にステータスを更新（停止されたら終了）
			select {
			case <-s.stopCh:
				return
			default:
			}
		}
	}
}

// CaptureFrameForTimelapse はタイムラプス用に1フレームをキャプチャする
func (s *USBCameraSource) CaptureFrameForTimelapse(_ context.Context) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.status != StatusActive {
		return nil, fmt.Errorf("カメラが非アクティブです")
	}

	// ストリーミング中の最新フレームを取得
	s.latestMutex.RLock()
	defer s.latestMutex.RUnlock()

	if s.latestFrame == nil {
		return nil, fmt.Errorf("フレームがまだ取得されていません")
	}

	// フレームのコピーを返す
	frame := make([]byte, len(s.latestFrame))
	copy(frame, s.latestFrame)
	return frame, nil
}
