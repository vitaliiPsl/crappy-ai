package runtime

import (
	"fmt"
	goruntime "runtime"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"
	xtool "github.com/vitaliiPsl/crappy-adk/x/tool"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/tools/bash"
	filesystem "github.com/vitaliiPsl/crappy-ai/internal/tools/fs"
)

func buildAgent(cfg config.Config, model kit.Model, mem kit.Memory, extra ...agent.Option) (*agent.Agent, error) {
	opts := []agent.Option{
		agent.WithInstructions(cfg.Prompt, envInstructions(cfg.Cwd)),
	}

	if cfg.Thinking != "" {
		opts = append(opts, agent.WithThinking(kit.ThinkingLevel(cfg.Thinking)))
	}

	if cfg.Temperature != nil {
		opts = append(opts, agent.WithTemperature(*cfg.Temperature))
	}

	if cfg.MaxOutputTokens != nil {
		opts = append(opts, agent.WithMaxOutputTokens(*cfg.MaxOutputTokens))
	}

	if cfg.CompactThreshold > 0 {
		opts = append(opts, agent.WithOnTurnStart(compactionHook(model, cfg.CompactThreshold)))
	}

	opts = append(opts, extra...)

	return agent.New(model, mem, coreTools(), opts...)
}

func coreTools() *xtool.Set {
	return xtool.NewSet(
		bash.NewBash(),
		filesystem.NewReadFile(),
		filesystem.NewWriteFile(),
		filesystem.NewEditFile(),
		filesystem.NewListDirectory(),
	)
}

func envInstructions(cwd string) string {
	return fmt.Sprintf(
		"# Environment\n- Working directory: %s\n- OS: %s/%s",
		cwd, goruntime.GOOS, goruntime.GOARCH,
	)
}
