package camera

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// LinuxDiscovery はLinux環境でのカメラデバイス検出を実装する
type LinuxDiscovery struct{}

// NewLinuxDiscovery は新しいLinuxDiscoveryを作成する
func NewLinuxDiscovery() Discovery {
	return &LinuxDiscovery{}
}

// ScanDevices はシステム内の利用可能なカメラデバイスをスキャンする
func (d *LinuxDiscovery) ScanDevices(ctx context.Context) ([]string, error) {
	var devices []string

	// /dev/video* パターンでデバイスを検索
	pattern := "/dev/video*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("デバイスのスキャンに失敗: %w", err)
	}

	// デバイス番号でソート
	sort.Slice(matches, func(i, j int) bool {
		numI := extractDeviceNumber(matches[i])
		numJ := extractDeviceNumber(matches[j])
		return numI < numJ
	})

	for _, match := range matches {
		// コンテキストのキャンセルをチェック
		select {
		case <-ctx.Done():
			return devices, ctx.Err()
		default:
		}

		// デバイスが実際に利用可能かチェック
		if d.IsDeviceAvailable(ctx, match) {
			devices = append(devices, match)
		}
	}

	return devices, nil
}

// IsDeviceAvailable は指定されたデバイスが利用可能かチェックする
func (d *LinuxDiscovery) IsDeviceAvailable(_ context.Context, device string) bool {
	// デバイスファイルの存在確認
	if _, err := os.Stat(device); os.IsNotExist(err) {
		return false
	}

	// デバイスファイルの読み取り権限チェック
	file, err := os.OpenFile(device, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer func() {
		_ = file.Close()
	}()

	// V4L2デバイスかどうかの簡易チェック
	// 実際のプロダクション環境では、より詳細なチェックが必要
	return d.isV4L2Device(device)
}

// GetDeviceInfo はデバイスの詳細情報を取得する
func (d *LinuxDiscovery) GetDeviceInfo(ctx context.Context, device string) (*DeviceInfo, error) {
	if !d.IsDeviceAvailable(ctx, device) {
		return nil, fmt.Errorf("デバイスが利用できません: %s", device)
	}

	info := &DeviceInfo{
		Device: device,
		Name:   d.generateDeviceName(device),
		Driver: "uvcvideo", // 仮の値、実際にはV4L2 APIから取得
		Resolutions: []Resolution{
			{Width: 640, Height: 480},
			{Width: 1280, Height: 720},
			{Width: 1920, Height: 1080},
		},
		Formats: []string{"MJPEG", "YUYV"},
	}

	return info, nil
}

// isV4L2Device はデバイスがV4L2デバイスかチェックする
// 簡易実装：実際にはV4L2のioctl呼び出しで確認する
func (d *LinuxDiscovery) isV4L2Device(device string) bool {
	// /dev/videoXX パターンかチェック
	matched, _ := regexp.MatchString(`^/dev/video\d+$`, device)
	return matched
}

// generateDeviceName はデバイスパスから表示名を生成する
func (d *LinuxDiscovery) generateDeviceName(device string) string {
	num := extractDeviceNumber(device)
	return fmt.Sprintf("カメラ %d", num)
}

// extractDeviceNumber はデバイスパスから番号を抽出する
func extractDeviceNumber(device string) int {
	// /dev/videoXX から XX を抽出
	re := regexp.MustCompile(`video(\d+)`)
	matches := re.FindStringSubmatch(device)
	if len(matches) < 2 {
		return 0
	}

	num, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}

	return num
}

// MockDiscovery はテスト用のモックDiscovery実装
type MockDiscovery struct {
	devices     []string
	deviceInfos map[string]*DeviceInfo
}

// NewMockDiscovery は新しいMockDiscoveryを作成する
func NewMockDiscovery(devices []string) *MockDiscovery {
	deviceInfos := make(map[string]*DeviceInfo)
	for i, device := range devices {
		deviceInfos[device] = &DeviceInfo{
			Device: device,
			Name:   fmt.Sprintf("テストカメラ %d", i+1),
			Driver: "mock",
			Resolutions: []Resolution{
				{Width: 640, Height: 480},
				{Width: 1280, Height: 720},
			},
			Formats: []string{"MJPEG"},
		}
	}

	return &MockDiscovery{
		devices:     devices,
		deviceInfos: deviceInfos,
	}
}

// ScanDevices はモックデバイス一覧を返す
func (m *MockDiscovery) ScanDevices(_ context.Context) ([]string, error) {
	return m.devices, nil
}

// IsDeviceAvailable はモックデバイスが利用可能かチェックする
func (m *MockDiscovery) IsDeviceAvailable(_ context.Context, device string) bool {
	for _, d := range m.devices {
		if d == device {
			return true
		}
	}
	return false
}

// GetDeviceInfo はモックデバイス情報を取得する
func (m *MockDiscovery) GetDeviceInfo(_ context.Context, device string) (*DeviceInfo, error) {
	info, exists := m.deviceInfos[device]
	if !exists {
		return nil, fmt.Errorf("デバイスが見つかりません: %s", device)
	}

	// コピーを返す
	result := *info
	return &result, nil
}

// AddDevice はテスト用にデバイスを追加する
func (m *MockDiscovery) AddDevice(device string) {
	// 重複チェック
	for _, d := range m.devices {
		if d == device {
			return
		}
	}

	m.devices = append(m.devices, device)
	deviceNum := len(m.devices)
	m.deviceInfos[device] = &DeviceInfo{
		Device: device,
		Name:   fmt.Sprintf("テストカメラ %d", deviceNum),
		Driver: "mock",
		Resolutions: []Resolution{
			{Width: 640, Height: 480},
			{Width: 1280, Height: 720},
		},
		Formats: []string{"MJPEG"},
	}
}

// RemoveDevice はテスト用にデバイスを削除する
func (m *MockDiscovery) RemoveDevice(device string) {
	// デバイス一覧から削除
	for i, d := range m.devices {
		if d == device {
			m.devices = append(m.devices[:i], m.devices[i+1:]...)
			break
		}
	}

	// デバイス情報も削除
	delete(m.deviceInfos, device)
}
