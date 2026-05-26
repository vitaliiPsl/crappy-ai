package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/server"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

const maxToolArgLen = 160

type Transport struct {
	srv    *server.Server
	prompt string
}

func NewTransport(srv *server.Server, prompt string) *Transport {
	return &Transport{srv: srv, prompt: prompt}
}

func (t *Transport) Run(ctx context.Context) error {
	sess, err := t.srv.CreateSession(ctx, "cli")
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	ch, err := t.srv.Subscribe(ctx, sess.ID)
	if err != nil {
		return fmt.Errorf("attach: %w", err)
	}

	defer t.srv.Unsubscribe(sess.ID, ch)

	if err := t.srv.Send(ctx, sess.ID, t.prompt); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	for ev := range ch {
		switch ev.Type {
		case session.EventContentDelta:
			renderContentDelta(ev.Content)
		case session.EventContentDone:
			renderContentDone(ev.Content)
		case session.EventTurnComplete:
			renderTurnComplete(ev.Stats)

			return nil
		case session.EventTurnCancelled:
			fmt.Fprintln(os.Stderr, "\n[cancelled]")

			return nil
		case session.EventError:
			fmt.Fprintf(os.Stderr, "\n[error] %s\n", ev.Error)

			return nil
		case session.EventPermissionPrompt:
			t.srv.CancelRun(sess.ID)

			return permissionPromptError(ev)
		}
	}

	return nil
}

func renderContentDelta(content *kit.Content) {
	if content == nil || content.Type != kit.ContentTypeText || content.Text == nil {
		return
	}

	fmt.Print(content.Text.Text)
}

func renderContentDone(content *kit.Content) {
	if content == nil {
		return
	}

	switch content.Type {
	case kit.ContentTypeToolCall:
		if content.ToolCall != nil {
			renderToolCall(*content.ToolCall)
		}
	case kit.ContentTypeToolResult:
		if content.ToolResult != nil {
			renderToolResult(*content.ToolResult)
		}
	}
}

func renderTurnComplete(stats *session.TurnStats) {
	fmt.Println()

	if stats == nil {
		return
	}

	fmt.Fprintf(os.Stderr, "[usage: in=%d out=%d]\n", stats.Usage.InputTokens, stats.Usage.OutputTokens)
}

func renderToolCall(call kit.ToolCall) {
	if arg := toolInlineArg(call.Arguments); arg != "" {
		fmt.Fprintf(os.Stderr, "[tool] %s %s\n", call.Name, arg)

		return
	}

	fmt.Fprintf(os.Stderr, "[tool] %s\n", call.Name)
}

func renderToolResult(result kit.ToolResult) {
	if result.Error == "" {
		return
	}

	fmt.Fprintf(os.Stderr, "[tool:error] %s %s\n", result.Call.Name, result.Error)
}

func toolInlineArg(args map[string]any) string {
	for _, key := range []string{"command", "path", "url", "description"} {
		if value, _ := args[key].(string); value != "" {
			return truncateInline(value)
		}
	}

	return ""
}

func truncateInline(value string) string {
	value = strings.TrimSpace(value)
	if before, _, ok := strings.Cut(value, "\n"); ok {
		value = before
	}

	if len(value) <= maxToolArgLen {
		return value
	}

	return value[:maxToolArgLen-3] + "..."
}

func permissionPromptError(ev session.Event) error {
	if ev.Prompt == nil {
		return fmt.Errorf("permission required in non-interactive CLI mode; use the TUI or rerun with -mode yolo")
	}

	return fmt.Errorf(
		"permission required for tool %q in non-interactive CLI mode; use the TUI or rerun with -mode yolo",
		ev.Prompt.ToolCall.Name,
	)
}
