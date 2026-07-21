package memory

import (
	"fmt"

	"github.com/vitaliiPsl/crappy-adk/kit"
	"github.com/vitaliiPsl/crappy-adk/x/tool"

	corememory "github.com/vitaliiPsl/crappy-ai/internal/memory"
)

const (
	toolList                = "memory_list"
	toolListDescription     = "List all persistent memories with IDs and timestamps."
	toolRemember            = "memory_remember"
	toolRememberDescription = "Save one persistent memory. Use only when the user explicitly asks you to remember something."
	toolUpdate              = "memory_update"
	toolUpdateDescription   = "Correct an existing persistent memory by exact ID. Call memory_list first."
	toolForget              = "memory_forget"
	toolForgetDescription   = "Delete a persistent memory by exact ID. Call memory_list first."
)

type rememberInput struct {
	Kind    string `json:"kind" jsonschema:"Memory kind: profile, preference, or instruction"`
	Content string `json:"content" jsonschema:"One concise, durable memory explicitly requested by the user"`
}

type updateInput struct {
	ID      string `json:"id" jsonschema:"Exact ID returned by memory_list"`
	Kind    string `json:"kind" jsonschema:"Memory kind: profile, preference, or instruction"`
	Content string `json:"content" jsonschema:"Complete corrected memory content"`
}

type forgetInput struct {
	ID string `json:"id" jsonschema:"Exact ID returned by memory_list"`
}

func newTools(store corememory.Store) []kit.Tool {
	return []kit.Tool{
		newListTool(store),
		newRememberTool(store),
		newUpdateTool(store),
		newForgetTool(store),
	}
}

func newListTool(store corememory.Store) kit.Tool {
	return tool.MustNew(toolList, toolListDescription,
		func(rc *kit.RunContext, _ struct{}) (string, error) {
			memories, err := store.List(rc.Context)
			if err != nil {
				return "", fmt.Errorf("list memories: %w", err)
			}

			return formatList(memories), nil
		})
}

func newRememberTool(store corememory.Store) kit.Tool {
	return tool.MustNew(toolRemember, toolRememberDescription,
		func(rc *kit.RunContext, input rememberInput) (string, error) {
			created, err := store.Create(rc.Context, corememory.CreateParams{
				Kind:    corememory.Kind(input.Kind),
				Content: input.Content,
			})
			if err != nil {
				return "", fmt.Errorf("remember: %w", err)
			}

			return fmt.Sprintf("Memory saved with ID %s.", created.ID), nil
		})
}

func newUpdateTool(store corememory.Store) kit.Tool {
	return tool.MustNew(toolUpdate, toolUpdateDescription,
		func(rc *kit.RunContext, input updateInput) (string, error) {
			updated, err := store.Update(rc.Context, corememory.UpdateParams{
				ID:      input.ID,
				Kind:    corememory.Kind(input.Kind),
				Content: input.Content,
			})
			if err != nil {
				return "", fmt.Errorf("update memory: %w", err)
			}

			return fmt.Sprintf("Memory %s updated.", updated.ID), nil
		})
}

func newForgetTool(store corememory.Store) kit.Tool {
	return tool.MustNew(toolForget, toolForgetDescription,
		func(rc *kit.RunContext, input forgetInput) (string, error) {
			if err := store.Delete(rc.Context, input.ID); err != nil {
				return "", fmt.Errorf("forget memory: %w", err)
			}

			return fmt.Sprintf("Memory %s forgotten.", input.ID), nil
		})
}
