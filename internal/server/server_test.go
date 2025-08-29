package server

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"senrigan/internal/config"
)

// TestServerStartAndShutdown はサーバーの起動とシャットダウンをテストする
func TestServerStartAndShutdown(t *testing.T) {
	// テスト用の設定を作成
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         0, // ランダムポートを使用
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		Camera: config.CameraConfig{
			Devices: []config.CameraDevice{
				{
					ID:     "test-camera",
					Name:   "テストカメラ",
					Device: "/dev/null", // テスト用のダミーデバイス
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

	// サーバーを作成
	srv := New(cfg)

	// テスト用のコンテキスト（タイムアウト付き）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// サーバーを別ゴルーチンで起動
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// サーバーが起動するまで少し待つ
	time.Sleep(100 * time.Millisecond)

	// コンテキストをキャンセルしてサーバーを停止
	cancel()

	// エラーチャンネルから結果を受信
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("サーバーの起動/停止でエラーが発生しました: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("サーバーの停止がタイムアウトしました")
	}
}

// TestServerEndpoints はサーバーのエンドポイントをテストする
func TestServerEndpoints(t *testing.T) {
	// テスト用の設定（利用可能なポートを使用）
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         8081, // 固定ポートでテスト
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

	// サーバーを作成
	srv := New(cfg)

	// テスト用のコンテキスト
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// サーバーを別ゴルーチンで起動
	go func() {
		srv.Start(ctx)
	}()

	// サーバーが起動するまで待つ
	time.Sleep(500 * time.Millisecond)

	baseURL := fmt.Sprintf("http://%s", cfg.ServerAddress())

	// テストケース
	testCases := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{"ルートエンドポイント", "/", http.StatusOK},
		{"ヘルスチェックエンドポイント", "/health", http.StatusOK},
		{"ステータスエンドポイント", "/api/status", http.StatusOK},
	}

	// 各エンドポイントをテスト
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(baseURL + tc.endpoint)
			if err != nil {
				t.Fatalf("HTTPリクエストでエラーが発生しました: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("予期しないステータスコード: got %d, want %d",
					resp.StatusCode, tc.expectedStatus)
			}
		})
	}
}
