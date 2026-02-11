package services

import (
	"net/http"
	"path/filepath"
	"strings"
)

// InferContentType determines the content type from explicit header, file extension, or body sniffing.
func InferContentType(explicit string, bodyFile string, body []byte) string {
	if explicit != "" {
		return explicit
	}

	if bodyFile != "" {
		ext := strings.ToLower(filepath.Ext(bodyFile))
		switch ext {
		case ".json":
			return "application/json"
		case ".xml":
			return "application/xml"
		case ".html", ".htm":
			return "text/html"
		case ".txt":
			return "text/plain"
		case ".csv":
			return "text/csv"
		}
	}

	if len(body) > 0 {
		return http.DetectContentType(body)
	}

	return "application/octet-stream"
}
