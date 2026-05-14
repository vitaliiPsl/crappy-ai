package assistant

import (
	"encoding/json"
	"math"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/summarization"
)

const (
	summarizationThresholdRatio = 0.75
	estimatedCharsPerToken      = 4
	messageTokenOverhead        = 4
)

const summarizationPrompt = `Create a detailed but compact summary of this conversation to use as context when continuing.

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
This summary must compact, but at the same time be sufficient to continue without the original conversation.`

func newSessionSummarization(model kit.Model) agent.Option {
	return summarization.WithSummarization(
		whenEstimatedContextExceeds(model.Config(), summarizationThresholdRatio),
		summarization.Summarize(model, summarizationPrompt),
	)
}

func whenEstimatedContextExceeds(config kit.ModelConfig, ratio float64) summarization.Trigger {
	limit := inputLimit(config)
	if limit <= 0 {
		return func(*kit.RunContext) bool { return false }
	}

	if ratio <= 0 || ratio > 1 {
		ratio = summarizationThresholdRatio
	}

	threshold := max(int(math.Floor(float64(limit)*ratio)), 1)

	return func(rc *kit.RunContext) bool {
		messages, err := rc.Memory.Context(rc.Context)
		if err != nil {
			return false
		}

		return estimateMessagesTokens(messages) >= threshold
	}
}

func inputLimit(config kit.ModelConfig) int {
	if config.InputLimit > 0 {
		return config.InputLimit
	}

	if config.ContextWindow <= 0 {
		return 0
	}

	if config.OutputLimit > 0 && config.OutputLimit < config.ContextWindow {
		return config.ContextWindow - config.OutputLimit
	}

	return config.ContextWindow
}

func estimateMessagesTokens(messages []kit.Message) int {
	total := 0
	for _, msg := range messages {
		total += messageTokenOverhead
		for _, content := range msg.Content {
			total += estimateContentTokens(content)
		}
	}

	return total
}

func estimateContentTokens(content kit.Content) int {
	chars := 0

	switch content.Type {
	case kit.ContentTypeText:
		if content.Text != nil {
			chars = len(content.Text.Text)
		}
	case kit.ContentTypeThinking:
		if content.Thinking != nil {
			chars = len(content.Thinking.Text)
		}
	case kit.ContentTypeSummary:
		if content.Summary != nil {
			chars = len(content.Summary.Text)
		}
	case kit.ContentTypeToolCall:
		chars = jsonLen(content.ToolCall)
	case kit.ContentTypeToolResult:
		chars = jsonLen(content.ToolResult)
	}

	if chars == 0 {
		return 0
	}

	return int(math.Ceil(float64(chars) / estimatedCharsPerToken))
}

func jsonLen(value any) int {
	if value == nil {
		return 0
	}

	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}

	return len(data)
}
