package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func newTestStore(t *testing.T) (*FileStore, string) {
	t.Helper()

	dir := t.TempDir()

	st, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	return st, dir
}

func TestNewFileStore_RequiresDir(t *testing.T) {
	if _, err := NewFileStore(""); err == nil {
		t.Fatal("NewFileStore(\"\"): want error, got nil")
	}
}

func TestNewFileStore_CreatesMissingDir(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sub", "sessions")

	if _, err := NewFileStore(target); err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if !info.IsDir() {
		t.Errorf("expected dir at %s", target)
	}
}

func TestCreate_ReturnsAndPersistsFields(t *testing.T) {
	st, _ := newTestStore(t)
	before := time.Now()

	sess, err := st.Create(context.Background(), "the title", "/work")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if sess.ID == "" {
		t.Error("Create: empty ID")
	}

	if sess.Title != "the title" {
		t.Errorf("Title = %q", sess.Title)
	}

	if sess.WorkDir != "/work" {
		t.Errorf("WorkDir = %q", sess.WorkDir)
	}

	if sess.CreatedAt.Before(before) {
		t.Errorf("CreatedAt %v predates test start %v", sess.CreatedAt, before)
	}

	got, err := st.Get(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}

	if got.ID != sess.ID || got.Title != sess.Title {
		t.Errorf("persisted session = %+v, want %+v", got, sess)
	}
}

func TestCreate_CleansWorkDir(t *testing.T) {
	st, _ := newTestStore(t)

	sess, err := st.Create(context.Background(), "t", "/work//foo/../bar")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if sess.WorkDir != "/work/bar" {
		t.Errorf("WorkDir = %q, want /work/bar", sess.WorkDir)
	}
}

func TestGet_Missing(t *testing.T) {
	st, _ := newTestStore(t)

	if _, err := st.Get(context.Background(), "missing-id"); err == nil {
		t.Fatal("Get on missing: want error, got nil")
	}
}

