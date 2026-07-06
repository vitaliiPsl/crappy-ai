package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestNewClientStartsDisconnected(t *testing.T) {
	state := NewClient(Config{Name: "docs", Transport: TransportHTTP, URL: "http://localhost:3000/mcp"}, testTransport).State()

	if state.Status != ClientDisconnected {
		t.Fatalf("Status = %q, want disconnected", state.Status)
	}

	if state.Error != "" {
		t.Fatalf("Error = %q, want empty", state.Error)
	}
}

func TestConnectRejectsDisabledClient(t *testing.T) {
	disabled := false

	err := NewClient(Config{Name: "docs", Enabled: &disabled}, testTransport).Connect(context.Background())
	if err == nil || err.Error() != "mcp: client is disabled" {
		t.Fatalf("Connect() error = %v, want disabled", err)
	}
}

func TestConnectUsesTransportFactory(t *testing.T) {
	want := errors.New("transport boom")
	factory := &fakeTransportFactory{err: want}
	client := NewClient(Config{Name: "test"}, factory.New)

	err := client.Connect(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Connect() error = %v, want %v", err, want)
	}

	if factory.calls != 1 {
		t.Fatalf("transport factory calls = %d, want 1", factory.calls)
	}

	if factory.config.Name != "test" {
		t.Fatalf("transport factory config = %+v, want test config", factory.config)
	}

	state := client.State()
	if state.Status != ClientFailed || state.Error != want.Error() {
		t.Fatalf("state = %+v, want failed with transport error", state)
	}
}

func TestConnectMapsSDKOAuthErrorStatus(t *testing.T) {
	factory := &fakeTransportFactory{err: fmt.Errorf("wrapped: %w", mcpauth.ErrOAuth)}
	client := NewClient(Config{Name: "test"}, factory.New)

	_ = client.Connect(context.Background())

	state := client.State()
	if state.Status != ClientAuthRequired {
		t.Fatalf("Status = %q, want auth_required", state.Status)
	}

	if !strings.Contains(state.Error, mcpauth.ErrOAuth.Error()) {
		t.Fatalf("Error = %q, want sdk oauth error", state.Error)
	}
}

func TestStatusLifecycle(t *testing.T) {
	client := newClient(t, newServer(t, "greet"))

	if got := client.State().Status; got != ClientDisconnected {
		t.Fatalf("initial Status = %q, want disconnected", got)
	}

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if got := client.State().Status; got != ClientConnected {
		t.Fatalf("connected Status = %q, want connected", got)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if got := client.State().Status; got != ClientDisconnected {
		t.Fatalf("closed Status = %q, want disconnected", got)
	}
}

func TestConnectIsIdempotent(t *testing.T) {
	client := connect(t, newServer(t, "greet"))

	session := client.session

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("second Connect() error = %v", err)
	}

	if client.session != session {
		t.Fatal("second Connect() replaced the live session")
	}
}

func TestConnectIsConcurrencySafe(t *testing.T) {
	client := newClient(t, newServer(t, "greet"))

	var wg sync.WaitGroup

	errs := make(chan error, 8)
	for range 8 {
		wg.Go(func() {
			errs <- client.Connect(context.Background())
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("Connect() error = %v", err)
		}
	}

	if got := client.State().Status; got != ClientConnected {
		t.Fatalf("Status = %q, want connected", got)
	}
}

func TestOperationsRequireConnection(t *testing.T) {
	client := NewClient(Config{Name: "test"}, testTransport)
	wantErr := "mcp: client is not connected"

	if _, err := client.ListTools(context.Background()); err == nil || err.Error() != wantErr {
		t.Fatalf("ListTools() error = %v, want %q", err, wantErr)
	}

	if _, err := client.ListPrompts(context.Background()); err == nil || err.Error() != wantErr {
		t.Fatalf("ListPrompts() error = %v, want %q", err, wantErr)
	}

	if _, err := client.ListResources(context.Background()); err == nil || err.Error() != wantErr {
		t.Fatalf("ListResources() error = %v, want %q", err, wantErr)
	}

	if _, err := client.ListResourceTemplates(context.Background()); err == nil || err.Error() != wantErr {
		t.Fatalf("ListResourceTemplates() error = %v, want %q", err, wantErr)
	}

	if _, err := client.CallTool(context.Background(), kit.NewToolCall("1", "greet", nil)); err == nil || err.Error() != wantErr {
		t.Fatalf("CallTool() error = %v, want %q", err, wantErr)
	}

	if _, err := client.GetPrompt(context.Background(), "review", nil); err == nil || err.Error() != wantErr {
		t.Fatalf("GetPrompt() error = %v, want %q", err, wantErr)
	}

	if _, err := client.ReadResource(context.Background(), "file:///README.md"); err == nil || err.Error() != wantErr {
		t.Fatalf("ReadResource() error = %v, want %q", err, wantErr)
	}
}

func TestToolLifecycle(t *testing.T) {
	client := connect(t, newServer(t, "greet"))

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}

	if got := tools[0].Definition().Name; got != "mcp__test__greet" {
		t.Fatalf("tool name = %q, want mcp__test__greet", got)
	}

	result, err := client.CallTool(context.Background(), kit.NewToolCall("call_1", "greet", map[string]any{"name": "Ada"}))
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	if !strings.Contains(result.Output, "Hi Ada") {
		t.Fatalf("output = %q, want greeting", result.Output)
	}
}

