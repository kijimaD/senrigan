import { useEffect, useState } from "react";
import { StatusApi, CameraApi } from "../generated/api";
import type { StatusResponse, CameraInfo, ErrorResponse } from "../generated/api";
import { AxiosError } from "axios";

export function MainPage() {
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [cameras, setCameras] = useState<CameraInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const statusApi = new StatusApi();
        const cameraApi = new CameraApi();

        // システム状態を取得
        const statusResponse = await statusApi.getStatus();
        setStatus(statusResponse.data);

        // カメラ一覧を取得
        const camerasResponse = await cameraApi.getCameras();
        setCameras(camerasResponse.data.cameras);
      } catch (err) {
        if (err instanceof AxiosError && err.response?.data) {
          const errorData = err.response.data as ErrorResponse;
          setError(`エラー: ${errorData.message}`);
          console.error("API Error:", errorData);
        } else {
          setError("データの取得に失敗しました");
          console.error("API fetch error:", err);
        }
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  if (loading) {
    return (
      <div style={{ padding: "20px", textAlign: "center" }}>
        <h1>Senrigan</h1>
        <p>読み込み中...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: "20px", textAlign: "center" }}>
        <h1>Senrigan</h1>
        <p style={{ color: "red" }}>エラー: {error}</p>
      </div>
    );
  }

  return (
    <div style={{ padding: "20px", maxWidth: "1200px", margin: "0 auto" }}>
      <header style={{ marginBottom: "30px" }}>
        <h1>Senrigan</h1>
      </header>

      {status && (
        <section style={{ marginBottom: "30px" }}>
          <h2>システム状態</h2>
          <div style={{ backgroundColor: "white", padding: "15px", borderRadius: "8px", border: "1px solid #ddd" }}>
            <p><strong>ステータス:</strong> {status.status}</p>
            <p><strong>サーバー:</strong> {status.server.host}:{status.server.port}</p>
            <p><strong>カメラ数:</strong> {status.cameras}台</p>
            <p><strong>最終更新:</strong> {new Date(status.timestamp).toLocaleString('ja-JP')}</p>
          </div>
        </section>
      )}

      <section>
        <h2>カメラ一覧</h2>
        {cameras.length === 0 ? (
          <p>設定されているカメラがありません</p>
        ) : (
          <div style={{ display: "grid", gap: "15px", gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))" }}>
            {cameras.map((camera) => (
              <div
                key={camera.id}
                style={{
                  backgroundColor: "white",
                  padding: "20px",
                  borderRadius: "8px",
                  border: "1px solid #ddd",
                  boxShadow: "0 2px 4px rgba(0,0,0,0.1)"
                }}
              >
                <h3 style={{ margin: "0 0 10px 0" }}>{camera.name}</h3>
                <p><strong>ID:</strong> {camera.id}</p>
                <p><strong>デバイス:</strong> {camera.device}</p>
                <p><strong>解像度:</strong> {camera.settings.width}x{camera.settings.height}</p>
                <p><strong>フレームレート:</strong> {camera.settings.fps}fps</p>
                <p><strong>状態:</strong> <span style={{ 
                  color: camera.status === 'active' ? 'green' : 
                        camera.status === 'error' ? 'red' : 'gray'
                }}>{camera.status || 'inactive'}</span></p>
                
                <div style={{ 
                  width: "100%", 
                  height: "200px", 
                  backgroundColor: "#f0f0f0", 
                  display: "flex", 
                  alignItems: "center", 
                  justifyContent: "center",
                  marginTop: "15px",
                  borderRadius: "4px",
                  border: "1px solid #ddd"
                }}>
                  <p style={{ color: "#666", margin: 0 }}>
                    ストリーミング機能は実装中です
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
