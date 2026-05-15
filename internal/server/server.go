package server

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
)

const eventBuffer = 64

type Transport interface {
	Run(ctx context.Context) error
}

type Assistant interface {
	Run(ctx context.Context, sessionID, text string) (*kit.Stream[session.Event, struct{}], error)
	Compact(ctx context.Context, sessionID string) (*kit.Stream[session.Event, struct{}], error)
}

type Server struct {
	assistant  Assistant
	transports []Transport

	settingsStore *settings.Store
	configStore   *config.Store
	sessionStore  session.Store
	registry      *models.Registry

	mu       sync.RWMutex
	sessions map[string]*sessionState
}

type sessionState struct {
	mu         sync.Mutex
	clients    []chan session.Event
	cancelTurn context.CancelFunc
}

func New(
	assistant Assistant,
	settingsStore *settings.Store,
	configStore *config.Store,
	sessionStore session.Store,
	registry *models.Registry,
) *Server {
	return &Server{
		assistant:     assistant,
		settingsStore: settingsStore,
		configStore:   configStore,
		sessionStore:  sessionStore,
		registry:      registry,
		sessions:      make(map[string]*sessionState),
	}
}

func (s *Server) AddTransport(t Transport) {
	s.transports = append(s.transports, t)
}

func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errc := make(chan error, len(s.transports))
	for _, t := range s.transports {
		go func() {
			errc <- t.Run(ctx)
		}()
	}

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) Attach(ctx context.Context, sessionID string) (<-chan session.Event, error) {
	if _, err := s.sessionStore.Get(ctx, sessionID); err != nil {
		return nil, err
	}

	ch := make(chan session.Event, eventBuffer)

	st := s.getOrCreateSessionState(sessionID)
	st.mu.Lock()
	st.clients = append(st.clients, ch)
	st.mu.Unlock()

	return ch, nil
}

func (s *Server) Detach(sessionID string, ch <-chan session.Event) {
	s.mu.RLock()
	st, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	st.mu.Lock()
	for i, c := range st.clients {
		if c == ch {
			st.clients = append(st.clients[:i], st.clients[i+1:]...)

			close(c)

			break
		}
	}

	isEmpty := len(st.clients) == 0
	st.mu.Unlock()

	if !isEmpty {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st2, ok := s.sessions[sessionID]
	if !ok {
		return
	}

	st2.mu.Lock()
	defer st2.mu.Unlock()

	if len(st2.clients) == 0 {
		delete(s.sessions, sessionID)
	}
}

func (s *Server) RunTurn(ctx context.Context, sessionID, text string) error {
	return s.startTurn(ctx, sessionID, func(turnCtx context.Context) (*kit.Stream[session.Event, struct{}], error) {
		return s.assistant.Run(turnCtx, sessionID, text)
	})
}

func (s *Server) Compact(ctx context.Context, sessionID string) error {
	return s.startTurn(ctx, sessionID, func(turnCtx context.Context) (*kit.Stream[session.Event, struct{}], error) {
		return s.assistant.Compact(turnCtx, sessionID)
	})
}

func (s *Server) startTurn(
	ctx context.Context,
	sessionID string,
	open func(context.Context) (*kit.Stream[session.Event, struct{}], error),
) error {
	st := s.getOrCreateSessionState(sessionID)

	st.mu.Lock()
	if st.cancelTurn != nil {
		st.mu.Unlock()

		return fmt.Errorf("session %q already has an active turn", sessionID)
	}

	turnCtx, cancel := context.WithCancel(ctx)
	st.cancelTurn = cancel
	st.mu.Unlock()

	stream, err := open(turnCtx)
	if err != nil {
		st.mu.Lock()
		st.cancelTurn = nil
		st.mu.Unlock()
		cancel()

		return err
	}

	go s.consumeTurnStream(sessionID, st, cancel, stream)

	return nil
}

func (s *Server) consumeTurnStream(
	sessionID string,
	st *sessionState,
	cancel context.CancelFunc,
	stream *kit.Stream[session.Event, struct{}],
) {
	defer func() {
		cancel()

		st.mu.Lock()
		st.cancelTurn = nil
		st.mu.Unlock()
	}()

	for event := range stream.Iter() {
		s.fanOut(sessionID, event)
	}

	if _, err := stream.Result(); err != nil {
		s.fanOut(sessionID, cancelledOrError(sessionID, err))
	}
}

func cancelledOrError(sessionID string, err error) session.Event {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return session.NewTurnCancelledEvent(sessionID)
	}

	return session.NewErrorEvent(sessionID, err)
}

func (s *Server) CancelTurn(sessionID string) {
	s.mu.RLock()
	st, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if st.cancelTurn != nil {
		st.cancelTurn()
	}
}

func (s *Server) fanOut(sessionID string, ev session.Event) {
	s.mu.RLock()
	st, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	st.mu.Lock()
	clients := make([]chan session.Event, len(st.clients))
	copy(clients, st.clients)
	st.mu.Unlock()

	for _, ch := range clients {
		safeSend(ch, ev)
	}
}

func safeSend(ch chan session.Event, ev session.Event) {
	defer func() { _ = recover() }()

	select {
	case ch <- ev:
	default:
	}
}
