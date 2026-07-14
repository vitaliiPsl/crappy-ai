package session

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"mvdan.cc/sh/v3/shell"

	"github.com/vitaliiPsl/crappy-adk/kit"

	"github.com/vitaliiPsl/crappy-ai/internal/utils"
)

const (
	fileURLPrefix         = "file://"
	localFileHost         = "localhost"
	imageMIMEPrefix       = "image/"
	audioMIMEPrefix       = "audio/"
	textMIMEPrefix        = "text/"
	jsonMIMEType          = "application/json"
	attachmentLabelPrefix = "["
	attachmentLabelSuffix = "]"
)

type attachment struct {
	Name    string
	Content kit.Content
}

func (a attachment) label() string {
	return attachmentLabelPrefix + a.Name + attachmentLabelSuffix
}

func loadAttachmentCmd(cwd, path string) tea.Cmd {
	return func() tea.Msg {
		result, err := loadAttachment(cwd, path)

		return attachmentLoadedMsg{result: result, err: err}
	}
}

func loadAttachment(cwd, path string) (attachment, error) {
	path, err := resolveAttachmentPath(cwd, path)
	if err != nil {
		return attachment{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return attachment{}, fmt.Errorf("read attachment %q: %w", path, err)
	}

	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	if mediaType, _, err := mime.ParseMediaType(mimeType); err == nil {
		mimeType = mediaType
	}

	name := filepath.Base(path)

	var content kit.Content
	switch {
	case strings.HasPrefix(mimeType, imageMIMEPrefix):
		content = kit.NewImageContent(mimeType, data)
	case strings.HasPrefix(mimeType, audioMIMEPrefix):
		content = kit.NewAudioContent(mimeType, data)
	default:
		resource := kit.Resource{Name: name, MIMEType: mimeType}
		if strings.HasPrefix(mimeType, textMIMEPrefix) || mimeType == jsonMIMEType {
			resource.Text = string(data)
		} else {
			resource.Blob = data
		}

		content = kit.NewResourceContent(resource)
	}

	return attachment{Name: name, Content: content}, nil
}

func attachmentPaths(cwd, value string) ([]string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}

	fields, err := shell.Fields(value, nil)
	if err == nil {
		if paths, ok := existingAttachmentPaths(cwd, fields); ok {
			return paths, true
		}
	}

	return existingAttachmentPaths(cwd, []string{value})
}

func existingAttachmentPaths(cwd string, candidates []string) ([]string, bool) {
	if len(candidates) == 0 {
		return nil, false
	}

	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		path, err := resolveAttachmentPath(cwd, candidate)
		if err != nil {
			return nil, false
		}

		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return nil, false
		}

		paths = append(paths, path)
	}

	return paths, true
}

func resolveAttachmentPath(cwd, path string) (string, error) {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, fileURLPrefix) {
		fileURL, err := url.Parse(path)
		if err != nil {
			return "", fmt.Errorf("parse attachment URL %q: %w", path, err)
		}

		if fileURL.Host != "" && fileURL.Host != localFileHost {
			return "", fmt.Errorf("attachment URL %q is not local", path)
		}

		path = fileURL.Path
	}

	path = utils.ExpandHome(path)

	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}

	return filepath.Clean(path), nil
}
