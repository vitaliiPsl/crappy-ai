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

func TestManagerForkSession(t *testing.T) {
	m := testManager(t)
	ctx := context.Background()

	source, err := m.CreateSession(ctx, "source")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	fork, err := m.ForkSession(ctx, source.ID, "branch")
	if err != nil {
		t.Fatalf("ForkSession: %v", err)
	}

	if fork.Title != "branch" || fork.ForkedFromID != source.ID {
		t.Fatalf("fork = %+v", fork)
	}
}

func TestManagerForkSessionRejectsActiveSession(t *testing.T) {
	m := testManager(t)
	ctx := context.Background()

	source, err := m.CreateSession(ctx, "source")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	live := m.getOrCreate(source.ID)
	live.mu.Lock()
	live.cancel = func() {}
	live.mu.Unlock()

	if _, err := m.ForkSession(ctx, source.ID, ""); err == nil {
		t.Fatal("ForkSession active session error = nil")
	}
}
