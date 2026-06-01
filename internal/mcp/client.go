package mcp

import (
	"context"
	"fmt"
	"sync"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

const (
	clientName    = "crappy"
	clientVersion = "0.1.0"
)

type sdkClient struct {
	config Config

	mu        sync.Mutex
	connectMu sync.Mutex
	session   *mcpsdk.ClientSession

	state ClientStatus
	tools []kit.Tool
	err   error
}

func NewClient(config Config) Client {
	return &sdkClient{
		config: config,
		state:  ClientDisconnected,
	}
}

func (c *sdkClient) State() ClientState {
	c.mu.Lock()
	defer c.mu.Unlock()

	status := ClientState{
		Config: c.config,
		Status: c.state,
	}

	if c.err != nil {
		status.Error = c.err.Error()
	}

	return status
}

func (c *sdkClient) Connect(ctx context.Context) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	if c.connected() {
		return nil
	}

	c.markConnecting()

	session, err := c.dial(ctx)
	if err != nil {
		c.markFailed(err)

		return err
	}

	c.markConnected(session)

	return nil
}

func (c *sdkClient) Close() error {
	session := c.reset()

	if session == nil {
		return nil
	}

	return session.Close()
}

func (c *sdkClient) ListTools(ctx context.Context) ([]kit.Tool, error) {
	c.mu.Lock()
	session := c.session
	connected := session != nil && c.state == ClientConnected
	tools := c.tools
	c.mu.Unlock()

	if !connected {
		return nil, fmt.Errorf("mcp: client %q is not connected", c.config.Name)
	}

	if tools != nil {
		return tools, nil
	}

	tools, err := c.fetchTools(ctx, session)
	if err != nil {
		c.failSession(err)

		return nil, err
	}

	c.mu.Lock()
	if c.session == session && c.state == ClientConnected {
		c.tools = tools
	}
	c.mu.Unlock()

	return tools, nil
}

func (c *sdkClient) CallTool(ctx context.Context, call kit.ToolCall) (kit.ToolResult, error) {
	c.mu.Lock()
	session := c.session
	connected := session != nil && c.state == ClientConnected
	c.mu.Unlock()

	if !connected {
		return kit.ToolResult{}, fmt.Errorf("mcp: client %q is not connected", c.config.Name)
	}

	res, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      call.Name,
		Arguments: call.Arguments,
	})
	if err != nil {
		c.failSession(err)

		return kit.ToolResult{}, err
	}

	return convertToolResult(call, res), nil
}

func (c *sdkClient) dial(ctx context.Context) (*mcpsdk.ClientSession, error) {
	transport, err := buildTransport(c.config)
	if err != nil {
		return nil, err
	}

	sdk := mcpsdk.NewClient(
		&mcpsdk.Implementation{
			Name:    clientName,
			Version: clientVersion,
		},
		&mcpsdk.ClientOptions{
			ToolListChangedHandler: func(context.Context, *mcpsdk.ToolListChangedRequest) {
				c.invalidateTools()
			},
		},
	)

	session, err := sdk.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp: connect %q: %w", c.config.Name, err)
	}

	return session, nil
}

func (c *sdkClient) fetchTools(ctx context.Context, session *mcpsdk.ClientSession) ([]kit.Tool, error) {
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	defs, err := convertTools(res.Tools)
	if err != nil {
		return nil, err
	}

	tools := make([]kit.Tool, 0, len(defs))
	for _, def := range defs {
		tools = append(tools, newTool(c.config.Name, c, def))
	}

	return tools, nil
}

func (c *sdkClient) invalidateTools() {
	c.mu.Lock()
	if c.state == ClientConnected {
		c.tools = nil
	}
	c.mu.Unlock()
}

func (c *sdkClient) connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.session != nil && c.state == ClientConnected
}

func (c *sdkClient) markConnecting() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.state = ClientConnecting
	c.err = nil
}

func (c *sdkClient) markFailed(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.state = ClientFailed
	c.err = err
}

func (c *sdkClient) markConnected(session *mcpsdk.ClientSession) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.session = session
	c.tools = nil
	c.state = ClientConnected
	c.err = nil
}

func (c *sdkClient) failSession(err error) {
	c.mu.Lock()

	session := c.session
	if session != nil {
		c.session = nil
		c.tools = nil
		c.state = ClientFailed
		c.err = err
	}
	c.mu.Unlock()

	if session != nil {
		_ = session.Close()
	}
}

func (c *sdkClient) reset() *mcpsdk.ClientSession {
	c.mu.Lock()
	defer c.mu.Unlock()

	session := c.session
	c.session = nil
	c.tools = nil
	c.state = ClientDisconnected
	c.err = nil

	return session
}
