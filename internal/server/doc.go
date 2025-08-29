// Package server は、HTTPサーバーとWebSocket通信を管理します。
//
// このパッケージは、HTTPサーバーの起動、ルーティング、
// WebSocket接続の管理、静的ファイルの配信を担当します。
//
// 責務:
//   - HTTPサーバーの起動と管理
//   - WebSocket接続の確立と管理
//   - 静的ファイル（HTML/CSS/JS）の配信
//   - クライアントからのリクエスト処理
//   - ストリーミングデータの配信
//
// 仕様:
//   - 標準ライブラリのnet/httpを使用
//   - WebSocketはgorilla/websocketを使用（将来的に）
//   - グレースフルシャットダウンに対応
//   - 複数クライアントの同時接続をサポート
package server
