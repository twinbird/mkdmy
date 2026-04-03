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
