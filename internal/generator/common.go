package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func renderName(pattern string, index int) string {
	if strings.Contains(pattern, "%") {
		return fmt.Sprintf(pattern, index)
	}
	return pattern
}

func ensureParentDir(path string) error {
	parent := filepath.Dir(path)
	if parent == "." || parent == "" {
		return nil
	}
	return os.MkdirAll(parent, 0o755)
}

func reserveOutputFile(path string) (*os.File, error) {
	if err := ensureParentDir(path); err != nil {
		return nil, fmt.Errorf("prepare parent dir: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	return file, nil
}
