package extension

import (
	"github.com/vitaliiPsl/crappy-adk/agent"
	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/config"
)

type Context struct {
	SessionID string
	Config    config.Config
	Model     kit.Model
}

type Extension interface {
	Name() string
	Options(ctx Context) (agent.Option, error)
}
