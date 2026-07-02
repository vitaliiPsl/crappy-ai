package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/vitaliiPsl/crappy-ai/internal/server"
)

type Transport struct {
	ctx context.Context
	srv *server.Server
}

func NewTransport(ctx context.Context, srv *server.Server) *Transport {
	return &Transport{ctx: ctx, srv: srv}
}

func (t *Transport) Start(_ context.Context) error {
	_, err := tea.NewProgram(New(t.ctx, t.srv)).Run()

	return err
}
