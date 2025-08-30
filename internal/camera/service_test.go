package camera

import (
	"context"
	"testing"
)

func TestCameraService_Basic(t *testing.T) {
	camera := &Camera{
		ID:     "test-camera-1",
		Name:   "Test Camera",
		Device: "/dev/video0",
		FPS:    30,
		Width:  1920,
		Height: 1080,
		Status: StatusInactive,
	}

	service := NewCameraService(camera)

	// 初期状態
	if service.GetStatus() != StatusInactive {
		t.Errorf("Expected initial status to be inactive, got %s", service.GetStatus())
	}

	settings := service.GetSettings()
	if settings.FPS != 30 {
		t.Errorf("Expected FPS 30, got %d", settings.FPS)
	}
}

func TestCameraService_StartStop(t *testing.T) {
	ctx := context.Background()
	camera := &Camera{
		ID:     "test-camera-1",
		Name:   "Test Camera",
		Device: "/dev/video0", // 実際には存在しない可能性があるがテスト用
		FPS:    30,
		Width:  1920,
		Height: 1080,
		Status: StatusInactive,
	}

	service := NewCameraService(camera)

	// 開始（存在しないデバイスなのでエラーになる可能性があるが、それで良い）
	err := service.Start(ctx)
	// エラーは無視してもOK（実際のデバイスがない環境でのテスト）

	// 停止
	err = service.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if service.GetStatus() != StatusInactive {
		t.Errorf("Expected status to be inactive after stop, got %s", service.GetStatus())
	}
}

func TestCameraService_UpdateSettings(t *testing.T) {
	ctx := context.Background()
	camera := &Camera{
		ID:     "test-camera-1",
		Name:   "Test Camera",
		Device: "/dev/video0",
		FPS:    30,
		Width:  1920,
		Height: 1080,
		Status: StatusInactive,
	}

	service := NewCameraService(camera)

	// 設定更新
	newSettings := Settings{FPS: 15, Width: 640, Height: 480}
	err := service.UpdateSettings(ctx, newSettings)
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// 確認
	settings := service.GetSettings()
	if settings.FPS != 15 {
		t.Errorf("Expected FPS 15, got %d", settings.FPS)
	}
	if settings.Width != 640 {
		t.Errorf("Expected width 640, got %d", settings.Width)
	}
	if settings.Height != 480 {
		t.Errorf("Expected height 480, got %d", settings.Height)
	}

	// カメラオブジェクトも更新されているか確認
	if camera.FPS != 15 {
		t.Errorf("Expected camera FPS 15, got %d", camera.FPS)
	}
}

func TestCameraService_InvalidSettings(t *testing.T) {
	ctx := context.Background()
	camera := &Camera{
		ID:     "test-camera-1",
		Name:   "Test Camera",
		Device: "/dev/video0",
		FPS:    30,
		Width:  1920,
		Height: 1080,
		Status: StatusInactive,
	}

	service := NewCameraService(camera)

	// 無効なFPS
	err := service.UpdateSettings(ctx, Settings{FPS: 0, Width: 640, Height: 480})
	if err == nil {
		t.Error("Expected error for invalid FPS")
	}

	err = service.UpdateSettings(ctx, Settings{FPS: 100, Width: 640, Height: 480})
	if err == nil {
		t.Error("Expected error for too high FPS")
	}

	// 無効な解像度
	err = service.UpdateSettings(ctx, Settings{FPS: 30, Width: 0, Height: 480})
	if err == nil {
		t.Error("Expected error for invalid width")
	}

	err = service.UpdateSettings(ctx, Settings{FPS: 30, Width: 640, Height: 0})
	if err == nil {
		t.Error("Expected error for invalid height")
	}

	err = service.UpdateSettings(ctx, Settings{FPS: 30, Width: 5000, Height: 480})
	if err == nil {
		t.Error("Expected error for too large width")
	}

	err = service.UpdateSettings(ctx, Settings{FPS: 30, Width: 640, Height: 5000})
	if err == nil {
		t.Error("Expected error for too large height")
	}
}

