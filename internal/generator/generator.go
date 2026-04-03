package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/twinbird/mkdmy/internal/cli"
)

func Generate(opts cli.Options) error {
	outputDir, err := prepareOutputDir(opts.OutputDir)
	if err != nil {
		return err
	}

	for index := 1; index <= opts.Count; index++ {
		name := renderName(opts.Name, index)
		path := filepath.Join(outputDir, filepath.Clean(name))

		switch opts.Type {
		case cli.KindDirectory:
			if err := createDirectory(path); err != nil {
				return fmt.Errorf("generate dir %q: %w", path, err)
			}
		case cli.KindText:
			if err := createTextFile(path, opts, index); err != nil {
				return fmt.Errorf("generate text %q: %w", path, err)
			}
		case cli.KindPNG:
			if err := createImageFile(path, opts); err != nil {
				return fmt.Errorf("generate png %q: %w", path, err)
			}
		default:
			return fmt.Errorf("unsupported type %q", opts.Type)
		}
	}

	return nil
}

func prepareOutputDir(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if err := os.MkdirAll(cleaned, 0o755); err != nil {
		return "", fmt.Errorf("prepare output dir %q: %w", cleaned, err)
	}
	return cleaned, nil
}
