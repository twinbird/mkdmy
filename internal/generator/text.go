package generator

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"os"

	"github.com/twinbird/mkdmy/internal/cli"
)

const randomAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func createTextFile(path string, opts cli.Options, index int) error {
	if err := ensureParentDir(path); err != nil {
		return fmt.Errorf("prepare parent dir: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	if err := writeTextContent(writer, opts, index); err != nil {
		file.Close()
		return err
	}
	if err := writer.Flush(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() != opts.SizeBytes {
		return fmt.Errorf("unexpected size: got %d bytes, want %d bytes", info.Size(), opts.SizeBytes)
	}

	return nil
}

func writeTextContent(writer *bufio.Writer, opts cli.Options, index int) error {
	remaining := opts.SizeBytes
	if remaining <= 0 {
		return nil
	}

	switch opts.ContentMode {
	case cli.ContentModeTemplate:
		return writeTemplateText(writer, []byte(fmt.Sprintf(opts.Content, index)), remaining)
	case cli.ContentModeRandom:
		return writeRandomText(writer, remaining)
	default:
		return fmt.Errorf("unsupported text mode %q", opts.ContentMode)
	}
}

func writeRepeatedText(writer *bufio.Writer, pattern []byte, size int64) error {
	if len(pattern) == 0 {
		return fmt.Errorf("content pattern must not be empty")
	}

	remaining := size
	for remaining > 0 {
		chunk := pattern
		if int64(len(chunk)) > remaining {
			chunk = chunk[:remaining]
		}
		if _, err := writer.Write(chunk); err != nil {
			return err
		}
		remaining -= int64(len(chunk))
	}

	return nil
}

func writeTemplateText(writer *bufio.Writer, content []byte, size int64) error {
	if len(content) == 0 {
		return fmt.Errorf("content pattern must not be empty")
	}

	remaining := size
	separator := []byte("\n")

	for remaining > 0 {
		chunk := content
		if int64(len(chunk)) > remaining {
			chunk = chunk[:remaining]
		}
		if _, err := writer.Write(chunk); err != nil {
			return err
		}
		remaining -= int64(len(chunk))
		if remaining == 0 || len(chunk) < len(content) {
			break
		}

		newline := separator
		if int64(len(newline)) > remaining {
			newline = newline[:remaining]
		}
		if _, err := writer.Write(newline); err != nil {
			return err
		}
		remaining -= int64(len(newline))
	}

	return nil
}

func writeRandomText(writer *bufio.Writer, size int64) error {
	const chunkSize = 4096

	remaining := size
	randomBytes := make([]byte, chunkSize)
	textBytes := make([]byte, chunkSize)

	for remaining > 0 {
		current := len(randomBytes)
		if int64(current) > remaining {
			current = int(remaining)
		}

		if _, err := rand.Read(randomBytes[:current]); err != nil {
			return err
		}
		for i := 0; i < current; i++ {
			textBytes[i] = randomAlphabet[int(randomBytes[i])%len(randomAlphabet)]
		}
		if _, err := writer.Write(textBytes[:current]); err != nil {
			return err
		}

		remaining -= int64(current)
	}

	return nil
}
