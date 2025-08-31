import { useState, useRef } from "react";

interface TimelapsePlayerProps {
  videoUrl: string;
  videoName: string;
}

export function TimelapsePlayer({ videoUrl, videoName }: TimelapsePlayerProps) {
  const [error, setError] = useState<string | null>(null);
  const videoRef = useRef<HTMLVideoElement>(null);

  const handleVideoError = () => {
    setError("動画の読み込みに失敗しました");
  };

  const handleVideoLoad = () => {
    setError(null);
  };

  return (
    <div style={{ marginTop: "10px" }}>
      <h4 style={{ margin: "0 0 10px 0", fontSize: "14px" }}>{videoName}</h4>
      {error && (
        <p style={{ color: "red", fontSize: "12px", margin: "5px 0" }}>
          {error}
        </p>
      )}
      <video
        ref={videoRef}
        src={videoUrl}
        style={{
          width: "100%",
          maxHeight: "400px",
          backgroundColor: "#000",
          borderRadius: "4px",
        }}
        controls
        onError={handleVideoError}
        onLoadedData={handleVideoLoad}
      >
        お使いのブラウザは動画タグをサポートしていません。
      </video>
    </div>
  );
}
