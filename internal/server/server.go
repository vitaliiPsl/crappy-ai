package server

import (
	"context"
	"sync"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	"github.com/vitaliiPsl/crappy-ai/internal/skills"
)

type Transport interface {
	Run(ctx context.Context) error
}

type Assistant interface {
	Run(ctx context.Context, sessionID string, req assistant.RunRequest) (*kit.Stream[session.Event, struct{}], error)
	Compact(ctx context.Context, sessionID string) (*kit.Stream[session.Event, struct{}], error)
}

type Server struct {
	assistant  Assistant
	transports []Transport

	settingsStore  *settings.Store
	configStore    *config.Store
	sessionStore   session.Store
	modelsRegistry *models.Registry
	skillRegistry  *skills.Registry

	mu       sync.RWMutex
	sessions map[string]*sessionState
}

func New(
	assistant Assistant,
	settingsStore *settings.Store,
	configStore *config.Store,
	sessionStore session.Store,
	modelsRegistry *models.Registry,
	skillRegistry *skills.Registry,
) *Server {
	return &Server{
		assistant:      assistant,
		settingsStore:  settingsStore,
		configStore:    configStore,
		sessionStore:   sessionStore,
		modelsRegistry: modelsRegistry,
		skillRegistry:  skillRegistry,
		sessions:       make(map[string]*sessionState),
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

func (s *Server) Subscribe(ctx context.Context, sessionID string) (<-chan session.Event, error) {
	if _, err := s.sessionStore.Get(ctx, sessionID); err != nil {
		return nil, err
	}

	st := s.getOrCreateSessionState(sessionID)

	sub, err := st.subscribe(ctx)
	if err != nil {
		return nil, err
	}

	return sub.events(), nil
}

func (s *Server) Unsubscribe(sessionID string, ch <-chan session.Event) {
	st, ok := s.getSessionState(sessionID)
	if !ok {
		return
	}

	removed, last := st.unsubscribe(ch)
	if !removed || !last {
		return
	}

	s.cleanup(sessionID)
}

func (s *Server) cleanup(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.sessions[sessionID]
	if !ok {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	if st.idle() {
		delete(s.sessions, sessionID)
	}
}

func (s *Server) broadcast(ctx context.Context, sessionID string, ev session.Event) error {
	st, ok := s.getSessionState(sessionID)
	if !ok {
		return nil
	}

	return st.broadcast(ctx, ev)
}
