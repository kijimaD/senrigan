# Senrigan - 監視カメラシステム

## 機能要件

### フェーズ1: リアルタイム再生（現在の実装対象）
- 複数の定点カメラで、ブラウザからリアルタイム再生できる（保存はしない）
- 低遅延でのストリーミング配信
- 複数カメラの同時表示

### フェーズ2: タイムラプス機能
- 数秒ごとにタイムラプス画像を保存しており、ブラウザ/アプリで任意時点にジャンプ、ズーム・パン操作可能
  - サムネイル＋タイムスタンプ管理で一覧表示
  - コンビニなどの監視カメラのUIのイメージ。カーソルを動かすと、それぞれの定点カメラの時系列が遡る

### フェーズ3: 動画生成
- 一定期間ごとにタイムラプス画像を使って、タイムラプス動画を生成する

---

# タイムラプス機能詳細設計 (フェーズ2拡張)

## 拡張仕様

### 基本機能
- **撮影間隔**: 2秒毎に1フレーム
- **動画更新**: 1時間毎に動画を延長
- **ファイル分割**: 日毎に新しい動画ファイル作成
- **リアルタイム視聴**: 作成途中の動画も再生可能
- **対象**: 全ての映像ソース（USBカメラ、X11画面キャプチャ）
- **フレーム結合**: 複数の映像ソースを1つの画面に結合してタイムラプス動画を作成

### 技術仕様
- **フレームバッファ**: メモリ上に最大1時間分（1800フレーム）保持
- **動画フォーマット**: MP4（H.264 + AAC）
- **解像度**: ソース解像度に応じて可変（最大1920x1080）
- **品質**: 設定可能（1-5段階）

## アーキテクチャ拡張

### 新規コンポーネント

#### 1. TimelapseManager
```go
type TimelapseManager interface {
    // システム制御
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // データ取得
    GetTimelapseVideos() ([]TimelapseVideo, error)
    GetTimelapseStatus() (TimelapseStatus, error)
    
    // 設定管理（ファイルベース）
    LoadConfig() (TimelapseConfig, error)
    ReloadConfig() error
}
```

#### 2. TimelapseCapture (統合キャプチャ)
```go
type TimelapseCapture struct {
    frameBuffer   []CombinedFrame // 結合フレーム保存
    outputDir     string          // 動画出力先
    currentVideo  string          // 現在の動画ファイル
    lastUpdate    time.Time       // 最後の動画更新時刻
    config        TimelapseConfig // 設定
    videoSources  []VideoSource   // 全ての映像ソース
    
    // 制御用
    stopCh        chan struct{}
    wg            sync.WaitGroup
    mu            sync.RWMutex
}
```

#### 3. データ構造
```go
// 単一ソースフレームデータ
type SourceFrame struct {
    SourceID  string
    Timestamp time.Time
    Data      []byte  // JPEG画像データ
    Size      int     // データサイズ
}

// 結合フレームデータ
type CombinedFrame struct {
    Timestamp    time.Time
    SourceFrames map[string]SourceFrame // ソースID毎のフレーム
    ComposedData []byte                 // 結合後のJPEG画像データ
    Size         int                    // データサイズ
}

// タイムラプス設定
type TimelapseConfig struct {
    Enabled          bool          // 有効/無効
    CaptureInterval  time.Duration // 撮影間隔 (デフォルト: 2秒)
    UpdateInterval   time.Duration // 動画更新間隔 (デフォルト: 1時間)
    OutputFormat     string        // 出力フォーマット ("mp4")
    Quality          int           // 動画品質 (1-5)
    Resolution       Resolution    // 出力解像度
    MaxFrameBuffer   int           // 最大バッファサイズ
    RetentionDays    int           // 保持期間（日数）
}

// タイムラプス動画情報
type TimelapseVideo struct {
    Date        time.Time         // 作成日
    FilePath    string            // ファイルパス
    FileSize    int64             // ファイルサイズ
    Duration    time.Duration     // 動画長
    FrameCount  int               // 総フレーム数
    StartTime   time.Time         // 録画開始時刻
    EndTime     time.Time         // 録画終了時刻
    Status      TimelapseStatus   // ステータス
    SourceCount int               // 結合された映像ソース数
}

// タイムラプスステータス
type TimelapseStatus string

const (
    TimelapseStatusRecording  TimelapseStatus = "recording"  // 録画中
    TimelapseStatusCompleted  TimelapseStatus = "completed"  // 完了
    TimelapseStatusError      TimelapseStatus = "error"      // エラー
    TimelapseStatusPaused     TimelapseStatus = "paused"     // 一時停止
)
```

