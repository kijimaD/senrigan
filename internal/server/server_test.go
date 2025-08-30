package server

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"senrigan/internal/config"
)

// TestGinServerEndpoints はGinサーバーのエンドポイントをテストする
func TestGinServerEndpoints(t *testing.T) {
	// テスト用の設定（利用可能なポートを使用）
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         8082, // 固定ポートでテスト
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		Camera: config.CameraConfig{
			Devices: []config.CameraDevice{
				{
					ID:     "test-camera",
					Name:   "テストカメラ",
					Device: "/dev/null",
					FPS:    15,
					Width:  1280,
					Height: 720,
				},
			},
		},
	}

	// Ginサーバーを作成
	srv := NewGin(cfg)

	// テスト用のコンテキスト
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// サーバーを別ゴルーチンで起動
	go func() {
		_ = srv.Start(ctx)
	}()

	// サーバーがリッスンを開始するまで待つ
	baseURL := fmt.Sprintf("http://%s", cfg.ServerAddress())
	for i := 0; i < 50; i++ { // 最大5秒待つ（50回 × 100ms）
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			_ = resp.Body.Close()
			break // サーバーが起動した
		}
		if i == 49 {
			t.Fatal("サーバーの起動がタイムアウトしました")
		}
		time.Sleep(100 * time.Millisecond)
	}

	// テストケース
	testCases := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{"ルートエンドポイント", "/", http.StatusOK},
		{"ヘルスチェックエンドポイント", "/health", http.StatusOK},
		{"ステータスエンドポイント", "/api/status", http.StatusOK},
		{"カメラ一覧エンドポイント", "/api/cameras", http.StatusOK},
		{"存在しないカメラストリーム", "/api/cameras/nonexistent/stream", http.StatusNotFound},
	}

	// 各エンドポイントをテスト
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(baseURL + tc.endpoint)
			if err != nil {
				t.Fatalf("HTTPリクエストでエラーが発生しました: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("予期しないステータスコード: got %d, want %d",
					resp.StatusCode, tc.expectedStatus)
			}
		})
	}
}
