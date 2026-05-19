package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAbsPath(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	got, err := AbsPath("src/../README.md")
	if err != nil {
		t.Fatalf("AbsPath: %v", err)
	}

	want := filepath.Join(root, "README.md")
	if got != want {
		t.Fatalf("AbsPath = %q, want %q", got, want)
	}
}

func TestAbsPathCleansAbsolutePath(t *testing.T) {
	got, err := AbsPath("/tmp/project/../README.md")
	if err != nil {
		t.Fatalf("AbsPath: %v", err)
	}

	want := filepath.Join(string(filepath.Separator), "tmp", "README.md")
	if got != want {
		t.Fatalf("AbsPath = %q, want %q", got, want)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skipf("home dir unavailable: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "expands home prefix",
			path: "~/notes/todo.md",
			want: filepath.Join(home, "notes", "todo.md"),
		},
		{
			name: "leaves bare tilde unchanged",
			path: "~",
			want: "~",
		},
		{
			name: "leaves non-home path unchanged",
			path: "/tmp/file.txt",
			want: "/tmp/file.txt",
		},
		{
			name: "leaves other user shorthand unchanged",
			path: "~other/file.txt",
			want: "~other/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExpandHome(tt.path); got != tt.want {
				t.Fatalf("ExpandHome(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestCompactHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skipf("home dir unavailable: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "compacts home directory",
			path: home,
			want: "~",
		},
		{
			name: "compacts child of home",
			path: filepath.Join(home, "notes", "todo.md"),
			want: "~" + string(filepath.Separator) + filepath.Join("notes", "todo.md"),
		},
		{
			name: "leaves sibling path unchanged",
			path: home + "-backup",
			want: home + "-backup",
		},
		{
			name: "leaves empty path unchanged",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompactHome(tt.path); got != tt.want {
				t.Fatalf("CompactHome(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