func TestGet_ReturnsIndependentSnapshot(t *testing.T) {
	st, _ := newTestStore(t)
	sess, _ := st.Create(context.Background(), "original", "")

	first, err := st.Get(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	first.Title = "mutated"

	second, err := st.Get(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if second.Title == "mutated" {
		t.Error("Get returned aliased pointer; mutation leaked through cache")
	}
}

func TestSave_PersistsAcrossStores(t *testing.T) {
	st, dir := newTestStore(t)
	sess, _ := st.Create(context.Background(), "t", "")

	sess.Title = "renamed"
	sess.Usage.Add(kit.Usage{InputTokens: 5, OutputTokens: 7})

	if err := st.Save(context.Background(), sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	st2, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	got, err := st2.Get(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Get on fresh store: %v", err)
	}

	if got.Title != "renamed" {
		t.Errorf("Title = %q, want renamed", got.Title)
	}

	if got.Usage.InputTokens != 5 || got.Usage.OutputTokens != 7 {
		t.Errorf("Usage = %+v, want input=5 output=7", got.Usage)
	}
}

func TestList_Empty(t *testing.T) {
	st, _ := newTestStore(t)

	got, err := st.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("List on empty store: got %d sessions", len(got))
	}
}

func TestList_SortsByUpdatedAtDesc(t *testing.T) {
	st, _ := newTestStore(t)
	ctx := context.Background()

	a, _ := st.Create(ctx, "a", "")

	time.Sleep(10 * time.Millisecond)

	b, _ := st.Create(ctx, "b", "")

	time.Sleep(10 * time.Millisecond)

	bump := session.NewErrorEvent(a.ID, errors.New("bump updated_at"))
	if err := st.AppendEvents(ctx, a.ID, bump); err != nil {
		t.Fatalf("AppendEvents: %v", err)
	}

	list, err := st.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("List: got %d sessions, want 2", len(list))
	}

	if list[0].ID != a.ID || list[1].ID != b.ID {
		t.Errorf("order = [%s, %s], want [%s, %s] (most recently updated first)",
			list[0].ID, list[1].ID, a.ID, b.ID)
	}
}

func TestList_IgnoresNonDirEntries(t *testing.T) {
	st, dir := newTestStore(t)

	stray := filepath.Join(dir, "stray.txt")
	if err := os.WriteFile(stray, []byte("not a session"), 0o600); err != nil {
		t.Fatalf("seed stray: %v", err)
	}

	sess, _ := st.Create(context.Background(), "real", "")

	list, err := st.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(list) != 1 || list[0].ID != sess.ID {
		t.Errorf("List: got %d sessions, want exactly the real one", len(list))
	}
}

func TestDelete_RemovesDirAndCache(t *testing.T) {
	st, dir := newTestStore(t)
	sess, _ := st.Create(context.Background(), "t", "")

	if err := st.Delete(context.Background(), sess.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, sess.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("session dir still exists; stat err = %v", err)
	}

	if _, err := st.Get(context.Background(), sess.ID); err == nil {
		t.Error("Get after Delete: expected error, got nil")
	}
}

func TestDelete_Idempotent(t *testing.T) {
	st, _ := newTestStore(t)

	if err := st.Delete(context.Background(), "never-existed"); err != nil {
		t.Errorf("Delete on missing id: %v", err)
	}
}

func TestAppendEvents_FiltersTransient(t *testing.T) {
	st, _ := newTestStore(t)
	ctx := context.Background()
	sess, _ := st.Create(ctx, "t", "")

	transient := session.NewContentDeltaEvent(sess.ID, kit.NewTextContent("x"))
	persistent := session.NewMessageEvent(sess.ID,
		kit.NewModelMessage([]kit.Content{kit.NewTextContent("hi")}))

	if err := st.AppendEvents(ctx, sess.ID, transient, persistent); err != nil {
		t.Fatalf("AppendEvents: %v", err)
	}

	events, err := st.LoadEvents(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("LoadEvents: got %d events, want 1", len(events))
	}

	if events[0].Type != session.EventMessage {
		t.Errorf("only event has type %q, want %q", events[0].Type, session.EventMessage)
	}
}

func TestAppendEvents_BumpsUpdatedAt(t *testing.T) {
	st, _ := newTestStore(t)
	ctx := context.Background()

	sess, _ := st.Create(ctx, "t", "")
	before := sess.UpdatedAt

	time.Sleep(5 * time.Millisecond)

	ev := session.NewMessageEvent(sess.ID,
		kit.NewModelMessage([]kit.Content{kit.NewTextContent("hi")}))
	if err := st.AppendEvents(ctx, sess.ID, ev); err != nil {
		t.Fatalf("AppendEvents: %v", err)
	}

	got, err := st.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if !got.UpdatedAt.After(before) {
		t.Errorf("UpdatedAt = %v, want after %v", got.UpdatedAt, before)
	}
}

func TestAppendEvents_PreservesOrderAcrossCalls(t *testing.T) {
	st, _ := newTestStore(t)
	ctx := context.Background()
	sess, _ := st.Create(ctx, "t", "")

	first := session.NewMessageEvent(sess.ID,
		kit.NewUserMessage([]kit.Content{kit.NewTextContent("one")}))
	second := session.NewMessageEvent(sess.ID,
		kit.NewModelMessage([]kit.Content{kit.NewTextContent("two")}))

	if err := st.AppendEvents(ctx, sess.ID, first); err != nil {
		t.Fatalf("AppendEvents first: %v", err)
	}

	if err := st.AppendEvents(ctx, sess.ID, second); err != nil {
		t.Fatalf("AppendEvents second: %v", err)
	}

	events, err := st.LoadEvents(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}

	if len(events) != 2 || events[0].ID != first.ID || events[1].ID != second.ID {
		t.Errorf("events out of order: %+v", events)
	}
}

func TestLoadEvents_Missing(t *testing.T) {
	st, _ := newTestStore(t)

	events, err := st.LoadEvents(context.Background(), "missing")
	if err != nil {
		t.Fatalf("LoadEvents on missing session: %v", err)
	}

	if events != nil {
		t.Errorf("got %d events, want nil", len(events))
	}
}

func TestLoadEvents_RoundTripsKitContent(t *testing.T) {
	st, _ := newTestStore(t)
	ctx := context.Background()
	sess, _ := st.Create(ctx, "t", "")

	msg := kit.NewModelMessage([]kit.Content{
		kit.NewTextContent("hello"),
		kit.NewToolCallContent(kit.NewToolCall("call-1", "tool", map[string]any{"k": "v"})),
	})

	if err := st.AppendEvents(ctx, sess.ID, session.NewMessageEvent(sess.ID, msg)); err != nil {
		t.Fatalf("AppendEvents: %v", err)
	}

	loaded, err := st.LoadEvents(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}

	if len(loaded) != 1 || loaded[0].Message == nil {
		t.Fatalf("loaded = %+v", loaded)
	}

	text := loaded[0].Message.TextContent()
	if text == nil || text.Text != "hello" {
		t.Errorf("text content lost: %v", text)
	}

	calls := loaded[0].Message.ToolCalls()
	if len(calls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(calls))
	}

	if calls[0].Name != "tool" || calls[0].Arguments["k"] != "v" {
		t.Errorf("tool call corrupted: %+v", calls[0])
	}
}
