import { useEffect, useState } from "react";
import { StatusApi, CameraApi, TimelapseApi } from "../generated/api";
import type {
  SystemStatusResponse,
  CameraInfo,
  ErrorResponse,
  StatusResponse as TimelapseStatusResponse,
  Config,
  Video,
} from "../generated/api";
import { AxiosError } from "axios";
import { CameraStream } from "../components/CameraStream";
import { TimelapsePlayer } from "../components/TimelapsePlayer";
import { API_CONFIG } from "../config/api";

export function MainPage() {
  const [status, setStatus] = useState<SystemStatusResponse | null>(null);
  const [cameras, setCameras] = useState<CameraInfo[]>([]);
  const [timelapseStatus, setTimelapseStatus] =
    useState<TimelapseStatusResponse | null>(null);
  const [timelapseConfig, setTimelapseConfig] = useState<Config | null>(null);
  const [timelapseVideos, setTimelapseVideos] = useState<Video[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        // 共通のAPI設定を使用
        const statusApi = new StatusApi(API_CONFIG);
        const cameraApi = new CameraApi(API_CONFIG);
        const timelapseApi = new TimelapseApi(API_CONFIG);

        // システム状態を取得
        const statusResponse = await statusApi.getStatus();
        setStatus(statusResponse.data);

        // カメラ一覧を取得
        const camerasResponse = await cameraApi.getCameras();
        setCameras(camerasResponse.data.cameras);

        // タイムラプス情報を取得
        const timelapseStatusResponse = await timelapseApi.getTimelapseStatus();
        setTimelapseStatus(timelapseStatusResponse.data);

        const timelapseConfigResponse = await timelapseApi.getTimelapseConfig();
        setTimelapseConfig(timelapseConfigResponse.data);

        const timelapseVideosResponse = await timelapseApi.getTimelapseVideos();
        setTimelapseVideos(timelapseVideosResponse.data);
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

    // 10秒ごとにタイムラプス情報を更新
    const interval = setInterval(async () => {
      try {
        const timelapseApi = new TimelapseApi(API_CONFIG);
        const timelapseStatusResponse = await timelapseApi.getTimelapseStatus();
        setTimelapseStatus(timelapseStatusResponse.data);

        const timelapseVideosResponse = await timelapseApi.getTimelapseVideos();
        setTimelapseVideos(timelapseVideosResponse.data);
      } catch (err) {
        console.error("Timelapse update error:", err);
      }
    }, 10000);

    return () => clearInterval(interval);
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
          <div
            style={{
              backgroundColor: "white",
              padding: "15px",
              borderRadius: "8px",
              border: "1px solid #ddd",
            }}
          >
            <p>
              <strong>ステータス:</strong> {status.status}
            </p>
            <p>
              <strong>サーバー:</strong> {status.server.host}:
              {status.server.port}
            </p>
            <p>
              <strong>カメラ数:</strong> {status.cameras}台
            </p>
          </div>
        </section>
      )}

      {timelapseStatus && timelapseConfig && (
        <section style={{ marginBottom: "30px" }}>
          <h2>タイムラプス</h2>
          <div
            style={{
              backgroundColor: "white",
              padding: "15px",
              borderRadius: "8px",
              border: "1px solid #ddd",
              marginBottom: "20px",
            }}
          >
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "1fr 1fr",
                gap: "20px",
              }}
            >
              <div>
                <h3 style={{ margin: "0 0 10px 0" }}>ステータス</h3>
                <p>
                  <strong>有効:</strong>{" "}
                  {timelapseStatus.enabled ? "はい" : "いいえ"}
                </p>
                <p>
                  <strong>アクティブなソース:</strong>{" "}
                  {timelapseStatus.active_sources}個
                </p>
                <p>
                  <strong>フレームバッファ:</strong>{" "}
                  {timelapseStatus.frame_buffer_size}フレーム
                </p>
                <p>
                  <strong>動画数:</strong> {timelapseStatus.total_videos}本
                </p>
              </div>
              <div>
                <h3 style={{ margin: "0 0 10px 0" }}>設定</h3>
                <p>
                  <strong>撮影間隔:</strong> {timelapseConfig.capture_interval}
                </p>
                <p>
                  <strong>更新間隔:</strong> {timelapseConfig.update_interval}
                </p>
                <p>
                  <strong>解像度:</strong>{" "}
                  {timelapseConfig.resolution?.width || 0}x
                  {timelapseConfig.resolution?.height || 0}
                </p>
                <p>
                  <strong>品質:</strong> {timelapseConfig.quality}
                </p>
              </div>
            </div>
          </div>

          {timelapseVideos.length > 0 && (
            <div>
              <h3>動画一覧</h3>
              <div style={{ display: "grid", gap: "15px" }}>
                {timelapseVideos.map((video, index) => {
                  // Generate video URL and name
                  const fileName =
                    video.file_path.split("/").pop() || `video_${index}.mp4`;
                  const videoUrl = `/api/timelapse/video/${fileName}`;
                  const videoName = `${new Date(video.date).toLocaleString()} (${(video.file_size / 1024).toFixed(1)}KB)`;

                  return (
                    <div
                      key={index}
                      style={{
                        backgroundColor: "white",
                        padding: "15px",
                        borderRadius: "8px",
                        border: "1px solid #ddd",
                        boxShadow: "0 2px 4px rgba(0,0,0,0.1)",
                      }}
                    >
                      <div style={{ marginBottom: "10px" }}>
                        <p>
                          <strong>日付:</strong>{" "}
                          {new Date(video.date).toLocaleString()}
                        </p>
                        <p>
                          <strong>フレーム数:</strong> {video.frame_count}
                        </p>
                        <p>
                          <strong>ファイルサイズ:</strong>{" "}
                          {(video.file_size / 1024).toFixed(1)}KB
                        </p>
                        <p>
                          <strong>ステータス:</strong> {video.status}
                        </p>
                      </div>

                      <TimelapsePlayer
                        videoUrl={videoUrl}
                        videoName={videoName}
                      />

                      {video.status !== "completed" && (
                        <div
                          style={{
                            padding: "8px 12px",
                            backgroundColor: "#fff3cd",
                            borderRadius: "4px",
                            border: "1px solid #ffeaa7",
                            color: "#856404",
                            fontSize: "12px",
                            marginTop: "8px",
                            textAlign: "center",
                          }}
                        >
                          ⚠️ 処理中のため、動画は途中までの内容です
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {timelapseVideos.length === 0 && (
            <p style={{ fontStyle: "italic", color: "#666" }}>
              まだタイムラプス動画は生成されていません。
              {timelapseConfig.update_interval} ごとに新しい動画が作成されます。
            </p>
          )}
        </section>
      )}

      <section>
        <h2>カメラ一覧</h2>
        {cameras.length === 0 ? (
          <p>設定されているカメラがありません</p>
        ) : (
          <div
            style={{
              display: "grid",
              gap: "15px",
              gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))",
            }}
          >
            {cameras.map((camera) => (
              <div
                key={camera.id}
                style={{
                  backgroundColor: "white",
                  padding: "20px",
                  borderRadius: "8px",
                  border: "1px solid #ddd",
                  boxShadow: "0 2px 4px rgba(0,0,0,0.1)",
                }}
              >
                <h3 style={{ margin: "0 0 10px 0" }}>{camera.name}</h3>
                <p>
                  <strong>ID:</strong> {camera.id}
                </p>
                <p>
                  <strong>デバイス:</strong> {camera.device}
                </p>
                <p>
                  <strong>解像度:</strong> {camera.settings.width}x
                  {camera.settings.height}
                </p>
                <p>
                  <strong>フレームレート:</strong> {camera.settings.fps}fps
                </p>
                <p>
                  <strong>状態:</strong>{" "}
                  <span
                    style={{
                      color:
                        camera.status === "active"
                          ? "green"
                          : camera.status === "error"
                            ? "red"
                            : "gray",
                    }}
                  >
                    {camera.status || "inactive"}
                  </span>
                </p>

                {camera.status === "active" ? (
                  <div style={{ marginTop: "15px" }}>
                    <CameraStream
                      cameraId={camera.id}
                      cameraName={camera.name}
                    />
                  </div>
                ) : (
                  <div
                    style={{
                      width: "100%",
                      height: "200px",
                      backgroundColor: "#f0f0f0",
                      display: "flex",
                      alignItems: "center",
                      justifyContent: "center",
                      marginTop: "15px",
                      borderRadius: "4px",
                      border: "1px solid #ddd",
                    }}
                  >
                    <p style={{ color: "#666", margin: 0 }}>
                      カメラが非アクティブです
                    </p>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
