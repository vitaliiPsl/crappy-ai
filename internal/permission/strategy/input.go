package strategy

import "github.com/vitaliiPsl/crappy-adk/kit"

const (
	ToolList      = "list"
	ToolReadFile  = "read_file"
	ToolWriteFile = "write_file"
	ToolEditFile  = "edit_file"
	ToolWebFetch  = "web_fetch"
	ToolBash      = "bash"
)

const (
	inputURL     = "url"
	inputPath    = "path"
	inputCommand = "command"
)

var toolInputKey = map[string]string{
	ToolList:      inputPath,
	ToolReadFile:  inputPath,
	ToolWriteFile: inputPath,
	ToolEditFile:  inputPath,
	ToolWebFetch:  inputURL,
	ToolBash:      inputCommand,
}

func extractInput(call kit.ToolCall) string {
	key, ok := toolInputKey[call.Name]
	if !ok || key == "" {
		return ""
	}

	value, _ := call.Arguments[key].(string)

	return value
}
