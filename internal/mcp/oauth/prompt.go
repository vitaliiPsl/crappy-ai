package oauth

import (
	"os/exec"
	"runtime"
)

type Prompter interface {
	Prompt(authURL string) error
}

type BrowserPrompter struct{}

func NewBrowserPrompter() BrowserPrompter {
	return BrowserPrompter{}
}

func (p BrowserPrompter) Prompt(authURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", authURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", authURL)
	default:
		cmd = exec.Command("xdg-open", authURL)
	}

	return cmd.Start()
}
