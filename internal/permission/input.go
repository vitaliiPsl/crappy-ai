package permission

import "github.com/vitaliiPsl/crappy-adk/kit"

var toolInputKey = map[string]string{
	"list":       "path",
	"read_file":  "path",
	"write_file": "path",
	"edit_file":  "path",
	"web_fetch":  "url",
}

func ExtractInput(call kit.ToolCall) string {
	key, ok := toolInputKey[call.Name]
	if !ok || key == "" {
		return ""
	}

	value, _ := call.Arguments[key].(string)

	return value
}
