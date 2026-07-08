package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"

	mcpcore "github.com/vitaliiPsl/crappy-ai/internal/mcp"
)

func TestResourceToolsListAndRead(t *testing.T) {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	server.AddResource(&mcpsdk.Resource{
		Name:     "docs",
		URI:      "docs://readme",
		MIMEType: "text/markdown",
	}, func(context.Context, *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		return &mcpsdk.ReadResourceResult{
			Contents: []*mcpsdk.ResourceContents{{
				URI:      "docs://readme",
				MIMEType: "text/markdown",
				Text:     "# Docs",
			}},
		}, nil
	})
	server.AddResourceTemplate(&mcpsdk.ResourceTemplate{
		Name:        "doc",
		URITemplate: "docs://{name}",
		MIMEType:    "text/markdown",
	}, func(context.Context, *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		return &mcpsdk.ReadResourceResult{}, nil
	})

	manager := mcpcore.NewManager(
		context.Background(),
		[]mcpcore.Config{{Name: "docs", Transport: mcpcore.TransportHTTP, URL: serve(t, server)}},
		nil,
		nil,
	)
	t.Cleanup(manager.Close)

	eventually(t, time.Second, func() bool {
		client, err := manager.Get("docs")

		return err == nil && client.State().Status == mcpcore.ClientConnected
	})

	tools := mapTools(newResourceTools(manager))
	rc := kit.NewRunContext(context.Background())

	resources, err := tools[listResourcesName].Execute(rc, kit.NewToolCall("call-1", listResourcesName, nil))
	if err != nil {
		t.Fatalf("list_mcp_resources: %v", err)
	}

	resourceText := kit.ContentsText(resources.Content)
	if !strings.Contains(resourceText, `"server": "docs"`) || !strings.Contains(resourceText, `"uri": "docs://readme"`) {
		t.Fatalf("list_mcp_resources output = %s, want docs resource", resourceText)
	}

	templates, err := tools[listResourceTemplatesName].Execute(rc, kit.NewToolCall("call-2", listResourceTemplatesName, nil))
	if err != nil {
		t.Fatalf("list_mcp_resource_templates: %v", err)
	}

	templateText := kit.ContentsText(templates.Content)
	if !strings.Contains(templateText, `"uri_template": "docs://{name}"`) {
		t.Fatalf("list_mcp_resource_templates output = %s, want docs template", templateText)
	}

	content, err := tools[readResourceName].Execute(rc, kit.NewToolCall("call-3", readResourceName, map[string]any{"server": "docs", "uri": "docs://readme"}))
	if err != nil {
		t.Fatalf("read_mcp_resource: %v", err)
	}

	contentText := kit.ContentsText(content.Content)
	if !strings.Contains(contentText, "# Docs") {
		t.Fatalf("read_mcp_resource output = %s, want resource text", contentText)
	}
}

func mapTools(tools []kit.Tool) map[string]kit.Tool {
	out := make(map[string]kit.Tool, len(tools))
	for _, tool := range tools {
		out[tool.Definition().Name] = tool
	}

	return out
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
