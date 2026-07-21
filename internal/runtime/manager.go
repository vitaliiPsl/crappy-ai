package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/eventbus"
	"github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/memory"
	"github.com/vitaliiPsl/crappy-ai/internal/models"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/skills"
)

type Manager struct {
	mu   sync.Mutex
	live map[string]*Session

	configStore       *config.Store
	sessionStore      session.Store
	memoryStore       memory.Store
	permissions       *permission.Service
	modelRegistry     *models.Registry
	skillRegistry     *skills.Registry
	mcpManager        *mcp.Manager
	backgroundManager *background.Manager
}

func NewManager(
	configStore *config.Store,
	sessionStore session.Store,
	memoryStore memory.Store,
	modelRegistry *models.Registry,
	skillRegistry *skills.Registry,
	mcpManager *mcp.Manager,
	backgroundManager *background.Manager,
) *Manager {
	return &Manager{
		live:              make(map[string]*Session),
		configStore:       configStore,
		sessionStore:      sessionStore,
		memoryStore:       memoryStore,
		permissions:       permission.NewService(configStore),
		modelRegistry:     modelRegistry,
		skillRegistry:     skillRegistry,
		mcpManager:        mcpManager,
		backgroundManager: backgroundManager,
	}
}

func (m *Manager) Close() {
	m.mu.Lock()

	sessions := make([]*Session, 0, len(m.live))
	for _, session := range m.live {
		sessions = append(sessions, session)
	}
	m.mu.Unlock()

	for _, session := range sessions {
		session.Close()
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
	sessionRuntime, err := m.session(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return sessionRuntime.Subscribe(), nil
}

func (m *Manager) ForkSession(ctx context.Context, sessionID, title string) (*session.Session, error) {
	sessionRuntime, err := m.session(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return sessionRuntime.Fork(ctx, title)
}

func (m *Manager) Run(ctx context.Context, sessionID string, req Request) error {
	sessionRuntime, err := m.session(ctx, sessionID)
	if err != nil {
		return err
	}

	return sessionRuntime.Run(ctx, req)
}

func (m *Manager) RunSubagent(ctx context.Context, sessionID string, req SubagentRequest) (SubagentResult, error) {
	sessionRuntime, err := m.session(ctx, sessionID)
	if err != nil {
		return SubagentResult{}, err
	}

	return sessionRuntime.RunSubagent(ctx, req)
}

func (m *Manager) Compact(ctx context.Context, sessionID string) error {
	sessionRuntime, err := m.session(ctx, sessionID)
	if err != nil {
		return err
	}

	return sessionRuntime.Compact(ctx)
}

func (m *Manager) Cancel(sessionID string) {
	if session, ok := m.get(sessionID); ok {
		session.Cancel()
	}
}

func (m *Manager) Queue(sessionID string) ([]QueuedRequest, error) {
	sessionRuntime, ok := m.get(sessionID)
	if !ok {
		return nil, fmt.Errorf("no live session %q", sessionID)
	}

	return sessionRuntime.Queue(), nil
}

func (m *Manager) UpdateQueued(sessionID, id string, req Request) error {
	sessionRuntime, ok := m.get(sessionID)
	if !ok {
		return fmt.Errorf("no live session %q", sessionID)
	}

	return sessionRuntime.UpdateQueued(id, req)
}

func (m *Manager) RemoveQueued(sessionID, id string) error {
	sessionRuntime, ok := m.get(sessionID)
	if !ok {
		return fmt.Errorf("no live session %q", sessionID)
	}

	return sessionRuntime.RemoveQueued(id)
}

func (m *Manager) Respond(sessionID string, resp ask.Response) error {
	session, ok := m.get(sessionID)
	if !ok {
		return fmt.Errorf("no live session %q", sessionID)
	}

	return session.Respond(resp)
}

func (m *Manager) session(ctx context.Context, sessionID string) (*Session, error) {
	if _, err := m.sessionStore.Get(ctx, sessionID); err != nil {
		return nil, err
	}

	return m.getOrCreate(sessionID), nil
}

func (m *Manager) getOrCreate(sessionID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.live[sessionID]; ok {
		return session
	}

	session := newSession(
		sessionID,
		m.configStore,
		m.sessionStore,
		m.memoryStore,
		m.permissions,
		m.modelRegistry,
		m.skillRegistry,
		m.mcpManager,
		m.backgroundManager,
	)

	m.live[sessionID] = session

	return session
}

func (m *Manager) get(sessionID string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.live[sessionID]

	return session, ok
}
