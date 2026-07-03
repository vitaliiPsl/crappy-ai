package runtime

import (
	"context"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	sessionstore "github.com/vitaliiPsl/crappy-ai/internal/session/store"
)

func testManager(t *testing.T) *Manager {
	t.Helper()

	store, err := sessionstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	cfg := config.NewStore(config.Config{Cwd: t.TempDir()}, "")

	return NewManager(cfg, store, nil, nil, nil, nil)
}

func TestManagerRunRequiresExistingSession(t *testing.T) {
	m := testManager(t)

	if err := m.Run(context.Background(), "missing", Request{Text: "hello"}); err == nil {
		t.Fatal("Run missing session error = nil")
	}

	if len(m.live) != 0 {
		t.Fatalf("live sessions = %d, want 0", len(m.live))
	}
}

func TestManagerRunSubagentRequiresExistingSession(t *testing.T) {
	m := testManager(t)

	_, err := m.RunSubagent(context.Background(), "missing", SubagentRequest{Agent: "worker", Task: "do it"})
	if err == nil {
		t.Fatal("RunSubagent missing session error = nil")
	}

	if len(m.live) != 0 {
		t.Fatalf("live sessions = %d, want 0", len(m.live))
	}
}

func TestManagerCompactRequiresExistingSession(t *testing.T) {
	m := testManager(t)

	if err := m.Compact(context.Background(), "missing"); err == nil {
		t.Fatal("Compact missing session error = nil")
	}

	if len(m.live) != 0 {
		t.Fatalf("live sessions = %d, want 0", len(m.live))
	}
}
