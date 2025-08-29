package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"senrigan/internal/config"
)

// Server はHTTPサーバーを管理する構造体
type Server struct {
	config     *config.Config
	httpServer *http.Server
	mux        *http.ServeMux
}

// New は新しいServerインスタンスを作成する
func New(cfg *config.Config) *Server {
	mux := http.NewServeMux()

	return &Server{
		config: cfg,
		mux:    mux,
		httpServer: &http.Server{
			Addr:         cfg.ServerAddress(),
			Handler:      mux,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		},
	}
}

// setupRoutes はHTTPルートを設定する
func (s *Server) setupRoutes() {
	// ヘルスチェックエンドポイント
	s.mux.HandleFunc("/health", s.handleHealth)

	// APIエンドポイント
	s.mux.HandleFunc("/api/status", s.handleStatus)

	// ルートハンドラ（簡単な確認用）
	s.mux.HandleFunc("/", s.handleRoot)
}

// handleHealth はヘルスチェックエンドポイント
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// handleStatus はステータス確認エンドポイント
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// カメラ設定情報を含めたステータスを返す
	fmt.Fprintf(w, `{
		"status": "running",
		"server": {
			"host": "%s",
			"port": %d
		},
		"cameras": %d,
		"timestamp": "%s"
	}`, s.config.Server.Host, s.config.Server.Port,
		len(s.config.Camera.Devices),
		time.Now().Format(time.RFC3339))
}

// handleRoot はルートパスのハンドラ
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>Senrigan - 監視カメラシステム</title>
</head>
<body>
    <h1>Senrigan 監視カメラシステム</h1>
    <p>サーバーが正常に起動しています。</p>
    <p>ステータス: <a href="/api/status">/api/status</a></p>
    <p>ヘルスチェック: <a href="/health">/health</a></p>
</body>
</html>`)
}

// Start はサーバーを起動する
func (s *Server) Start(ctx context.Context) error {
	// ルートを設定
	s.setupRoutes()

	// シャットダウン用のチャンネル
	shutdownCh := make(chan error, 1)

	// サーバーを別ゴルーチンで起動
	go func() {
		log.Printf("HTTPサーバーを起動しています: %s", s.config.ServerAddress())
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			shutdownCh <- fmt.Errorf("サーバーの起動に失敗: %w", err)
		}
	}()

	// シグナルハンドリング
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// コンテキストかシグナルを待つ
	select {
	case <-ctx.Done():
		log.Println("コンテキストがキャンセルされました")
	case sig := <-sigCh:
		log.Printf("シグナルを受信しました: %v", sig)
	case err := <-shutdownCh:
		return err
	}

	// グレースフルシャットダウン
	return s.Shutdown()
}

// Shutdown はサーバーをグレースフルにシャットダウンする
func (s *Server) Shutdown() error {
	log.Println("サーバーをシャットダウンしています...")

	// 5秒のタイムアウトを設定
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("サーバーのシャットダウンに失敗: %w", err)
	}

	log.Println("サーバーが正常にシャットダウンされました")
	return nil
}
