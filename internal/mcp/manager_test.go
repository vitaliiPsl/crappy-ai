package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type fakeClient struct {
	config Config
	state  ClientState
	tools  []kit.ToolDefinition
	err    error

	connects int
	called   kit.ToolCall
	result   kit.ToolResult
}

func (c *fakeClient) Config() Config {
	return c.config
}

func (c *fakeClient) Status() ClientStatus {
	state := c.state
	if state == "" {
		state = ClientConnected
	}

	status := ClientStatus{Config: c.config, State: state}
	if c.err != nil {
		status.Error = c.err.Error()
	}

	return status
}

func (c *fakeClient) Connect(context.Context) error {
	c.connects++

	return c.err
}

func (c *fakeClient) Close() error {
	return c.err
}

func (c *fakeClient) ListTools(context.Context) ([]kit.ToolDefinition, error) {
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

func TestManagerToolsWrapsClientTools(t *testing.T) {
	client := &fakeClient{
		config: Config{Name: "github"},
		tools: []kit.ToolDefinition{{
			Name:        "search",
			Description: "Search issues",
			Schema:      map[string]any{"type": "object"},
		}},
	}

	tools := NewWithClients(client).Tools(context.Background())

	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}

	def := tools[0].Definition()
	if def.Name != "mcp__github__search" {
		t.Fatalf("tool name = %q, want mcp__github__search", def.Name)
	}

	if def.Description != "Search issues" {
		t.Fatalf("description = %q, want Search issues", def.Description)
	}
}

func TestManagerToolsSkipsClientsThatFailToList(t *testing.T) {
	client := &fakeClient{
		config: Config{Name: "github"},
		tools:  []kit.ToolDefinition{{Name: "search"}},
		err:    errors.New("boom"),
	}

	tools := NewWithClients(client).Tools(context.Background())
	if len(tools) != 0 {
		t.Fatalf("len(tools) = %d, want 0", len(tools))
	}
}

func TestManagerStatusesReturnsClientStatuses(t *testing.T) {
	client := &fakeClient{
		config: Config{Name: "github"},
		state:  ClientFailed,
		err:    errors.New("boom"),
	}

	statuses := NewWithClients(client).Statuses()
	if len(statuses) != 1 {
		t.Fatalf("len(statuses) = %d, want 1", len(statuses))
	}

	if statuses[0].Config.Name != "github" || statuses[0].State != ClientFailed || statuses[0].Error != "boom" {
		t.Fatalf("status = %+v, want github failed boom", statuses[0])
	}
}

func TestToolExecuteCallsServerTool(t *testing.T) {
	client := &fakeClient{
		config: Config{Name: "github"},
		result: kit.NewToolResult(kit.NewToolCall("", "search", nil), "done", nil),
	}

	output, err := newTool(client, kit.ToolDefinition{Name: "search"}).Execute(kit.NewRunContext(context.Background()), map[string]any{"q": "x"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if output != "done" {
		t.Fatalf("output = %q, want done", output)
	}

	if client.called.Name != "search" {
		t.Fatalf("called name = %q, want search", client.called.Name)
	}

	if client.called.Arguments["q"] != "x" {
		t.Fatalf("called arguments = %#v, want q=x", client.called.Arguments)
	}
}
