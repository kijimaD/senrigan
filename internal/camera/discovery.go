package camera

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
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
			// カメラの種類を判定して、メインカメラのみを追加
			if d.IsMainCamera(ctx, match) {
				devices = append(devices, match)
			}
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
	// v4l2-ctlを使って実際のカメラ名を取得
	if realName := d.getV4L2DeviceName(device); realName != "" {
		return realName
	}

	// フォールバック: デバイス番号から生成
	num := extractDeviceNumber(device)
	return fmt.Sprintf("カメラ %d", num)
}

// getV4L2DeviceName はv4l2-ctlを使って実際のデバイス名を取得する
func (d *LinuxDiscovery) getV4L2DeviceName(device string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "v4l2-ctl", "--device", device, "--info")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// "Card type" の行からカメラ名を抽出
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Card type") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cardType := strings.TrimSpace(parts[1])
				if cardType != "" {
					return cardType
				}
			}
		}
	}

	return ""
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

// IsMainCamera はデバイスがメインカメラ（カラー）かどうかを判定する
func (d *LinuxDiscovery) IsMainCamera(ctx context.Context, device string) bool {
	// v4l2-ctlでサポートフォーマットを取得
	cmd := exec.CommandContext(ctx, "v4l2-ctl", "--device", device, "--list-formats-ext")
	output, err := cmd.Output()
	if err != nil {
		// コマンドが失敗した場合は除外
		return false
	}

	outputStr := string(output)
	
	// グレースケールのみのデバイスは除外
	if strings.Contains(outputStr, "GREY") && !strings.Contains(outputStr, "YUYV") && !strings.Contains(outputStr, "MJPG") {
		return false
	}
	
	// カラーフォーマットをサポートしているかチェック
	hasColor := strings.Contains(outputStr, "YUYV") || strings.Contains(outputStr, "MJPG")
	
	// 同じ物理デバイスの複数チャンネルの場合、最も小さい番号を選択
	if hasColor {
		deviceNum := extractDeviceNumber(device)
		
		// 同じカメラの他のチャンネルをチェック
		// 例: video0, video1が同じカメラの場合、video0を選択
		for i := 0; i < deviceNum; i++ {
			siblingDevice := fmt.Sprintf("/dev/video%d", i)
			if d.IsDeviceAvailable(ctx, siblingDevice) {
				// より小さい番号のデバイスがカラーをサポートしている場合は現在のデバイスをスキップ
				siblingCmd := exec.CommandContext(ctx, "v4l2-ctl", "--device", siblingDevice, "--list-formats-ext")
				if siblingOutput, err := siblingCmd.Output(); err == nil {
					siblingStr := string(siblingOutput)
					if strings.Contains(siblingStr, "YUYV") || strings.Contains(siblingStr, "MJPG") {
						// 同じカメラ名かチェック
						if d.haveSameCameraName(ctx, device, siblingDevice) {
							return false // より小さい番号のデバイスを優先
						}
					}
				}
			}
		}
		
		return true
	}
	
	return false
}

// haveSameCameraName は2つのデバイスが同じカメラかチェック
func (d *LinuxDiscovery) haveSameCameraName(ctx context.Context, device1, device2 string) bool {
	name1 := d.getV4L2DeviceName(device1)
	name2 := d.getV4L2DeviceName(device2)
	
	if name1 == "" || name2 == "" {
		return false
	}
	
	return name1 == name2
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
