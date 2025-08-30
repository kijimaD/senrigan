.DEFAULT_GOAL := help

.PHONY: build
build: ## ビルドする
	go build -o ./bin/senrigan .

.PHONY: run
run: ## 実行する
	go run .

.PHONY: test
test: ## テストを実行する
	go test -v -cover -shuffle=on ./...

.PHONY: fmt
fmt: ## フォーマットする
	goimports -w .

.PHONY: lint
lint: ## Linterを実行する
	@golangci-lint run -v ./...

.PHONY: tools-install
tools-install: ## 開発ツールをインストールする
	@go install golang.org/x/tools/cmd/goimports@latest
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.2.2)

.PHONY: check
check: test build fmt lint ## 一気にチェックする

.PHONY: frontend-install
frontend-install: ## フロントエンドの依存関係をインストール
	cd frontend && npm install

.PHONY: frontend-build
frontend-build: ## フロントエンドを本番用にビルド
	cd frontend && npm run build

.PHONY: frontend-dev
frontend-dev: ## フロントエンド開発サーバを起動（ポート3000）
	cd frontend && npm run dev

.PHONY: backend-dev
backend-dev: ## バックエンド開発サーバを起動（ポート8080）
	go run .

.PHONY: production
production: frontend-build build ## 本番用ビルド（フロントエンド + バックエンド統合）
	@echo "本番用ビルドが完了しました！"
	@echo "バイナリ: ./bin/senrigan"
	@echo "起動: ./bin/senrigan"
	@echo "アクセス: http://localhost:8080"

# ================

.PHONY: help
help: ## ヘルプを表示する
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
