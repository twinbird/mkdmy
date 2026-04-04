package generator

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/twinbird/mkdmy/internal/cli"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	labelFontOnce sync.Once
	labelFont     *opentype.Font
	labelFontErr  error
)

func createImageFile(path string, opts cli.Options, index int) error {
	var (
		data []byte
		err  error
	)

	switch opts.ContentMode {
	case cli.ContentModeRandom:
		data, err = buildRandomPNG(opts.SizeBytes)
	case cli.ContentModeTemplate:
		data, err = buildLabeledPNG(opts.SizeBytes, templateLabel(opts.Content, index))
	default:
		return fmt.Errorf("unsupported png mode %q", opts.ContentMode)
	}
	if err := ensureParentDir(path); err != nil {
		return fmt.Errorf("prepare parent dir: %w", err)
	}
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return err
	}

	return nil
}

func buildRandomPNG(targetSize int64) ([]byte, error) {
	return buildPNG(targetSize, encodeRandomPNG)
}

func buildLabeledPNG(targetSize int64, label string) ([]byte, error) {
	return buildPNG(targetSize, func(side int) ([]byte, error) {
		return encodeLabeledPNG(side, label)
	})
}

func buildPNG(targetSize int64, encode func(side int) ([]byte, error)) ([]byte, error) {
	side := estimateImageSide(targetSize)
	bestSide := side
	var best []byte
	bestDelta := int64(math.MaxInt64)

	for attempt := 0; attempt < 5; attempt++ {
		data, err := encode(side)
		if err != nil {
			return nil, err
		}

		delta := absInt64(int64(len(data)) - targetSize)
		if best == nil || delta < bestDelta {
			best = append(best[:0], data...)
			bestDelta = delta
			bestSide = side
		}

		if targetSize <= 0 {
			break
		}
		if delta == 0 {
			break
		}

		scale := math.Sqrt(float64(targetSize) / float64(len(data)))
		nextSide := int(math.Round(float64(side) * scale))
		if nextSide < 1 {
			nextSide = 1
		}
		if nextSide == side {
			if int64(len(data)) < targetSize {
				nextSide++
			} else if side > 1 {
				nextSide--
			}
		}
		if nextSide == side || nextSide == bestSide && attempt > 0 {
			break
		}
		side = nextSide
	}

	if len(best) == 0 {
		return nil, fmt.Errorf("failed to encode png")
	}

	return best, nil
}

func encodeRandomPNG(side int) ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, side, side))

	rgb := make([]byte, side*side*3)
	if _, err := rand.Read(rgb); err != nil {
		return nil, err
	}

	for pixel := 0; pixel < side*side; pixel++ {
		src := pixel * 3
		dst := pixel * 4
		img.Pix[dst] = rgb[src]
		img.Pix[dst+1] = rgb[src+1]
		img.Pix[dst+2] = rgb[src+2]
		img.Pix[dst+3] = 0xFF
	}

	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func encodeLabeledPNG(side int, label string) ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	imagedraw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, imagedraw.Src)

	lines := splitLabelLines(label)
	face, lineHeight, err := newLabelFace(side, lines)
	if err != nil {
		return nil, err
	}
	if closer, ok := face.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	drawer := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: face,
	}

	metrics := face.Metrics()
	ascent := metrics.Ascent.Ceil()
	blockHeight := lineHeight * len(lines)
	y := (side-blockHeight)/2 + ascent

	for _, line := range lines {
		width := drawer.MeasureString(line).Ceil()
		x := (side - width) / 2
		drawer.Dot = fixed.P(x, y)
		drawer.DrawString(line)
		y += lineHeight
	}

	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func newLabelFace(side int, lines []string) (font.Face, int, error) {
	ttf, err := loadLabelFont()
	if err != nil {
		return nil, 0, err
	}

	minSize := 8.0
	maxSize := math.Max(minSize, float64(side)*0.45)
	bestSize := minSize

	for size := maxSize; size >= minSize; size *= 0.85 {
		face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			return nil, 0, err
		}

		metrics := face.Metrics()
		lineHeight := (metrics.Ascent + metrics.Descent).Ceil()
		blockHeight := lineHeight * len(lines)

		if labelLinesFit(face, lines, side, blockHeight) {
			return face, lineHeight, nil
		}

		if closer, ok := face.(interface{ Close() error }); ok {
			closer.Close()
		}
		bestSize = size * 0.85
	}

	if bestSize < minSize {
		bestSize = minSize
	}

	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
		Size:    bestSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, 0, err
	}

	metrics := face.Metrics()
	lineHeight := (metrics.Ascent + metrics.Descent).Ceil()

	return face, lineHeight, nil
}

func labelLinesFit(face font.Face, lines []string, side int, blockHeight int) bool {
	if blockHeight > int(float64(side)*0.75) {
		return false
	}

	drawer := font.Drawer{Face: face}
	for _, line := range lines {
		width := drawer.MeasureString(line).Ceil()
		if width > int(float64(side)*0.8) {
			return false
		}
	}

	return true
}

func loadLabelFont() (*opentype.Font, error) {
	labelFontOnce.Do(func() {
		labelFont, labelFontErr = opentype.Parse(goregular.TTF)
	})
	return labelFont, labelFontErr
}

func templateLabel(content string, index int) string {
	if strings.TrimSpace(content) != "" {
		return fmt.Sprintf(content, index)
	}
	return strconv.Itoa(index)
}

func splitLabelLines(label string) []string {
	parts := strings.Split(label, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		lines = append(lines, part)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func estimateImageSide(targetSize int64) int {
	if targetSize <= 0 {
		return 1
	}

	usable := float64(targetSize - 128)
	if usable < 16 {
		usable = 16
	}

	side := int(math.Sqrt(usable / 4))
	if side < 1 {
		return 1
	}
	return side
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
