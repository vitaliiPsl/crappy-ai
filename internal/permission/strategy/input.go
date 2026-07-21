package strategy

import "github.com/vitaliiPsl/crappy-adk/kit"

const (
	ToolList           = "list"
	ToolReadFile       = "read_file"
	ToolWriteFile      = "write_file"
	ToolEditFile       = "edit_file"
	ToolWebFetch       = "web_fetch"
	ToolBash           = "bash"
	ToolJobStatus      = "job_status"
	ToolJobResult      = "job_result"
	ToolJobCancel      = "job_cancel"
	ToolJobList        = "job_list"
	ToolMemoryList     = "memory_list"
	ToolMemoryRemember = "memory_remember"
	ToolMemoryUpdate   = "memory_update"
	ToolMemoryForget   = "memory_forget"
)

const (
	inputURL     = "url"
	inputPath    = "path"
	inputCommand = "command"
	inputContent = "content"
	inputID      = "id"
)

var toolInputKey = map[string]string{
	ToolList:           inputPath,
	ToolReadFile:       inputPath,
	ToolWriteFile:      inputPath,
	ToolEditFile:       inputPath,
	ToolWebFetch:       inputURL,
	ToolBash:           inputCommand,
	ToolMemoryRemember: inputContent,
	ToolMemoryUpdate:   inputContent,
	ToolMemoryForget:   inputID,
}

func extractInput(call kit.ToolCall) string {
	key, ok := toolInputKey[call.Name]
	if !ok || key == "" {
		return ""
	}

	value, _ := call.Arguments[key].(string)

	return value
}
