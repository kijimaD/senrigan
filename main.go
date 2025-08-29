package main

import (
	"context"
	"log"
	"os"

	"senrigan/internal/config"
	"senrigan/internal/server"
)

func main() {
	// 設定を読み込む
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	// サーバーを作成
	srv := server.New(cfg)

	// コンテキストを作成
	ctx := context.Background()

	// サーバーを起動
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
		os.Exit(1)
	}
}
