package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/server"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

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

	ch, err := t.srv.Attach(ctx, sess.ID)
	if err != nil {
		return fmt.Errorf("attach: %w", err)
	}

	defer t.srv.Detach(sess.ID, ch)

	if err := t.srv.RunTurn(ctx, sess.ID, t.prompt); err != nil {
		return fmt.Errorf("run turn: %w", err)
	}

	for ev := range ch {
		switch ev.Type {
		case session.EventContentDelta:
			if ev.Content != nil && ev.Content.Type == kit.ContentTypeText && ev.Content.Text != nil {
				fmt.Print(ev.Content.Text.Text)
			}
		case session.EventTurnComplete:
			fmt.Println()

			if ev.Stats != nil {
				fmt.Fprintf(os.Stderr, "[usage: in=%d out=%d]\n", ev.Stats.Usage.InputTokens, ev.Stats.Usage.OutputTokens)
			}

			return nil
		case session.EventTurnCancelled:
			fmt.Fprintln(os.Stderr, "\n[cancelled]")

			return nil
		case session.EventError:
			fmt.Fprintf(os.Stderr, "\n[error] %s\n", ev.Error)

			return nil
		}
	}

	return nil
}
