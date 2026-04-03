package cli

import "testing"

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
