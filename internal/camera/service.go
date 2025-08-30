package camera

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// defaultCameraService は個別カメラの制御を担う実装
type defaultCameraService struct {
	camera   *Camera
	status   Status
	settings Settings
	mu       sync.RWMutex

	// 制御用チャンネル
	stopCh   chan struct{}
	statusCh chan Status

	// 監視ゴルーチン用
	wg sync.WaitGroup

	// V4L2キャプチャ
	capturer *V4L2Capturer

	// ストリーミング用チャンネル
	frameChan chan []byte
	errorChan chan error
}

// NewCameraService は新しいdefaultCameraServiceを作成する
func NewCameraService(camera *Camera) Service {
	capturer := NewV4L2Capturer(camera.Device, camera.Width, camera.Height, camera.FPS)

	return &defaultCameraService{
		camera: camera,
		status: StatusInactive,
		settings: Settings{
			FPS:    camera.FPS,
			Width:  camera.Width,
			Height: camera.Height,
		},
		stopCh:    make(chan struct{}),
		statusCh:  make(chan Status, 1),
		capturer:  capturer,
		frameChan: make(chan []byte, 10), // 最大10フレームをバッファ
		errorChan: make(chan error, 5),   // エラーチャンネル
	}
}

// Start はカメラサービスを開始する
func (s *defaultCameraService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == StatusActive {
		return fmt.Errorf("カメラ %s は既に開始されています", s.camera.ID)
	}

	// カメラデバイスの初期化を試行（モック実装）
	if err := s.initializeCamera(ctx); err != nil {
		s.status = StatusError
		return fmt.Errorf("カメラ %s の初期化に失敗: %w", s.camera.ID, err)
	}

	// 監視ゴルーチンを開始
	s.wg.Add(1)
	go s.monitorCamera(ctx)

	s.status = StatusActive
	s.camera.Status = StatusActive
	s.camera.LastSeen = time.Now()

	return nil
}

// Stop はカメラサービスを停止する
func (s *defaultCameraService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == StatusInactive {
		return nil // 既に停止している
	}

	// 停止シグナルを送信
	close(s.stopCh)

	// ゴルーチンの終了を待機
	s.wg.Wait()

	// リソースのクリーンアップ
	s.cleanupCamera(ctx)

	s.status = StatusInactive
	s.camera.Status = StatusInactive
	s.camera.LastSeen = time.Now()

	// 新しいチャンネルを作成（再開可能にするため）
	s.stopCh = make(chan struct{})

	return nil
}

// GetStatus は現在の状態を取得する
func (s *defaultCameraService) GetStatus() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// GetSettings は現在の設定を取得する
func (s *defaultCameraService) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// UpdateSettings は設定を更新する
func (s *defaultCameraService) UpdateSettings(ctx context.Context, settings Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 設定の検証
	if err := s.validateSettings(settings); err != nil {
		return fmt.Errorf("設定が無効: %w", err)
	}

	// 設定を適用（モック実装）
	if err := s.applySettings(ctx, settings); err != nil {
		return fmt.Errorf("設定の適用に失敗: %w", err)
	}

	s.settings = settings
	s.camera.FPS = settings.FPS
	s.camera.Width = settings.Width
	s.camera.Height = settings.Height

	return nil
}

// initializeCamera はカメラデバイスを初期化する
func (s *defaultCameraService) initializeCamera(ctx context.Context) error {
	// V4L2デバイスの利用可能性をチェック
	if !s.capturer.IsDeviceAvailable(ctx) {
		return fmt.Errorf("V4L2デバイスが利用できません: %s", s.camera.Device)
	}

	// テストキャプチャを実行してデバイスが正常に動作するかチェック
	if err := s.capturer.TestCapture(ctx); err != nil {
		return fmt.Errorf("デバイスのテストキャプチャに失敗: %w", err)
	}

	// ストリーミングを開始
	go s.capturer.StartStream(ctx, s.frameChan, s.errorChan)

	// エラー監視ゴルーチンを開始
	s.wg.Add(1)
	go s.monitorErrors(ctx)

	return nil
}

// cleanupCamera はカメラデバイスをクリーンアップする
func (s *defaultCameraService) cleanupCamera(_ context.Context) {
	// フレームチャンネルをクリア
	select {
	case <-s.frameChan:
		// チャンネルに残っているデータを消費
	default:
	}

	// エラーチャンネルをクリア
	select {
	case <-s.errorChan:
		// チャンネルに残っているエラーを消費
	default:
	}
}

// monitorCamera はカメラの状態を監視する
func (s *defaultCameraService) monitorCamera(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second) // 5秒間隔で監視
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkCameraHealth(ctx)
		}
	}
}

// checkCameraHealth はカメラの健全性をチェックする
func (s *defaultCameraService) checkCameraHealth(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// V4L2デバイスの利用可能性をチェック
	if !s.capturer.IsDeviceAvailable(ctx) {
		if s.status == StatusActive {
			s.status = StatusError
			s.camera.Status = StatusError
		}
	} else {
		if s.status == StatusError {
			// デバイスが復旧した場合、再初期化を試行
			if err := s.initializeCamera(ctx); err == nil {
				s.status = StatusActive
				s.camera.Status = StatusActive
			}
		}
	}

	s.camera.LastSeen = time.Now()
}