func TestMockCameraService(t *testing.T) {
	ctx := context.Background()
	camera := &Camera{
		ID:     "test-camera-1",
		Name:   "Test Camera",
		Device: "/dev/video0",
		FPS:    30,
		Width:  1920,
		Height: 1080,
		Status: StatusInactive,
	}

	service := NewMockCameraService(camera)

	// 正常開始
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if service.GetStatus() != StatusActive {
		t.Errorf("Expected status to be active, got %s", service.GetStatus())
	}

	if camera.Status != StatusActive {
		t.Errorf("Expected camera status to be active, got %s", camera.Status)
	}

	// 正常停止
	err = service.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if service.GetStatus() != StatusInactive {
		t.Errorf("Expected status to be inactive, got %s", service.GetStatus())
	}

	// 設定更新
	newSettings := Settings{FPS: 15, Width: 640, Height: 480}
	err = service.UpdateSettings(ctx, newSettings)
	if err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	settings := service.GetSettings()
	if settings.FPS != 15 {
		t.Errorf("Expected FPS 15, got %d", settings.FPS)
	}
}

func TestMockCameraService_Failures(t *testing.T) {
	ctx := context.Background()
	camera := &Camera{
		ID:     "test-camera-1",
		Name:   "Test Camera",
		Device: "/dev/video0",
		FPS:    30,
		Width:  1920,
		Height: 1080,
		Status: StatusInactive,
	}

	service := NewMockCameraService(camera)

	// 開始失敗を設定
	service.SetShouldFailStart(true)
	err := service.Start(ctx)
	if err == nil {
		t.Error("Expected start to fail")
	}

	if service.GetStatus() != StatusError {
		t.Errorf("Expected status to be error, got %s", service.GetStatus())
	}

	// 開始失敗を無効にして成功させる
	service.SetShouldFailStart(false)
	err = service.Start(ctx)
	if err != nil {
		t.Fatalf("Start should succeed now: %v", err)
	}

	// 停止失敗を設定
	service.SetShouldFailStop(true)
	err = service.Stop(ctx)
	if err == nil {
		t.Error("Expected stop to fail")
	}

	// 停止失敗を無効にして成功させる
	service.SetShouldFailStop(false)
	err = service.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop should succeed now: %v", err)
	}
}

func TestCameraService_DoubleStart(t *testing.T) {
	ctx := context.Background()
	camera := &Camera{
		ID:     "test-camera-1",
		Name:   "Test Camera",
		Device: "/dev/video0",
		FPS:    30,
		Width:  1920,
		Height: 1080,
		Status: StatusInactive,
	}

	service := NewMockCameraService(camera)

	// 最初の開始
	err := service.Start(ctx)
	if err != nil {
		t.Fatalf("First start failed: %v", err)
	}

	// 二回目の開始（エラーになるべき）
	err = service.Start(ctx)
	if err == nil {
		t.Error("Expected error for double start")
	}

	// 停止
	err = service.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// 停止後の再開始（成功するべき）
	err = service.Start(ctx)
	if err != nil {
		t.Fatalf("Restart failed: %v", err)
	}

	// クリーンアップ
	service.Stop(ctx)
}

func TestCameraService_StopInactive(t *testing.T) {
	ctx := context.Background()
	camera := &Camera{
		ID:     "test-camera-1",
		Name:   "Test Camera",
		Device: "/dev/video0",
		FPS:    30,
		Width:  1920,
		Height: 1080,
		Status: StatusInactive,
	}

	service := NewMockCameraService(camera)

	// 開始せずに停止（エラーにならないはず）
	err := service.Stop(ctx)
	if err != nil {
		t.Fatalf("Stop of inactive camera failed: %v", err)
	}

	if service.GetStatus() != StatusInactive {
		t.Errorf("Expected status to remain inactive, got %s", service.GetStatus())
	}
}
