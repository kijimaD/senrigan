package camera

import (
	"context"
	"testing"
	"time"
)

func TestDefaultCameraManager_Basic(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{"/dev/video0", "/dev/video1"})
	defaultSettings := Settings{FPS: 30, Width: 1920, Height: 1080}

	manager := NewDefaultCameraManager(mockDiscovery, defaultSettings)

	// Start
	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// 自動検出されたカメラを確認
	cameras := manager.GetCameras()
	if len(cameras) != 2 {
		t.Fatalf("Expected 2 cameras, got %d", len(cameras))
	}

	// カメラの詳細確認
	for _, cam := range cameras {
		if cam.Status != StatusInactive {
			t.Errorf("Expected camera %s to be inactive, got %s", cam.ID, cam.Status)
		}

		if cam.FPS != defaultSettings.FPS {
			t.Errorf("Expected camera %s FPS to be %d, got %d", cam.ID, defaultSettings.FPS, cam.FPS)
		}
	}

	// Stop
	if err := manager.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestDefaultCameraManager_AddRemoveCamera(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{})
	defaultSettings := Settings{FPS: 15, Width: 640, Height: 480}

	manager := NewDefaultCameraManager(mockDiscovery, defaultSettings)

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 初期状態では0台
	cameras := manager.GetCameras()
	if len(cameras) != 0 {
		t.Fatalf("Expected 0 cameras initially, got %d", len(cameras))
	}

	// カメラを追加
	mockDiscovery.AddDevice("/dev/video0")
	camera, err := manager.AddCamera(ctx, "/dev/video0", Settings{FPS: 30, Width: 1280, Height: 720})
	if err != nil {
		t.Fatalf("AddCamera failed: %v", err)
	}

	if camera.ID == "" {
		t.Error("Expected camera ID to be set")
	}

	if camera.Device != "/dev/video0" {
		t.Errorf("Expected device /dev/video0, got %s", camera.Device)
	}

	// カメラ一覧を確認
	cameras = manager.GetCameras()
	if len(cameras) != 1 {
		t.Fatalf("Expected 1 camera after addition, got %d", len(cameras))
	}

	// 個別取得
	retrievedCamera, found := manager.GetCamera(camera.ID)
	if !found {
		t.Fatal("Camera not found by ID")
	}

	if retrievedCamera.Device != camera.Device {
		t.Errorf("Retrieved camera device mismatch: expected %s, got %s", camera.Device, retrievedCamera.Device)
	}

	// カメラを削除
	if err := manager.RemoveCamera(ctx, camera.ID); err != nil {
		t.Fatalf("RemoveCamera failed: %v", err)
	}

	// 削除確認
	cameras = manager.GetCameras()
	if len(cameras) != 0 {
		t.Fatalf("Expected 0 cameras after removal, got %d", len(cameras))
	}

	_, found = manager.GetCamera(camera.ID)
	if found {
		t.Error("Camera should not be found after removal")
	}
}

func TestDefaultCameraManager_StartStopCamera(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{"/dev/video0"})
	defaultSettings := Settings{FPS: 15, Width: 640, Height: 480}

	manager := NewDefaultCameraManager(mockDiscovery, defaultSettings)

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	cameras := manager.GetCameras()
	if len(cameras) != 1 {
		t.Fatalf("Expected 1 camera, got %d", len(cameras))
	}

	cameraID := cameras[0].ID

	// カメラを開始
	if err := manager.StartCamera(ctx, cameraID); err != nil {
		t.Fatalf("StartCamera failed: %v", err)
	}

	// 状態確認
	camera, found := manager.GetCamera(cameraID)
	if !found {
		t.Fatal("Camera not found after start")
	}

	if camera.Status != StatusActive {
		t.Errorf("Expected camera to be active, got %s", camera.Status)
	}

	// カメラを停止
	if err := manager.StopCamera(ctx, cameraID); err != nil {
		t.Fatalf("StopCamera failed: %v", err)
	}

	// 状態確認
	camera, found = manager.GetCamera(cameraID)
	if !found {
		t.Fatal("Camera not found after stop")
	}

	if camera.Status != StatusInactive {
		t.Errorf("Expected camera to be inactive, got %s", camera.Status)
	}
}

func TestDefaultCameraManager_DiscoverCameras(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{"/dev/video0"})
	defaultSettings := Settings{FPS: 15, Width: 640, Height: 480}

	manager := NewDefaultCameraManager(mockDiscovery, defaultSettings)

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 初期状態で1台
	cameras := manager.GetCameras()
	if len(cameras) != 1 {
		t.Fatalf("Expected 1 camera initially, got %d", len(cameras))
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

	// カメラが自動追加されているか確認
	cameras = manager.GetCameras()
	if len(cameras) != 2 {
		t.Fatalf("Expected 2 cameras after discovery, got %d", len(cameras))
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

	// カメラが自動削除されているか確認
	cameras = manager.GetCameras()
	if len(cameras) != 1 {
		t.Fatalf("Expected 1 camera after device removal, got %d", len(cameras))
	}
}

func TestDefaultCameraManager_ErrorCases(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{})
	defaultSettings := Settings{FPS: 15, Width: 640, Height: 480}

	manager := NewDefaultCameraManager(mockDiscovery, defaultSettings)

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 存在しないデバイスを追加
	_, err := manager.AddCamera(ctx, "/dev/video99", defaultSettings)
	if err == nil {
		t.Error("Expected error for non-existent device")
	}

	// 存在しないカメラを操作
	err = manager.StartCamera(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent camera")
	}

	err = manager.StopCamera(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent camera")
	}

	err = manager.RemoveCamera(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent camera")
	}

	// 重複デバイス追加
	mockDiscovery.AddDevice("/dev/video0")
	_, err = manager.AddCamera(ctx, "/dev/video0", defaultSettings)
	if err != nil {
		t.Fatalf("First AddCamera failed: %v", err)
	}

	_, err = manager.AddCamera(ctx, "/dev/video0", defaultSettings)
	if err == nil {
		t.Error("Expected error for duplicate device")
	}
}

func TestDefaultCameraManager_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	mockDiscovery := NewMockDiscovery([]string{"/dev/video0", "/dev/video1"})
	defaultSettings := Settings{FPS: 15, Width: 640, Height: 480}

	manager := NewDefaultCameraManager(mockDiscovery, defaultSettings)

	if err := manager.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = manager.Stop(ctx) }()

	// 複数のゴルーチンで同時アクセス
	done := make(chan bool, 2)

	go func() {
		defer func() { done <- true }()
		for i := 0; i < 10; i++ {
			manager.GetCameras()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	go func() {
		defer func() { done <- true }()
		cameras := manager.GetCameras()
		for _, camera := range cameras {
			manager.GetCamera(camera.ID)
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// 完了を待つ
	<-done
	<-done

	// 最終状態確認
	cameras := manager.GetCameras()
	if len(cameras) != 2 {
		t.Fatalf("Expected 2 cameras after concurrent access, got %d", len(cameras))
	}
}
