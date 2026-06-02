package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type fakeClient struct {
	config Config
	status ClientStatus
	tools  []kit.Tool
	err    error

	connects int
	closes   int
	called   kit.ToolCall
	result   kit.ToolResult
}

func (c *fakeClient) Config() Config {
	return c.config
}

func (c *fakeClient) State() ClientState {
	status := c.status
	if status == "" {
		status = ClientConnected
	}

	state := ClientState{Status: status}
	if c.err != nil {
		state.Error = c.err.Error()
	}

	return state
}

func (c *fakeClient) Connect(context.Context) error {
	c.connects++

	return c.err
}

func (c *fakeClient) Close() error {
	c.closes++

	return c.err
}

func (c *fakeClient) ListTools(context.Context) ([]kit.Tool, error) {
	return c.tools, c.err
}

func (c *fakeClient) CallTool(_ context.Context, call kit.ToolCall) (kit.ToolResult, error) {
	c.called = call

	return c.result, c.err
}

func newManager(clients ...Client) *Manager {
	byName := make(map[string]Client, len(clients))
	for _, client := range clients {
		byName[client.Config().Name] = client
	}

	return &Manager{clients: byName}
}

func TestManagerConnectConnectsClients(t *testing.T) {
	first := &fakeClient{config: Config{Name: "first"}}
	second := &fakeClient{config: Config{Name: "second"}}

	if err := newManager(first, second).Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if first.connects != 1 || second.connects != 1 {
		t.Fatalf("connects = %d/%d, want 1/1", first.connects, second.connects)
	}
}

func TestManagerConnectReturnsErrors(t *testing.T) {
	want := errors.New("boom")

	err := newManager(&fakeClient{err: want}).Connect(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Connect error = %v, want %v", err, want)
	}
}

func TestManagerStatesReturnsClientStates(t *testing.T) {
	client := &fakeClient{
		status: ClientFailed,
		err:    errors.New("boom"),
	}

	states := newManager(client).States()
	if len(states) != 1 {
		t.Fatalf("len(states) = %d, want 1", len(states))
	}

	if states[0].Status != ClientFailed || states[0].Error != "boom" {
		t.Fatalf("state = %+v, want failed boom", states[0])
	}
}

func TestManagerReconnectClosesThenConnects(t *testing.T) {
	client := &fakeClient{config: Config{Name: "github"}}

	if err := newManager(client).Reconnect(context.Background(), "github"); err != nil {
		t.Fatalf("Reconnect: %v", err)
	}

	if client.closes != 1 || client.connects != 1 {
		t.Fatalf("closes/connects = %d/%d, want 1/1", client.closes, client.connects)
	}
}

func TestManagerReconnectUnknownClient(t *testing.T) {
	err := newManager(&fakeClient{config: Config{Name: "github"}}).Reconnect(context.Background(), "missing")
	if err == nil || err.Error() != `mcp: unknown client "missing"` {
		t.Fatalf("Reconnect error = %v, want unknown client", err)
	}
}
