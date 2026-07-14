package command

import (
	"context"
	"testing"
)

func TestAttachCommandEmitsPath(t *testing.T) {
	msg := NewAttachCommand().Execute(context.Background(), Request{
		Args: []string{"docs/my", "file.txt"},
	})()

	attach, ok := msg.(AttachFileMsg)
	if !ok {
		t.Fatalf("message = %#v, want AttachFileMsg", msg)
	}

	if attach.Path != "docs/my file.txt" {
		t.Fatalf("path = %q, want joined path", attach.Path)
	}
}
