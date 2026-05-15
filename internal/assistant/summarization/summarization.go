package summarization

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/agent"
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

const (
	thresholdRatio         = 0.75
	estimatedCharsPerToken = 4
	messageTokenOverhead   = 4
)

func New(model kit.Model) agent.Option {
	summarizer := NewSummarizer(model)

	return xsummarization.WithSummarization(
		whenEstimatedContextExceeds(model.Config(), thresholdRatio),
		strategy(summarizer),
	)
}

func strategy(summarizer *Summarizer) xsummarization.Strategy {
	return func(rc *kit.RunContext) error {
		messages, err := rc.Memory.Context(rc.Context)
		if err != nil {
			return fmt.Errorf("summarization: read memory context: %w", err)
		}

		if len(messages) == 0 {
			return nil
		}

		if err := rc.Emit(kit.NewAgentContentStartedEvent(kit.NewSummaryContent(""))); err != nil {
			return err
		}

		result, err := summarizer.Summarize(rc.Context, messages)
		if err != nil {
			return err
		}

		content := kit.NewSummaryContent(result.Text)
		summary := kit.NewUserMessage(content)

		rc.Usage.Add(result.Usage)
		rc.Append(summary)

		if err := rc.Memory.Record(rc.Context, summary); err != nil {
			return fmt.Errorf("summarization: record summary: %w", err)
		}

		if err := rc.Emit(kit.NewAgentContentDoneEvent(content)); err != nil {
			return err
		}

		return rc.Emit(kit.NewAgentMessageEvent(summary))
	}
}
