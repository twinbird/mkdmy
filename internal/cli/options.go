package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const appName = "mkdmy"

type FileKind string

const (
	KindUnknown   FileKind = ""
	KindText      FileKind = "text"
	KindImage     FileKind = "image"
	KindDirectory FileKind = "dir"
)

func (k *FileKind) String() string {
	if k == nil {
		return ""
	}
	return string(*k)
}

func (k *FileKind) Set(value string) error {
	switch normalized := strings.ToLower(strings.TrimSpace(value)); normalized {
	case string(KindText), string(KindImage), string(KindDirectory):
		*k = FileKind(normalized)
		return nil
	case "directory":
		*k = KindDirectory
		return nil
	default:
		return fmt.Errorf("invalid -type %q: must be one of text, image, dir", value)
	}
}

type ContentMode string

const (
	ContentModeUnset    ContentMode = ""
	ContentModeTemplate ContentMode = "template"
	ContentModeRandom   ContentMode = "random"
	ContentModeLorem    ContentMode = "lorem"
)

func (m *ContentMode) String() string {
	if m == nil {
		return ""
	}
	return string(*m)
}

func (m *ContentMode) Set(value string) error {
	switch normalized := strings.ToLower(strings.TrimSpace(value)); normalized {
	case string(ContentModeTemplate), string(ContentModeRandom), string(ContentModeLorem):
		*m = ContentMode(normalized)
		return nil
	default:
		return fmt.Errorf("invalid -mode %q: must be one of template, random, lorem", value)
	}
}

type byteSize int64

func (s *byteSize) String() string {
	if s == nil {
		return ""
	}
	return strconv.FormatInt(int64(*s), 10)
}

func (s *byteSize) Set(value string) error {
	parsed, err := parseByteSize(value)
	if err != nil {
		return err
	}
	*s = byteSize(parsed)
	return nil
}

type Options struct {
	Type        FileKind    `json:"type,omitempty"`
	SizeBytes   int64       `json:"sizeBytes,omitempty"`
	Count       int         `json:"count"`
	Name        string      `json:"name"`
	ContentMode ContentMode `json:"contentMode,omitempty"`
	Content     string      `json:"content,omitempty"`
	OutputDir   string      `json:"outputDir"`
}

func Parse(args []string) (Options, bool, error) {
	var opts Options
	var size byteSize

	fs := flag.NewFlagSet(commandName(), flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	fs.Var(&opts.Type, "type", "output kind: text, image, dir")
	fs.Var(&size, "size", "size per generated item, e.g. 512, 4KB, 10MiB")
	fs.IntVar(&opts.Count, "count", 0, "number of files or directories to create")
	fs.IntVar(&opts.Count, "n", 0, "alias of -count")
	fs.StringVar(&opts.Name, "name", "", "name template; fmt.Sprintf style numbering is supported")
	fs.Var(&opts.ContentMode, "mode", "content mode: template, random, lorem")
	fs.Var(&opts.ContentMode, "m", "alias of -mode")
	fs.StringVar(&opts.Content, "content", "", "content string used when -mode=template")
	fs.StringVar(&opts.Content, "c", "", "alias of -content")
	fs.StringVar(&opts.OutputDir, "o", ".", "destination directory")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Options{}, true, nil
		}
		return Options{}, false, err
	}

	if fs.NArg() > 0 {
		return Options{}, false, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	visited := visitedFlags(fs)
	opts.SizeBytes = int64(size)

	applyDefaults(&opts, visited)

	if err := validateOptions(&opts, visited); err != nil {
		return Options{}, false, err
	}

	return opts, false, nil
}

func validateOptions(opts *Options, visited map[string]bool) error {
	if opts.Count < 1 {
		return errors.New("-count or -n is required and must be 1 or greater")
	}

	if strings.TrimSpace(opts.OutputDir) == "" {
		return errors.New("-o must not be empty")
	}

	if opts.Type == KindUnknown {
		return errors.New("-type is required")
	}

	if opts.Type == KindDirectory {
		if visited["size"] {
			return errors.New("-size cannot be used when -type=dir")
		}
		if visited["mode"] || visited["m"] || visited["content"] || visited["c"] {
			return errors.New("content options cannot be used when -type=dir")
		}
	}

	if opts.Type == KindImage {
		if visited["content"] || visited["c"] {
			return errors.New("-content cannot be used when -type=image")
		}
		if opts.ContentMode != ContentModeRandom {
			return errors.New("-type=image only supports -mode=random")
		}
	}

	if opts.Content != "" && opts.ContentMode != ContentModeTemplate {
		return errors.New("-content requires -mode=template")
	}

	if opts.ContentMode == ContentModeTemplate && strings.TrimSpace(opts.Content) == "" {
		return errors.New("-content is required when -mode=template")
	}

	return nil
}

