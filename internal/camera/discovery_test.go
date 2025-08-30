package camera

import (
	"context"
	"testing"
)

func TestLinuxDiscovery_ScanDevices(t *testing.T) {
	ctx := context.Background()
	discovery := NewLinuxDiscovery()

	devices, err := discovery.ScanDevices(ctx)
	if err != nil {
		t.Fatalf("ScanDevices failed: %v", err)
	}

	// デバイスが見つからない場合もあるため、エラーがないことを確認
	t.Logf("Found %d video devices", len(devices))
	for _, device := range devices {
		t.Logf("Device: %s", device)
	}
}

func TestLinuxDiscovery_IsDeviceAvailable(t *testing.T) {
	ctx := context.Background()
	discovery := NewLinuxDiscovery()

	// 存在しないデバイスをテスト
	if discovery.IsDeviceAvailable(ctx, "/dev/video999") {
		t.Error("Expected non-existent device to be unavailable")
	}

	// 無効なパスをテスト
	if discovery.IsDeviceAvailable(ctx, "/invalid/path") {
		t.Error("Expected invalid path to be unavailable")
	}
}

func TestMockDiscovery(t *testing.T) {
	ctx := context.Background()
	mockDevices := []string{"/dev/video0", "/dev/video1"}
	discovery := NewMockDiscovery(mockDevices)

	// ScanDevicesのテスト
	devices, err := discovery.ScanDevices(ctx)
	if err != nil {
		t.Fatalf("ScanDevices failed: %v", err)
	}

	if len(devices) != len(mockDevices) {
		t.Fatalf("Expected %d devices, got %d", len(mockDevices), len(devices))
	}

	for i, device := range devices {
		if device != mockDevices[i] {
			t.Errorf("Expected device %s, got %s", mockDevices[i], device)
		}
	}

	// IsDeviceAvailableのテスト
	if !discovery.IsDeviceAvailable(ctx, "/dev/video0") {
		t.Error("Expected /dev/video0 to be available")
	}

	if discovery.IsDeviceAvailable(ctx, "/dev/video2") {
		t.Error("Expected /dev/video2 to be unavailable")
	}

	// GetDeviceInfoのテスト
	info, err := discovery.GetDeviceInfo(ctx, "/dev/video0")
	if err != nil {
		t.Fatalf("GetDeviceInfo failed: %v", err)
	}

	if info.Device != "/dev/video0" {
		t.Errorf("Expected device /dev/video0, got %s", info.Device)
	}

	if info.Name == "" {
		t.Error("Expected device name to be set")
	}

	// 存在しないデバイスの情報取得
	_, err = discovery.GetDeviceInfo(ctx, "/dev/video99")
	if err == nil {
		t.Error("Expected error for non-existent device")
	}
}

func TestMockDiscovery_AddRemoveDevice(t *testing.T) {
	ctx := context.Background()
	discovery := NewMockDiscovery([]string{"/dev/video0"})

	// デバイス追加
	discovery.AddDevice("/dev/video1")

	devices, err := discovery.ScanDevices(ctx)
	if err != nil {
		t.Fatalf("ScanDevices failed: %v", err)
	}

	if len(devices) != 2 {
		t.Fatalf("Expected 2 devices after addition, got %d", len(devices))
	}

	if !discovery.IsDeviceAvailable(ctx, "/dev/video1") {
		t.Error("Expected /dev/video1 to be available after addition")
	}

	// デバイス削除
	discovery.RemoveDevice("/dev/video0")

	devices, err = discovery.ScanDevices(ctx)
	if err != nil {
		t.Fatalf("ScanDevices failed: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("Expected 1 device after removal, got %d", len(devices))
	}

	if discovery.IsDeviceAvailable(ctx, "/dev/video0") {
		t.Error("Expected /dev/video0 to be unavailable after removal")
	}

	// 重複追加のテスト
	discovery.AddDevice("/dev/video1") // 既に存在
	devices, err = discovery.ScanDevices(ctx)
	if err != nil {
		t.Fatalf("ScanDevices failed: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("Expected 1 device after duplicate addition, got %d", len(devices))
	}
}
