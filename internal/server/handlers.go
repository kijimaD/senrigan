package server

import (
	"net/http"
	"time"

	"senrigan/internal/config"
	"senrigan/internal/generated"

	"github.com/gin-gonic/gin"
)

// SenriganHandler は生成されたServerInterfaceを実装する
type SenriganHandler struct {
	config *config.Config
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
	cameras := make([]generated.CameraInfo, 0, len(h.config.Camera.Devices))

	for _, device := range h.config.Camera.Devices {
		// カメラ設定を生成されたスキーマに変換
		settings := generated.CameraSettings{
			Fps:    getFpsOrDefault(device.FPS, h.config.Camera.DefaultFPS),
			Width:  getIntOrDefault(device.Width, h.config.Camera.DefaultWidth),
			Height: getIntOrDefault(device.Height, h.config.Camera.DefaultHeight),
		}

		// カメラ情報を作成
		cameraInfo := generated.CameraInfo{
			Id:       device.ID,
			Name:     device.Name,
			Device:   device.Device,
			Settings: settings,
		}

		// カメラの状態を設定（現在は固定でinactive）
		status := generated.Inactive
		cameraInfo.Status = &status

		cameras = append(cameras, cameraInfo)
	}

	response := generated.CamerasResponse{
		Cameras: cameras,
	}

	c.JSON(http.StatusOK, response)
}

// GetCameraStream はカメラストリーム接続エンドポイントの実装
// 現在はWebSocket機能未実装のため、404を返す
func (h *SenriganHandler) GetCameraStream(c *gin.Context, cameraID string) {
	// カメラIDの存在確認
	found := false
	for _, device := range h.config.Camera.Devices {
		if device.ID == cameraID {
			found = true
			break
		}
	}

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
		Message:   "ストリーミング機能は未実装です",
		Details:   stringPtr("WebSocket機能の実装が必要です"),
		Timestamp: time.Now(),
	}
	c.JSON(http.StatusNotImplemented, errorResponse)
}

// ヘルパー関数

// getFpsOrDefault はFPS値を取得し、0の場合はデフォルト値を返す
func getFpsOrDefault(fps, defaultFps int) int {
	if fps <= 0 {
		return defaultFps
	}
	return fps
}

// getIntOrDefault は整数値を取得し、0の場合はデフォルト値を返す
func getIntOrDefault(value, defaultValue int) int {
	if value <= 0 {
		return defaultValue
	}
	return value
}

// stringPtr は文字列のポインタを返すヘルパー関数
func stringPtr(s string) *string {
	return &s
}
