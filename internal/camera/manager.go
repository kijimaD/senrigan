package camera

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DefaultCameraManager はCamera Managerのデフォルト実装
type DefaultCameraManager struct {
	discovery Discovery
	cameras   map[string]*Camera
	services  map[string]Service
	mu        sync.RWMutex

	// デフォルト設定
	defaultSettings Settings

	// 制御用
	stopCh chan struct{}
	wg     sync.WaitGroup

	// 自動検出設定
	autoDiscovery bool
	scanInterval  time.Duration
}

// NewDefaultCameraManager は新しいDefaultCameraManagerを作成する
func NewDefaultCameraManager(discovery Discovery, defaultSettings Settings) Manager {
	return &DefaultCameraManager{
		discovery:       discovery,
		cameras:         make(map[string]*Camera),
		services:        make(map[string]Service),
		defaultSettings: defaultSettings,
		stopCh:          make(chan struct{}),
		autoDiscovery:   true,
		scanInterval:    30 * time.Second, // 30秒間隔で自動スキャン
	}
}

// Start はカメラマネージャーを開始する
func (m *DefaultCameraManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 初期スキャンを実行
	if _, err := m.performDiscovery(ctx); err != nil {
		return fmt.Errorf("初期スキャンに失敗: %w", err)
	}

	// 自動検出が有効な場合、バックグラウンドスキャンを開始
	if m.autoDiscovery {
		m.wg.Add(1)
		go m.backgroundScan(ctx)
	}

	return nil
}

// Stop はカメラマネージャーを停止する
func (m *DefaultCameraManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// バックグラウンドスキャンを停止
	close(m.stopCh)
	m.wg.Wait()

	// 全カメラサービスを停止
	var stopErrors []error
	for id, service := range m.services {
		if err := service.Stop(ctx); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("カメラ %s の停止に失敗: %w", id, err))
		}
	}

	if len(stopErrors) > 0 {
		return fmt.Errorf("一部のカメラ停止に失敗: %v", stopErrors)
	}

	// リソースをクリア
	m.cameras = make(map[string]*Camera)
	m.services = make(map[string]Service)
	m.stopCh = make(chan struct{})

	return nil
}

// GetCameras は現在管理されているカメラ一覧を取得する
func (m *DefaultCameraManager) GetCameras() []Camera {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cameras := make([]Camera, 0, len(m.cameras))
	for _, camera := range m.cameras {
		cameras = append(cameras, *camera)
	}

	return cameras
}

// GetCamera は指定されたIDのカメラを取得する
func (m *DefaultCameraManager) GetCamera(id string) (*Camera, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	camera, exists := m.cameras[id]
	if !exists {
		return nil, false
	}

	// コピーを返す
	result := *camera
	return &result, true
}

// AddCamera はカメラを動的に追加する
func (m *DefaultCameraManager) AddCamera(ctx context.Context, device string, settings Settings) (*Camera, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// デバイスの利用可能性をチェック
	if !m.discovery.IsDeviceAvailable(ctx, device) {
		return nil, fmt.Errorf("デバイスが利用できません: %s", device)
	}

	// 既に同じデバイスが登録されているかチェック
	for _, camera := range m.cameras {
		if camera.Device == device {
			return nil, fmt.Errorf("デバイス %s は既に追加されています", device)
		}
	}

	// デバイス情報を取得
	deviceInfo, err := m.discovery.GetDeviceInfo(ctx, device)
	if err != nil {
		return nil, fmt.Errorf("デバイス情報の取得に失敗: %w", err)
	}

	// 新しいカメラを作成
	camera := &Camera{
		ID:       uuid.New().String(),
		Name:     deviceInfo.Name,
		Device:   device,
		FPS:      settings.FPS,
		Width:    settings.Width,
		Height:   settings.Height,
		Status:   StatusInactive,
		LastSeen: time.Now(),
	}

	// カメラサービスを作成
	service := NewCameraService(camera)

	// 管理対象に追加
	m.cameras[camera.ID] = camera
	m.services[camera.ID] = service

	return camera, nil
}

