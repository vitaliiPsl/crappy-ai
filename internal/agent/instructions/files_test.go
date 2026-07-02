package instructions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesLoadsInstructionFilesFromRootToCwd(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "pkg", "api")
	mkdirAll(t, child)

	writeFile(t, filepath.Join(root, "AGENTS.md"), "root agents")
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "root claude")
	writeFile(t, filepath.Join(child, "AGENTS.md"), "child agents")
	writeFile(t, filepath.Join(child, "CLAUDE.md"), "child claude")

	got := Files(child)

	for _, want := range []string{
		"# File Instructions",
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, "CLAUDE.md"),
		filepath.Join(child, "AGENTS.md"),
		filepath.Join(child, "CLAUDE.md"),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Files output missing %q:\n%s", want, got)
		}
	}

	assertOrder(t, got,
		"root agents",
		"root claude",
		"child agents",
		"child claude",
	)
}

func TestFilesIgnoresClaudeSpecificFiles(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "work")
	mkdirAll(t, filepath.Join(root, ".claude"))
	mkdirAll(t, cwd)

	writeFile(t, filepath.Join(root, "CLAUDE.local.md"), "local private")
	writeFile(t, filepath.Join(root, ".claude", "claude.md"), "lowercase user style")
	writeFile(t, filepath.Join(root, ".claude", "CLAUDE.md"), "dot claude")
	writeFile(t, filepath.Join(root, "CLAUDE.md"), "supported claude")

	got := Files(cwd)

	if strings.Contains(got, "local private") {
		t.Fatalf("Files loaded CLAUDE.local.md:\n%s", got)
	}

	if strings.Contains(got, "lowercase user style") {
		t.Fatalf("Files loaded lowercase .claude/claude.md:\n%s", got)
	}

	if strings.Contains(got, "dot claude") {
		t.Fatalf("Files loaded .claude/CLAUDE.md:\n%s", got)
	}

	if !strings.Contains(got, "supported claude") {
		t.Fatalf("Files did not load CLAUDE.md:\n%s", got)
	}
}

func TestFilesReturnsEmptyWithoutInstructionFiles(t *testing.T) {
	if got := Files(t.TempDir()); got != "" {
		t.Fatalf("Files = %q, want empty", got)
	}
}

func TestFilesSkipsTooLargeInstructionFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "AGENTS.md"), strings.Repeat("x", maxInstructionFileBytes+1))

	if got := Files(root); got != "" {
		t.Fatalf("Files = %q, want empty for too-large file", got)
	}
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func assertOrder(t *testing.T, text string, values ...string) {
	t.Helper()

	last := -1
	for _, value := range values {
		idx := strings.Index(text, value)
		if idx < 0 {
			t.Fatalf("text missing %q:\n%s", value, text)
		}

		if idx <= last {
			t.Fatalf("%q appeared out of order:\n%s", value, text)
		}

		last = idx
	}
}
