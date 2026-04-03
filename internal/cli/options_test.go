package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseAppliesDefaultsForText(t *testing.T) {
	opts, helpRequested, err := Parse([]string{"-type", "text", "-n", "1"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if helpRequested {
		t.Fatal("helpRequested = true, want false")
	}
	if opts.Type != KindText {
		t.Fatalf("Type = %q, want %q", opts.Type, KindText)
	}
	if opts.Name != "text-%03d.txt" {
		t.Fatalf("Name = %q", opts.Name)
	}
	if opts.SizeBytes != 1000 {
		t.Fatalf("SizeBytes = %d, want 1000", opts.SizeBytes)
	}
	if opts.ContentMode != ContentModeLorem {
		t.Fatalf("ContentMode = %q, want %q", opts.ContentMode, ContentModeLorem)
	}
}

func TestParseAutoSetsTemplateMode(t *testing.T) {
	opts, _, err := Parse([]string{"-type", "text", "-n", "1", "-content", "dummy-%02d"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if opts.ContentMode != ContentModeTemplate {
		t.Fatalf("ContentMode = %q, want %q", opts.ContentMode, ContentModeTemplate)
	}
}

func TestParseRejectsDirSize(t *testing.T) {
	_, _, err := Parse([]string{"-type", "dir", "-n", "1", "-size", "1KB"})
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
}

func TestParseRejectsImageLorem(t *testing.T) {
	_, _, err := Parse([]string{"-type", "image", "-n", "1", "-mode", "lorem"})
	if err == nil {
		t.Fatal("Parse() error = nil, want error")
	}
}

func TestParseRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing type",
			args:    []string{"-n", "1"},
			wantErr: "-type is required",
		},
		{
			name:    "invalid type",
			args:    []string{"-type", "pdf", "-n", "1"},
			wantErr: `invalid -type "pdf"`,
		},
		{
			name:    "invalid mode",
			args:    []string{"-type", "text", "-n", "1", "-mode", "json"},
			wantErr: `invalid -mode "json"`,
		},
		{
			name:    "count zero",
			args:    []string{"-type", "text", "-count", "0"},
			wantErr: "-count or -n is required and must be 1 or greater",
		},
		{
			name:    "count negative",
			args:    []string{"-type", "text", "-count", "-1"},
			wantErr: "-count or -n is required and must be 1 or greater",
		},
		{
			name:    "empty output dir",
			args:    []string{"-type", "text", "-n", "1", "-o", "   "},
			wantErr: "-o must not be empty",
		},
		{
			name:    "template mode without content",
			args:    []string{"-type", "text", "-n", "1", "-mode", "template"},
			wantErr: "-content is required when -mode=template",
		},
		{
			name:    "dir with content option",
			args:    []string{"-type", "dir", "-n", "1", "-content", "dummy"},
			wantErr: "content options cannot be used when -type=dir",
		},
		{
			name:    "image with content option",
			args:    []string{"-type", "image", "-n", "1", "-content", "dummy", "-mode", "random"},
			wantErr: "-content cannot be used when -type=image",
		},
		{
			name:    "unexpected positional args",
			args:    []string{"-type", "text", "-n", "1", "extra"},
			wantErr: "unexpected positional arguments: extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Parse(tt.args)
			if err == nil {
				t.Fatal("Parse() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Parse() error = %q, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseByteSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw     string
		want    int64
		wantErr string
	}{
		{raw: "512", want: 512},
		{raw: "4KB", want: 4000},
		{raw: "4KiB", want: 4096},
		{raw: "10MiB", want: 10 * 1024 * 1024},
		{raw: " 7 mb ", want: 7 * 1000 * 1000},
		{raw: "", wantErr: "size must not be empty"},
		{raw: "-1", wantErr: `invalid size "-1"`},
		{raw: "12XB", wantErr: `invalid size unit "XB"`},
		{raw: "abc", wantErr: `invalid size "abc"`},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := parseByteSize(tt.raw)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("parseByteSize() error = nil, want error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseByteSize() error = %q, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseByteSize() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseByteSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPrintUsageIncludesKeyLines(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	PrintUsage(&buf)

	for _, want := range []string{
		"mkdmy - ダミーファイルを作成するコマンド",
		"生成対象の種類。text, image, dir (required)",
		"mkdmy -type text -n 1",
		"mkdmy -type image -count 3 -name 'img-%02d.png' -mode random",
	} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("PrintUsage() missing %q", want)
		}
	}
}