// RemoveCamera はカメラを削除する
func (m *DefaultCameraManager) RemoveCamera(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.cameras[id]
	if !exists {
		return fmt.Errorf("カメラが見つかりません: %s", id)
	}

	// カメラが動作中の場合は停止
	service := m.services[id]
	if service.GetStatus() == StatusActive {
		if err := service.Stop(ctx); err != nil {
			return fmt.Errorf("カメラの停止に失敗: %w", err)
		}
	}

	// 管理対象から削除
	delete(m.cameras, id)
	delete(m.services, id)

	return nil
}

// StartCamera はカメラを開始する
func (m *DefaultCameraManager) StartCamera(ctx context.Context, id string) error {
	m.mu.RLock()
	service, exists := m.services[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("カメラが見つかりません: %s", id)
	}

	return service.Start(ctx)
}

// StopCamera はカメラを停止する
func (m *DefaultCameraManager) StopCamera(ctx context.Context, id string) error {
	m.mu.RLock()
	service, exists := m.services[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("カメラが見つかりません: %s", id)
	}

	return service.Stop(ctx)
}

// DiscoverCameras はシステム内のカメラデバイスを再検出する
func (m *DefaultCameraManager) DiscoverCameras(ctx context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.performDiscovery(ctx)
}

// performDiscovery は実際の検出処理を実行する（ロック済み前提）
func (m *DefaultCameraManager) performDiscovery(ctx context.Context) ([]string, error) {
	devices, err := m.discovery.ScanDevices(ctx)
	if err != nil {
		return nil, err
	}

	// 新しく検出されたデバイスを自動追加
	for _, device := range devices {
		// 既に登録済みかチェック
		isRegistered := false
		for _, camera := range m.cameras {
			if camera.Device == device {
				isRegistered = true
				break
			}
		}

		if !isRegistered {
			// デフォルト設定で自動追加
			_, err := m.addCameraInternal(ctx, device, m.defaultSettings)
			if err != nil {
				// ログ出力は実際の実装で行う
				continue
			}
		}
	}

	// 存在しなくなったデバイスを検出
	var toRemove []string
	for id, camera := range m.cameras {
		deviceExists := false
		for _, device := range devices {
			if camera.Device == device {
				deviceExists = true
				break
			}
		}

		if !deviceExists {
			toRemove = append(toRemove, id)
		}
	}

	// 存在しないデバイスを削除
	for _, id := range toRemove {
		m.removeCameraInternal(ctx, id)
	}

	return devices, nil
}

// addCameraInternal は内部でカメラを追加する（ロック済み前提）
func (m *DefaultCameraManager) addCameraInternal(ctx context.Context, device string, settings Settings) (*Camera, error) {
	// デバイス情報を取得
	deviceInfo, err := m.discovery.GetDeviceInfo(ctx, device)
	if err != nil {
		return nil, err
	}

	// 新しいカメラを作成
	cam := &Camera{
		ID:       uuid.New().String(),
		Name:     deviceInfo.Name,
		Device:   device,
		FPS:      settings.FPS,
		Width:    settings.Width,
		Height:   settings.Height,
		Status:   StatusInactive,
		LastSeen: time.Now(),
	}

	// カメラサービスを作成
	service := NewCameraService(cam)

	// 管理対象に追加
	m.cameras[cam.ID] = cam
	m.services[cam.ID] = service

	return cam, nil
}

// removeCameraInternal は内部でカメラを削除する（ロック済み前提）
func (m *DefaultCameraManager) removeCameraInternal(ctx context.Context, id string) error {
	service, exists := m.services[id]
	if !exists {
		return nil
	}

	// カメラが動作中の場合は停止
	if service.GetStatus() == StatusActive {
		service.Stop(ctx) // エラーは無視
	}

	// 管理対象から削除
	delete(m.cameras, id)
	delete(m.services, id)

	return nil
}

// backgroundScan は定期的なデバイススキャンを実行する
func (m *DefaultCameraManager) backgroundScan(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 定期的にデバイスをスキャン
			m.mu.Lock()
			m.performDiscovery(ctx)
			m.mu.Unlock()
		}
	}
}

// SetAutoDiscovery は自動検出の有効/無効を設定する
func (m *DefaultCameraManager) SetAutoDiscovery(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.autoDiscovery = enabled
}

// SetScanInterval はスキャン間隔を設定する
func (m *DefaultCameraManager) SetScanInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanInterval = interval
}
