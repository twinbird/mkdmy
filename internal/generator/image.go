package generator

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"

	"github.com/twinbird/mkdmy/internal/cli"
)

func createImageFile(path string, opts cli.Options) error {
	if opts.ContentMode != cli.ContentModeRandom {
		return fmt.Errorf("unsupported png mode %q", opts.ContentMode)
	}

	if err := ensureParentDir(path); err != nil {
		return fmt.Errorf("prepare parent dir: %w", err)
	}

	data, err := buildRandomPNG(opts.SizeBytes)
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
	side := estimateImageSide(targetSize)
	bestSide := side
	var best []byte
	bestDelta := int64(math.MaxInt64)

	for attempt := 0; attempt < 5; attempt++ {
		data, err := encodeRandomPNG(side)
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
