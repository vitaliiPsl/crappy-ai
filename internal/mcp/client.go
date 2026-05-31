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

	ctx    context.Context
	cancel context.CancelFunc

	mu      sync.Mutex
	session *mcpsdk.ClientSession
	state   ClientState
	err     error
}

func NewClient(config Config) Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &sdkClient{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *sdkClient) Config() Config {
	return c.config
}

func (c *sdkClient) Status() ClientStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	status := ClientStatus{
		Config: c.config,
		State:  c.state,
	}

	if status.State == "" {
		status.State = ClientDisconnected
	}

	if c.err != nil {
		status.Error = c.err.Error()
	}

	return status
}

func (c *sdkClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.session != nil {
		c.mu.Unlock()

		return nil
	}

	if c.state == ClientConnecting {
		c.mu.Unlock()

		return nil
	}

	c.state = ClientConnecting
	c.err = nil
	c.mu.Unlock()

	session, err := c.dial(ctx)

	c.mu.Lock()
	defer c.mu.Unlock()

	if err != nil {
		c.state = ClientFailed
		c.err = err

		return err
	}

	if c.ctx.Err() != nil {
		_ = session.Close()
		c.state = ClientDisconnected

		return fmt.Errorf("mcp: client %q closed during connect", c.config.Name)
	}

	c.session = session
	c.state = ClientConnected
	c.err = nil

	return nil
}

func (c *sdkClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer c.cancel()

	if c.session == nil {
		c.state = ClientDisconnected

		return nil
	}

	err := c.session.Close()
	c.session = nil
	c.state = ClientDisconnected
	c.err = err

	return err
}

func (c *sdkClient) ListTools(ctx context.Context) ([]kit.ToolDefinition, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return nil, err
	}

	res, err := session.ListTools(ctx, nil)
	if err != nil {
		c.setFailed(err)

		return nil, err
	}

	return convertTools(res.Tools)
}

func (c *sdkClient) CallTool(ctx context.Context, call kit.ToolCall) (kit.ToolResult, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return kit.ToolResult{}, err
	}

	res, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      call.Name,
		Arguments: call.Arguments,
	})
	if err != nil {
		c.setFailed(err)

		return kit.ToolResult{}, err
	}

	return convertToolResult(call, res), nil
}

func (c *sdkClient) ensureSession(ctx context.Context) (*mcpsdk.ClientSession, error) {
	c.mu.Lock()
	session := c.session
	c.mu.Unlock()

	if session != nil {
		return session, nil
	}

	if err := c.Connect(ctx); err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session == nil {
		return nil, fmt.Errorf("mcp: client %q is not connected", c.config.Name)
	}

	return c.session, nil
}

func (c *sdkClient) dial(ctx context.Context) (*mcpsdk.ClientSession, error) {
	transport, err := c.transport()
	if err != nil {
		return nil, err
	}

	sdk := mcpsdk.NewClient(
		&mcpsdk.Implementation{
			Name:    clientName,
			Version: clientVersion,
		}, nil,
	)

	session, err := sdk.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp: connect %q: %w", c.config.Name, err)
	}

	return session, nil
}

func (c *sdkClient) setFailed(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session == nil {
		return
	}

	_ = c.session.Close()
	c.session = nil
	c.state = ClientFailed
	c.err = err
}
