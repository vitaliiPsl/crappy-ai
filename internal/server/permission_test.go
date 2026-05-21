package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/strategy"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func toolCall(id string) kit.ToolCall {
	return kit.NewToolCall(id, "read_file", map[string]any{"path": "/tmp/x"})
}

func askRequest(id string) model.AskRequest {
	result := strategy.Resolve(model.Permissions{Default: model.Ask}, toolCall(id))
	if result.AskRequest == nil {
		panic("ask request is nil")
	}

	return *result.AskRequest
}

func TestAsk_DeliversPromptAndResponse(t *testing.T) {
	srv, sess := newTestServer(t, &fakeAssistant{})

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch)

	want := model.AskResponse{OptionID: model.OptionAllowOnce}

	type result struct {
		resp model.AskResponse
		err  error
	}

	resultCh := make(chan result, 1)
	go func() {
		resp, err := srv.Ask(context.Background(), sess.ID, askRequest("call-1"))
		resultCh <- result{resp, err}
	}()

	ev := readEvent(t, ch)
	if ev.Type != session.EventPermissionPrompt {
		t.Fatalf("event type = %q, want %q", ev.Type, session.EventPermissionPrompt)
	}

	if ev.Prompt == nil || ev.Prompt.ToolCall.ID != "call-1" {
		t.Fatalf("event prompt = %+v, want tool call call-1", ev.Prompt)
	}

	if ev.Prompt.Request.Call.ID != "call-1" || len(ev.Prompt.Request.Options) == 0 {
		t.Fatalf("event request = %+v, want populated request for call-1", ev.Prompt.Request)
	}

	if err := srv.RespondPrompt(sess.ID, "call-1", want); err != nil {
		t.Fatalf("RespondPrompt: %v", err)
	}

	select {
	case r := <-resultCh:
		if r.err != nil {
			t.Fatalf("Ask err = %v", r.err)
		}

		if r.resp != want {
			t.Fatalf("Ask resp = %+v, want %+v", r.resp, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Ask to return")
	}
}

func TestAsk_ReturnsCtxErrorOnCancel(t *testing.T) {
	srv, sess := newTestServer(t, &fakeAssistant{})

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		_, err := srv.Ask(ctx, sess.ID, askRequest("call-2"))
		errCh <- err
	}()

	_ = readEvent(t, ch)

	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Ask err = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Ask to return after cancel")
	}

	if err := srv.RespondPrompt(sess.ID, "call-2", model.AskResponse{}); err == nil {
		t.Fatal("RespondPrompt after cancel should fail")
	}
}

func TestAttach_ReplaysPendingPrompts(t *testing.T) {
	srv, sess := newTestServer(t, &fakeAssistant{})

	primary, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, primary)

	go func() {
		_, _ = srv.Ask(context.Background(), sess.ID, askRequest("call-3"))
	}()

	_ = readEvent(t, primary)

	late, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("late Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, late)

	ev := readEvent(t, late)
	if ev.Type != session.EventPermissionPrompt || ev.Prompt == nil || ev.Prompt.ToolCall.ID != "call-3" {
		t.Fatalf("replayed event = %+v, want prompt for call-3", ev)
	}

	if err := srv.RespondPrompt(sess.ID, "call-3", model.AskResponse{OptionID: model.OptionAllowOnce}); err != nil {
		t.Fatalf("RespondPrompt: %v", err)
	}
}

func TestRespondPrompt_UnknownReturnsError(t *testing.T) {
	srv, sess := newTestServer(t, &fakeAssistant{})

	err := srv.RespondPrompt(sess.ID, "missing", model.AskResponse{})
	if err == nil {
		t.Fatal("RespondPrompt for missing session should fail")
	}
}

func TestRespondPrompt_TwiceReturnsError(t *testing.T) {
	srv, sess := newTestServer(t, &fakeAssistant{})

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	defer srv.Unsubscribe(sess.ID, ch)

	go func() {
		_, _ = srv.Ask(context.Background(), sess.ID, askRequest("call-twice"))
	}()

	_ = readEvent(t, ch)

	resp := model.AskResponse{OptionID: model.OptionAllowOnce}
	if err := srv.RespondPrompt(sess.ID, "call-twice", resp); err != nil {
		t.Fatalf("first RespondPrompt: %v", err)
	}

	if err := srv.RespondPrompt(sess.ID, "call-twice", resp); err == nil {
		t.Fatal("second RespondPrompt for same call should fail")
	}
}

func TestDetach_KeepsSessionWithPendingPrompt(t *testing.T) {
	srv, sess := newTestServer(t, &fakeAssistant{})

	ch, err := srv.Subscribe(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	go func() {
		_, _ = srv.Ask(context.Background(), sess.ID, askRequest("call-pending"))
	}()

	_ = readEvent(t, ch)

	srv.Unsubscribe(sess.ID, ch)

	if _, ok := srv.getSessionState(sess.ID); !ok {
		t.Fatal("session state should be kept while a prompt is pending")
	}

	if err := srv.RespondPrompt(sess.ID, "call-pending", model.AskResponse{OptionID: model.OptionAllowOnce}); err != nil {
		t.Fatalf("RespondPrompt: %v", err)
	}
}
