package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

func TestLoadAttachmentLoadsImageContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pixel.png")

	data := []byte("\x89PNG\r\n\x1a\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	attachment, err := loadAttachment(dir, "pixel.png")
	if err != nil {
		t.Fatalf("loadAttachment: %v", err)
	}

	if attachment.Name != "pixel.png" {
		t.Fatalf("attachment name = %q, want pixel.png", attachment.Name)
	}

	if attachment.Content.Type != kit.ContentTypeImage || attachment.Content.Image == nil {
		t.Fatalf("attachment content = %+v, want image", attachment.Content)
	}

	if attachment.Content.Image.MIMEType != "image/png" || string(attachment.Content.Image.Data) != string(data) {
		t.Fatalf("attachment image = %+v, want inline png", attachment.Content.Image)
	}
}

func TestLoadAttachmentLoadsTextResource(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	attachment, err := loadAttachment(dir, "notes.txt")
	if err != nil {
		t.Fatalf("loadAttachment: %v", err)
	}

	resource := attachment.Content.Resource
	if attachment.Content.Type != kit.ContentTypeResource || resource == nil {
		t.Fatalf("attachment content = %+v, want resource", attachment.Content)
	}

	if resource.Name != "notes.txt" || resource.MIMEType != "text/plain" || resource.Text != "hello" {
		t.Fatalf("attachment resource = %+v, want text file", resource)
	}
}

func TestAttachmentPathsParsesDroppedFiles(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first image.png")

	second := filepath.Join(dir, "second.png")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("image"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	value := strings.ReplaceAll(first, " ", `\ `) + " " + strings.ReplaceAll(second, " ", `\ `)

	paths, ok := attachmentPaths(dir, value)
	if !ok {
		t.Fatal("attachmentPaths did not recognize dropped files")
	}

	if len(paths) != 2 || paths[0] != first || paths[1] != second {
		t.Fatalf("paths = %#v, want both dropped files", paths)
	}
}

func TestAttachmentPathsRejectsOrdinaryPaste(t *testing.T) {
	if paths, ok := attachmentPaths(t.TempDir(), "please inspect /missing/file.png"); ok {
		t.Fatalf("attachmentPaths = %#v, want ordinary paste", paths)
	}
}
