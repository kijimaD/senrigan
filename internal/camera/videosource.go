package camera

import (
	"context"
	"sync"
)

// VideoSourceType はソースタイプを定義
type VideoSourceType string

const (
	// SourceTypeUSBCamera はUSBカメラソースを表す
	SourceTypeUSBCamera VideoSourceType = "usb_camera"
	// SourceTypeX11Screen はX11画面キャプチャソースを表す
	SourceTypeX11Screen VideoSourceType = "x11_screen"
)

// VideoSource は全ての動画源を統一するインターフェース
type VideoSource interface {
	// 基本操作
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsAvailable(ctx context.Context) bool

	// ストリーミング
	GetFrameChannel() <-chan []byte
	GetErrorChannel() <-chan error

	// タイムラプス用フレーム取得
	CaptureFrameForTimelapse(ctx context.Context) ([]byte, error)

	// メタデータ
	GetInfo() VideoSourceInfo
	GetCapabilities() VideoCapabilities

	// 設定
	ApplySettings(ctx context.Context, settings VideoSettings) error
	GetCurrentSettings() VideoSettings

	// ステータス取得
	GetStatus() Status
}

// VideoSourceInfo はソース情報を表す
type VideoSourceInfo struct {
	ID          string
	Name        string
	Type        VideoSourceType
	Driver      string
	Description string
	Device      string // デバイスパス（USBカメラ等）
}

// VideoCapabilities はソースの能力を表す
type VideoCapabilities struct {
	SupportedResolutions []Resolution
	SupportedFrameRates  []int
	SupportedFormats     []string
}

// VideoSettings は動画設定を統一
type VideoSettings struct {
	Width      int
	Height     int
	FrameRate  int
	Format     string
	Quality    int
	Properties map[string]interface{}
}

// BaseVideoSource は共通実装を提供
type BaseVideoSource struct {
	info         VideoSourceInfo
	capabilities VideoCapabilities
	settings     VideoSettings
	frameChan    chan []byte
	errorChan    chan error
	status       Status
	mu           sync.RWMutex
}

// GetInfo は基本情報を返す
func (b *BaseVideoSource) GetInfo() VideoSourceInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.info
}

// GetCapabilities は機能情報を返す
func (b *BaseVideoSource) GetCapabilities() VideoCapabilities {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.capabilities
}

// GetCurrentSettings は現在の設定を返す
func (b *BaseVideoSource) GetCurrentSettings() VideoSettings {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.settings
}

// GetStatus はステータスを返す
func (b *BaseVideoSource) GetStatus() Status {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.status
}

// GetFrameChannel はフレームチャンネルを返す
func (b *BaseVideoSource) GetFrameChannel() <-chan []byte {
	return b.frameChan
}

// GetErrorChannel はエラーチャンネルを返す
func (b *BaseVideoSource) GetErrorChannel() <-chan error {
	return b.errorChan
}
