package summarization

import (
	"math"

	"github.com/vitaliiPsl/crappy-adk/kit"
	xsummarization "github.com/vitaliiPsl/crappy-adk/x/summarization"
)

func whenLastInputExceeds(config kit.ModelConfig, ratio float64) xsummarization.Trigger {
	limit := inputLimit(config)
	if limit <= 0 {
		return func(*kit.RunContext) bool { return false }
	}

	if ratio <= 0 || ratio > 1 {
		ratio = thresholdRatio
	}

	threshold := int64(max(int(math.Floor(float64(limit)*ratio)), 1))

	return func(rc *kit.RunContext) bool {
		return rc.LastUsage.InputTokens >= threshold
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
