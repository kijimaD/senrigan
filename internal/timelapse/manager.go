package timelapse

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"senrigan/internal/camera"
)

// Manager はタイムラプス機能全体を管理するインターフェース
type Manager interface {
	// システム制御
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// データ取得
	GetTimelapseVideos() ([]Video, error)
	GetTimelapseStatus() (StatusInfo, error)

	// 設定取得
	GetConfig() Config
}

// StatusInfo はタイムラプスシステムの状態情報
type StatusInfo struct {
	Enabled         bool      `json:"enabled"`
	ActiveSources   int       `json:"active_sources"`
	TotalVideos     int       `json:"total_videos"`
	StorageUsed     int64     `json:"storage_used"`
	CurrentVideo    string    `json:"current_video"`
	FrameBufferSize int       `json:"frame_buffer_size"`
	LastUpdate      time.Time `json:"last_update"`
}

// DefaultManager はTimelapseManagerのデフォルト実装
type DefaultManager struct {
	cameraManager camera.Manager
	capture       *Capture
	config        Config
	outputDir     string
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewDefaultManager は新しいDefaultManagerを作成する
func NewDefaultManager(cameraManager camera.Manager, outputDir string, config Config) *DefaultManager {
	return &DefaultManager{
		cameraManager: cameraManager,
		outputDir:     outputDir,
		config:        config,
	}
}

// Start はタイムラプス機能を開始する
func (m *DefaultManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled {
		log.Println("タイムラプス機能は無効です")
		return nil
	}

	// コンテキストを保存
	m.ctx, m.cancel = context.WithCancel(ctx)

	// 出力ディレクトリを作成
	if err := os.MkdirAll(m.outputDir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリの作成に失敗: %w", err)
	}

	// 利用可能な全ての映像ソースを取得
	videoSources := m.cameraManager.GetVideoSources()
	activeVideoSources := make([]camera.VideoSource, 0)

	for _, source := range videoSources {
		if source.GetStatus() == camera.StatusActive {
			activeVideoSources = append(activeVideoSources, source)
		}
	}

	if len(activeVideoSources) == 0 {
		return fmt.Errorf("アクティブな映像ソースがありません")
	}

	// 統合タイムラプスキャプチャを作成して開始
	m.capture = NewCapture(m.outputDir, m.config, activeVideoSources)

	if err := m.capture.Start(m.ctx); err != nil {
		return fmt.Errorf("タイムラプスキャプチャの開始に失敗: %w", err)
	}

	log.Printf("タイムラプスマネージャーを開始しました (%d個の映像ソースを結合)", len(activeVideoSources))
	return nil
}

// Stop はタイムラプス機能を停止する
func (m *DefaultManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}

	// タイムラプスキャプチャを停止
	if m.capture != nil {
		if err := m.capture.Stop(ctx); err != nil {
			log.Printf("タイムラプス停止に失敗: %v", err)
		}
		m.capture = nil
	}

	log.Println("タイムラプスマネージャーを停止しました")
	return nil
}

// GetTimelapseVideos は結合タイムラプス動画一覧を取得する
func (m *DefaultManager) GetTimelapseVideos() ([]Video, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.capture == nil {
		return []Video{}, nil
	}

	return m.capture.GetVideos()
}

// GetTimelapseStatus はタイムラプスシステムの状態を取得する
func (m *DefaultManager) GetTimelapseStatus() (StatusInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := StatusInfo{
		Enabled:     m.config.Enabled,
		TotalVideos: 0,
		StorageUsed: 0,
	}

	if m.capture != nil {
		videos, err := m.capture.GetVideos()
		if err == nil {
			status.TotalVideos = len(videos)

			// ストレージ使用量を計算
			for _, video := range videos {
				status.StorageUsed += video.FileSize
			}
		}

		// 現在の状態を取得
		captureStatus := m.capture.GetStatus()
		status.CurrentVideo = captureStatus.CurrentVideo
		status.FrameBufferSize = captureStatus.FrameBufferSize
		status.LastUpdate = captureStatus.LastUpdate
	}

	// アクティブソース数を取得
	videoSources := m.cameraManager.GetVideoSources()
	for _, source := range videoSources {
		if source.GetStatus() == camera.StatusActive {
			status.ActiveSources++
		}
	}

	return status, nil
}

// GetConfig は設定を取得する
func (m *DefaultManager) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}
