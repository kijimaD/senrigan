package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"senrigan/internal/camera"
	"senrigan/internal/config"
	"senrigan/internal/generated"
	"senrigan/internal/timelapse"

	"github.com/gin-gonic/gin"
)

// GinServer はGinベースのHTTPサーバーを管理する構造体
type GinServer struct {
	config           *config.Config
	httpServer       *http.Server
	router           *gin.Engine
	cameraManager    camera.Manager
	timelapseManager timelapse.Manager
}

// NewGin は新しいGinServerインスタンスを作成する
func NewGin(cfg *config.Config) *GinServer {
	// 開発モードで動作
	gin.SetMode(gin.DebugMode)

	router := gin.New()

	// デフォルトミドルウェア
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS設定（開発環境用に緩和）
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// カメラマネージャーを初期化
	discovery := camera.NewLinuxDiscovery()
	cameraManager := camera.NewDefaultCameraManager(discovery)

	// タイムラプスマネージャーを初期化
	timelapseOutputDir := "/data/timelapse" // TODO: 設定ファイルから読み込み
	timelapseManager := timelapse.NewDefaultManager(cameraManager, timelapseOutputDir, cfg.Timelapse)

	return &GinServer{
		config:           cfg,
		router:           router,
		cameraManager:    cameraManager,
		timelapseManager: timelapseManager,
		httpServer: &http.Server{
			Addr:         cfg.ServerAddress(),
			Handler:      router,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		},
	}
}

// Start はサーバーを起動する
func (s *GinServer) Start(ctx context.Context) error {
	// カメラマネージャーを開始
	if err := s.cameraManager.Start(ctx); err != nil {
		return fmt.Errorf("カメラマネージャーの起動に失敗: %w", err)
	}

	// タイムラプスマネージャーを開始
	if err := s.timelapseManager.Start(ctx); err != nil {
		log.Printf("タイムラプスマネージャーの起動に失敗: %v", err)
		// タイムラプスはオプション機能なので失敗してもサーバー起動を続行
	}

	// ルートを設定
	s.setupRoutes()

	// シャットダウン用のチャンネル
	shutdownCh := make(chan error, 1)

	// サーバーを別ゴルーチンで起動
	go func() {
		log.Printf("Gin HTTPサーバーを起動しています: %s", s.config.ServerAddress())
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
func (s *GinServer) Shutdown() error {
	log.Println("サーバーをシャットダウンしています...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// タイムラプスマネージャーを停止
	log.Println("タイムラプスマネージャーを停止中...")
	if err := s.timelapseManager.Stop(ctx); err != nil {
		log.Printf("タイムラプスマネージャーの停止に失敗: %v", err)
	} else {
		log.Println("タイムラプスマネージャーを停止しました")
	}

	// カメラマネージャーを停止
	log.Println("カメラマネージャーを停止中...")
	if err := s.cameraManager.Stop(ctx); err != nil {
		log.Printf("カメラマネージャーの停止に失敗: %v", err)
	} else {
		log.Println("カメラマネージャーを停止しました")
	}

	// HTTPサーバーをシャットダウン
	log.Println("HTTPサーバーを停止中...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("サーバーのシャットダウンに失敗: %w", err)
	}

	log.Println("サーバーが正常にシャットダウンされました")
	return nil
}

// setupRoutes はHTTPルートを設定する
func (s *GinServer) setupRoutes() {
	// ServerInterfaceを実装したハンドラーを作成
	handler := &SenriganHandler{
		config:           s.config,
		cameraManager:    s.cameraManager,
		timelapseManager: s.timelapseManager,
	}

	// 生成されたルートを登録（OpenAPI仕様に基づく）
	generated.RegisterHandlers(s.router, handler)

	// フロントエンドの静的ファイルを配信（embed）
	s.router.StaticFS("/assets", GetAssetsFS())
	s.router.GET("/favicon.ico", func(c *gin.Context) {
		c.FileFromFS("favicon.ico", GetStaticFS())
	})

	// SPAのためのフォールバック（APIルート以外はindex.htmlを返す）
	s.router.NoRoute(func(c *gin.Context) {
		// APIルートの場合は404を返す
		if strings.HasPrefix(c.Request.URL.Path, "/api/") || strings.HasPrefix(c.Request.URL.Path, "/health") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}
		// それ以外はindex.htmlを配信（SPAルーティング用）
		c.Data(http.StatusOK, "text/html; charset=utf-8", getIndexHTML())
	})
}
