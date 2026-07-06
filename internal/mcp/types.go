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
	Close() error

	ListTools(ctx context.Context) ([]kit.Tool, error)
	ListPrompts(ctx context.Context) ([]Prompt, error)
	ListResources(ctx context.Context) ([]Resource, error)
	ListResourceTemplates(ctx context.Context) ([]ResourceTemplate, error)
	CallTool(ctx context.Context, call kit.ToolCall) (kit.ToolResult, error)
	GetPrompt(ctx context.Context, name string, args map[string]string) (PromptResult, error)
	ReadResource(ctx context.Context, uri string) (ResourceResult, error)
}

type Authenticator interface {
	Authenticate(ctx context.Context, cfg Config) error
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

type ClientSnapshot struct {
	Config Config
	State  ClientState
}

type Prompt struct {
	Name        string
	Title       string
	Description string
	Arguments   []PromptArgument
}

type ServerPrompt struct {
	Server string
	Prompt
}

type PromptArgument struct {
	Name        string
	Title       string
	Description string
	Required    bool
}

type PromptResult struct {
	Description string
	Messages    []PromptMessage
}

type PromptMessage struct {
	Role    string
	Content []PromptContent
}

type PromptContent struct {
	Type        string
	Text        string
	MIMEType    string
	URI         string
	Name        string
	Title       string
	Description string
	Data        []byte
	Resource    *ResourceContent
}

type Resource struct {
	Name        string
	Title       string
	Description string
	URI         string
	MIMEType    string
	Size        int64
}

type ResourceTemplate struct {
	Name        string
	Title       string
	Description string
	URITemplate string
	MIMEType    string
}

type ResourceResult struct {
	Contents []ResourceContent
}

type ResourceContent struct {
	URI      string
	MIMEType string
	Text     string
	Blob     []byte
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
