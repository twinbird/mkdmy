# mkdmy

`mkdmy` is a small CLI for generating dummy text files, PNG files, and directories.

## Features

- Generate text files with exact byte sizes
- Generate random PNG files near a target size
- Generate directories in bulk
- Use sequential names with `fmt.Sprintf`-style templates

## Requirements

- Go 1.25 or later

## Build

```bash
go build -o mkdmy ./cmd/mkdmy
```

## Run

```bash
go run ./cmd/mkdmy -type text -n 1
```

## Usage

```text
mkdmy - Generate dummy files and directories

Usage:
  mkdmy [options]

Options:
  -type string
        Output kind: text, png, dir (required)
  -size value
        Size per generated item. Accepts bytes or units like 4KB, 10MiB
        Default by type: text=1KB, png=256KB
  -count int
        Number of files or directories to create (required)
  -n int
        Alias for -count
  -name string
        Name template. fmt.Sprintf-style numbering is supported
        Default by type: text=text-%03d.txt, png=image-%03d.png, dir=dir-%03d
  -mode string
        Content mode: template, random, lorem
        Default by type: text=lorem, png=random
  -m string
        Alias for -mode
  -content string
        Content string for -mode=template. Sets -mode=template automatically when provided
  -c string
        Alias for -content
  -o string
        Output directory (default .)
```

## Examples

Create one text file with default settings:

```bash
mkdmy -type text -n 1
```

Create ten 4KB lorem text files:

```bash
mkdmy -type text -n 10 -size 4KB -name 'note-%03d.txt' -mode lorem
```

Create three random PNG files:

```bash
mkdmy -type png -count 3 -name 'img-%02d.png' -mode random
```

Create template-based text files:

```bash
mkdmy -type text -n 3 -name 'memo-%02d.txt' -content 'dummy-%02d'
```

Create directories:

```bash
mkdmy -type dir -count 5 -name 'batch-%02d'
```

## Notes

- `-size` is not available for `-type=dir`
- `-type=png` supports only `-mode=random`
- `-content` requires `-mode=template`, and setting `-content` enables template mode automatically
