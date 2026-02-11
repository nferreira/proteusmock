package services_test

import (
	"testing"

	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
)

func TestInferContentType(t *testing.T) {
	tests := []struct {
		name     string
		explicit string
		bodyFile string
		body     []byte
		expected string
	}{
		{"explicit header wins", "text/plain", "file.json", []byte(`{"a":1}`), "text/plain"},
		{".json extension", "", "data.json", nil, "application/json"},
		{".xml extension", "", "data.xml", nil, "application/xml"},
		{".html extension", "", "page.html", nil, "text/html"},
		{".htm extension", "", "page.htm", nil, "text/html"},
		{".txt extension", "", "readme.txt", nil, "text/plain"},
		{".csv extension", "", "data.csv", nil, "text/csv"},
		{".unknown extension falls through to sniff", "", "file.xyz", []byte(`{"a":1}`), "text/plain; charset=utf-8"},
		{"body sniffing JSON-like", "", "", []byte(`{"key":"val"}`), "text/plain; charset=utf-8"},
		{"body sniffing HTML", "", "", []byte(`<html><body>hi</body></html>`), "text/html; charset=utf-8"},
		{"empty body no file", "", "", nil, "application/octet-stream"},
		{"empty body empty file", "", "", []byte{}, "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := services.InferContentType(tt.explicit, tt.bodyFile, tt.body)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
