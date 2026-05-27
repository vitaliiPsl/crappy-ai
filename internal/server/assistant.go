package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func (s *Server) Send(ctx context.Context, sessionID string, req assistant.RunRequest) error {
	return s.callAssistant(ctx, sessionID, func(turnCtx context.Context) (*kit.Stream[session.Event, struct{}], error) {
		return s.assistant.Run(turnCtx, sessionID, req)
	})
}

func (s *Server) Compact(ctx context.Context, sessionID string) error {
	return s.callAssistant(ctx, sessionID, func(turnCtx context.Context) (*kit.Stream[session.Event, struct{}], error) {
		return s.assistant.Compact(turnCtx, sessionID)
	})
}

func (s *Server) CancelRun(sessionID string) {
	st, ok := s.getSessionState(sessionID)
	if !ok {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if st.cancel != nil {
		st.cancel()
	}
}

func (s *Server) callAssistant(
	ctx context.Context,
	sessionID string,
	open func(context.Context) (*kit.Stream[session.Event, struct{}], error),
) error {
	st := s.getOrCreateSessionState(sessionID)

	st.mu.Lock()
	if st.cancel != nil {
		st.mu.Unlock()

		return fmt.Errorf("session %q already has an active turn", sessionID)
	}

	turnCtx, cancel := context.WithCancel(ctx)
	st.cancel = cancel
	st.mu.Unlock()

	stream, err := open(turnCtx)
	if err != nil {
		st.mu.Lock()
		st.cancel = nil
		st.mu.Unlock()
		cancel()

		return err
	}

	go s.consumeAssistantStream(turnCtx, sessionID, st, cancel, stream)

	return nil
}

func (s *Server) consumeAssistantStream(
	ctx context.Context,
	sessionID string,
	st *sessionState,
	cancel context.CancelFunc,
	stream *kit.Stream[session.Event, struct{}],
) {
	defer func() {
		cancel()

		st.mu.Lock()
		st.cancel = nil
		st.mu.Unlock()
	}()

	for event := range stream.Iter() {
		_ = s.broadcast(ctx, sessionID, event)
	}

	if _, err := stream.Result(); err != nil {
		_ = s.broadcast(context.Background(), sessionID, cancelledOrError(sessionID, err))
	}
}

func cancelledOrError(sessionID string, err error) session.Event {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return session.NewTurnCancelledEvent(sessionID)
	}

	return session.NewErrorEvent(sessionID, err)
}
