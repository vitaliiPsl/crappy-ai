package runtime

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

// agentMemory is a kit.Memory backed by a session's persisted event log. Context
// returns the full history truncated to the most recent summary, so the agent
// continues from a compaction rather than replaying everything.
type agentMemory struct {
	store     session.Store
	sessionID string
}

func newMemory(store session.Store, sessionID string) kit.Memory {
	return &agentMemory{store: store, sessionID: sessionID}
}

func (m *agentMemory) Context(ctx context.Context) ([]kit.Message, error) {
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

func (m *agentMemory) History(ctx context.Context) ([]kit.Message, error) {
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

func (m *agentMemory) Record(ctx context.Context, messages ...kit.Message) error {
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
