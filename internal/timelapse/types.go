package timelapse

import (
	"time"
)

// SourceFrame は単一映像ソースのフレームデータ
type SourceFrame struct {
	SourceID  string    `json:"source_id"` // 映像ソースID
	Timestamp time.Time `json:"timestamp"` // フレームのタイムスタンプ
	Data      []byte    `json:"data"`      // JPEG画像データ
	Size      int       `json:"size"`      // データサイズ
}

// CombinedFrame は複数映像ソースを結合したフレームデータ
type CombinedFrame struct {
	Timestamp    time.Time              `json:"timestamp"`     // 結合フレームのタイムスタンプ
	SourceFrames map[string]SourceFrame `json:"source_frames"` // ソースID毎のフレーム
	ComposedData []byte                 `json:"composed_data"` // 結合後のJPEG画像データ
	Size         int                    `json:"size"`          // データサイズ
}

// Config はタイムラプス設定
type Config struct {
	Enabled         bool          `json:"enabled"`          // 有効/無効
	CaptureInterval time.Duration `json:"capture_interval"` // 撮影間隔 (デフォルト: 2秒)
	UpdateInterval  time.Duration `json:"update_interval"`  // 動画更新間隔 (デフォルト: 1時間)
	OutputFormat    string        `json:"output_format"`    // 出力フォーマット ("mp4")
	Quality         int           `json:"quality"`          // 動画品質 (1-5)
	Resolution      Resolution    `json:"resolution"`       // 出力解像度
	MaxFrameBuffer  int           `json:"max_frame_buffer"` // 最大バッファサイズ
	RetentionDays   int           `json:"retention_days"`   // 保持期間（日数）
}

// Resolution は解像度設定
type Resolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Video はタイムラプス動画情報
type Video struct {
	Date        time.Time     `json:"date"`         // 作成日
	FilePath    string        `json:"file_path"`    // ファイルパス
	FileSize    int64         `json:"file_size"`    // ファイルサイズ
	Duration    time.Duration `json:"duration"`     // 動画長
	FrameCount  int           `json:"frame_count"`  // 総フレーム数
	StartTime   time.Time     `json:"start_time"`   // 録画開始時刻
	EndTime     time.Time     `json:"end_time"`     // 録画終了時刻
	Status      Status        `json:"status"`       // ステータス
	SourceCount int           `json:"source_count"` // 結合された映像ソース数
}

// Status はタイムラプスのステータス
type Status string

// Status の定数定義
const (
	StatusRecording Status = "recording" // 録画中
	StatusCompleted Status = "completed" // 完了
	StatusError     Status = "error"     // エラー
	StatusPaused    Status = "paused"    // 一時停止
)

// DefaultConfig はデフォルトのタイムラプス設定を返す
func DefaultConfig() Config {
	return Config{
		Enabled:         true,
		CaptureInterval: 2 * time.Second,
		UpdateInterval:  1 * time.Hour,
		OutputFormat:    "mp4",
		Quality:         3,
		Resolution: Resolution{
			Width:  1920,
			Height: 1080,
		},
		MaxFrameBuffer: 1800, // 1時間分（2秒間隔）
		RetentionDays:  30,
	}
}