func TestMCPMetadataLifecycle(t *testing.T) {
	server := newServer(t, "greet")
	server.AddPrompt(&mcpsdk.Prompt{
		Name:        "review",
		Title:       "Review",
		Description: "Review code",
		Arguments: []*mcpsdk.PromptArgument{{
			Name:        "path",
			Description: "Path to review",
			Required:    true,
		}},
	}, func(_ context.Context, req *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
		return &mcpsdk.GetPromptResult{
			Description: "Review code",
			Messages: []*mcpsdk.PromptMessage{{
				Role:    "user",
				Content: &mcpsdk.TextContent{Text: "Review " + req.Params.Arguments["path"]},
			}},
		}, nil
	})
	server.AddResource(&mcpsdk.Resource{
		Name:        "readme",
		Title:       "README",
		Description: "Project README",
		URI:         "file:///README.md",
		MIMEType:    "text/markdown",
		Size:        42,
	}, func(context.Context, *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		return &mcpsdk.ReadResourceResult{
			Contents: []*mcpsdk.ResourceContents{{
				URI:      "file:///README.md",
				MIMEType: "text/markdown",
				Text:     "# Test",
			}},
		}, nil
	})
	server.AddResourceTemplate(&mcpsdk.ResourceTemplate{
		Name:        "file",
		Description: "Project file",
		URITemplate: "file:///{path}",
		MIMEType:    "text/plain",
	}, func(context.Context, *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		return &mcpsdk.ReadResourceResult{}, nil
	})

	client := connect(t, server)

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if len(tools) != 1 || tools[0].Definition().Name != "mcp__test__greet" {
		t.Fatalf("Tools = %+v, want namespaced greet", tools)
	}

	prompts, err := client.ListPrompts(context.Background())
	if err != nil {
		t.Fatalf("ListPrompts() error = %v", err)
	}

	if len(prompts) != 1 || prompts[0].Name != "review" {
		t.Fatalf("Prompts = %+v, want review", prompts)
	}

	if len(prompts[0].Arguments) != 1 || !prompts[0].Arguments[0].Required {
		t.Fatalf("Prompt arguments = %+v, want required path", prompts[0].Arguments)
	}

	promptResult, err := client.GetPrompt(context.Background(), "review", map[string]string{"path": "main.go"})
	if err != nil {
		t.Fatalf("GetPrompt() error = %v", err)
	}

	if len(promptResult.Messages) != 1 || len(promptResult.Messages[0].Content) != 1 || promptResult.Messages[0].Content[0].Text != "Review main.go" {
		t.Fatalf("GetPrompt() = %+v, want review message", promptResult)
	}

	resources, err := client.ListResources(context.Background())
	if err != nil {
		t.Fatalf("ListResources() error = %v", err)
	}

	if len(resources) != 1 || resources[0].URI != "file:///README.md" {
		t.Fatalf("Resources = %+v, want README resource", resources)
	}

	resourceTemplates, err := client.ListResourceTemplates(context.Background())
	if err != nil {
		t.Fatalf("ListResourceTemplates() error = %v", err)
	}

	if len(resourceTemplates) != 1 || resourceTemplates[0].URITemplate != "file:///{path}" {
		t.Fatalf("ResourceTemplates = %+v, want file template", resourceTemplates)
	}

	result, err := client.ReadResource(context.Background(), "file:///README.md")
	if err != nil {
		t.Fatalf("ReadResource() error = %v", err)
	}

	if len(result.Contents) != 1 || result.Contents[0].Text != "# Test" {
		t.Fatalf("ReadResource() = %+v, want README text", result)
	}
}

func TestListToolsCachesBetweenCalls(t *testing.T) {
	client := connect(t, newServer(t, "greet"))

	first, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	second, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if !sameTools(first, second) {
		t.Fatal("ListTools() refetched instead of returning the cache")
	}
}

func TestToolListChangedRefreshesCache(t *testing.T) {
	server := newServer(t, "greet")
	client := connect(t, server)

	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "farewell"}, greet)

	eventually(t, time.Second, func() bool {
		tools, err := client.ListTools(context.Background())
		if err != nil || len(tools) != 2 {
			return false
		}

		names := map[string]bool{}
		for _, tool := range tools {
			names[tool.Definition().Name] = true
		}

		return names["mcp__test__greet"] && names["mcp__test__farewell"]
	})
}

