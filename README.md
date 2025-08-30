# Senrigan

リアルタイムストリーミング対応の監視カメラシステムです。

## 構成

- **バックエンド**: Go + Gin（APIサーバー）
- **フロントエンド**: TypeScript + React + Vite
- **API仕様**: OpenAPI 3.0

## 開発環境セットアップ

### システム要件

V4L2対応カメラを使用するため、以下のライブラリが必要です：

```bash
# Ubuntu/Debian
sudo apt install v4l-utils

# Red Hat/CentOS/Fedora  
sudo yum install v4l-utils
# または
sudo dnf install v4l-utils
```

### カメラデバイスの確認

```bash
# 利用可能なカメラデバイスを確認
ls /dev/video*

# カメラの詳細情報を確認
v4l2-ctl --device /dev/video0 --info

# サポートされているフォーマットを確認
v4l2-ctl --device /dev/video0 --list-formats-ext
```

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

## カメラ機能

### V4L2対応

このシステムはV4L2（Video4Linux2）を使用してカメラデバイスにアクセスします：

- **自動デバイス検出**: `/dev/video*` デバイスを自動検出
- **実デバイス名取得**: `v4l2-ctl`を使用して実際のカメラ名を表示
- **フォールバック対応**: v4l-utilsが無い環境でも動作（デバイス番号で表示）
- **リアルタイムストリーミング**: ffmpegを使用した低遅延ストリーミング

### 対応カメラ

- USBウェブカメラ（UVC対応）
- 内蔵カメラ
- その他V4L2対応デバイス

### トラブルシューティング

#### カメラが認識されない場合

```bash
# デバイスの存在確認
ls -la /dev/video*

# 権限確認
ls -la /dev/video0

# ユーザーをvideoグループに追加
sudo usermod -a -G video $USER
# 再ログインが必要
```

#### ffmpegが見つからない場合

```bash
# Ubuntu/Debian
sudo apt install ffmpeg

# Red Hat/CentOS/Fedora
sudo yum install ffmpeg
# または
sudo dnf install ffmpeg
```
