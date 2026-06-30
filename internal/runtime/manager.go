package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/eventbus"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

type Manager struct {
	configStore   *config.Store
	sessionStore  session.Store
	modelRegistry *models.Registry
	permissions   *permission.Service

	mu   sync.Mutex
	live map[string]*Session
}

func NewManager(configStore *config.Store, sessionStore session.Store, modelRegistry *models.Registry) *Manager {
	return &Manager{
		configStore:   configStore,
		sessionStore:  sessionStore,
		modelRegistry: modelRegistry,
		permissions:   permission.NewService(configStore),
		live:          make(map[string]*Session),
	}
}

func (m *Manager) CreateSession(ctx context.Context, title string) (*session.Session, error) {
	return m.sessionStore.Create(ctx, session.CreateParams{
		Title: title,
		Cwd:   m.configStore.Get().Cwd,
	})
}

func (m *Manager) GetSession(ctx context.Context, id string) (*session.Session, error) {
	return m.sessionStore.Get(ctx, id)
}

func (m *Manager) ListSessions(ctx context.Context) ([]*session.Session, error) {
	return m.sessionStore.List(ctx)
}

func (m *Manager) DeleteSession(ctx context.Context, id string) error {
	return m.sessionStore.Delete(ctx, id)
}

func (m *Manager) LoadEvents(ctx context.Context, id string) ([]session.Event, error) {
	return m.sessionStore.LoadEvents(ctx, id)
}

func (m *Manager) Subscribe(ctx context.Context, sessionID string) (*eventbus.Subscription[session.Event], error) {
	if _, err := m.sessionStore.Get(ctx, sessionID); err != nil {
		return nil, err
	}

	return m.getOrCreate(sessionID).Subscribe(), nil
}

func (m *Manager) Send(ctx context.Context, sessionID string, req Request) error {
	return m.getOrCreate(sessionID).Send(ctx, req)
}

func (m *Manager) Compact(ctx context.Context, sessionID string) error {
	return m.getOrCreate(sessionID).Compact(ctx)
}

func (m *Manager) CancelRun(sessionID string) {
	if s, ok := m.get(sessionID); ok {
		s.Cancel()
	}
}

func (m *Manager) Respond(sessionID string, resp ask.Response) error {
	s, ok := m.get(sessionID)
	if !ok {
		return fmt.Errorf("no live session %q", sessionID)
	}

	return s.Respond(resp)
}

func (m *Manager) getOrCreate(sessionID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.live[sessionID]; ok {
		return s
	}

	s := newSession(sessionID, m.configStore, m.sessionStore, m.modelRegistry, m.permissions)
	m.live[sessionID] = s

	return s
}

func (m *Manager) get(sessionID string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.live[sessionID]

	return s, ok
}
