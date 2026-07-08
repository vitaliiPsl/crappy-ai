package mcp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/tool"

	mcpcore "github.com/vitaliiPsl/crappy-ai/internal/mcp"
)

const (
	listResourcesName         = "list_mcp_resources"
	listResourceTemplatesName = "list_mcp_resource_templates"
	readResourceName          = "read_mcp_resource"
)

type listResourcesInput struct {
	Server string `json:"server,omitempty" jsonschema:"Optional MCP server name. When omitted, lists resources from every connected server."`
}

type readResourceInput struct {
	Server string `json:"server" jsonschema:"MCP server name exactly as returned by list_mcp_resources"`
	URI    string `json:"uri" jsonschema:"Resource URI to read. Use the exact URI returned by list_mcp_resources or a filled resource template URI."`
}

type resourceOutput struct {
	Server      string `json:"server"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	URI         string `json:"uri"`
	MIMEType    string `json:"mime_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

type resourceTemplateOutput struct {
	Server      string `json:"server"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	URITemplate string `json:"uri_template"`
	MIMEType    string `json:"mime_type,omitempty"`
}

type readResourceOutput struct {
	Server string `json:"server"`
	URI    string `json:"uri"`
}

func newResourceTools(manager *mcpcore.Manager) []kit.Tool {
	return []kit.Tool{
		newListResourcesTool(manager),
		newListResourceTemplatesTool(manager),
		newReadResourceTool(manager),
	}
}

func newListResourcesTool(manager *mcpcore.Manager) kit.Tool {
	return tool.MustNew(
		listResourcesName,
		"Lists resources provided by connected MCP servers. Resources provide context such as files, database schemas, documentation, or application-specific information.",
		func(rc *kit.RunContext, input listResourcesInput) (kit.ToolOutput, error) {
			resources, err := listResources(rc, manager, strings.TrimSpace(input.Server))
			if err != nil {
				return kit.ToolOutput{}, err
			}

			return kit.NewStructuredToolOutput(map[string]any{"resources": resources}), nil
		},
	)
}

func newListResourceTemplatesTool(manager *mcpcore.Manager) kit.Tool {
	return tool.MustNew(
		listResourceTemplatesName,
		"Lists resource templates provided by connected MCP servers. Resource templates are parameterized resource URI patterns that can be filled and read with read_mcp_resource.",
		func(rc *kit.RunContext, input listResourcesInput) (kit.ToolOutput, error) {
			templates, err := listResourceTemplates(rc, manager, strings.TrimSpace(input.Server))
			if err != nil {
				return kit.ToolOutput{}, err
			}

			return kit.NewStructuredToolOutput(map[string]any{"resource_templates": templates}), nil
		},
	)
}

func newReadResourceTool(manager *mcpcore.Manager) kit.Tool {
	return tool.MustNew(
		readResourceName,
		"Read a specific MCP resource using the server name and resource URI. Use list_mcp_resources or list_mcp_resource_templates first when you need to discover available resource URIs.",
		func(rc *kit.RunContext, input readResourceInput) (kit.ToolOutput, error) {
			server := strings.TrimSpace(input.Server)
			if server == "" {
				return kit.ToolOutput{}, fmt.Errorf("server is required")
			}

			uri := strings.TrimSpace(input.URI)
			if uri == "" {
				return kit.ToolOutput{}, fmt.Errorf("uri is required")
			}

			client, err := manager.Get(server)
			if err != nil {
				return kit.ToolOutput{}, err
			}

			result, err := client.ReadResource(rc.Context, uri)
			if err != nil {
				return kit.ToolOutput{}, err
			}

			return kit.ToolOutput{
				Content: result,
				Structured: readResourceOutput{
					Server: server,
					URI:    uri,
				},
			}, nil
		},
	)
}

func listResources(rc *kit.RunContext, manager *mcpcore.Manager, server string) ([]resourceOutput, error) {
	var out []resourceOutput
	for _, client := range manager.List() {
		if client.State().Status != mcpcore.ClientConnected {
			continue
		}

		if server != "" && client.Config().Name != server {
			continue
		}

		resources, err := client.ListResources(rc.Context)
		if err != nil {
			continue
		}

		for _, resource := range resources {
			out = append(out, resourceOutput{
				Server:      client.Config().Name,
				Name:        resource.Name,
				Title:       resource.Title,
				Description: resource.Description,
				URI:         resource.URI,
				MIMEType:    resource.MIMEType,
				Size:        resource.Size,
			})
		}
	}

	if server != "" && len(out) == 0 {
		if _, err := manager.Get(server); err != nil {
			return nil, err
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Server+"\x00"+out[i].Name+"\x00"+out[i].URI < out[j].Server+"\x00"+out[j].Name+"\x00"+out[j].URI
	})

	return out, nil
}

func listResourceTemplates(rc *kit.RunContext, manager *mcpcore.Manager, server string) ([]resourceTemplateOutput, error) {
	var out []resourceTemplateOutput
	for _, client := range manager.List() {
		if client.State().Status != mcpcore.ClientConnected {
			continue
		}

		if server != "" && client.Config().Name != server {
			continue
		}

		templates, err := client.ListResourceTemplates(rc.Context)
		if err != nil {
			continue
		}

		for _, template := range templates {
			out = append(out, resourceTemplateOutput{
				Server:      client.Config().Name,
				Name:        template.Name,
				Title:       template.Title,
				Description: template.Description,
				URITemplate: template.URITemplate,
				MIMEType:    template.MIMEType,
			})
		}
	}

	if server != "" && len(out) == 0 {
		if _, err := manager.Get(server); err != nil {
			return nil, err
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Server+"\x00"+out[i].Name+"\x00"+out[i].URITemplate < out[j].Server+"\x00"+out[j].Name+"\x00"+out[j].URITemplate
	})

	return out, nil
}
