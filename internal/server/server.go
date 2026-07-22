package server

import (
	"context"
	"sync"

	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/memory"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/runtime"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/settings"
	"github.com/vitaliiPsl/crappy-ai/internal/skills"
)

type Transport interface {
	Start(ctx context.Context) error
}

type Server struct {
	runtime    *runtime.Manager
	transports []Transport

	settingsStore  *settings.Store
	configStore    *config.Store
	modelsRegistry *models.Registry
	skillRegistry  *skills.Registry
	mcpManager     *mcp.Manager
	background     *background.Manager
	memoryStore    memory.Store

	mu            sync.Mutex
	subscriptions map[<-chan session.Event]func()
}

func New(
	runtimeManager *runtime.Manager,
	settingsStore *settings.Store,
	configStore *config.Store,
	modelsRegistry *models.Registry,
	skillRegistry *skills.Registry,
	mcpManager *mcp.Manager,
	backgroundManager *background.Manager,
	memoryStore memory.Store,
) *Server {
	return &Server{
		runtime:        runtimeManager,
		settingsStore:  settingsStore,
		configStore:    configStore,
		modelsRegistry: modelsRegistry,
		skillRegistry:  skillRegistry,
		mcpManager:     mcpManager,
		background:     backgroundManager,
		memoryStore:    memoryStore,
		subscriptions:  make(map[<-chan session.Event]func()),
	}
}

func (s *Server) AddTransport(t Transport) {
	s.transports = append(s.transports, t)
}

func (s *Server) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errc := make(chan error, len(s.transports))
	for _, t := range s.transports {
		go func() {
			errc <- t.Start(ctx)
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
	sub, err := s.runtime.Subscribe(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	ch := sub.Events()

	s.mu.Lock()
	s.subscriptions[ch] = sub.Close
	s.mu.Unlock()

	return ch, nil
}

func (s *Server) Unsubscribe(_ string, ch <-chan session.Event) {
	s.mu.Lock()

	closeSub, ok := s.subscriptions[ch]
	if ok {
		delete(s.subscriptions, ch)
	}
	s.mu.Unlock()

	if ok {
		closeSub()
	}
}
