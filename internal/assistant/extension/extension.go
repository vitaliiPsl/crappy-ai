package extension

import (
	"context"

	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
)

type Context struct {
	Ctx       context.Context
	SessionID string
	Config    config.Config
	Model     kit.Model
}

type Extension interface {
	Name() string
	Options(ctx Context) (agent.Option, error)
}
