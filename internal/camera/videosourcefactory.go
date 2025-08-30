package camera

import (
	"context"
	"fmt"
	"time"
)

// SourceConfig はソース作成設定
type SourceConfig struct {
	Device     string                 // デバイスパス
	URL        string                 // IP カメラの場合
	Settings   VideoSettings          // 設定
	Properties map[string]interface{} // 追加プロパティ
}

// VideoSourceFactory はソース作成ファクトリー
type VideoSourceFactory interface {
	CreateSource(sourceType VideoSourceType, config SourceConfig) (VideoSource, error)
	GetSupportedTypes() []VideoSourceType
}

// SourceCreator はソース作成関数の型
type SourceCreator func(config SourceConfig) (VideoSource, error)

// DefaultVideoSourceFactory は標準実装
type DefaultVideoSourceFactory struct {
	creators map[VideoSourceType]SourceCreator
}

// NewVideoSourceFactory は新しいファクトリーを作成する
func NewVideoSourceFactory() VideoSourceFactory {
	factory := &DefaultVideoSourceFactory{
		creators: make(map[VideoSourceType]SourceCreator),
	}

	// USBカメラの作成関数を登録
	factory.Register(SourceTypeUSBCamera, NewUSBCameraSourceFromConfig)

	// X11画面キャプチャの作成関数を登録
	factory.Register(SourceTypeX11Screen, NewX11ScreenSourceFromConfig)

	return factory
}

// Register はソース作成関数を登録する
func (f *DefaultVideoSourceFactory) Register(sourceType VideoSourceType, creator SourceCreator) {
	f.creators[sourceType] = creator
}

// CreateSource はソースを作成する
func (f *DefaultVideoSourceFactory) CreateSource(sourceType VideoSourceType, config SourceConfig) (VideoSource, error) {
	creator, exists := f.creators[sourceType]
	if !exists {
		return nil, fmt.Errorf("サポートされていないソースタイプ: %s", sourceType)
	}

	return creator(config)
}

// GetSupportedTypes はサポートされているソースタイプを返す
func (f *DefaultVideoSourceFactory) GetSupportedTypes() []VideoSourceType {
	types := make([]VideoSourceType, 0, len(f.creators))
	for sourceType := range f.creators {
		types = append(types, sourceType)
	}
	return types
}

// NewUSBCameraSourceFromConfig は設定からUSBCameraSourceを作成する
func NewUSBCameraSourceFromConfig(config SourceConfig) (VideoSource, error) {
	if config.Device == "" {
		return nil, fmt.Errorf("USBカメラの作成にはデバイスパスが必要です")
	}

	// デフォルト設定
	width := 1280
	height := 720
	fps := 15

	// 設定が指定されている場合は使用
	if config.Settings.Width > 0 {
		width = config.Settings.Width
	}
	if config.Settings.Height > 0 {
		height = config.Settings.Height
	}
	if config.Settings.FrameRate > 0 {
		fps = config.Settings.FrameRate
	}

	// デバイス名を生成（既存のdiscovery.goの機能を使用）
	discovery := NewLinuxDiscovery()
	deviceInfo, err := discovery.GetDeviceInfo(context.TODO(), config.Device)
	var name string
	if err != nil || deviceInfo == nil {
		name = fmt.Sprintf("USB Camera (%s)", config.Device)
	} else {
		name = deviceInfo.Name
	}

	// VideoSourceInfo を設定
	info := VideoSourceInfo{
		ID:          generateCameraID(),
		Name:        name,
		Type:        SourceTypeUSBCamera,
		Driver:      "v4l2",
		Description: fmt.Sprintf("USB Camera: %s", name),
		Device:      config.Device,
	}

	// VideoCapabilities を設定
	capabilities := VideoCapabilities{
		SupportedResolutions: []Resolution{
			{Width: 640, Height: 480},
			{Width: 1280, Height: 720},
			{Width: 1920, Height: 1080},
		},
		SupportedFrameRates: []int{5, 10, 15, 30},
		SupportedFormats:    []string{"MJPEG", "YUYV"},
	}

	// VideoSettings を設定
	settings := VideoSettings{
		Width:      width,
		Height:     height,
		FrameRate:  fps,
		Format:     "MJPEG",
		Quality:    3,
		Properties: make(map[string]interface{}),
	}

	return NewDirectUSBCameraSource(info, capabilities, settings), nil
}

// generateCameraID はユニークなカメラIDを生成する
func generateCameraID() string {
	// 既存のコードでUUIDを使用しているため同じ方式を使用
	// 簡単のため現在時刻ベースのIDを生成
	return fmt.Sprintf("camera_%d", time.Now().UnixNano())
}
