package summarization

import (
	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	xsummarization "github.com/vitaliiPsl/crappy-adk/x/summarization"
)

const thresholdRatio = 0.75

func New(model kit.Model) agent.Option {
	return xsummarization.WithSummarization(
		whenLastInputExceeds(model.Config(), thresholdRatio),
		strategy(NewSummarizer(model)),
	)
}

func strategy(summarizer *Summarizer) xsummarization.Strategy {
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
