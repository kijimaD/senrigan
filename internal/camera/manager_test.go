package camera

import (
	"context"
	"testing"
	"time"
)

func TestDefaultCameraManager_Basic(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{"/dev/video0", "/dev/video1"})

	manager := NewDefaultCameraManager(mockDiscovery)

	// Start
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// 自動検出されたVideoSourceを確認（USBカメラ2台 + X11画面録画1台）
	sources := manager.GetVideoSources()
	if len(sources) != 3 {
		t.Fatalf("Expected 3 video sources (2 USB + 1 X11), got %d", len(sources))
	}

	// VideoSourceの詳細確認
	for _, source := range sources {
		// X11は必ずアクティブになるはず
		if source.GetInfo().Type == SourceTypeX11Screen {
			if source.GetStatus() != StatusActive {
				t.Errorf("Expected X11 video source %s to be active, got %s",
					source.GetInfo().ID, source.GetStatus())
			}
		}
		// USBカメラはテスト環境では起動に失敗する可能性がある（実デバイスがないため）

		settings := source.GetCurrentSettings()
		if settings.FrameRate != 15 { // デフォルト値（X11もUSBカメラも15fps）
			t.Errorf("Expected video source %s FrameRate to be 15, got %d",
				source.GetInfo().ID, settings.FrameRate)
		}
	}

	// Stop
	if err := manager.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDefaultCameraManager_AddRemoveVideoSource(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{})

	manager := NewDefaultCameraManager(mockDiscovery)

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 初期状態ではX11の1台
	sources := manager.GetVideoSources()
	if len(sources) != 1 {
		t.Fatalf("Expected 1 video source initially (X11), got %d", len(sources))
	}

	// VideoSourceを追加
	mockDiscovery.AddDevice("/dev/video0")
	settings := VideoSettings{
		Width:      1280,
		Height:     720,
		FrameRate:  30,
		Format:     "MJPEG",
		Quality:    3,
		Properties: make(map[string]interface{}),
	}
	config := SourceConfig{
		Device:   "/dev/video0",
		Settings: settings,
	}

	source, err := manager.AddVideoSource(ctx, SourceTypeUSBCamera, config)
	if err != nil {
		t.Fatalf("AddVideoSource failed: %v", err)
	}

	info := source.GetInfo()
	if info.ID == "" {
		t.Error("Expected video source ID to be set")
	}

	if info.Device != "/dev/video0" {
		t.Errorf("Expected device /dev/video0, got %s", info.Device)
	}

	// VideoSource一覧を確認（X11 + 追加したUSB）
	sources = manager.GetVideoSources()
	if len(sources) != 2 {
		t.Fatalf("Expected 2 video sources after addition (X11 + USB), got %d", len(sources))
	}

	// 個別取得
	retrievedSource, found := manager.GetVideoSource(info.ID)
	if !found {
		t.Fatal("VideoSource not found by ID")
	}

	if retrievedSource.GetInfo().Device != info.Device {
		t.Errorf("Retrieved video source device mismatch: expected %s, got %s",
			info.Device, retrievedSource.GetInfo().Device)
	}

	// VideoSourceを削除
	if err := manager.RemoveVideoSource(ctx, info.ID); err != nil {
		t.Fatalf("RemoveVideoSource failed: %v", err)
	}

	// 削除確認（X11は残る）
	sources = manager.GetVideoSources()
	if len(sources) != 1 {
		t.Fatalf("Expected 1 video source after removal (X11), got %d", len(sources))
	}

	_, found = manager.GetVideoSource(info.ID)
	if found {
		t.Error("VideoSource should not be found after removal")
	}
}

func TestDefaultCameraManager_DiscoverCameras(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{"/dev/video0"})

	manager := NewDefaultCameraManager(mockDiscovery)

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 初期状態で2台（USB1台 + X11）
	sources := manager.GetVideoSources()
	if len(sources) != 2 {
		t.Fatalf("Expected 2 video sources initially (1 USB + X11), got %d", len(sources))
	}

	// デバイスを追加
	mockDiscovery.AddDevice("/dev/video1")

	// 再検出実行
	devices, err := manager.DiscoverCameras(ctx)
	if err != nil {
		t.Fatalf("DiscoverCameras failed: %v", err)
	}

	if len(devices) != 2 {
		t.Fatalf("Expected 2 devices, got %d", len(devices))
	}

	// VideoSourceが自動追加されているか確認（USB2台 + X11）
	sources = manager.GetVideoSources()
	if len(sources) != 3 {
		t.Fatalf("Expected 3 video sources after discovery (2 USB + X11), got %d", len(sources))
	}

	// デバイスを削除
	mockDiscovery.RemoveDevice("/dev/video0")

	// 再検出実行
	devices, err = manager.DiscoverCameras(ctx)
	if err != nil {
		t.Fatalf("DiscoverCameras failed: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("Expected 1 device after removal, got %d", len(devices))
	}

	// VideoSourceが自動削除されているか確認（USB1台 + X11）
	sources = manager.GetVideoSources()
	if len(sources) != 2 {
		t.Fatalf("Expected 2 video sources after device removal (1 USB + X11), got %d", len(sources))
	}
}

func TestDefaultCameraManager_ErrorCases(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{})

	manager := NewDefaultCameraManager(mockDiscovery)

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 存在しないデバイスでVideoSourceを追加
	settings := VideoSettings{
		Width:      1280,
		Height:     720,
		FrameRate:  30,
		Format:     "MJPEG",
		Quality:    3,
		Properties: make(map[string]interface{}),
	}
	config := SourceConfig{
		Device:   "/dev/video99",
		Settings: settings,
	}

	_, err := manager.AddVideoSource(ctx, SourceTypeUSBCamera, config)
	if err == nil {
		t.Error("Expected error for non-existent device")
	}

	// 存在しないVideoSourceを削除
	err = manager.RemoveVideoSource(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent video source")
	}
}

func TestDefaultCameraManager_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{"/dev/video0", "/dev/video1"})

	manager := NewDefaultCameraManager(mockDiscovery)

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 複数のゴルーチンで同時アクセス
	done := make(chan bool, 2)

	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			manager.GetVideoSources()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	go func() {
		defer func() { done <- true }()
		sources := manager.GetVideoSources()
		for _, source := range sources {
			manager.GetVideoSource(source.GetInfo().ID)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// 完了を待つ
	<-done
	<-done

	// 最終状態確認（USB2台 + X11）
	sources := manager.GetVideoSources()
	if len(sources) != 3 {
		t.Fatalf("Expected 3 video sources after concurrent access (2 USB + X11), got %d", len(sources))
	}
}
