// API設定の共通定義
export const API_CONFIG = {
  basePath: __BACKEND_BASE_URL__,
};

// ストリームURL構築のヘルパー関数
export function buildStreamUrl(cameraId: string): string {
  return `${API_CONFIG.basePath}/api/cameras/${cameraId}/stream`;
}
