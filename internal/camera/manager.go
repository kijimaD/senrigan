package camera

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// DefaultCameraManager はCamera Managerのデフォルト実装
type DefaultCameraManager struct {
	discovery Discovery
	mu        sync.RWMutex

	// デフォルト設定
	defaultSettings VideoSettings

	// 制御用
	stopCh chan struct{}
	wg     sync.WaitGroup

	// 自動検出設定
	autoDiscovery bool
	scanInterval  time.Duration

	// VideoSource管理用
	videoSources  map[string]VideoSource
	sourceFactory VideoSourceFactory
}

// NewDefaultCameraManager は新しいDefaultCameraManagerを作成する
func NewDefaultCameraManager(discovery Discovery) Manager {
	// デフォルトのVideoSettings
	defaultSettings := VideoSettings{
		Width:      1280,
		Height:     720,
		FrameRate:  15,
		Format:     "MJPEG",
		Quality:    3,
		Properties: make(map[string]interface{}),
	}

	return &DefaultCameraManager{
		discovery:       discovery,
		defaultSettings: defaultSettings,
		stopCh:          make(chan struct{}),
		autoDiscovery:   true,
		scanInterval:    30 * time.Second,
		videoSources:    make(map[string]VideoSource),
		sourceFactory:   NewVideoSourceFactory(),
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

	// X11画面録画をデフォルトで追加（USBカメラの前に追加）
	x11Config := SourceConfig{
		Device: "x11:screen",
		Settings: VideoSettings{
			Width:      1920,
			Height:     1080,
			FrameRate:  10,
			Format:     "MJPEG",
			Quality:    3,
			Properties: make(map[string]interface{}),
		},
	}

	x11Source, err := m.sourceFactory.CreateSource(SourceTypeX11Screen, x11Config)
	if err != nil {
		log.Printf("X11画面録画の作成に失敗: %v", err)
	} else {
		// VideoSourceを管理対象に追加
		sourceID := x11Source.GetInfo().ID
		m.videoSources[sourceID] = x11Source

		// X11画面録画を自動的に開始
		if err := x11Source.Start(ctx); err != nil {
			log.Printf("X11画面録画 %s の自動開始に失敗: %v", sourceID, err)
		} else {
			log.Printf("X11画面録画 %s を自動開始しました", sourceID)
		}
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

	// 全VideoSourceを停止
	var stopErrors []error
	for id, videoSource := range m.videoSources {
		if err := videoSource.Stop(ctx); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("VideoSource %s の停止に失敗: %w", id, err))
		}
	}

	if len(stopErrors) > 0 {
		return fmt.Errorf("一部のVideoSource停止に失敗: %v", stopErrors)
	}

	// リソースをクリア
	m.videoSources = make(map[string]VideoSource)
	m.stopCh = make(chan struct{})

	return nil
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
		for _, source := range m.videoSources {
			if source.GetInfo().Device == device {
				isRegistered = true
				break
			}
		}

		if !isRegistered {
			// デフォルト設定で自動追加
			_, err := m.addVideoSourceInternal(ctx, device, m.defaultSettings)
			if err != nil {
				// ログ出力は実際の実装で行う
				continue
			}
		}
	}

	// 存在しなくなったデバイスを検出
	var toRemove []string
	for id, source := range m.videoSources {
		// X11ScreenSourceは削除対象から除外
		if source.GetInfo().Type == SourceTypeX11Screen {
			continue
		}

		deviceExists := false
		for _, device := range devices {
			if source.GetInfo().Device == device {
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
		m.removeVideoSourceInternal(ctx, id)
	}

	return devices, nil
}

// addVideoSourceInternal は内部でVideoSourceを追加する（ロック済み前提）
func (m *DefaultCameraManager) addVideoSourceInternal(ctx context.Context, device string, settings VideoSettings) (VideoSource, error) {
	config := SourceConfig{
		Device:   device,
		Settings: settings,
	}

	videoSource, err := m.sourceFactory.CreateSource(SourceTypeUSBCamera, config)
	if err != nil {
		return nil, err
	}

	// VideoSourceを管理対象に追加
	sourceID := videoSource.GetInfo().ID
	m.videoSources[sourceID] = videoSource

	// VideoSourceを自動的に開始
	if err := videoSource.Start(ctx); err != nil {
		log.Printf("VideoSource %s の自動開始に失敗: %v", sourceID, err)
	} else {
		log.Printf("VideoSource %s を自動開始しました", sourceID)
	}

	return videoSource, nil
}

// removeVideoSourceInternal は内部でVideoSourceを削除する（ロック済み前提）
func (m *DefaultCameraManager) removeVideoSourceInternal(ctx context.Context, id string) {
	source, exists := m.videoSources[id]
	if !exists {
		return
	}

	// VideoSourceが動作中の場合は停止
	if source.GetStatus() == StatusActive {
		_ = source.Stop(ctx) // エラーは無視
	}

	// 管理対象から削除
	delete(m.videoSources, id)
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
			_, _ = m.performDiscovery(ctx)
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

// AddVideoSource はVideoSourceを追加する
func (m *DefaultCameraManager) AddVideoSource(_ context.Context, sourceType VideoSourceType, config SourceConfig) (VideoSource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// VideoSourceを作成
	source, err := m.sourceFactory.CreateSource(sourceType, config)
	if err != nil {
		return nil, fmt.Errorf("VideoSourceの作成に失敗: %w", err)
	}

	// VideoSourceを管理対象に追加
	sourceID := source.GetInfo().ID
	m.videoSources[sourceID] = source

	return source, nil
}

// GetVideoSource は指定されたIDのVideoSourceを取得する
func (m *DefaultCameraManager) GetVideoSource(id string) (VideoSource, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	source, exists := m.videoSources[id]
	return source, exists
}

// GetVideoSources は現在管理されているVideoSource一覧を取得する
func (m *DefaultCameraManager) GetVideoSources() []VideoSource {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sources := make([]VideoSource, 0, len(m.videoSources))
	for _, source := range m.videoSources {
		sources = append(sources, source)
	}

	return sources
}

// RemoveVideoSource はVideoSourceを削除する
func (m *DefaultCameraManager) RemoveVideoSource(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	source, exists := m.videoSources[id]
	if !exists {
		return fmt.Errorf("VideoSourceが見つかりません: %s", id)
	}

	// VideoSourceが動作中の場合は停止
	if source.GetStatus() == StatusActive {
		if err := source.Stop(ctx); err != nil {
			return fmt.Errorf("VideoSourceの停止に失敗: %w", err)
		}
	}

	// 管理対象から削除
	delete(m.videoSources, id)

	return nil
}
