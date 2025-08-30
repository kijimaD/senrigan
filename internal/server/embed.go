package server

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed all:dist
var embedFS embed.FS

// GetStaticFS returns the static files filesystem
func GetStaticFS() http.FileSystem {
	// dist のサブディレクトリを取得
	staticFS, err := fs.Sub(embedFS, "dist")
	if err != nil {
		log.Fatalf("埋め込み静的ファイルシステムの作成に失敗: %v", err)
	}
	return http.FS(staticFS)
}

// GetAssetsFS returns the assets filesystem
func GetAssetsFS() http.FileSystem {
	// dist/assets のサブディレクトリを取得
	assetsFS, err := fs.Sub(embedFS, "dist/assets")
	if err != nil {
		log.Fatalf("埋め込みアセットファイルシステムの作成に失敗: %v", err)
	}
	return http.FS(assetsFS)
}

// getIndexHTML returns the index.html content as bytes
func getIndexHTML() []byte {
	data, err := embedFS.ReadFile("dist/index.html")
	if err != nil {
		log.Fatalf("埋め込みindex.htmlの読み込みに失敗: %v", err)
	}
	return data
}