## 動作フロー

### 1. フレーム撮影フロー
```
[VideoSource] --2秒毎--> [TimelapseCapture] 
                               ↓
                    [Frame Buffer (メモリ)]
                               ↓
                      (1時間毎にトリガー)
                               ↓
                    [Video Generator (ffmpeg)]
                               ↓
                      [MP4ファイル更新/作成]
```

### 2. 動画生成フロー
1. **フレーム収集**: Frame Bufferから新フレーム取得
2. **一時ファイル作成**: 新フレームを画像ファイル群として保存
3. **動画生成/更新**:
   - 既存動画: ffmpegで新フレーム追加
   - 新しい日: 新規動画ファイル作成
4. **クリーンアップ**: 一時ファイル・バッファクリア

### 3. 日次ローテーション
- 00:00:00に新動画ファイル開始
- 前日動画は完了ステータスに変更
- ファイル名: `{sourceName}_YYYY-MM-DD.mp4`

## ファイル構造

```
/data/timelapse/
├── YYYY-MM-DD/
│   ├── camera_123456_YYYY-MM-DD.mp4
│   ├── screen_capture_YYYY-MM-DD.mp4
│   └── temp/
│       ├── camera_123456/
│       └── screen_capture/
├── config/
│   ├── timelapse_config.json
│   └── source_configs/
└── metadata/
    ├── videos.db (SQLite)
    └── logs/
```

## API設計

### REST API エンドポイント
```
GET    /api/timelapse/videos    # タイムラプス動画一覧
GET    /api/timelapse/config    # 設定取得（ファイルベース）
GET    /api/timelapse/status    # システムステータス
DELETE /api/timelapse/videos/{videoId} # 動画削除
```

### WebSocket API
```
/ws/timelapse/status    # 進行状況のリアルタイム配信
```

## 実装ステップ

### Phase 1: 基本構造 (1-2日)
1. `TimelapseManager`インターフェース定義
2. `TimelapseCapture`実装
3. フレーム撮影・バッファリング機能

### Phase 2: 動画生成 (2-3日)
1. ffmpeg統合（動画作成・延長）
2. スケジューリング（1時間毎更新・日次分割）
3. バックグラウンド処理

### Phase 3: API・UI (2-3日)
1. REST API実装
2. フロントエンド（設定画面・プレイヤー）
3. リアルタイム進行状況表示

### Phase 4: 最適化 (1-2日)
1. ストレージ管理（古い動画削除）
2. パフォーマンス最適化
3. 監視・ログ機能強化

## 技術的考慮事項

### メモリ管理
- フレームバッファサイズ制限（デフォルト1800フレーム）
- 定期的なクリーンアップ
- メモリ使用量監視

### 並行処理
- 映像ソース毎の独立goroutine
- 動画生成の非同期実行
- 競合状態の回避

### エラーハンドリング
- ffmpegプロセス失敗時の再試行
- ディスク容量不足の検知
- 映像ソース停止時の処理

### パフォーマンス目標
- **メモリ使用量**: 映像ソース1つあたり最大100MB
- **CPU使用率**: 動画生成時除き5%以下
- **ディスク使用量**: 1日あたり約500MB/映像ソース
- **レスポンス時間**: API応答200ms以下

## 設定例

