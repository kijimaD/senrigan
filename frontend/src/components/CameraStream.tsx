import { useRef, useEffect, useState } from "react";
import { buildStreamUrl } from "../config/api";

interface CameraStreamProps {
  cameraId: string;
  cameraName: string;
  width?: number;
  height?: number;
}

export function CameraStream({
  cameraId,
  cameraName,
  width = 640,
  height = 480,
}: CameraStreamProps) {
  const imgRef = useRef<HTMLImageElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);

  useEffect(() => {
    if (!imgRef.current) return;

    const img = imgRef.current;

    // 共通設定を使用してストリームURLを構築
    const streamUrl = buildStreamUrl(cameraId);

    // MJPEGストリームを直接img要素のsrcに設定
    img.src = streamUrl;

    const handleLoad = () => {
      setIsLoading(false);
      setHasError(false);
    };

    const handleError = () => {
      setIsLoading(false);
      setHasError(true);
      console.error(`カメラストリームの読み込みに失敗: ${cameraId}`);
    };

    img.addEventListener("load", handleLoad);
    img.addEventListener("error", handleError);

    return () => {
      // クリーンアップ: ストリームを停止
      img.removeEventListener("load", handleLoad);
      img.removeEventListener("error", handleError);
      img.src = "";
    };
  }, [cameraId]);

  // フルスクリーンのトグル
  const toggleFullscreen = async () => {
    if (!containerRef.current) return;

    try {
      if (!document.fullscreenElement) {
        await containerRef.current.requestFullscreen();
        setIsFullscreen(true);
      } else {
        await document.exitFullscreen();
        setIsFullscreen(false);
      }
    } catch (err) {
      console.error("フルスクリーンの切り替えに失敗:", err);
    }
  };

  // フルスクリーンの変更を監視
  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };

    document.addEventListener("fullscreenchange", handleFullscreenChange);
    return () => {
      document.removeEventListener("fullscreenchange", handleFullscreenChange);
    };
  }, []);

  return (
    <div
      ref={containerRef}
      style={{
        position: "relative",
        width: "100%",
        paddingBottom: isFullscreen ? "0" : `${(height / width) * 100}%`,
        height: isFullscreen ? "100vh" : "auto",
        backgroundColor: "#000",
        borderRadius: isFullscreen ? "0" : "4px",
        overflow: "hidden",
      }}
    >
      {isLoading && (
        <div
          style={{
            position: "absolute",
            top: "50%",
            left: "50%",
            transform: "translate(-50%, -50%)",
            color: "#fff",
            fontSize: "14px",
            textAlign: "center",
          }}
        >
          <div style={{ marginBottom: "10px" }}>
            {cameraName} を読み込み中...
          </div>
          <div
            style={{
              width: "40px",
              height: "40px",
              border: "3px solid rgba(255,255,255,0.3)",
              borderTop: "3px solid #fff",
              borderRadius: "50%",
              margin: "0 auto",
              animation: "spin 1s linear infinite",
            }}
          />
        </div>
      )}

      {hasError && (
        <div
          style={{
            position: "absolute",
            top: "50%",
            left: "50%",
            transform: "translate(-50%, -50%)",
            color: "#ff6b6b",
            fontSize: "14px",
            textAlign: "center",
          }}
        >
          <div style={{ marginBottom: "10px" }}>⚠</div>
          <div>ストリームに接続できません</div>
          <div style={{ fontSize: "12px", color: "#aaa", marginTop: "5px" }}>
            カメラID: {cameraId}
          </div>
        </div>
      )}

      <img
        ref={imgRef}
        alt={`${cameraName} ストリーム`}
        style={{
          position: "absolute",
          top: 0,
          left: 0,
          width: "100%",
          height: "100%",
          objectFit: "contain",
          display: hasError ? "none" : "block",
        }}
      />

      {/* フルスクリーンボタン */}
      {!hasError && !isLoading && (
        <button
          onClick={toggleFullscreen}
          style={{
            position: "absolute",
            top: "10px",
            right: "10px",
            width: "40px",
            height: "40px",
            borderRadius: "4px",
            backgroundColor: "rgba(0, 0, 0, 0.5)",
            border: "1px solid rgba(255, 255, 255, 0.3)",
            color: "#fff",
            cursor: "pointer",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            fontSize: "20px",
            transition: "background-color 0.2s",
            zIndex: 10,
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.backgroundColor = "rgba(0, 0, 0, 0.7)";
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.backgroundColor = "rgba(0, 0, 0, 0.5)";
          }}
          title={isFullscreen ? "フルスクリーンを終了" : "フルスクリーン"}
        >
          {isFullscreen ? "◱" : "◰"}
        </button>
      )}

      {/* アニメーション用のスタイル */}
      <style>{`
        @keyframes spin {
          0% { transform: rotate(0deg); }
          100% { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  );
}
