package camera

import (
	"context"
)

// Status はカメラの動作状態を表す
type Status string

// カメラの状態定数
const (
	StatusInactive Status = "inactive" // カメラは停止中
	StatusActive   Status = "active"   // カメラは動作中
	StatusError    Status = "error"    // カメラでエラーが発生
)

// Manager はカメラの動的管理を担うインターフェース
type Manager interface {
	// Start はカメラマネージャーを開始する
	Start(ctx context.Context) error

	// Stop はカメラマネージャーを停止する
	Stop(ctx context.Context) error

	// DiscoverCameras はシステム内のカメラデバイスを再検出する
	DiscoverCameras(ctx context.Context) ([]string, error)

	// VideoSource関連のメソッド
	// AddVideoSource はVideoSourceを追加する
	AddVideoSource(ctx context.Context, sourceType VideoSourceType, config SourceConfig) (VideoSource, error)

	// GetVideoSource は指定されたIDのVideoSourceを取得する
	GetVideoSource(id string) (VideoSource, bool)

	// GetVideoSources は現在管理されているVideoSource一覧を取得する
	GetVideoSources() []VideoSource

	// RemoveVideoSource はVideoSourceを削除する
	RemoveVideoSource(ctx context.Context, id string) error
}

// Discovery はカメラデバイスの検出機能を提供する
type Discovery interface {
	// ScanDevices はシステム内の利用可能なカメラデバイスをスキャンする
	ScanDevices(ctx context.Context) ([]string, error)

	// IsDeviceAvailable は指定されたデバイスが利用可能かチェックする
	IsDeviceAvailable(ctx context.Context, device string) bool

	// GetDeviceInfo はデバイスの詳細情報を取得する
	GetDeviceInfo(ctx context.Context, device string) (*DeviceInfo, error)
}

// DeviceInfo はカメラデバイスの詳細情報を表す
type DeviceInfo struct {
	Device      string       // デバイスパス
	Name        string       // デバイス名
	Driver      string       // ドライバー名
	Resolutions []Resolution // サポートされる解像度
	Formats     []string     // サポートされるフォーマット
}

// Resolution はカメラの解像度を表す
type Resolution struct {
	Width  int // 幅
	Height int // 高さ
}
