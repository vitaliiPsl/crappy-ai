package utils

import (
	"os"
	"path/filepath"
	"strings"
)

func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, path[2:])
}
