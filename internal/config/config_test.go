package config

import (
	"os"
	"testing"
)

// TestConfigLoad は設定の読み込みをテストする
func TestConfigLoad(t *testing.T) {
	// 設定を読み込む
	cfg, err := Load()
	if err != nil {
		t.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	// 基本的な設定値を検証
	if cfg == nil {
		t.Fatal("設定がnilです")
	}

	// サーバー設定の検証
	if cfg.Server.Host == "" {
		t.Error("サーバーホストが設定されていません")
	}
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		t.Errorf("無効なポート番号: %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout <= 0 {
		t.Error("読み込みタイムアウトが設定されていません")
	}
	// WriteTimeout は 0（無効）でも正常
	if cfg.Server.WriteTimeout < 0 {
		t.Error("書き込みタイムアウトが負の値です")
	}

	// カメラ設定の検証
	if len(cfg.Camera.Devices) == 0 {
		t.Error("カメラデバイスが設定されていません")
	}

	// デフォルト値の検証
	if cfg.Camera.DefaultFPS <= 0 {
		t.Error("デフォルトFPSが設定されていません")
	}
	if cfg.Camera.DefaultWidth <= 0 {
		t.Error("デフォルト幅が設定されていません")
	}
	if cfg.Camera.DefaultHeight <= 0 {
		t.Error("デフォルト高さが設定されていません")
	}
}

// TestConfigValidation は設定の検証をテストする
func TestConfigValidation(t *testing.T) {
	testCases := []struct {
		name      string
		config    *Config
		expectErr bool
	}{
		{
			name: "正常な設定",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
				Camera: CameraConfig{
					Devices: []CameraDevice{
						{
							ID:     "camera1",
							Name:   "メインカメラ",
							Device: "/dev/video0",
						},
					},
				},
			},
			expectErr: false,
		},
		{
			name: "無効なポート番号",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 99999, // 無効なポート
				},
				Camera: CameraConfig{
					Devices: []CameraDevice{
						{
							ID:     "camera1",
							Device: "/dev/video0",
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "カメラデバイスなし",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
				Camera: CameraConfig{
					Devices: []CameraDevice{}, // 空のデバイスリスト
				},
			},
			expectErr: true,
		},
		{
			name: "カメラIDなし",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
				Camera: CameraConfig{
					Devices: []CameraDevice{
						{
							ID:     "", // 空のID
							Device: "/dev/video0",
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "カメラデバイスパスなし",
			config: &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8080,
				},
				Camera: CameraConfig{
					Devices: []CameraDevice{
						{
							ID:     "camera1",
							Device: "", // 空のデバイスパス
						},
					},
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectErr && err == nil {
				t.Error("エラーが期待されましたが、エラーが発生しませんでした")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("予期しないエラーが発生しました: %v", err)
			}
		})
	}
}

// TestServerAddress はサーバーアドレスの生成をテストする
func TestServerAddress(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "192.168.1.100",
			Port: 9090,
		},
	}

	expected := "192.168.1.100:9090"
	actual := cfg.ServerAddress()

	if actual != expected {
		t.Errorf("サーバーアドレスが一致しません: got %s, want %s", actual, expected)
	}
}

// TestEnvironmentVariables は環境変数の処理をテストする
// 注意: このテストは環境変数を変更するため、parallelは使わない
func TestEnvironmentVariables(t *testing.T) {
	// テスト用の環境変数を設定
	originalHost := os.Getenv("SERVER_HOST")
	originalPort := os.Getenv("SERVER_PORT")

	defer func() {
		// テスト後に環境変数を復元
		_ = os.Setenv("SERVER_HOST", originalHost)
		_ = os.Setenv("SERVER_PORT", originalPort)
	}()

	// 環境変数を設定
	_ = os.Setenv("SERVER_HOST", "test.example.com")
	_ = os.Setenv("SERVER_PORT", "9999")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	if cfg.Server.Host != "test.example.com" {
		t.Errorf("環境変数のホストが反映されていません: got %s, want test.example.com", cfg.Server.Host)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("環境変数のポートが反映されていません: got %d, want 9999", cfg.Server.Port)
	}
}
