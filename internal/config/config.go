package config

import (
	"fmt"
	"os"
	"time"
)

// Config はアプリケーション全体の設定を保持する構造体
type Config struct {
	Server ServerConfig `yaml:"server"`
	Camera CameraConfig `yaml:"camera"`
}

// ServerConfig はHTTPサーバーの設定
type ServerConfig struct {
	Host string `yaml:"host"` // リッスンするホスト
	Port int    `yaml:"port"` // リッスンするポート番号

	// タイムアウト設定
	ReadTimeout  time.Duration `yaml:"read_timeout"`  // 読み込みタイムアウト
	WriteTimeout time.Duration `yaml:"write_timeout"` // 書き込みタイムアウト
}

// CameraConfig はカメラ関連の設定
type CameraConfig struct {
	// 複数カメラ対応のための設定
	Devices []CameraDevice `yaml:"devices"`

	// デフォルト設定
	DefaultFPS    int `yaml:"default_fps"`    // フレームレート (fps)
	DefaultWidth  int `yaml:"default_width"`  // 画像幅
	DefaultHeight int `yaml:"default_height"` // 画像高さ
}

// CameraDevice は個別カメラの設定
type CameraDevice struct {
	ID     string `yaml:"id"`     // カメラID
	Name   string `yaml:"name"`   // カメラ名
	Device string `yaml:"device"` // デバイスパス (例: /dev/video0)

	// カメラ固有の設定（デフォルト値より優先）
	FPS    int `yaml:"fps"`
	Width  int `yaml:"width"`
	Height int `yaml:"height"`
}

// Load は設定を読み込む
// 現在はデフォルト値を返すシンプルな実装
func Load() (*Config, error) {
	// デフォルト設定を作成
	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvAsIntOrDefault("SERVER_PORT", 8080),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 0, // ストリーミング用にタイムアウト無効化
		},
		Camera: CameraConfig{
			Devices: []CameraDevice{
				{
					ID:     "camera1",
					Name:   "メインカメラ",
					Device: "/dev/video0",
					FPS:    15,
					Width:  1280,
					Height: 720,
				},
			},
			DefaultFPS:    15,
			DefaultWidth:  1280,
			DefaultHeight: 720,
		},
	}

	// 設定の検証
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("設定の検証に失敗: %w", err)
	}

	return cfg, nil
}

// Validate は設定の妥当性を検証する
func (c *Config) Validate() error {
	// サーバー設定の検証
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("無効なポート番号: %d", c.Server.Port)
	}

	// カメラ設定の検証
	if len(c.Camera.Devices) == 0 {
		return fmt.Errorf("カメラデバイスが設定されていません")
	}

	for _, device := range c.Camera.Devices {
		if device.ID == "" {
			return fmt.Errorf("カメラIDが設定されていません")
		}
		if device.Device == "" {
			return fmt.Errorf("カメラデバイスパスが設定されていません: %s", device.ID)
		}
	}

	return nil
}

// ServerAddress はサーバーのリッスンアドレスを返す
func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// getEnvOrDefault は環境変数を取得し、設定されていない場合はデフォルト値を返す
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsIntOrDefault は環境変数を整数として取得し、設定されていない場合はデフォルト値を返す
func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}
