import { Configuration } from "../generated/configuration";

// API設定の共通定義
export const API_CONFIG = new Configuration({
  basePath: __BACKEND_BASE_URL__,
});

// ストリームURL構築のヘルパー関数
export function buildStreamUrl(cameraId: string): string {
  return `${__BACKEND_BASE_URL__}/api/cameras/${cameraId}/stream`;
}
