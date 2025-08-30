package server

import (
	"net/http"
	"sort"

	"senrigan/internal/camera"
	"senrigan/internal/config"
	"senrigan/internal/generated"

	"github.com/gin-gonic/gin"
)

// SenriganHandler は生成されたServerInterfaceを実装する
type SenriganHandler struct {
	config        *config.Config
	cameraManager camera.Manager
}

// HealthCheck はヘルスチェックエンドポイントの実装
func (h *SenriganHandler) HealthCheck(c *gin.Context) {
	response := generated.HealthResponse{
		Status: generated.Healthy,
	}

	c.JSON(http.StatusOK, response)
}

// GetStatus はシステム状態取得エンドポイントの実装
func (h *SenriganHandler) GetStatus(c *gin.Context) {
	response := generated.StatusResponse{
		Status: generated.Running,
		Server: generated.ServerInfo{
			Host: h.config.Server.Host,
			Port: h.config.Server.Port,
		},
		Cameras: len(h.cameraManager.GetVideoSources()),
	}

	c.JSON(http.StatusOK, response)
}

// GetCameras はカメラ一覧取得エンドポイントの実装
func (h *SenriganHandler) GetCameras(c *gin.Context) {
	// VideoSourceマネージャーから現在のVideoSource一覧を取得
	videoSources := h.cameraManager.GetVideoSources()
	cameras := make([]generated.CameraInfo, 0, len(videoSources))

	for _, source := range videoSources {
		info := source.GetInfo()
		settings := source.GetCurrentSettings()

		// カメラ設定を生成されたスキーマに変換
		cameraSettings := generated.CameraSettings{
			Fps:    settings.FrameRate,
			Width:  settings.Width,
			Height: settings.Height,
		}

		// カメラ情報を作成
		cameraInfo := generated.CameraInfo{
			Id:       info.ID,
			Name:     info.Name,
			Device:   info.Device,
			Settings: cameraSettings,
		}

		// カメラの状態を変換
		status := convertCameraStatus(source.GetStatus())
		cameraInfo.Status = &status

		cameras = append(cameras, cameraInfo)
	}

	// カメラを名前順でソート
	sort.Slice(cameras, func(i, j int) bool {
		return cameras[i].Name < cameras[j].Name
	})

	response := generated.CamerasResponse{
		Cameras: cameras,
	}

	c.JSON(http.StatusOK, response)
}

// GetCameraStream はMJPEGストリーミングエンドポイントの実装
func (h *SenriganHandler) GetCameraStream(c *gin.Context, cameraID string) {
	// VideoSourceの存在確認
	source, found := h.cameraManager.GetVideoSource(cameraID)
	if !found {
		errorResponse := generated.ErrorResponse{
			Error:   "camera_not_found",
			Message: "指定されたカメラが見つかりません",
		}
		c.JSON(http.StatusNotFound, errorResponse)
		return
	}

	// VideoSourceがアクティブか確認
	if source.GetStatus() != camera.StatusActive {
		errorResponse := generated.ErrorResponse{
			Error:   "camera_not_active",
			Message: "カメラがアクティブではありません",
		}
		c.JSON(http.StatusServiceUnavailable, errorResponse)
		return
	}

	// MJPEGストリーミングを配信
	h.streamMJPEG(c, cameraID)
}

// GetCameraWebSocket はWebSocketストリーミングエンドポイントの実装（未実装）
func (h *SenriganHandler) GetCameraWebSocket(c *gin.Context, cameraID string) {
	// VideoSourceの存在確認
	_, found := h.cameraManager.GetVideoSource(cameraID)
	if !found {
		errorResponse := generated.ErrorResponse{
			Error:   "camera_not_found",
			Message: "指定されたカメラが見つかりません",
		}
		c.JSON(http.StatusNotFound, errorResponse)
		return
	}

	// WebSocket機能は未実装
	errorResponse := generated.ErrorResponse{
		Error:   "not_implemented",
		Message: "WebSocketストリーミング機能は未実装です",
		Details: stringPtr("将来的に実装予定です"),
	}
	c.JSON(http.StatusNotImplemented, errorResponse)
}

// ヘルパー関数

// convertCameraStatus はカメラステータスを変換する
func convertCameraStatus(status camera.Status) generated.CameraInfoStatus {
	switch status {
	case camera.StatusActive:
		return generated.Active
	case camera.StatusInactive:
		return generated.Inactive
	case camera.StatusError:
		return generated.Error
	default:
		return generated.Inactive
	}
}

// stringPtr は文字列のポインタを返すヘルパー関数
func stringPtr(s string) *string {
	return &s
}

// streamMJPEG はMJPEGストリームを配信する
func (h *SenriganHandler) streamMJPEG(c *gin.Context, cameraID string) {
	// VideoSourceを取得
	source, exists := h.cameraManager.GetVideoSource(cameraID)
	if !exists {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// レスポンスヘッダーを設定
	c.Header("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// レスポンスライターを取得
	writer := c.Writer
	flusher, ok := writer.(http.Flusher)
	if !ok {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// フレームチャンネルを取得
	frameChan := source.GetFrameChannel()

	// クライアント切断を検知するためのコンテキスト
	clientGone := c.Request.Context().Done()

	// ストリーミングループ
	for {
		select {
		case <-clientGone:
			// クライアントが切断された
			return

		case frame, ok := <-frameChan:
			if !ok {
				// チャンネルがクローズされた
				return
			}

			// MJPEGフレームを書き込み
			_, err := writer.Write([]byte("--frame\r\n"))
			if err != nil {
				return
			}

			_, err = writer.Write([]byte("Content-Type: image/jpeg\r\n\r\n"))
			if err != nil {
				return
			}

			_, err = writer.Write(frame)
			if err != nil {
				return
			}

			_, err = writer.Write([]byte("\r\n"))
			if err != nil {
				return
			}

			// バッファをフラッシュ
			flusher.Flush()
		}
	}
}