func applyDefaults(opts *Options, visited map[string]bool) {
	if opts.Content != "" && opts.ContentMode == ContentModeUnset {
		opts.ContentMode = ContentModeTemplate
	}

	if opts.Type == KindUnknown {
		return
	}

	if opts.Name != "" {
		goto applyModeAndSize
	}

	switch opts.Type {
	case KindText:
		opts.Name = "text-%03d.txt"
	case KindImage:
		opts.Name = "image-%03d.png"
	case KindDirectory:
		opts.Name = "dir-%03d"
	}

applyModeAndSize:
	if opts.Type == KindDirectory {
		return
	}

	if opts.ContentMode == ContentModeUnset {
		opts.ContentMode = defaultContentModeForType(opts.Type)
	}

	if !visited["size"] {
		opts.SizeBytes = defaultSizeForType(opts.Type)
	}
}

func PrintUsage(w io.Writer) {
	fmt.Fprintf(w, "%s - Generate dummy files and directories\n\n", commandName())
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s [options]\n\n", commandName())
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -type string")
	fmt.Fprintln(w, "        Output kind: text, image, dir (required)")
	fmt.Fprintln(w, "  -size value")
	fmt.Fprintln(w, "        Size per generated item. Accepts bytes or units like 4KB, 10MiB")
	fmt.Fprintln(w, "        Default by type: text=1KB, image=256KB")
	fmt.Fprintln(w, "  -count int")
	fmt.Fprintln(w, "        Number of files or directories to create (required)")
	fmt.Fprintln(w, "  -n int")
	fmt.Fprintln(w, "        Alias for -count")
	fmt.Fprintln(w, "  -name string")
	fmt.Fprintln(w, "        Name template. fmt.Sprintf-style numbering is supported")
	fmt.Fprintln(w, "        Default by type: text=text-%03d.txt, image=image-%03d.png, dir=dir-%03d")
	fmt.Fprintln(w, "  -mode string")
	fmt.Fprintln(w, "        Content mode: template, random, lorem")
	fmt.Fprintln(w, "        Default by type: text=lorem, image=random")
	fmt.Fprintln(w, "  -m string")
	fmt.Fprintln(w, "        Alias for -mode")
	fmt.Fprintln(w, "  -content string")
	fmt.Fprintln(w, "        Content string for -mode=template. Sets -mode=template automatically when provided")
	fmt.Fprintln(w, "  -c string")
	fmt.Fprintln(w, "        Alias for -content")
	fmt.Fprintln(w, "  -o string")
	fmt.Fprintln(w, "        Output directory (default .)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintf(w, "  %s -type text -n 1\n", commandName())
	fmt.Fprintf(w, "  %s -type text -n 10 -size 4KB -name 'note-%%03d.txt' -mode lorem\n", commandName())
	fmt.Fprintf(w, "  %s -type image -count 3 -name 'img-%%02d.png' -mode random\n", commandName())
	fmt.Fprintf(w, "  %s -type text -n 3 -name 'memo-%%02d.txt' -content 'dummy-%%02d'\n", commandName())
	fmt.Fprintf(w, "  %s -type dir -count 5 -name 'batch-%%02d'\n", commandName())
}

func commandName() string {
	return appName
}

func parseByteSize(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("size must not be empty")
	}

	split := 0
	for split < len(value) && value[split] >= '0' && value[split] <= '9' {
		split++
	}

	if split == 0 {
		return 0, fmt.Errorf("invalid size %q", raw)
	}

	numberPart := value[:split]
	unitPart := strings.ToUpper(strings.TrimSpace(value[split:]))

	number, err := strconv.ParseInt(numberPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", raw, err)
	}
	if number < 0 {
		return 0, errors.New("size must not be negative")
	}

	multiplier, ok := sizeUnitMultiplier(unitPart)
	if !ok {
		return 0, fmt.Errorf("invalid size unit %q", unitPart)
	}

	return number * multiplier, nil
}

func sizeUnitMultiplier(unit string) (int64, bool) {
	switch unit {
	case "", "B":
		return 1, true
	case "K", "KB":
		return 1000, true
	case "KI", "KIB":
		return 1024, true
	case "M", "MB":
		return 1000 * 1000, true
	case "MI", "MIB":
		return 1024 * 1024, true
	case "G", "GB":
		return 1000 * 1000 * 1000, true
	case "GI", "GIB":
		return 1024 * 1024 * 1024, true
	default:
		return 0, false
	}
}

func defaultContentModeForType(kind FileKind) ContentMode {
	switch kind {
	case KindText:
		return ContentModeLorem
	case KindImage:
		return ContentModeRandom
	default:
		return ContentModeUnset
	}
}

func defaultSizeForType(kind FileKind) int64 {
	switch kind {
	case KindText:
		return 1000
	case KindImage:
		return 256 * 1000
	default:
		return 0
	}
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}
