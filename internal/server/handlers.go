package server

import (
	"net/http"
	"time"

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
		Status:    generated.Healthy,
		Timestamp: time.Now(),
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
		Cameras:   len(h.config.Camera.Devices),
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetCameras はカメラ一覧取得エンドポイントの実装
func (h *SenriganHandler) GetCameras(c *gin.Context) {
	// カメラマネージャーから現在のカメラ一覧を取得
	managedCameras := h.cameraManager.GetCameras()
	cameras := make([]generated.CameraInfo, 0, len(managedCameras))

	for _, cam := range managedCameras {
		// カメラ設定を生成されたスキーマに変換
		settings := generated.CameraSettings{
			Fps:    cam.FPS,
			Width:  cam.Width,
			Height: cam.Height,
		}

		// カメラ情報を作成
		cameraInfo := generated.CameraInfo{
			Id:       cam.ID,
			Name:     cam.Name,
			Device:   cam.Device,
			Settings: settings,
		}

		// カメラの状態を変換
		status := convertCameraStatus(cam.Status)
		cameraInfo.Status = &status

		cameras = append(cameras, cameraInfo)
	}

	response := generated.CamerasResponse{
		Cameras: cameras,
	}

	c.JSON(http.StatusOK, response)
}

// GetCameraStream はMJPEGストリーミングエンドポイントの実装
func (h *SenriganHandler) GetCameraStream(c *gin.Context, cameraID string) {
	// カメラIDの存在確認
	cam, found := h.cameraManager.GetCamera(cameraID)
	if !found {
		errorResponse := generated.ErrorResponse{
			Error:     "camera_not_found",
			Message:   "指定されたカメラが見つかりません",
			Timestamp: time.Now(),
		}
		c.JSON(http.StatusNotFound, errorResponse)
		return
	}

	// カメラがアクティブか確認
	if cam.Status != camera.StatusActive {
		errorResponse := generated.ErrorResponse{
			Error:     "camera_not_active",
			Message:   "カメラがアクティブではありません",
			Timestamp: time.Now(),
		}
		c.JSON(http.StatusServiceUnavailable, errorResponse)
		return
	}

	// MJPEGストリーミングを配信
	h.streamMJPEG(c, cameraID)
}

// GetCameraWebSocket はWebSocketストリーミングエンドポイントの実装（未実装）
func (h *SenriganHandler) GetCameraWebSocket(c *gin.Context, cameraID string) {
	// カメラIDの存在確認
	_, found := h.cameraManager.GetCamera(cameraID)
	if !found {
		errorResponse := generated.ErrorResponse{
			Error:     "camera_not_found",
			Message:   "指定されたカメラが見つかりません",
			Timestamp: time.Now(),
		}
		c.JSON(http.StatusNotFound, errorResponse)
		return
	}

	// WebSocket機能は未実装
	errorResponse := generated.ErrorResponse{
		Error:     "not_implemented",
		Message:   "WebSocketストリーミング機能は未実装です",
		Details:   stringPtr("将来的に実装予定です"),
		Timestamp: time.Now(),
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
	// カメラサービスを取得
	service, exists := h.cameraManager.GetCameraService(cameraID)
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
	frameChan := service.GetFrameChannel()

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
