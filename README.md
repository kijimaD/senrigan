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

### 開発サーバの起動

開発時は2つのサーバを別々に起動します：

1. **バックエンドサーバ** (ポート8009)
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
- バックエンドAPI直接: http://localhost:8009/api/status

### 本番サーバの起動

```
make production
```

http://localhost:8009

(embedによって、バックエンド単体で動く)
