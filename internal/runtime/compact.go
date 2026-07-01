package runtime

import (
	"context"
	"fmt"

	adk "github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/summarization"

	appagent "github.com/vitaliiPsl/crappy-ai/internal/agent"
)

const compactInstructions = `You are compacting a conversation between a user and an AI coding assistant so it can continue with less context.

Write a summary that preserves everything needed to keep working:
- the user's goals and any explicit instructions or constraints
- decisions made and the current state of the work
- files, commands, and findings that matter going forward
- open questions and the next steps

Be factual and concise. Do not address the user or add commentary; write the summary as notes for your future self.`

type compactionContributor struct{}

func (compactionContributor) Contribute(_ context.Context, req appagent.Request) (appagent.Contribution, error) {
	if req.Config.CompactThreshold <= 0 {
		return appagent.Contribution{}, nil
	}

	return appagent.Contribution{
		Options: []adk.Option{
			adk.WithOnTurnStart(compactionHook(req.Model, req.Config.CompactThreshold)),
		},
	}, nil
}

func compactionHook(model kit.Model, percent int) kit.OnTurnStart {
	threshold := int64(model.Config().InputLimit) * int64(percent) / 100

	return func(rc *kit.RunContext) error {
		if threshold <= 0 || rc.LastUsage.InputTokens <= threshold {
			return nil
		}

		if err := rc.Emit(kit.NewAgentContentStartedEvent(kit.NewSummaryContent(""))); err != nil {
			return err
		}

		usage, err := summarize(rc.Context, model, rc.Memory)
		if err != nil {
			return err
		}

		rc.Usage.Add(usage)

		return rc.Emit(kit.NewAgentContentDoneEvent(kit.NewSummaryContent("")))
	}
}

func summarize(ctx context.Context, model kit.Model, mem kit.Memory) (kit.Usage, error) {
	messages, err := mem.Context(ctx)
	if err != nil {
		return kit.Usage{}, fmt.Errorf("load context: %w", err)
	}

	if len(messages) == 0 {
		return kit.Usage{}, nil
	}

	resp, err := model.Generate(ctx, kit.ModelRequest{
		Instructions: compactInstructions,
		Messages: []kit.Message{
			kit.NewUserMessage(kit.NewTextContent(summarization.Flatten(messages))),
		},
	})
	if err != nil {
		return kit.Usage{}, fmt.Errorf("summarize: %w", err)
	}

	text := resp.Message.TextContent()
	if text == nil || text.Text == "" {
		return kit.Usage{}, fmt.Errorf("summarizer returned an empty response")
	}

	summary := kit.NewUserMessage(kit.NewSummaryContent(text.Text))
	if err := mem.Record(ctx, summary); err != nil {
		return kit.Usage{}, fmt.Errorf("record summary: %w", err)
	}

	return resp.Usage, nil
}
