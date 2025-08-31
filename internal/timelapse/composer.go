package timelapse

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"sort"
	"time"

	"senrigan/internal/camera"
)

// FrameComposer は複数の映像ソースのフレームを結合する
type FrameComposer struct {
	outputWidth  int
	outputHeight int
	quality      int
}

// NewFrameComposer は新しいFrameComposerを作成する
func NewFrameComposer(outputWidth, outputHeight, quality int) *FrameComposer {
	return &FrameComposer{
		outputWidth:  outputWidth,
		outputHeight: outputHeight,
		quality:      quality,
	}
}

// ComposeFrames は複数の映像ソースからフレームを取得して結合する
func (fc *FrameComposer) ComposeFrames(ctx context.Context, videoSources []camera.VideoSource) (CombinedFrame, error) {
	timestamp := time.Now()
	sourceFrames := make(map[string]SourceFrame)
	sourceTypeMap := make(map[string]camera.VideoSourceType) // タイプ情報を保持

	// 各映像ソースからフレームを取得
	for _, source := range videoSources {
		if source.GetStatus() != camera.StatusActive {
			continue // 非アクティブなソースはスキップ
		}

		info := source.GetInfo()
		sourceTypeMap[info.ID] = info.Type // タイプ情報を保存

		frameData, err := source.CaptureFrameForTimelapse(ctx)
		if err != nil {
			log.Printf("映像ソース %s のフレーム取得に失敗: %v", info.ID, err)
			continue // エラーがあってもスキップして続行
		}

		if len(frameData) > 0 {
			sourceFrames[info.ID] = SourceFrame{
				SourceID:  info.ID,
				Timestamp: timestamp,
				Data:      frameData,
				Size:      len(frameData),
			}
		}
	}

	if len(sourceFrames) == 0 {
		return CombinedFrame{}, fmt.Errorf("有効なフレームが取得できませんでした")
	}

	// フレームを結合
	composedData, err := fc.combineFrames(sourceFrames, sourceTypeMap, videoSources)
	if err != nil {
		return CombinedFrame{}, fmt.Errorf("フレーム結合に失敗: %w", err)
	}

	return CombinedFrame{
		Timestamp:    timestamp,
		SourceFrames: sourceFrames,
		ComposedData: composedData,
		Size:         len(composedData),
	}, nil
}

// combineFrames は複数のJPEGフレームを1つの画像に結合する
func (fc *FrameComposer) combineFrames(sourceFrames map[string]SourceFrame, _ map[string]camera.VideoSourceType, videoSources []camera.VideoSource) ([]byte, error) {
	if len(sourceFrames) == 0 {
		return nil, fmt.Errorf("結合するフレームがありません")
	}

	// 結合レイアウトを決定
	layout := fc.calculateLayout(len(sourceFrames))

	// 出力画像を作成
	outputImg := image.NewRGBA(image.Rect(0, 0, fc.outputWidth, fc.outputHeight))

	// カメラ名でソートしてカメラ位置を固定（シンプルな名前順）
	type SourceInfo struct {
		ID   string
		Name string
	}

	sourceInfos := make([]SourceInfo, 0, len(sourceFrames))
	for sourceID := range sourceFrames {
		// sourceTypeMapから名前を取得するためにVideoSourceを探す
		name := sourceID // デフォルトはID
		for _, source := range videoSources {
			if source.GetInfo().ID == sourceID {
				name = source.GetInfo().Name
				break
			}
		}
		sourceInfos = append(sourceInfos, SourceInfo{ID: sourceID, Name: name})
	}

	// 名前でソート、同じ名前の場合はIDでソート
	sort.SliceStable(sourceInfos, func(i, j int) bool {
		if sourceInfos[i].Name == sourceInfos[j].Name {
			return sourceInfos[i].ID < sourceInfos[j].ID
		}
		return sourceInfos[i].Name < sourceInfos[j].Name
	})

	// ソートされた順序でIDリストを作成
	sortedSourceIDs := make([]string, len(sourceInfos))
	for i, info := range sourceInfos {
		sortedSourceIDs[i] = info.ID
	}

	// 各フレームをソート順で配置
	frameIndex := 0
	for _, sourceID := range sortedSourceIDs {
		sourceFrame := sourceFrames[sourceID]
		if len(sourceFrame.Data) == 0 {
			continue
		}

		// JPEGデータを画像にデコード
		img, err := jpeg.Decode(bytes.NewReader(sourceFrame.Data))
		if err != nil {
			log.Printf("JPEG デコードエラー (ソース %s): %v", sourceFrame.SourceID, err)
			continue
		}

		// 配置位置を計算
		pos := fc.calculatePosition(frameIndex, layout)

		// 画像を配置
		fc.drawImageAt(outputImg, img, pos)

		frameIndex++
	}

	// 結合画像をJPEGにエンコード
	var buf bytes.Buffer
	encodeOptions := &jpeg.Options{Quality: fc.quality * 20} // 1-5 を 20-100 に変換

	if err := jpeg.Encode(&buf, outputImg, encodeOptions); err != nil {
		return nil, fmt.Errorf("JPEG エンコードに失敗: %w", err)
	}

	return buf.Bytes(), nil
}

// LayoutInfo はレイアウト情報
type LayoutInfo struct {
	Cols       int
	Rows       int
	CellWidth  int
	CellHeight int
}

// calculateLayout はフレーム数に基づいてレイアウトを計算する
func (fc *FrameComposer) calculateLayout(frameCount int) LayoutInfo {
	var cols, rows int

	switch frameCount {
	case 1:
		cols, rows = 1, 1
	case 2:
		cols, rows = 2, 1
	case 3:
		cols, rows = 2, 2 // 3つの場合も2x2で1つ空き
	case 4:
		cols, rows = 2, 2
	default:
		// 5つ以上の場合は適切な格子を計算
		cols = int(float64(frameCount)*0.6) + 1 // 横を多めに
		rows = (frameCount + cols - 1) / cols
	}

	return LayoutInfo{
		Cols:       cols,
		Rows:       rows,
		CellWidth:  fc.outputWidth / cols,
		CellHeight: fc.outputHeight / rows,
	}
}

// Position は配置位置
type Position struct {
	X, Y          int
	Width, Height int
}

// calculatePosition は指定したインデックスの配置位置を計算する
func (fc *FrameComposer) calculatePosition(index int, layout LayoutInfo) Position {
	row := index / layout.Cols
	col := index % layout.Cols

	return Position{
		X:      col * layout.CellWidth,
		Y:      row * layout.CellHeight,
		Width:  layout.CellWidth,
		Height: layout.CellHeight,
	}
}

// drawImageAt は指定した位置に画像を描画する（簡単なリサイズあり）
func (fc *FrameComposer) drawImageAt(dst *image.RGBA, src image.Image, pos Position) {
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// 簡単なニアレストネイバー法でリサイズしながら配置
	for y := 0; y < pos.Height; y++ {
		for x := 0; x < pos.Width; x++ {
			// ソース画像の対応する座標を計算
			srcX := x * srcWidth / pos.Width
			srcY := y * srcHeight / pos.Height

			if srcX < srcWidth && srcY < srcHeight {
				color := src.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY)
				dst.Set(pos.X+x, pos.Y+y, color)
			}
		}
	}
}
