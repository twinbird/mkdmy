package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const appName = "mkdmy"

type fileKind string

const (
	kindUnknown   fileKind = ""
	kindText      fileKind = "text"
	kindImage     fileKind = "image"
	kindDirectory fileKind = "dir"
)

func (k *fileKind) String() string {
	if k == nil {
		return ""
	}
	return string(*k)
}

func (k *fileKind) Set(value string) error {
	switch normalized := strings.ToLower(strings.TrimSpace(value)); normalized {
	case string(kindText), string(kindImage), string(kindDirectory):
		*k = fileKind(normalized)
		return nil
	case "directory":
		*k = kindDirectory
		return nil
	default:
		return fmt.Errorf("invalid -type %q: must be one of text, image, dir", value)
	}
}

type contentMode string

const (
	contentModeUnset    contentMode = ""
	contentModeTemplate contentMode = "template"
	contentModeRandom   contentMode = "random"
	contentModeLorem    contentMode = "lorem"
)

func (m *contentMode) String() string {
	if m == nil {
		return ""
	}
	return string(*m)
}

func (m *contentMode) Set(value string) error {
	switch normalized := strings.ToLower(strings.TrimSpace(value)); normalized {
	case string(contentModeTemplate), string(contentModeRandom), string(contentModeLorem):
		*m = contentMode(normalized)
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

type options struct {
	Type        fileKind    `json:"type,omitempty"`
	SizeBytes   int64       `json:"sizeBytes,omitempty"`
	Count       int         `json:"count"`
	Name        string      `json:"name"`
	ContentMode contentMode `json:"contentMode,omitempty"`
	Content     string      `json:"content,omitempty"`
	OutputDir   string      `json:"outputDir"`
}

func main() {
	opts, helpRequested, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
		printUsage(os.Stderr)
		os.Exit(2)
	}

	if helpRequested {
		printUsage(os.Stdout)
		return
	}

	fmt.Println("option parsing is implemented; generation is not implemented yet.")
	fmt.Printf("parsed options: %+v\n", opts)
}

func parseOptions(args []string) (options, bool, error) {
	var opts options
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
			return options{}, true, nil
		}
		return options{}, false, err
	}

	if fs.NArg() > 0 {
		return options{}, false, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	visited := visitedFlags(fs)
	opts.SizeBytes = int64(size)

	applyDefaults(&opts, visited)

	if err := validateOptions(&opts, visited); err != nil {
		return options{}, false, err
	}

	return opts, false, nil
}

func validateOptions(opts *options, visited map[string]bool) error {
	if opts.Count < 1 {
		return errors.New("-count or -n is required and must be 1 or greater")
	}

	if strings.TrimSpace(opts.OutputDir) == "" {
		return errors.New("-o must not be empty")
	}

	if opts.Type == kindUnknown {
		return errors.New("-type is required")
	}

	if opts.Type == kindDirectory {
		if visited["size"] {
			return errors.New("-size cannot be used when -type=dir")
		}
		if visited["mode"] || visited["m"] || visited["content"] || visited["c"] {
			return errors.New("content options cannot be used when -type=dir")
		}
	}

	if opts.Content != "" && opts.ContentMode != contentModeTemplate {
		return errors.New("-content requires -mode=template")
	}

	if opts.ContentMode == contentModeTemplate && strings.TrimSpace(opts.Content) == "" {
		return errors.New("-content is required when -mode=template")
	}

	return nil
}

func applyDefaults(opts *options, visited map[string]bool) {
	if opts.Content != "" && opts.ContentMode == contentModeUnset {
		opts.ContentMode = contentModeTemplate
	}

	if opts.Type == kindUnknown {
		return
	}

	if opts.Name != "" {
		goto applyModeAndSize
	}

	switch opts.Type {
	case kindText:
		opts.Name = "text-%03d.txt"
	case kindImage:
		opts.Name = "image-%03d.png"
	case kindDirectory:
		opts.Name = "dir-%03d"
	}

applyModeAndSize:
	if opts.Type == kindDirectory {
		return
	}

	if opts.ContentMode == contentModeUnset {
		opts.ContentMode = defaultContentModeForType(opts.Type)
	}

	if !visited["size"] {
		opts.SizeBytes = defaultSizeForType(opts.Type)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "%s - ダミーファイルを作成するコマンド\n\n", commandName())
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s [options]\n\n", commandName())
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  -type string")
	fmt.Fprintln(w, "        生成対象の種類。text, image, dir (required)")
	fmt.Fprintln(w, "  -size value")
	fmt.Fprintln(w, "        1件ごとのサイズ。整数バイトまたは 4KB, 10MiB のような単位付き")
	fmt.Fprintln(w, "        type ごとの既定値: text=1KB, image=256KB")
	fmt.Fprintln(w, "  -count int")
	fmt.Fprintln(w, "        生成するファイル/ディレクトリ数 (required)")
	fmt.Fprintln(w, "  -n int")
	fmt.Fprintln(w, "        -count のエイリアス")
	fmt.Fprintln(w, "  -name string")
	fmt.Fprintln(w, "        名前テンプレート。fmt.Sprintf の連番フォーマットを利用可能")
	fmt.Fprintln(w, "        type ごとの既定値: text=text-%03d.txt, image=image-%03d.png, dir=dir-%03d")
	fmt.Fprintln(w, "  -mode string")
	fmt.Fprintln(w, "        内容指定。template, random, lorem")
	fmt.Fprintln(w, "        type ごとの既定値: text=lorem, image=random")
	fmt.Fprintln(w, "  -m string")
	fmt.Fprintln(w, "        -mode のエイリアス")
	fmt.Fprintln(w, "  -content string")
	fmt.Fprintln(w, "        -mode=template のときに使う文字列。指定時は -mode=template を自動適用")
	fmt.Fprintln(w, "  -c string")
	fmt.Fprintln(w, "        -content のエイリアス")
	fmt.Fprintln(w, "  -o string")
	fmt.Fprintln(w, "        出力先ディレクトリ (default .)")
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

func defaultContentModeForType(kind fileKind) contentMode {
	switch kind {
	case kindText:
		return contentModeLorem
	case kindImage:
		return contentModeRandom
	default:
		return contentModeUnset
	}
}

func defaultSizeForType(kind fileKind) int64 {
	switch kind {
	case kindText:
		return 1000
	case kindImage:
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
