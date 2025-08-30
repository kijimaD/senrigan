package camera

import (
	"context"
	"time"
)

// Status はカメラの動作状態を表す
type Status string

const (
	StatusInactive Status = "inactive" // カメラは停止中
	StatusActive   Status = "active"   // カメラは動作中
	StatusError    Status = "error"    // カメラでエラーが発生
)

// Camera は動的に管理されるカメラの情報と制御機能を提供する
type Camera struct {
	ID       string    // カメラの一意識別子
	Name     string    // カメラの表示名
	Device   string    // デバイスパス（例: /dev/video0）
	FPS      int       // フレームレート
	Width    int       // 画像幅
	Height   int       // 画像高さ
	Status   Status    // 現在の状態
	LastSeen time.Time // 最後に確認された時刻
}

// Settings はカメラの設定を表す
type Settings struct {
	FPS    int // フレームレート
	Width  int // 画像幅
	Height int // 画像高さ
}

// Manager はカメラの動的管理を担うインターフェース
type Manager interface {
	// Start はカメラマネージャーを開始する
	Start(ctx context.Context) error

	// Stop はカメラマネージャーを停止する
	Stop(ctx context.Context) error

	// GetCameras は現在管理されているカメラ一覧を取得する
	GetCameras() []Camera

	// GetCamera は指定されたIDのカメラを取得する
	GetCamera(id string) (*Camera, bool)

	// AddCamera はカメラを動的に追加する
	AddCamera(ctx context.Context, device string, settings Settings) (*Camera, error)

	// RemoveCamera はカメラを削除する
	RemoveCamera(ctx context.Context, id string) error

	// StartCamera はカメラを開始する
	StartCamera(ctx context.Context, id string) error

	// StopCamera はカメラを停止する
	StopCamera(ctx context.Context, id string) error

	// DiscoverCameras はシステム内のカメラデバイスを再検出する
	DiscoverCameras(ctx context.Context) ([]string, error)
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

// Service は個別カメラの制御を担うインターフェース
type Service interface {
	// Start はカメラサービスを開始する
	Start(ctx context.Context) error

	// Stop はカメラサービスを停止する
	Stop(ctx context.Context) error

	// GetStatus は現在の状態を取得する
	GetStatus() Status

	// GetSettings は現在の設定を取得する
	GetSettings() Settings

	// UpdateSettings は設定を更新する
	UpdateSettings(ctx context.Context, settings Settings) error
}
