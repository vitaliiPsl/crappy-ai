package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/ask"
	mcpcore "github.com/vitaliiPsl/crappy-ai/internal/mcp"
	"github.com/vitaliiPsl/crappy-ai/internal/session"
	"github.com/vitaliiPsl/crappy-ai/internal/skills/skillstest"
)

func bareSession() *Session {
	return newSession("s1", nil, nil, nil, nil, nil, nil, nil)
}

func recv(t *testing.T, ch <-chan session.Event) session.Event {
	t.Helper()

	select {
	case ev := <-ch:
		return ev
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")

		return session.Event{}
	}
}

func TestBroadcastReachesSubscribers(t *testing.T) {
	s := bareSession()

	sub := s.Subscribe()

	want := session.NewMessageEvent("s1", kit.NewModelMessage(kit.NewTextContent("hi")))
	s.events.Publish(want)

	if got := recv(t, sub.Events()); got.Type != session.EventMessage {
		t.Fatalf("event type = %q, want %q", got.Type, session.EventMessage)
	}
}

func TestAskRoundTrip(t *testing.T) {
	s := bareSession()
	sub := s.Subscribe()

	req := ask.Request{ID: "r1", Title: "Allow bash?", Options: []ask.Option{{ID: "allow", Label: "Allow"}}}

	answered := make(chan ask.Response, 1)
	go func() {
		resp, _ := s.Ask(context.Background(), req)
		answered <- resp
	}()

	got := recv(t, sub.Events())
	if got.Type != session.EventAsk || got.Ask == nil || got.Ask.ID != "r1" {
		t.Fatalf("event = %+v, want an ask event for r1", got)
	}

	if err := s.Respond(ask.Response{RequestID: "r1", OptionID: "allow"}); err != nil {
		t.Fatalf("Respond: %v", err)
	}

	select {
	case resp := <-answered:
		if resp.OptionID != "allow" {
			t.Fatalf("Ask returned %q, want allow", resp.OptionID)
		}
	case <-time.After(time.Second):
		t.Fatal("Ask did not return after Respond")
	}
}

func TestBuildAgentInputWithSkill(t *testing.T) {
	root := t.TempDir()
	skillstest.WriteSkill(t, filepath.Join(root, "review", "SKILL.md"), "review", "Review code", "Read the diff carefully.")

	s := bareSession()
	s.skillRegistry = skillstest.NewRegistry(root)

	req := Request{
		Text: "ignored",
		Skill: &SkillInvocation{
			Name: "review",
			Args: []string{"file.go"},
		},
	}

	msg, ev, err := s.buildAgentInput(context.Background(), req)
	if err != nil {
		t.Fatalf("buildAgentInput: %v", err)
	}

	content := msg.TextContent()
	if content == nil {
		t.Fatal("message has no text content")
	}

	for _, want := range []string{
		"Loaded skill: review",
		"Arguments:\nfile.go",
		"Read the diff carefully.",
	} {
		if !strings.Contains(content.Text, want) {
			t.Fatalf("message text missing %q:\n%s", want, content.Text)
		}
	}

	if ev.Skill == nil || ev.Skill.Name != "review" || strings.Join(ev.Skill.Args, " ") != "file.go" {
		t.Fatalf("skill event = %+v, want review invocation", ev.Skill)
	}
}

func TestBuildAgentInputWithMCPPrompt(t *testing.T) {
	manager := newPromptManager(t)
	defer manager.Close()

	s := bareSession()
	s.mcpManager = manager

	req := Request{
		Text: "ignored",
		MCPPrompt: &MCPPromptInvocation{
			Server: "prompts",
			Name:   "review",
			Args:   map[string]string{"path": "main.go"},
		},
	}

	msg, ev, err := s.buildAgentInput(context.Background(), req)
	if err != nil {
		t.Fatalf("buildAgentInput: %v", err)
	}

	content := msg.TextContent()
	if content == nil {
		t.Fatal("message has no text content")
	}

	if content.Text != "Review main.go" {
		t.Fatalf("message text = %q, want resolved mcp prompt text", content.Text)
	}

	if ev.MCPPrompt == nil ||
		ev.MCPPrompt.Server != "prompts" ||
		ev.MCPPrompt.Name != "review" ||
		ev.MCPPrompt.Args["path"] != "main.go" {
		t.Fatalf("mcp prompt event = %+v, want prompts review path=main.go", ev.MCPPrompt)
	}
}

func TestSecondTurnRejectedWhileActive(t *testing.T) {
	s := bareSession()

	block := make(chan struct{})
	defer close(block)

	if err := s.start(context.Background(), func(context.Context) error {
		<-block

		return nil
	}); err != nil {
		t.Fatalf("first start: %v", err)
	}

	if err := s.start(context.Background(), func(context.Context) error { return nil }); err == nil {
		t.Fatal("second start during an active turn should fail")
	}
}

func newPromptManager(t *testing.T) *mcpcore.Manager {
	t.Helper()

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "prompts", Version: "0.1.0"}, nil)
	server.AddPrompt(&mcpsdk.Prompt{
		Name: "review",
		Arguments: []*mcpsdk.PromptArgument{{
			Name:     "path",
			Required: true,
		}},
	}, func(_ context.Context, req *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
		return &mcpsdk.GetPromptResult{
			Messages: []*mcpsdk.PromptMessage{{
				Role:    "user",
				Content: &mcpsdk.TextContent{Text: "Review " + req.Params.Arguments["path"]},
			}},
		}, nil
	})

	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return server
	}, nil)
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	manager := mcpcore.NewManager(
		context.Background(),
		[]mcpcore.Config{{Name: "prompts", Transport: mcpcore.TransportHTTP, URL: httpServer.URL}},
		nil,
		nil,
	)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		for _, client := range manager.Clients() {
			if client.State().Status == mcpcore.ClientConnected {
				return manager
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("mcp prompt manager did not connect")

	return nil
}

func TestTurnGuardClearsAfterCompletion(t *testing.T) {
	s := bareSession()

	done := make(chan struct{})
	if err := s.start(context.Background(), func(context.Context) error {
		close(done)

		return nil
	}); err != nil {
		t.Fatalf("start: %v", err)
	}

	<-done
	// Wait for the guard to clear after the turn goroutine returns.
	deadline := time.After(time.Second)
	for {
		s.mu.Lock()
		cleared := s.cancel == nil
		s.mu.Unlock()

		if cleared {
			break
		}

		select {
		case <-deadline:
			t.Fatal("turn guard never cleared")
		case <-time.After(time.Millisecond):
		}
	}

	if err := s.start(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatalf("start after completion should succeed: %v", err)
	}
}
