package generator

import (
	"fmt"
	"os"
)

func createDirectory(path string) error {
	if err := ensureParentDir(path); err != nil {
		return fmt.Errorf("prepare parent dir: %w", err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		return err
	}
	return nil
}
