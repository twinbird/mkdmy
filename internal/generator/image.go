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
	"path/filepath"
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
	case cli.ContentModeIndex:
		data, err = buildIndexedPNG(opts.SizeBytes, sequenceLabel(filepath.Base(path), index))
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

func buildIndexedPNG(targetSize int64, label string) ([]byte, error) {
	return buildPNG(targetSize, func(side int) ([]byte, error) {
		return encodeIndexedPNG(side, label)
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

func encodeIndexedPNG(side int, label string) ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	imagedraw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, imagedraw.Src)

	face, width, textHeight, ascent, err := newLabelFace(side, label)
	if err != nil {
		return nil, err
	}
	if closer, ok := face.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	x := (side - width) / 2
	y := (side-textHeight)/2 + ascent

	drawer := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	drawer.DrawString(label)

	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func newLabelFace(side int, label string) (font.Face, int, int, int, error) {
	ttf, err := loadLabelFont()
	if err != nil {
		return nil, 0, 0, 0, err
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
			return nil, 0, 0, 0, err
		}

		drawer := font.Drawer{Face: face}
		width := drawer.MeasureString(label).Ceil()
		metrics := face.Metrics()
		textHeight := (metrics.Ascent + metrics.Descent).Ceil()

		if width <= int(float64(side)*0.8) && textHeight <= int(float64(side)*0.6) {
			ascent := metrics.Ascent.Ceil()
			return face, width, textHeight, ascent, nil
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
		return nil, 0, 0, 0, err
	}

	drawer := font.Drawer{Face: face}
	width := drawer.MeasureString(label).Ceil()
	metrics := face.Metrics()
	textHeight := (metrics.Ascent + metrics.Descent).Ceil()
	ascent := metrics.Ascent.Ceil()

	return face, width, textHeight, ascent, nil
}

func loadLabelFont() (*opentype.Font, error) {
	labelFontOnce.Do(func() {
		labelFont, labelFontErr = opentype.Parse(goregular.TTF)
	})
	return labelFont, labelFontErr
}

func sequenceLabel(fileName string, index int) string {
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	label := strconv.Itoa(index)
	best := ""

	start := -1
	for i := 0; i < len(name); i++ {
		if name[i] >= '0' && name[i] <= '9' {
			if start == -1 {
				start = i
			}
			continue
		}

		if start != -1 {
			best = chooseLabelCandidate(name[start:i], index, best)
			start = -1
		}
	}
	if start != -1 {
		best = chooseLabelCandidate(name[start:], index, best)
	}

	if best != "" {
		return best
	}
	return label
}

func chooseLabelCandidate(candidate string, index int, best string) string {
	value, err := strconv.Atoi(candidate)
	if err != nil || value != index {
		return best
	}
	if len(candidate) > len(best) {
		return candidate
	}
	return best
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
