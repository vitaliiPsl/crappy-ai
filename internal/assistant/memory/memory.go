package memory

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

type sessionMemory struct {
	store     session.Store
	sessionID string
}

func New(store session.Store, sessionID string) kit.Memory {
	return &sessionMemory{
		store:     store,
		sessionID: sessionID,
	}
}

func (m *sessionMemory) Context(ctx context.Context) ([]kit.Message, error) {
	history, err := m.History(ctx)
	if err != nil {
		return nil, err
	}

	for i := len(history) - 1; i >= 0; i-- {
		if isSummary(history[i]) {
			return history[i:], nil
		}
	}

	return history, nil
}

func (m *sessionMemory) History(ctx context.Context) ([]kit.Message, error) {
	events, err := m.store.LoadEvents(ctx, m.sessionID)
	if err != nil {
		return nil, err
	}

	var messages []kit.Message
	for _, event := range events {
		if event.Type == session.EventMessage && event.Message != nil {
			messages = append(messages, *event.Message)
		}
	}

	return messages, nil
}

func (m *sessionMemory) Record(ctx context.Context, messages ...kit.Message) error {
	if len(messages) == 0 {
		return nil
	}

	events := make([]session.Event, 0, len(messages))
	for _, msg := range messages {
		events = append(events, session.NewMessageEvent(m.sessionID, msg))
	}

	return m.store.AppendEvents(ctx, m.sessionID, events...)
}

func isSummary(msg kit.Message) bool {
	for _, content := range msg.Content {
		if content.Type == kit.ContentTypeSummary && content.Summary != nil {
			return true
		}
	}

	return false
}