```json
{
  "global": {
    "output_directory": "/data/timelapse",
    "temp_directory": "/tmp/senrigan-timelapse",
    "retention_days": 30,
    "max_concurrent_processing": 2
  },
  "default_config": {
    "capture_interval": "2s",
    "update_interval": "1h",
    "output_format": "mp4",
    "quality": 3,
    "resolution": {
      "width": 1920,
      "height": 1080
    },
    "max_frame_buffer": 1800
  },
  "source_configs": {
    "camera_123456": {
      "enabled": true,
      "capture_interval": "2s",
      "quality": 4
    },
    "screen_capture": {
      "enabled": true,
      "capture_interval": "5s",
      "resolution": {
        "width": 1280,
        "height": 720
      }
    }
  }
}
```

## アーキテクチャ設計

### 全体方針
- **段階的な実装**: まずモノリシックで開始し、必要に応じてハイブリッド構成へ移行
- **将来の拡張性を考慮**: インターフェースを適切に設計し、後からの機能追加を容易に
- **シンプルさ優先**: 過度な抽象化を避け、必要最小限の実装から開始

### フェーズ1の設計（現在）

#### 技術スタック
- **言語**: Go
- **ストリーミング方式**: WebSocket + MJPEG
  - 理由: シンプルで実装が容易、ブラウザ互換性が高い、遅延が少ない
- **Webフレームワーク**: 標準ライブラリ (net/http) + gorilla/websocket
- **カメラ入力**: V4L2 (Video4Linux2) または FFmpeg経由

#### システム構成
```
[カメラ] → [Goサーバー] → [WebSocket] → [ブラウザ]
```

#### プロジェクト構造
```
senrigan/
├── cmd/
│   └── server.go
├── internal/
│   ├── camera/              # カメラ制御・キャプチャ
│   │   ├── doc.go
│   │   ├── manager.go      # 複数カメラの管理
│   │   ├── capture.go      # カメラからの画像取得
│   │   └── stream.go       # ストリーム生成
│   ├── server/              # HTTPサーバー
│   │   ├── doc.go
│   │   ├── server.go       # HTTPサーバー本体
│   │   ├── websocket.go    # WebSocketハンドラ
│   │   └── static.go       # 静的ファイル配信
│   └── config/              # 設定管理
│       ├── doc.go
│       └── config.go       # 設定構造体
├── front/                     # フロントエンド
│   ├── index.html          # メインページ
│   ├── style.css           # スタイル
│   └── app.js              # WebSocket接続・表示制御
├── go.mod
├── go.sum
├── main.go
├── Makefile
├── README.md
└── config.yaml              # 設定ファイル
```

#### 主要コンポーネント

1. **Camera Manager**
   - 複数カメラの管理
   - カメラの初期化・終了処理
   - フレームレートの制御

2. **Stream Handler**
   - カメラからの画像をJPEGにエンコード
   - WebSocketクライアントへの配信
   - バックプレッシャー制御

3. **WebSocket Server**
   - クライアント接続管理
   - ストリームのルーティング
   - 切断処理

### 将来のハイブリッド構成への移行パス

フェーズ2以降で必要に応じて以下の分離を検討：

1. **ストリーミングサーバー** (現在のコア)
   - リアルタイム配信に特化
   - 軽量・低遅延を維持

2. **ストレージサービス** (フェーズ2で追加)
   - タイムラプス画像の保存
   - メタデータ管理
   - サムネイル生成

3. **ワーカープロセス** (フェーズ3で追加)
   - 動画生成処理
   - バッチ処理

## 実装の優先順位

1. 単一カメラのストリーミング実装
2. 複数カメラ対応
3. Web UIの改善
4. 設定ファイル対応
5. エラーハンドリング・ロギング強化

## 非機能要件

- **遅延**: 1秒以内のストリーミング遅延
- **同時接続数**: 10クライアント程度を想定
- **画質**: 720p (1280x720) @ 15-30fps
- **カメラ数**: 最大4台程度