// validateSettings は設定値の妥当性を検証する
func (s *defaultCameraService) validateSettings(settings Settings) error {
	if settings.FPS <= 0 || settings.FPS > 60 {
		return fmt.Errorf("無効なFPS値: %d", settings.FPS)
	}

	if settings.Width <= 0 || settings.Width > 4096 {
		return fmt.Errorf("無効な幅: %d", settings.Width)
	}

	if settings.Height <= 0 || settings.Height > 4096 {
		return fmt.Errorf("無効な高さ: %d", settings.Height)
	}

	return nil
}

// applySettings は設定をカメラデバイスに適用する
func (s *defaultCameraService) applySettings(ctx context.Context, settings Settings) error {
	// 新しい設定でキャプチャを再作成
	s.capturer = NewV4L2Capturer(s.camera.Device, settings.Width, settings.Height, settings.FPS)

	// アクティブな場合は再初期化
	if s.status == StatusActive {
		return s.initializeCamera(ctx)
	}

	return nil
}

// monitorErrors はエラーチャンネルを監視する
func (s *defaultCameraService) monitorErrors(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case err := <-s.errorChan:
			if err != nil {
				// エラーログの出力（実装は後で追加）
				s.mu.Lock()
				if s.status == StatusActive {
					s.status = StatusError
					s.camera.Status = StatusError
				}
				s.mu.Unlock()
			}
		}
	}
}

// GetLatestFrame は最新のフレームを取得する
func (s *defaultCameraService) GetLatestFrame() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.status != StatusActive {
		return nil, fmt.Errorf("カメラが非アクティブです")
	}

	// 非ブロッキングでフレームを取得
	select {
	case frame := <-s.frameChan:
		return frame, nil
	default:
		return nil, fmt.Errorf("利用可能なフレームがありません")
	}
}

// GetFrameChannel はフレームチャンネルを取得する（ストリーミング用）
func (s *defaultCameraService) GetFrameChannel() <-chan []byte {
	return s.frameChan
}

// MockCameraService はテスト用のモックサービス実装
type MockCameraService struct {
	camera   *Camera
	status   Status
	settings Settings
	mu       sync.RWMutex

	// テスト制御用
	shouldFailStart bool
	shouldFailStop  bool
}

// NewMockCameraService は新しいMockCameraServiceを作成する
func NewMockCameraService(camera *Camera) *MockCameraService {
	return &MockCameraService{
		camera: camera,
		status: StatusInactive,
		settings: Settings{
			FPS:    camera.FPS,
			Width:  camera.Width,
			Height: camera.Height,
		},
	}
}

// Start はモックカメラサービスを開始する
func (m *MockCameraService) Start(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status == StatusActive {
		return fmt.Errorf("カメラ %s は既に開始されています", m.camera.ID)
	}

	if m.shouldFailStart {
		m.status = StatusError
		return fmt.Errorf("モック: カメラ開始に失敗")
	}

	m.status = StatusActive
	m.camera.Status = StatusActive
	m.camera.LastSeen = time.Now()
	return nil
}

// Stop はモックカメラサービスを停止する
func (m *MockCameraService) Stop(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailStop {
		return fmt.Errorf("モック: カメラ停止に失敗")
	}

	m.status = StatusInactive
	m.camera.Status = StatusInactive
	m.camera.LastSeen = time.Now()
	return nil
}

// GetStatus は現在の状態を取得する
func (m *MockCameraService) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// GetSettings は現在の設定を取得する
func (m *MockCameraService) GetSettings() Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.settings
}

// UpdateSettings は設定を更新する
func (m *MockCameraService) UpdateSettings(_ context.Context, settings Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.settings = settings
	m.camera.FPS = settings.FPS
	m.camera.Width = settings.Width
	m.camera.Height = settings.Height
	return nil
}

// SetShouldFailStart はテスト用にStart失敗を設定する
func (m *MockCameraService) SetShouldFailStart(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailStart = shouldFail
}

// SetShouldFailStop はテスト用にStop失敗を設定する
func (m *MockCameraService) SetShouldFailStop(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailStop = shouldFail
}

// GetLatestFrame は最新のフレームを取得する（モック実装）
func (m *MockCameraService) GetLatestFrame() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.status != StatusActive {
		return nil, fmt.Errorf("カメラが非アクティブです")
	}

	// モックフレームデータを返す（1x1の白いJPEG）
	return []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46,
		0x00, 0x01, 0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, 0x00,
		0xFF, 0xC0, 0x00, 0x11, 0x08, 0x00, 0x01, 0x00, 0x01, 0x01,
		0x01, 0x11, 0x00, 0x02, 0x11, 0x01, 0x03, 0x11, 0x01, 0xFF,
		0xC4, 0x00, 0x14, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x08, 0xFF, 0xC4, 0x00, 0x14, 0x10, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0xFF, 0xDA, 0x00, 0x0C, 0x03, 0x01, 0x00,
		0x02, 0x11, 0x03, 0x11, 0x00, 0x3F, 0x00, 0x80, 0xFF, 0xD9,
	}, nil
}

// GetFrameChannel はフレームチャンネルを取得する（モック実装）
func (m *MockCameraService) GetFrameChannel() <-chan []byte {
	// モック用のチャンネルを作成
	frameChan := make(chan []byte, 1)

	// モックフレームを1つ送信
	go func() {
		defer close(frameChan)
		frame, _ := m.GetLatestFrame()
		if frame != nil {
			frameChan <- frame
		}
	}()

	return frameChan
}
