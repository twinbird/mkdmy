package generator

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/twinbird/mkdmy/internal/cli"
)

func TestGenerateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	opts := cli.Options{
		Type:      cli.KindDirectory,
		Count:     2,
		Name:      "dir-%02d",
		OutputDir: tmpDir,
	}

	if err := Generate(opts); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	for _, name := range []string{"dir-01", "dir-02"} {
		info, err := os.Stat(filepath.Join(tmpDir, name))
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", name, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", name)
		}
	}
}

func TestGenerateTextTemplateExactSize(t *testing.T) {
	tmpDir := t.TempDir()
	opts := cli.Options{
		Type:        cli.KindText,
		Count:       1,
		Name:        "memo-%02d.txt",
		SizeBytes:   16,
		ContentMode: cli.ContentModeTemplate,
		Content:     "dummy-%02d",
		OutputDir:   tmpDir,
	}

	if err := Generate(opts); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "memo-01.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) != 16 {
		t.Fatalf("len(data) = %d, want 16", len(data))
	}
	if string(data) != "dummy-01dummy-01" {
		t.Fatalf("data = %q", data)
	}
}

func TestGenerateImageProducesPNG(t *testing.T) {
	tmpDir := t.TempDir()
	opts := cli.Options{
		Type:        cli.KindImage,
		Count:       1,
		Name:        "image-%02d.png",
		SizeBytes:   6000,
		ContentMode: cli.ContentModeRandom,
		OutputDir:   tmpDir,
	}

	if err := Generate(opts); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "image-01.png"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	pngSignature := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if !bytes.HasPrefix(data, pngSignature) {
		t.Fatalf("missing PNG signature")
	}
	if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}

	sizeDelta := absInt64(int64(len(data)) - opts.SizeBytes)
	if sizeDelta > opts.SizeBytes/2 {
		t.Fatalf("image size delta = %d, want <= %d", sizeDelta, opts.SizeBytes/2)
	}
}
