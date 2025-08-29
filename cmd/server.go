// Package main はSenriganサーバーコマンドの実装です
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"senrigan/internal/config"
	"senrigan/internal/server"
)

func main() {
	// コマンドラインオプション
	var (
		host = flag.String("host", "", "サーバーのホスト (デフォルト: 0.0.0.0)")
		port = flag.Int("port", 0, "サーバーのポート (デフォルト: 8080)")
		help = flag.Bool("help", false, "ヘルプを表示")
	)

	flag.Parse()

	// ヘルプ表示
	if *help {
		fmt.Println("Senrigan")
		fmt.Println()
		fmt.Println("使用方法:")
		fmt.Println("  server [オプション]")
		fmt.Println()
		fmt.Println("オプション:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// 設定を読み込む
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	// コマンドラインオプションで設定を上書き
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}

	// Ginサーバーを作成
	srv := server.NewGin(cfg)

	// コンテキストを作成
	ctx := context.Background()

	// サーバーを起動
	log.Printf("Senrigan サーバーを起動します: %s", cfg.ServerAddress())
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
	}
}
