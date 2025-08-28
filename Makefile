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

# ================

.PHONY: help
help: ## ヘルプを表示する
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
