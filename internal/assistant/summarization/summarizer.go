package summarization

import (
	"context"
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"
	xsummarization "github.com/vitaliiPsl/crappy-adk/x/summarization"
)

const Prompt = `Create a detailed but compact summary of this conversation to use as context when continuing.

Include these sections:

1. Goal — what the user is trying to accomplish
2. Key concepts — domain terms, constraints, and requirements established
3. Artifacts — any important data, content, decisions, or resources produced or referenced
4. Corrections and decisions — every significant correction or deliberate choice made, what was wrong or considered, what was agreed
5. User messages — every user message verbatim, in order
6. Current state — exactly where the conversation ended and what question or task is next
7. Pending work — concrete remaining tasks

Use bullet points and inline code or content blocks rather than prose.
Every line should carry information a new agent needs to continue.
Omit greetings, filler, duplicate wording, and anything with no future relevance.
This summary must be compact, but at the same time be sufficient to continue without the original conversation.`

type Summarizer struct {
	model kit.Model
}

type Result struct {
	Summary kit.Message
	Usage   kit.Usage
}

func NewSummarizer(model kit.Model) *Summarizer {
	return &Summarizer{model: model}
}

func (s *Summarizer) Summarize(ctx context.Context, mem kit.Memory, emit func(kit.AgentEvent) error) (*Result, error) {
	messages, err := mem.Context(ctx)
	if err != nil {
		return nil, fmt.Errorf("summarization: read memory context: %w", err)
	}

	if len(messages) == 0 {
		return nil, nil
	}

	if err := emit(kit.NewAgentContentStartedEvent(kit.NewSummaryContent(""))); err != nil {
		return nil, err
	}

	text, usage, err := generateSummary(ctx, s.model, messages)
	if err != nil {
		return nil, err
	}

	content := kit.NewSummaryContent(text)
	summary := kit.NewUserMessage(content)

	if err := mem.Record(ctx, summary); err != nil {
		return nil, fmt.Errorf("summarization: record summary: %w", err)
	}

	if err := emit(kit.NewAgentContentDoneEvent(content)); err != nil {
		return nil, err
	}

	if err := emit(kit.NewAgentMessageEvent(summary)); err != nil {
		return nil, err
	}

	return &Result{Summary: summary, Usage: usage}, nil
}

func generateSummary(ctx context.Context, model kit.Model, messages []kit.Message) (string, kit.Usage, error) {
	resp, err := model.Generate(ctx, kit.ModelRequest{
		Instructions: Prompt,
		Messages: []kit.Message{
			kit.NewUserMessage(kit.NewTextContent(xsummarization.Flatten(messages))),
		},
	})
	if err != nil {
		return "", kit.Usage{}, fmt.Errorf("summarization: model call failed: %w", err)
	}

	text := resp.Message.TextContent()
	if text == nil || text.Text == "" {
		return "", kit.Usage{}, fmt.Errorf("summarization: model returned no text (finish=%q)", resp.FinishReason)
	}

	return text.Text, resp.Usage, nil
}
