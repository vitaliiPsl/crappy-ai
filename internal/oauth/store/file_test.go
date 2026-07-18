package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilePreservesOtherFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth.json")
	if err := os.WriteFile(path, []byte(`{"sessions":{"mcp":{"token":"saved"}}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	file, err := NewFile[string](path, "credentials")
	if err != nil {
		t.Fatalf("NewFile() error = %v", err)
	}

	if err := file.Save(context.Background(), "openai", "credential"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), `"sessions"`) || !strings.Contains(string(data), `"credentials"`) {
		t.Fatalf("oauth file = %s, want both fields", data)
	}
}
