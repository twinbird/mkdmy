package generator

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
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

func TestGenerateTextLoremExactSize(t *testing.T) {
	tmpDir := t.TempDir()
	opts := cli.Options{
		Type:        cli.KindText,
		Count:       1,
		Name:        "lorem-%02d.txt",
		SizeBytes:   int64(len(loremText) + 5),
		ContentMode: cli.ContentModeLorem,
		OutputDir:   tmpDir,
	}

	if err := Generate(opts); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "lorem-01.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) != int(opts.SizeBytes) {
		t.Fatalf("len(data) = %d, want %d", len(data), opts.SizeBytes)
	}
	if !strings.HasPrefix(string(data), loremText) {
		t.Fatalf("data prefix = %q, want %q", string(data[:len(loremText)]), loremText)
	}
}

func TestGenerateTextRandomExactSize(t *testing.T) {
	tmpDir := t.TempDir()
	opts := cli.Options{
		Type:        cli.KindText,
		Count:       1,
		Name:        "random-%02d.txt",
		SizeBytes:   257,
		ContentMode: cli.ContentModeRandom,
		OutputDir:   tmpDir,
	}

	if err := Generate(opts); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "random-01.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) != int(opts.SizeBytes) {
		t.Fatalf("len(data) = %d, want %d", len(data), opts.SizeBytes)
	}
	for i, b := range data {
		if !strings.ContainsRune(randomAlphabet, rune(b)) {
			t.Fatalf("byte[%d] = %q not in random alphabet", i, b)
		}
	}
}

func TestGenerateTextMultipleFilesWithSequentialNames(t *testing.T) {
	tmpDir := t.TempDir()
	opts := cli.Options{
		Type:        cli.KindText,
		Count:       3,
		Name:        "note-%02d.txt",
		SizeBytes:   8,
		ContentMode: cli.ContentModeLorem,
		OutputDir:   tmpDir,
	}

	if err := Generate(opts); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	for _, name := range []string{"note-01.txt", "note-02.txt", "note-03.txt"} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); err != nil {
			t.Fatalf("Stat(%q) error = %v", name, err)
		}
	}
}

func TestGenerateTextWithoutFormatCollides(t *testing.T) {
	tmpDir := t.TempDir()
	opts := cli.Options{
		Type:        cli.KindText,
		Count:       2,
		Name:        "same.txt",
		SizeBytes:   8,
		ContentMode: cli.ContentModeLorem,
		OutputDir:   tmpDir,
	}

	err := Generate(opts)
	if err == nil {
		t.Fatal("Generate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "file exists") {
		t.Fatalf("Generate() error = %q, want file exists", err)
	}
}

func TestGenerateRejectsExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "text-01.txt")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	opts := cli.Options{
		Type:        cli.KindText,
		Count:       1,
		Name:        "text-%02d.txt",
		SizeBytes:   8,
		ContentMode: cli.ContentModeLorem,
		OutputDir:   tmpDir,
	}

	err := Generate(opts)
	if err == nil {
		t.Fatal("Generate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "file exists") {
		t.Fatalf("Generate() error = %q, want file exists", err)
	}
}

func TestGenerateCreatesNestedOutputDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "nested", "output")
	opts := cli.Options{
		Type:      cli.KindDirectory,
		Count:     1,
		Name:      "dir-%02d",
		OutputDir: outputDir,
	}

	if err := Generate(opts); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(outputDir, "dir-01"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if !info.IsDir() {
		t.Fatal("generated path is not a directory")
	}
}

func TestGenerateImageRejectsUnsupportedMode(t *testing.T) {
	tmpDir := t.TempDir()
	opts := cli.Options{
		Type:        cli.KindImage,
		Count:       1,
		Name:        "image-%02d.png",
		SizeBytes:   2000,
		ContentMode: cli.ContentModeLorem,
		OutputDir:   tmpDir,
	}

	err := Generate(opts)
	if err == nil {
		t.Fatal("Generate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `unsupported image mode "lorem"`) {
		t.Fatalf("Generate() error = %q", err)
	}
}

func TestGenerateImageTinySizeStillProducesPNG(t *testing.T) {
	tmpDir := t.TempDir()
	opts := cli.Options{
		Type:        cli.KindImage,
		Count:       1,
		Name:        "tiny-%02d.png",
		SizeBytes:   1,
		ContentMode: cli.ContentModeRandom,
		OutputDir:   tmpDir,
	}

	if err := Generate(opts); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "tiny-01.png"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if _, err := png.DecodeConfig(bytes.NewReader(data)); err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
}
