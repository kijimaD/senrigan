import { Configuration } from "../generated/configuration";

// API設定の共通定義
export const API_CONFIG = new Configuration({
  basePath: "", // 相対パスを使用（プロキシ経由）
});

// ストリームURL構築のヘルパー関数
export function buildStreamUrl(cameraId: string): string {
  return `/api/cameras/${cameraId}/stream`;
}
