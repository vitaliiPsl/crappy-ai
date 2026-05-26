package instructions

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	maxInstructionFileBytes  = 64 * 1024
	maxInstructionTotalBytes = 256 * 1024
)

var instructionFileNames = []string{
	"AGENTS.md",
	"CLAUDE.md",
}

type instructionFile struct {
	path    string
	content string
}

func Files(cwd string) string {
	if cwd == "" {
		return ""
	}

	abs, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}

	files := loadInstructionFiles(abs)
	if len(files) == 0 {
		return ""
	}

	return formatInstructions(files)
}

func loadInstructionFiles(cwd string) []instructionFile {
	var (
		files []instructionFile
		total int
	)

	for _, dir := range ancestorDirs(cwd) {
		for _, name := range instructionFileNames {
			path := filepath.Join(dir, name)

			info, err := os.Stat(path)
			if err != nil {
				continue
			}

			if info.IsDir() {
				continue
			}

			if info.Size() > maxInstructionFileBytes {
				continue
			}

			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			content := strings.TrimSpace(string(data))
			if content == "" {
				continue
			}

			total += len(content)
			if total > maxInstructionTotalBytes {
				return files
			}

			files = append(files, instructionFile{path: path, content: content})
		}
	}

	return files
}

func ancestorDirs(cwd string) []string {
	var dirs []string

	for dir := cwd; ; dir = filepath.Dir(dir) {
		dirs = append(dirs, dir)

		if parent := filepath.Dir(dir); parent == dir {
			break
		}
	}

	slices.Reverse(dirs)

	return dirs
}

func formatInstructions(files []instructionFile) string {
	var b strings.Builder

	b.WriteString("# File Instructions\n\n")
	b.WriteString("The following instruction files were found by walking upward from the session working directory. ")
	b.WriteString("Treat them as context, not enforced configuration. Follow them unless they conflict with higher-priority instructions, user requests, or permission policy.")

	for _, file := range files {
		fmt.Fprintf(&b, "\n\n## %s\n\n%s", file.path, file.content)
	}

	return b.String()
}
