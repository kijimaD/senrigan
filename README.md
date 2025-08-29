# Senrigan - 監視カメラシステム

リアルタイムストリーミング対応の監視カメラシステムです。

## 構成

- **バックエンド**: Go + Gin（APIサーバー）
- **フロントエンド**: TypeScript + React + Vite
- **API仕様**: OpenAPI 3.0

## 開発環境セットアップ

### 依存関係のインストール

```bash
# フロントエンド依存関係
make frontend-install
```

### 開発サーバの起動

開発時は2つのサーバを別々に起動します：

1. **バックエンドサーバ** (ポート8080)
```bash
make backend-dev
# または
go run .
```

2. **フロントエンドサーバ** (ポート3000) - 別ターミナル
```bash
make frontend-dev
# または
cd frontend && npm run dev
```

フロントエンドサーバが自動的にバックエンドAPIにプロキシします。

アクセス方法：
- フロントエンド: http://localhost:3000
- バックエンドAPI直接: http://localhost:8080/api/status

## API仕様

OpenAPI仕様は `/oas/openapi.yml` にあります。

主要エンドポイント：
- `GET /health` - ヘルスチェック
- `GET /api/status` - システム状態
- `GET /api/cameras` - カメラ一覧
- `GET /api/cameras/{id}/stream` - カメラストリーム（未実装）

## 開発用コマンド

```bash
make help           # 利用可能なコマンド一覧
make check          # テスト + ビルド + フォーマット + Lint
make test           # テスト実行
make fmt            # コードフォーマット
make lint           # Linter実行
```

## プロジェクト構成

```
senrigan/
├── cmd/                # エントリーポイント
├── internal/          # Goパッケージ
│   ├── camera/       # カメラ制御
│   ├── config/       # 設定管理
│   ├── server/       # HTTPサーバ
│   └── generated/    # OpenAPI生成コード
├── frontend/         # React アプリケーション
│   ├── src/         # TypeScriptソース
│   └── dist/        # ビルド出力
├── oas/             # OpenAPI仕様
└── front/           # 静的HTML（従来版）
```

## 設定

設定ファイルは環境変数で指定します：
- `CONFIG_PATH`: 設定ファイルパス（デフォルト: config/development.yml）

設定例は `internal/config/` を参照してください。