// Package camera カメラデバイスの動的管理を担う
//
// # 責務
// - カメラデバイスの自動検出と管理
// - カメラの動的な追加・削除機能
// - カメラ状態の監視とライフサイクル管理
// - カメラサービスの統合管理
// - V4L2デバイスからのリアルタイム画像取得
//
// # 使い分け
// このパッケージは以下の場合に使用する：
// - カメラデバイスを動的に管理したい
// - カメラの状態をリアルタイムで監視したい
// - カメラの追加・削除を実行時に行いたい
// - V4L2デバイスから画像をストリーミングしたい
//
// # 仕様
// - Camera Manager: 複数カメラの統合管理
// - Camera Discovery: V4L2デバイスの自動検出・実名取得
// - Camera Service: 個別カメラの制御・状態管理・ストリーミング
// - V4L2 Capturer: ffmpeg経由での画像キャプチャ
// - Thread-safe な操作をサポート
// - エラーハンドリングとログ出力を統合
//
// # 前提要件
//   - v4l-utils: カメラ名の取得とデバイス制御に使用
//     Ubuntu/Debian: sudo apt install v4l-utils
//     Red Hat/Fedora: sudo dnf install v4l-utils
//   - ffmpeg: 画像キャプチャとストリーミングに使用
//     Ubuntu/Debian: sudo apt install ffmpeg
//     Red Hat/Fedora: sudo dnf install ffmpeg
//   - videoグループへの参加: デバイスアクセス権限
//     sudo usermod -a -G video $USER
package camera
