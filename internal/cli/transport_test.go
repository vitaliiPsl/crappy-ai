package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	sessionstore "github.com/vitaliiPsl/crappy-ai/internal/session/store"
)

type fakeAssistant struct {
	streamFn func(ctx context.Context, sessionID string) *kit.Stream[session.Event, struct{}]
	err      error
}

func (a fakeAssistant) Run(ctx context.Context, sessionID, _ string) (*kit.Stream[session.Event, struct{}], error) {
	if a.err != nil {
		return nil, a.err
	}

	return a.streamFn(ctx, sessionID), nil
}

func (a fakeAssistant) Compact(ctx context.Context, sessionID string) (*kit.Stream[session.Event, struct{}], error) {
	return a.Run(ctx, sessionID, "")
}

func TestTransport_PrintsTextAndUsageOnTurnComplete(t *testing.T) {
	srv := newTestServer(t, fakeAssistant{
		streamFn: func(_ context.Context, sessionID string) *kit.Stream[session.Event, struct{}] {
			return eventStream(
				session.NewContentDeltaEvent(sessionID, kit.NewTextContent("hello")),
				session.NewContentDeltaEvent(sessionID, kit.NewTextContent(" world")),
				session.NewTurnCompleteEvent(sessionID, session.TurnStats{
					Usage: kit.Usage{InputTokens: 7, OutputTokens: 11},
				}),
			)
		},
	})

	stdout, stderr, err := captureOutput(t, func() error {
		return NewTransport(srv, "say hello").Run(context.Background())
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if stdout != "hello world\n" {
		t.Fatalf("stdout = %q, want streamed text plus newline", stdout)
	}

	if stderr != "[usage: in=7 out=11]\n" {
		t.Fatalf("stderr = %q, want usage line", stderr)
	}
}

func TestTransport_TurnCompleteWithZeroStatsPrintsUsage(t *testing.T) {
	srv := newTestServer(t, fakeAssistant{
		streamFn: func(_ context.Context, sessionID string) *kit.Stream[session.Event, struct{}] {
			return eventStream(session.NewTurnCompleteEvent(sessionID, session.TurnStats{}))
		},
	})

	stdout, stderr, err := captureOutput(t, func() error {
		return NewTransport(srv, "done").Run(context.Background())
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if stdout != "\n" {
		t.Fatalf("stdout = %q, want completion newline", stdout)
	}

	if stderr != "[usage: in=0 out=0]\n" {
		t.Fatalf("stderr = %q, want zero usage line", stderr)
	}
}

func TestTransport_PrintsCancelledAndReturnsNil(t *testing.T) {
	srv := newTestServer(t, fakeAssistant{
		streamFn: func(_ context.Context, sessionID string) *kit.Stream[session.Event, struct{}] {
			return eventStream(session.NewTurnCancelledEvent(sessionID))
		},
	})

	stdout, stderr, err := captureOutput(t, func() error {
		return NewTransport(srv, "cancel").Run(context.Background())
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}

	if stderr != "\n[cancelled]\n" {
		t.Fatalf("stderr = %q, want cancellation notice", stderr)
	}
}

func TestTransport_PrintsErrorEventAndReturnsNil(t *testing.T) {
	srv := newTestServer(t, fakeAssistant{
		streamFn: func(_ context.Context, sessionID string) *kit.Stream[session.Event, struct{}] {
			return eventStream(session.NewErrorEvent(sessionID, errors.New("model down")))
		},
	})

	stdout, stderr, err := captureOutput(t, func() error {
		return NewTransport(srv, "fail").Run(context.Background())
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}

	if stderr != "\n[error] model down\n" {
		t.Fatalf("stderr = %q, want error notice", stderr)
	}
}

func TestTransport_ReturnsErrorOnPermissionPromptAndCancelsRun(t *testing.T) {
	cancelled := make(chan struct{})

	srv := newTestServer(t, fakeAssistant{
		streamFn: func(ctx context.Context, sessionID string) *kit.Stream[session.Event, struct{}] {
			return kit.NewStream(func(emit kit.Emitter[session.Event]) (struct{}, error) {
				call := kit.NewToolCall("call-1", "bash", map[string]any{"command": "go test ./..."})
				req := model.NewAskRequest(call, "go test ./...", nil)
				if err := emit.Emit(session.NewPermissionPromptEvent(sessionID, req)); err != nil {
					return struct{}{}, err
				}

				<-ctx.Done()
				close(cancelled)

				return struct{}{}, ctx.Err()
			})
		},
	})

	stdout, stderr, err := captureOutput(t, func() error {
		return NewTransport(srv, "run tests").Run(context.Background())
	})
	if err == nil {
		t.Fatal("Run error = nil, want permission prompt error")
	}

	if stdout != "" || stderr != "" {
		t.Fatalf("stdout/stderr = %q/%q, want no output before permission error", stdout, stderr)
	}

	got := err.Error()
	for _, want := range []string{"permission required", "bash", "-mode yolo"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Run error = %q, want it to mention %q", got, want)
		}
	}

	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("permission prompt did not cancel active run")
	}
}

func TestTransport_WrapsSendError(t *testing.T) {
	wantErr := errors.New("assistant failed")
	srv := newTestServer(t, fakeAssistant{err: wantErr})

	_, _, err := captureOutput(t, func() error {
		return NewTransport(srv, "fail before streaming").Run(context.Background())
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run error = %v, want wraps %v", err, wantErr)
	}

	if !strings.Contains(err.Error(), "send:") {
		t.Fatalf("Run error = %q, want send context", err)
	}
}

func TestPermissionPromptErrorWithoutPromptPayload(t *testing.T) {
	err := permissionPromptError(session.Event{})
	if err == nil {
		t.Fatal("permissionPromptError = nil, want error")
	}

	for _, want := range []string{"permission required", "non-interactive", "-mode yolo"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to mention %q", err, want)
		}
	}
}

func eventStream(events ...session.Event) *kit.Stream[session.Event, struct{}] {
	return kit.NewStream(func(emit kit.Emitter[session.Event]) (struct{}, error) {
		for _, ev := range events {
			if err := emit.Emit(ev); err != nil {
				return struct{}{}, err
			}
		}

		return struct{}{}, nil
	})
}

func newTestServer(t *testing.T, asst server.Assistant) *server.Server {
	t.Helper()

	store, err := sessionstore.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	configStore := config.NewStore(config.Config{Cwd: t.TempDir()}, "")

	return server.New(asst, nil, configStore, store, nil)
}

func captureOutput(t *testing.T, fn func() error) (string, string, error) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}

	stdoutCh := readAll(t, stdoutR)
	stderrCh := readAll(t, stderrR)

	os.Stdout = stdoutW
	os.Stderr = stderrW

	runErr := fn()

	_ = stdoutW.Close()
	_ = stderrW.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return <-stdoutCh, <-stderrCh, runErr
}

func readAll(t *testing.T, r *os.File) <-chan string {
	t.Helper()

	ch := make(chan string, 1)
	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- ""

			return
		}

		ch <- string(data)
	}()

	return ch
}
