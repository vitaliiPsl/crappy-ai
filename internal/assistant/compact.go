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
		result, runErr := a.compactSession(ctx, sessionID, mem, summarizer, emit)

		var (
			ev         session.Event
			handlerErr error
		)

		if runErr != nil {
			ev, handlerErr = a.handleRunError(ctx, sessionID, runErr)
		} else {
			ev, handlerErr = a.handleRunResult(ctx, sessionID, model.Config(), result.Usage, result.Usage)
		}

		if handlerErr != nil {
			return struct{}{}, handlerErr
		}

		if err := emit.Emit(ev); err != nil {
			return struct{}{}, err
		}

		return struct{}{}, nil
	}), nil
}

func (a *Assistant) compactSession(
	ctx context.Context,
	sessionID string,
	mem kit.Memory,
	summarizer *summarization.Summarizer,
	emit kit.Emitter[session.Event],
) (summarization.Result, error) {
	messages, err := mem.Context(ctx)
	if err != nil {
		return summarization.Result{}, fmt.Errorf("compact: read memory context: %w", err)
	}

	if len(messages) == 0 {
		return summarization.Result{}, nil
	}

	if err := emit.Emit(session.NewContentStartedEvent(sessionID, kit.NewSummaryContent(""))); err != nil {
		return summarization.Result{}, err
	}

	result, err := summarizer.Summarize(ctx, messages)
	if err != nil {
		return summarization.Result{}, err
	}

	content := kit.NewSummaryContent(result.Text)
	summary := kit.NewUserMessage(content)

	if err := mem.Record(ctx, summary); err != nil {
		return summarization.Result{}, fmt.Errorf("compact: record summary: %w", err)
	}

	if err := emit.Emit(session.NewContentDoneEvent(sessionID, content)); err != nil {
		return result, err
	}

	if err := emit.Emit(session.NewMessageEvent(sessionID, summary)); err != nil {
		return result, err
	}

	return result, nil
}
