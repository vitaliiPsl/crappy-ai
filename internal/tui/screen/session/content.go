package session

import (
	"strings"

	"github.com/vitaliiPsl/crappy-adk/kit"
)

const (
	contentSeparator  = "\n"
	fileLabelPrefix   = "[file: "
	fileLabelSuffix   = "]"
	fileMIMESeparator = ", "
)

func userContentText(content []kit.Content) string {
	parts := make([]string, 0, len(content))
	for _, item := range content {
		if item.Type == kit.ContentTypeResource && item.Resource != nil && item.Resource.Name != "" {
			label := item.Resource.Name
			if item.Resource.MIMEType != "" {
				label += fileMIMESeparator + item.Resource.MIMEType
			}

			parts = append(parts, fileLabelPrefix+label+fileLabelSuffix)

			continue
		}

		text, ok := kit.ContentText(item)
		if ok && text != "" {
			parts = append(parts, text)
		}
	}

	return strings.Join(parts, contentSeparator)
}
