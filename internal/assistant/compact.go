package assistant

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/memory"
	"github.com/vitaliiPsl/crappy-ai/internal/assistant/summarization"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
)

func (a *Assistant) Compact(ctx context.Context, sessionID string) (*kit.Stream[session.Event, struct{}], error) {
	cfg := a.configStore.Get()

	model, err := a.modelRegistry.Build(cfg)
	if err != nil {
		return nil, fmt.Errorf("build model: %w", err)
	}

	mem := memory.New(a.sessionStore, sessionID)
	summarizer := summarization.NewSummarizer(model)

	return kit.NewStream(func(emit kit.Emitter[session.Event]) (struct{}, error) {
		result, runErr := summarizer.Summarize(ctx, mem, sessionEventEmitter(sessionID, emit))

		var usage kit.Usage
		if result != nil {
			usage = result.Usage
		}

		return struct{}{}, a.handleResult(ctx, sessionID, model.Config(), usage, usage, runErr, emit)
	}), nil
}

func sessionEventEmitter(sessionID string, emit kit.Emitter[session.Event]) func(kit.AgentEvent) error {
	return func(ev kit.AgentEvent) error {
		sessEv, ok := session.FromKitEvent(sessionID, ev)
		if !ok {
			return nil
		}

		return emit.Emit(sessEv)
	}
}
