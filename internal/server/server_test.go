package server

import (
	"context"
	"testing"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/runtime"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	sessionstore "github.com/vitaliiPsl/crappy-ai/internal/session/store"
)

func TestSubscribe_UnknownSessionReturnsError(t *testing.T) {
	srv, _ := newTestServer(t)

	if _, err := srv.Subscribe(context.Background(), "missing"); err == nil {
		t.Fatal("Subscribe for missing session should fail")
	}
}

func TestUnsubscribeRemovesSubscription(t *testing.T) {
	srv, sess := newTestServer(t)

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	srv.Unsubscribe(sess.ID, ch)

	if len(srv.subscriptions) != 0 {
		t.Fatalf("subscriptions len = %d, want 0", len(srv.subscriptions))
	}
}

func newTestServer(t *testing.T) (*Server, *session.Session) {
	t.Helper()

	store, err := sessionstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	cfg := config.NewStore(config.Config{Cwd: t.TempDir()}, "")
	rt := runtime.NewManager(cfg, store, nil, nil, nil, nil, nil)
	t.Cleanup(rt.Close)

	srv := New(rt, nil, cfg, nil, nil, nil, nil)

	sess, err := srv.CreateSession(context.Background(), "test")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	return srv, sess
}
