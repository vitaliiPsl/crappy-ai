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
	called   kit.ToolCall
	result   kit.ToolResult
}

func (c *fakeClient) State() ClientState {
	status := c.status
	if status == "" {
		status = ClientConnected
	}

	state := ClientState{Config: c.config, Status: status}
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
	return c.err
}

func (c *fakeClient) ListTools(context.Context) ([]kit.Tool, error) {
	return c.tools, c.err
}

func (c *fakeClient) CallTool(_ context.Context, call kit.ToolCall) (kit.ToolResult, error) {
	c.called = call

	return c.result, c.err
}

func TestManagerConnectConnectsClients(t *testing.T) {
	first := &fakeClient{}
	second := &fakeClient{}

	if err := NewWithClients(first, second).Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if first.connects != 1 || second.connects != 1 {
		t.Fatalf("connects = %d/%d, want 1/1", first.connects, second.connects)
	}
}

func TestManagerConnectReturnsErrors(t *testing.T) {
	want := errors.New("boom")

	err := NewWithClients(&fakeClient{err: want}).Connect(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Connect error = %v, want %v", err, want)
	}
}

func TestManagerStatesReturnsClientStates(t *testing.T) {
	client := &fakeClient{
		config: Config{Name: "github"},
		status: ClientFailed,
		err:    errors.New("boom"),
	}

	states := NewWithClients(client).States()
	if len(states) != 1 {
		t.Fatalf("len(states) = %d, want 1", len(states))
	}

	if states[0].Config.Name != "github" || states[0].Status != ClientFailed || states[0].Error != "boom" {
		t.Fatalf("state = %+v, want github failed boom", states[0])
	}
}
