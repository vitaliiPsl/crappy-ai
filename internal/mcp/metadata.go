package mcp

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
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

func fetchOptional[T any](ctx context.Context, session *mcpsdk.ClientSession, fetch func(context.Context, *mcpsdk.ClientSession) ([]T, error)) ([]T, error) {
	items, err := fetch(ctx, session)
	if err != nil {
		if isUnsupportedCapability(err) {
			return nil, nil
		}

		return nil, err
	}

	return items, nil
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

func convertPromptResult(result *mcpsdk.GetPromptResult) PromptResult {
	if result == nil {
		return PromptResult{}
	}

	out := PromptResult{
		Description: result.Description,
		Messages:    make([]PromptMessage, 0, len(result.Messages)),
	}

	for _, message := range result.Messages {
		out.Messages = append(out.Messages, convertPromptMessage(message))
	}

	return out
}

func convertPromptMessage(message *mcpsdk.PromptMessage) PromptMessage {
	if message == nil {
		return PromptMessage{}
	}

	return PromptMessage{
		Role:    string(message.Role),
		Content: []PromptContent{convertPromptContent(message.Content)},
	}
}

func convertPromptContent(content mcpsdk.Content) PromptContent {
	switch c := content.(type) {
	case *mcpsdk.TextContent:
		return PromptContent{Type: "text", Text: c.Text}
	case *mcpsdk.ImageContent:
		return PromptContent{Type: "image", MIMEType: c.MIMEType, Data: append([]byte(nil), c.Data...)}
	case *mcpsdk.AudioContent:
		return PromptContent{Type: "audio", MIMEType: c.MIMEType, Data: append([]byte(nil), c.Data...)}
	case *mcpsdk.ResourceLink:
		return PromptContent{
			Type:        "resource_link",
			URI:         c.URI,
			Name:        c.Name,
			Title:       c.Title,
			Description: c.Description,
			MIMEType:    c.MIMEType,
		}
	case *mcpsdk.EmbeddedResource:
		resource := convertResourceContent(c.Resource)

		return PromptContent{
			Type:     "resource",
			URI:      resource.URI,
			MIMEType: resource.MIMEType,
			Text:     resource.Text,
			Resource: &resource,
		}
	default:
		return PromptContent{Type: "unknown"}
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

func convertResourceResult(result *mcpsdk.ReadResourceResult) ResourceResult {
	if result == nil {
		return ResourceResult{}
	}

	out := ResourceResult{
		Contents: make([]ResourceContent, 0, len(result.Contents)),
	}

	for _, content := range result.Contents {
		out.Contents = append(out.Contents, convertResourceContent(content))
	}

	return out
}

func convertResourceContent(content *mcpsdk.ResourceContents) ResourceContent {
	if content == nil {
		return ResourceContent{}
	}

	return ResourceContent{
		URI:      content.URI,
		MIMEType: content.MIMEType,
		Text:     content.Text,
		Blob:     append([]byte(nil), content.Blob...),
	}
}

func isUnsupportedCapability(err error) bool {
	var rpcErr *jsonrpc.Error
	return errors.As(err, &rpcErr) && rpcErr.Code == jsonrpc.CodeMethodNotFound
}
