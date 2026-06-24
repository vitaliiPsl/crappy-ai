package summarization

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	xsummarization "github.com/vitaliiPsl/crappy-adk/x/summarization"

	"github.com/vitaliiPsl/crappy-ai/internal/assistant/factory"
)

const thresholdRatio = 0.75

type ext struct{}

func New() factory.Extension {
	return &ext{}
}

func (e *ext) Name() string {
	return "summarization"
}

func (e *ext) Options(_ context.Context, req factory.BuildRequest) ([]kit.Tool, []agent.Option, error) {
	return nil, []agent.Option{
		xsummarization.WithSummarization(
			whenLastInputExceeds(req.Model.Config(), thresholdRatio),
			strategy(NewSummarizer(req.Model)),
		),
	}, nil
}

func strategy(summarizer *Summarizer) func(*kit.RunContext) error {
	return func(rc *kit.RunContext) error {
		result, err := summarizer.Summarize(rc.Context, rc.Memory, rc.Emit)
		if err != nil || result == nil {
			return err
		}

		rc.Usage.Add(result.Usage)
		rc.Append(result.Summary)

		return nil
	}
}
