package session

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/server"
	sessiondata "github.com/vitaliiPsl/crappy-ai/internal/session"
)

func loadHistoryCmd(ctx context.Context, srv *server.Server, sessionID string) tea.Cmd {
	return func() tea.Msg {
		events, err := srv.LoadEvents(ctx, sessionID)

		return historyLoadedMsg{events: events, err: err}
	}
}

func waitForEventCmd(ch <-chan sessiondata.Event) tea.Cmd {
	if ch == nil {
		return nil
	}

	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}

		return sessionEventMsg{event: ev}
	}
}

func sendCmd(ctx context.Context, srv *server.Server, sessionID, text string) tea.Cmd {
	return func() tea.Msg {
		if err := srv.Send(ctx, sessionID, text); err != nil {
			return effectErrorMsg{err: err}
		}

		return nil
	}
}

func compactCmd(ctx context.Context, srv *server.Server, sessionID string) tea.Cmd {
	return func() tea.Msg {
		if err := srv.Compact(ctx, sessionID); err != nil {
			return effectErrorMsg{err: err}
		}

		return nil
	}
}

func respondPromptCmd(srv *server.Server, sessionID, toolCallID string, resp model.AskResponse) tea.Cmd {
	return func() tea.Msg {
		if err := srv.RespondPrompt(sessionID, toolCallID, resp); err != nil {
			return effectErrorMsg{err: err}
		}

		return nil
	}
}

func updateModeCmd(srv *server.Server, mode config.Mode) tea.Cmd {
	return func() tea.Msg {
		cfg := srv.GetConfig()
		cfg.Mode = mode

		if err := srv.UpdateConfig(cfg); err != nil {
			return modeUpdatedMsg{err: err}
		}

		return modeUpdatedMsg{mode: mode}
	}
}
