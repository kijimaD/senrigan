import { useRef, useEffect, useState, useCallback } from "react";
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
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const watchdogIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const lastLoadTimeRef = useRef<number>(0);
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);
  const [isReconnecting, setIsReconnecting] = useState(false);

  const maxReconnectAttempts = 10;
  const watchdogInterval = 5000; // 5秒ごとに生存確認
  const maxIdleTime = 15000; // 15秒以上フレームが来ない場合は再接続

  // クリーンアップ関数
  const cleanup = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (watchdogIntervalRef.current) {
      clearInterval(watchdogIntervalRef.current);
      watchdogIntervalRef.current = null;
    }
  }, []);

  // 再接続を試行する関数
  const attemptReconnect = useCallback(() => {
    if (reconnectAttempts >= maxReconnectAttempts) {
      console.error(`カメラ ${cameraId} の最大再接続試行回数を超過しました`);
      setHasError(true);
      setIsReconnecting(false);
      return;
    }

    const backoffDelay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000); // 指数バックオフ、最大30秒
    console.log(
      `カメラ ${cameraId} の再接続を ${backoffDelay}ms 後に試行します (試行 ${reconnectAttempts + 1}/${maxReconnectAttempts})`,
    );

    setIsReconnecting(true);
    setReconnectAttempts((prev) => prev + 1);

    reconnectTimeoutRef.current = setTimeout(() => {
      if (!imgRef.current) return;

      const img = imgRef.current;
      const streamUrl = buildStreamUrl(cameraId);

      // キャッシュバスターを追加して強制的に再読み込み
      const separator = streamUrl.includes("?") ? "&" : "?";
      img.src = `${streamUrl}${separator}t=${Date.now()}`;
    }, backoffDelay);
  }, [cameraId, reconnectAttempts, maxReconnectAttempts]);

  // ウォッチドッグタイマー：定期的に生存確認
  const startWatchdog = useCallback(() => {
    if (watchdogIntervalRef.current) {
      clearInterval(watchdogIntervalRef.current);
    }

    watchdogIntervalRef.current = setInterval(() => {
      const now = Date.now();
      const timeSinceLastLoad = now - lastLoadTimeRef.current;

      if (timeSinceLastLoad > maxIdleTime && !isReconnecting && !hasError) {
        console.warn(
          `カメラ ${cameraId} が ${timeSinceLastLoad}ms 間無応答です。再接続を開始します。`,
        );
        attemptReconnect();
      }
    }, watchdogInterval);
  }, [
    cameraId,
    isReconnecting,
    hasError,
    attemptReconnect,
    maxIdleTime,
    watchdogInterval,
  ]);

  // ストリームを初期化
  const initializeStream = useCallback(() => {
    if (!imgRef.current) return;

    const img = imgRef.current;
    const streamUrl = buildStreamUrl(cameraId);

    const handleLoad = () => {
      lastLoadTimeRef.current = Date.now();
      setIsLoading(false);
      setHasError(false);
      setIsReconnecting(false);
      setReconnectAttempts(0);

      // ウォッチドッグタイマーを開始
      startWatchdog();
    };

    const handleError = () => {
      setIsLoading(false);
      console.error(`カメラストリームの読み込みに失敗: ${cameraId}`);

      // 既に再接続中でなければ再接続を試行
      if (!isReconnecting) {
        attemptReconnect();
      }
    };

    img.addEventListener("load", handleLoad);
    img.addEventListener("error", handleError);

    // 初回ロード
    img.src = streamUrl;
    lastLoadTimeRef.current = Date.now();

    return () => {
      img.removeEventListener("load", handleLoad);
      img.removeEventListener("error", handleError);
      img.src = "";
    };
  }, [cameraId, isReconnecting, attemptReconnect, startWatchdog]);

  useEffect(() => {
    const cleanupStream = initializeStream();

    return () => {
      cleanup();
      if (cleanupStream) {
        cleanupStream();
      }
    };
  }, [cameraId, initializeStream, cleanup]);

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
      {(isLoading || isReconnecting) && (
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
            {isReconnecting
              ? `${cameraName} に再接続中... (${reconnectAttempts}/${maxReconnectAttempts})`
              : `${cameraName} を読み込み中...`}
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
