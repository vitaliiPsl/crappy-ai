package mcp

import (
	"context"
	"time"

	"github.com/vitaliiPsl/crappy-ai/internal/mcp/oauth"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type Client interface {
	Config() Config
	State() ClientState

	Connect(ctx context.Context) error
	Authenticate(ctx context.Context) error
	Close() error

	ListTools(ctx context.Context) ([]kit.Tool, error)
	CallTool(ctx context.Context, call kit.ToolCall) (kit.ToolResult, error)
}

type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportHTTP  TransportType = "http"
)

type ClientStatus string

const (
	ClientDisconnected ClientStatus = "disconnected"
	ClientConnecting   ClientStatus = "connecting"
	ClientConnected    ClientStatus = "connected"
	ClientAuthRequired ClientStatus = "auth_required"
	ClientFailed       ClientStatus = "failed"
)

type ClientState struct {
	Status ClientStatus
	Error  string
}

type Config struct {
	Name    string `yaml:"name"`
	Enabled *bool  `yaml:"enabled,omitempty"`

	Transport TransportType `yaml:"type,omitempty"`

	Command string   `yaml:"command,omitempty"`
	Args    []string `yaml:"args,omitempty"`
	Env     []string `yaml:"env,omitempty"`

	URL       string            `yaml:"url,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty"`
	HeaderEnv map[string]string `yaml:"header_env,omitempty"`
	OAuth     *oauth.Config     `yaml:"oauth,omitempty"`

	ConnectTimeout time.Duration `yaml:"connect_timeout,omitempty"`
	RequestTimeout time.Duration `yaml:"request_timeout,omitempty"`
}

func (c Config) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}

	return *c.Enabled
}
