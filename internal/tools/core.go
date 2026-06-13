package tools

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/tools/bash"
	filesystem "github.com/vitaliiPsl/crappy-ai/internal/tools/fs"
	"github.com/vitaliiPsl/crappy-ai/internal/tools/web"
)

func Core(jobs background.Jobs) []kit.Tool {
	return []kit.Tool{
		wrapBackground(bash.NewBash(), jobs),
		web.NewFetch(),
		filesystem.NewReadFile(),
		filesystem.NewWriteFile(),
		filesystem.NewEditFile(),
		filesystem.NewListDirectory(),
	}
}

func wrapBackground(t kit.Tool, jobs background.Jobs) kit.Tool {
	wrapped, err := background.Wrap(t, jobs)
	if err != nil {
		panic(fmt.Sprintf("wrap tool %q for background: %v", t.Definition().Name, err))
	}

	return wrapped
}