func TestRefetchFailureKeepsConnectedAndKeepsCache(t *testing.T) {
	var failCalls atomic.Bool

	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return newServer(t, "greet")
	}, nil)

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failCalls.Load() && r.Method == http.MethodPost {
			http.Error(w, "boom", http.StatusInternalServerError)

			return
		}

		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(httpServer.Close)

	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: httpServer.URL}, testTransport).(*sdkClient)
	t.Cleanup(func() { _ = client.Close() })

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	before, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	failCalls.Store(true)

	client.refetchLists(context.Background())

	if got := client.State().Status; got != ClientConnected {
		t.Fatalf("Status = %q, want connected", got)
	}

	if !sameTools(before, client.tools) {
		t.Fatal("refetch failure replaced cached tools")
	}
}

func TestRefetchCancellationKeepsConnected(t *testing.T) {
	client := connect(t, newServer(t, "greet"))

	before, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client.refetchLists(ctx)

	state := client.State()
	if state.Status != ClientConnected {
		t.Fatalf("Status = %q, want connected", state.Status)
	}

	if state.Error != "" {
		t.Fatalf("Error = %q, want empty", state.Error)
	}

	if !sameTools(before, client.tools) {
		t.Fatal("cancelled refetch replaced cached tools")
	}
}

func TestCallToolHonorsTimeout(t *testing.T) {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	mcpsdk.AddTool(server, &mcpsdk.Tool{Name: "slow"}, func(ctx context.Context, _ *mcpsdk.CallToolRequest, _ greetInput) (*mcpsdk.CallToolResult, greetOutput, error) {
		<-ctx.Done()

		return nil, greetOutput{}, ctx.Err()
	})

	client := NewClient(Config{
		Name:           "test",
		Transport:      TransportHTTP,
		URL:            serve(t, server),
		RequestTimeout: 20 * time.Millisecond,
	}, testTransport)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	t.Cleanup(func() { _ = client.Close() })

	_, err := client.CallTool(context.Background(), kit.NewToolCall("1", "slow", map[string]any{"name": "Ada"}))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CallTool() error = %v, want context.DeadlineExceeded", err)
	}

	if got := client.State().Status; got != ClientConnected {
		t.Fatalf("Status = %q, want connected", got)
	}
}

func TestCallToolFailureKeepsConnected(t *testing.T) {
	var failCalls atomic.Bool

	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return newServer(t, "greet")
	}, nil)

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failCalls.Load() && r.Method == http.MethodPost {
			http.Error(w, "boom", http.StatusInternalServerError)

			return
		}

		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(httpServer.Close)

	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: httpServer.URL}, testTransport).(*sdkClient)
	t.Cleanup(func() { _ = client.Close() })

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	failCalls.Store(true)

	if _, err := client.CallTool(context.Background(), kit.NewToolCall("1", "greet", map[string]any{"name": "Ada"})); err == nil {
		t.Fatal("CallTool() error = nil, want request failure")
	}

	if got := client.State().Status; got != ClientConnected {
		t.Fatalf("Status = %q, want connected", got)
	}
}

type greetInput struct {
	Name string `json:"name" jsonschema:"Name to greet"`
}

type greetOutput struct {
	Greeting string `json:"greeting" jsonschema:"Greeting text"`
}

func greet(_ context.Context, _ *mcpsdk.CallToolRequest, in greetInput) (*mcpsdk.CallToolResult, greetOutput, error) {
	return nil, greetOutput{Greeting: "Hi " + in.Name}, nil
}

func newServer(t *testing.T, tools ...string) *mcpsdk.Server {
	t.Helper()

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	for _, name := range tools {
		mcpsdk.AddTool(server, &mcpsdk.Tool{Name: name}, greet)
	}

	return server
}

func serve(t *testing.T, server *mcpsdk.Server) string {
	t.Helper()

	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return server
	}, nil)

	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	return httpServer.URL
}

func newClient(t *testing.T, server *mcpsdk.Server) *sdkClient {
	t.Helper()

	client := NewClient(Config{Name: "test", Transport: TransportHTTP, URL: serve(t, server)}, testTransport).(*sdkClient)
	t.Cleanup(func() { _ = client.Close() })

	return client
}

func connect(t *testing.T, server *mcpsdk.Server) *sdkClient {
	t.Helper()

	client := newClient(t, server)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	return client
}

func sameTools(a, b []kit.Tool) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

func eventually(t *testing.T, timeout time.Duration, ok func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	if !ok() {
		t.Fatal("condition was not met before timeout")
	}
}

type fakeTransportFactory struct {
	config Config
	err    error
	calls  int
}

func (f *fakeTransportFactory) New(config Config) (mcpsdk.Transport, error) {
	f.config = config
	f.calls++

	return nil, f.err
}

func testTransport(cfg Config) (mcpsdk.Transport, error) {
	return newTransport(cfg, nil, nil)
}
