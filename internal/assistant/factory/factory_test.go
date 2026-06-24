package factory

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/kittest"
	xmemory "github.com/vitaliiPsl/crappy-adk/x/memory"
	"github.com/vitaliiPsl/crappy-adk/x/tool"

	"github.com/vitaliiPsl/crappy-ai/internal/background"
	"github.com/vitaliiPsl/crappy-ai/internal/config"
	"github.com/vitaliiPsl/crappy-ai/internal/permission"
	permissionmodel "github.com/vitaliiPsl/crappy-ai/internal/permission/model"
	"github.com/vitaliiPsl/crappy-ai/internal/permission/strategy"
)

func testTool(name string) kit.Tool {
	return tool.MustNew(name, "test tool", func(_ *kit.RunContext, _ struct{}) (string, error) {
		return "", nil
	})
}

func names(tools []kit.Tool) []string {
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = t.Definition().Name
	}

	return out
}

func TestAllowedToolsEmptyAllowlistKeepsAll(t *testing.T) {
	tools := []kit.Tool{testTool("read_file"), testTool("bash")}

	got := allowedTools(tools, nil)
	if len(got) != 2 {
		t.Fatalf("tools = %v, want all kept when allowlist empty", names(got))
	}
}

func TestAllowedToolsFiltersToAllowlist(t *testing.T) {
	tools := []kit.Tool{testTool("read_file"), testTool("bash"), testTool("list")}

	got := allowedTools(tools, []string{"read_file", "list"})

	want := map[string]bool{"read_file": true, "list": true}
	if len(got) != len(want) {
		t.Fatalf("tools = %v, want %v", names(got), want)
	}

	for _, t2 := range got {
		if !want[t2.Definition().Name] {
			t.Fatalf("unexpected tool %q in filtered set", t2.Definition().Name)
		}
	}
}

func TestBuildUsesRequestPermissionsForToolCalls(t *testing.T) {
	ctx := context.Background()

	bg := background.NewManager(ctx)
	defer bg.Close()

	configStore := config.NewStore(config.Config{
		Mode: config.ModeDefault,
		Agent: config.Agent{
			Permissions: permissionmodel.Permissions{
				Default: permissionmodel.Allow,
			},
		},
	}, filepath.Join(t.TempDir(), "config.yaml"))

	f := New(permission.NewService(configStore), bg)

	call := kit.NewToolCall("call-1", strategy.ToolReadFile, map[string]any{"path": "/tmp/project/main.go"})
	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{
			Message:      kit.NewModelMessage(kit.NewToolCallContent(call)),
			FinishReason: kit.FinishReasonToolCall,
		},
	}, kittest.ModelResult{
		Response: kit.ModelResponse{
			Message:      kit.NewModelMessage(kit.NewTextContent("done")),
			FinishReason: kit.FinishReasonStop,
		},
	})

	ag, err := f.Build(ctx, BuildRequest{
		SessionID: "child-session",
		Config: config.Config{
			Mode: config.ModeDefault,
			Agent: config.Agent{
				Permissions: permissionmodel.Permissions{
					Default: permissionmodel.Deny,
				},
			},
		},
		Model:  model,
		Memory: xmemory.NewHistory(),
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if _, err := ag.Run(ctx, kit.NewUserMessage(kit.NewTextContent("read the file"))); err != nil {
		t.Fatalf("Run: %v", err)
	}

	secondRequest := model.CallAt(1)
	lastMessage := secondRequest.Messages[len(secondRequest.Messages)-1]

	results := lastMessage.ToolResults()
	if len(results) != 1 {
		t.Fatalf("tool results = %d, want 1", len(results))
	}

	if !strings.Contains(results[0].Error, permission.ErrDenied.Error()) {
		t.Fatalf("tool error = %q, want permission denial from request permissions", results[0].Error)
	}
}

func TestBuildAppliesGenerationConfig(t *testing.T) {
	ctx := context.Background()

	bg := background.NewManager(ctx)
	defer bg.Close()

	configStore := config.NewStore(config.Config{}, filepath.Join(t.TempDir(), "config.yaml"))
	f := New(permission.NewService(configStore), bg)

	model := kittest.NewModel(t, kittest.ModelResult{
		Response: kit.ModelResponse{
			Message:      kit.NewModelMessage(kit.NewTextContent("done")),
			FinishReason: kit.FinishReasonStop,
		},
	})

	temperature := float32(0)
	maxOutputTokens := int32(1234)

	ag, err := f.Build(ctx, BuildRequest{
		SessionID: "session-1",
		Config: config.Config{
			Agent: config.Agent{
				Thinking:        "high",
				Temperature:     &temperature,
				MaxOutputTokens: &maxOutputTokens,
			},
		},
		Model:  model,
		Memory: xmemory.NewHistory(),
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if _, err := ag.Run(ctx, kit.NewUserMessage(kit.NewTextContent("hello"))); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := model.CallAt(0).Config
	if got.Thinking == nil || *got.Thinking != kit.ThinkingLevelHigh {
		t.Fatalf("thinking = %v, want high", got.Thinking)
	}

	if got.Temperature == nil || *got.Temperature != temperature {
		t.Fatalf("temperature = %v, want %v", got.Temperature, temperature)
	}

	if got.MaxOutputTokens == nil || *got.MaxOutputTokens != maxOutputTokens {
		t.Fatalf("max output tokens = %v, want %v", got.MaxOutputTokens, maxOutputTokens)
	}
}
