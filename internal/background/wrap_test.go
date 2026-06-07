package background

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

type testTool struct {
	def     kit.ToolDefinition
	execute func(rc *kit.RunContext, input map[string]any) (string, error)
	calls   []map[string]any
}

func (t *testTool) Definition() kit.ToolDefinition {
	return t.def
}

func (t *testTool) Execute(rc *kit.RunContext, input map[string]any) (string, error) {
	t.calls = append(t.calls, input)

	return t.execute(rc, input)
}

func TestWrapAddsBackgroundArgument(t *testing.T) {
	base := &testTool{
		def: kit.ToolDefinition{
			Name: "bash",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{"type": "string"},
				},
			},
		},
	}

	wrapped, err := Wrap(base, NewManager(context.Background()))
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}

	props := wrapped.Definition().Schema["properties"].(map[string]any)
	if _, ok := props[ArgName]; !ok {
		t.Fatalf("background argument missing")
	}

	originalProps := base.Definition().Schema["properties"].(map[string]any)
	if _, ok := originalProps[ArgName]; ok {
		t.Fatalf("original schema was mutated")
	}
}

func TestWrapForegroundStripsBackgroundArgument(t *testing.T) {
	base := &testTool{
		def: kit.ToolDefinition{Name: "bash"},
		execute: func(_ *kit.RunContext, input map[string]any) (string, error) {
			if _, ok := input[ArgName]; ok {
				t.Fatalf("background argument leaked into wrapped tool")
			}

			return input["command"].(string), nil
		},
	}

	wrapped, err := Wrap(base, NewManager(context.Background()))
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}

	output, err := wrapped.Execute(kit.NewRunContext(context.Background()), map[string]any{
		"command": "echo hi",
		ArgName:   false,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if output != "echo hi" {
		t.Fatalf("output = %q, want echo hi", output)
	}
}

func TestWrapBackgroundStartsJob(t *testing.T) {
	manager := NewManager(context.Background())
	defer manager.Close()

	started := make(chan struct{})
	release := make(chan struct{})
	base := &testTool{
		def: kit.ToolDefinition{Name: "bash"},
		execute: func(rc *kit.RunContext, input map[string]any) (string, error) {
			close(started)

			select {
			case <-release:
				return input["command"].(string) + " done", nil
			case <-rc.Done():
				return "", rc.Err()
			}
		},
	}

	wrapped, err := Wrap(base, manager)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}

	rc := kit.NewRunContext(WithSessionID(context.Background(), "session-1"))

	output, err := wrapped.Execute(rc, map[string]any{
		"command": "go test ./...",
		ArgName:   true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var job Job
	if err := json.Unmarshal([]byte(output), &job); err != nil {
		t.Fatalf("unmarshal job: %v", err)
	}

	if job.ID == "" || job.SessionID != "session-1" || job.Status != StatusRunning {
		t.Fatalf("job = %+v, want running job with session ID", job)
	}

	<-started
	close(release)

	done, err := manager.ForSession("session-1").Wait(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if done.Output != "go test ./... done" {
		t.Fatalf("output = %q, want command output", done.Output)
	}
}

func TestWrapRejectsBackgroundArgumentCollision(t *testing.T) {
	base := &testTool{
		def: kit.ToolDefinition{
			Name: "custom",
			Schema: map[string]any{
				"properties": map[string]any{
					ArgName: map[string]any{"type": "string"},
				},
			},
		},
	}

	_, err := Wrap(base, NewManager(context.Background()))
	if err == nil {
		t.Fatal("Wrap should reject existing background argument")
	}
}

func TestWrapBackgroundArgumentMustBeBoolean(t *testing.T) {
	base := &testTool{def: kit.ToolDefinition{Name: "bash"}}

	wrapped, err := Wrap(base, NewManager(context.Background()))
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}

	_, err = wrapped.Execute(kit.NewRunContext(context.Background()), map[string]any{
		ArgName: "true",
	})
	if err == nil {
		t.Fatal("Execute should reject non-boolean background argument")
	}
}

func TestWrapRequiresManager(t *testing.T) {
	_, err := Wrap(&testTool{}, nil)
	if err == nil {
		t.Fatal("Wrap should require a manager")
	}
}
