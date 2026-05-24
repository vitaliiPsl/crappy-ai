package instructions

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

func Env(cwd string) string {
	var b strings.Builder
	b.WriteString("# Environment\n")

	if cwd != "" {
		fmt.Fprintf(&b, "- Working directory: %s\n", cwd)
	}

	fmt.Fprintf(&b, "- OS: %s\n", runtime.GOOS)
	fmt.Fprintf(&b, "- Arch: %s\n", runtime.GOARCH)

	if shell := os.Getenv("SHELL"); shell != "" {
		fmt.Fprintf(&b, "- Shell: %s\n", shell)
	}

	return strings.TrimRight(b.String(), "\n")
}
