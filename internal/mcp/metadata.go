package mcp

import (
	"context"
	"encoding/json"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func fetchTools(ctx context.Context, cfg Config, client Client, session *mcpsdk.ClientSession) ([]kit.Tool, error) {
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	defs, err := convertTools(res.Tools)
	if err != nil {
		return nil, err
	}

	tools := make([]kit.Tool, 0, len(defs))
	for _, def := range defs {
		tools = append(tools, newTool(cfg.Name, client, def))
	}

	return tools, nil
}

func fetchPrompts(ctx context.Context, session *mcpsdk.ClientSession) ([]Prompt, error) {
	var prompts []Prompt
	for prompt, err := range session.Prompts(ctx, nil) {
		if err != nil {
			return nil, err
		}

		prompts = append(prompts, convertPrompt(prompt))
	}

	return prompts, nil
}

func fetchResources(ctx context.Context, session *mcpsdk.ClientSession) ([]Resource, error) {
	var resources []Resource
	for resource, err := range session.Resources(ctx, nil) {
		if err != nil {
			return nil, err
		}

		resources = append(resources, convertResource(resource))
	}

	return resources, nil
}

func fetchResourceTemplates(ctx context.Context, session *mcpsdk.ClientSession) ([]ResourceTemplate, error) {
	var templates []ResourceTemplate
	for template, err := range session.ResourceTemplates(ctx, nil) {
		if err != nil {
			return nil, err
		}

		templates = append(templates, convertResourceTemplate(template))
	}

	return templates, nil
}

func convertPrompt(prompt *mcpsdk.Prompt) Prompt {
	if prompt == nil {
		return Prompt{}
	}

	out := Prompt{
		Name:        prompt.Name,
		Title:       prompt.Title,
		Description: prompt.Description,
		Arguments:   make([]PromptArgument, 0, len(prompt.Arguments)),
	}

	for _, arg := range prompt.Arguments {
		out.Arguments = append(out.Arguments, convertPromptArgument(arg))
	}

	return out
}

func convertPromptArgument(arg *mcpsdk.PromptArgument) PromptArgument {
	if arg == nil {
		return PromptArgument{}
	}

	return PromptArgument{
		Name:        arg.Name,
		Title:       arg.Title,
		Description: arg.Description,
		Required:    arg.Required,
	}
}

func convertPromptResult(result *mcpsdk.GetPromptResult) []kit.Message {
	if result == nil {
		return nil
	}

	out := make([]kit.Message, 0, len(result.Messages))
	for _, message := range result.Messages {
		out = append(out, convertPromptMessage(message))
	}

	return out
}

func convertPromptMessage(message *mcpsdk.PromptMessage) kit.Message {
	if message == nil {
		return kit.Message{}
	}

	return kit.Message{
		Role:    convertPromptRole(message.Role),
		Content: []kit.Content{convertMCPContent(message.Content)},
	}
}

func convertPromptRole(role mcpsdk.Role) kit.Role {
	switch string(role) {
	case string(kit.RoleModel), "assistant":
		return kit.RoleModel
	case string(kit.RoleTool):
		return kit.RoleTool
	default:
		return kit.RoleUser
	}
}

func convertMCPContent(content mcpsdk.Content) kit.Content {
	switch c := content.(type) {
	case *mcpsdk.TextContent:
		return kit.NewTextContent(c.Text)
	case *mcpsdk.ImageContent:
		return kit.NewImageContent(c.MIMEType, c.Data)
	case *mcpsdk.AudioContent:
		return kit.NewAudioContent(c.MIMEType, c.Data)
	case *mcpsdk.ResourceLink:
		return kit.NewResourceContent(kit.Resource{
			URI:         c.URI,
			Name:        c.Name,
			Title:       c.Title,
			Description: c.Description,
			MIMEType:    c.MIMEType,
		})
	case *mcpsdk.EmbeddedResource:
		return convertResourceContent(c.Resource)
	default:
		data, err := json.Marshal(content)
		if err != nil {
			return kit.Content{}
		}

		return kit.NewTextContent(string(data))
	}
}

func convertResource(resource *mcpsdk.Resource) Resource {
	if resource == nil {
		return Resource{}
	}

	return Resource{
		Name:        resource.Name,
		Title:       resource.Title,
		Description: resource.Description,
		URI:         resource.URI,
		MIMEType:    resource.MIMEType,
		Size:        resource.Size,
	}
}

func convertResourceTemplate(template *mcpsdk.ResourceTemplate) ResourceTemplate {
	if template == nil {
		return ResourceTemplate{}
	}

	return ResourceTemplate{
		Name:        template.Name,
		Title:       template.Title,
		Description: template.Description,
		URITemplate: template.URITemplate,
		MIMEType:    template.MIMEType,
	}
}

func convertResourceResult(result *mcpsdk.ReadResourceResult) []kit.Content {
	if result == nil {
		return nil
	}

	out := make([]kit.Content, 0, len(result.Contents))
	for _, content := range result.Contents {
		out = append(out, convertResourceContent(content))
	}

	return out
}

func convertResourceContent(content *mcpsdk.ResourceContents) kit.Content {
	if content == nil {
		return kit.Content{}
	}

	return kit.NewResourceContent(kit.Resource{
		URI:      content.URI,
		MIMEType: content.MIMEType,
		Text:     content.Text,
		Blob:     content.Blob,
	})
}
