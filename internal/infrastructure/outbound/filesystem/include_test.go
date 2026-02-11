package filesystem_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/filesystem"
)

func TestIncludeResolver_DepthLimit(t *testing.T) {
	dir := t.TempDir()

	// Create a self-referencing include chain that exceeds max depth.
	content := "body: !include self.yaml\n"
	if err := os.WriteFile(filepath.Join(dir, "self.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := filesystem.NewIncludeResolver(dir)

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		t.Fatal(err)
	}

	err := resolver.ResolveIncludes(&node, dir)
	if err == nil {
		t.Error("expected error for exceeding include depth")
	}
}

func TestIncludeResolver_EmptyValue(t *testing.T) {
	dir := t.TempDir()

	content := "body: !include \"\"\n"
	resolver := filesystem.NewIncludeResolver(dir)

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		t.Fatal(err)
	}

	err := resolver.ResolveIncludes(&node, dir)
	if err == nil {
		t.Error("expected error for empty !include value")
	}
}

func TestIncludeResolver_AbsolutePathRejected(t *testing.T) {
	dir := t.TempDir()

	content := "body: !include /etc/passwd\n"
	resolver := filesystem.NewIncludeResolver(dir)

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		t.Fatal(err)
	}

	err := resolver.ResolveIncludes(&node, dir)
	if err == nil {
		t.Error("expected error for absolute path in !include")
	}
}

func TestIncludeResolver_NonYAMLFile(t *testing.T) {
	dir := t.TempDir()

	// Create a non-YAML file to include.
	if err := os.WriteFile(filepath.Join(dir, "response.json"), []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	content := "body: !include response.json\n"
	resolver := filesystem.NewIncludeResolver(dir)

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		t.Fatal(err)
	}

	err := resolver.ResolveIncludes(&node, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The body node should now contain the raw file content.
	bodyNode := node.Content[0].Content[1] // mapping → value
	if bodyNode.Value != `{"ok":true}` {
		t.Errorf("expected raw JSON content, got %q", bodyNode.Value)
	}
}

func TestIncludeResolver_AtRootReference(t *testing.T) {
	dir := t.TempDir()

	// Create a file at root.
	if err := os.WriteFile(filepath.Join(dir, "shared.json"), []byte(`{"shared":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	content := "body: !include \"@root/shared.json\"\n"
	resolver := filesystem.NewIncludeResolver(dir)

	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0o755)

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		t.Fatal(err)
	}

	err := resolver.ResolveIncludes(&node, subdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIncludeResolver_AtHereReference(t *testing.T) {
	dir := t.TempDir()

	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0o755)

	if err := os.WriteFile(filepath.Join(subdir, "local.json"), []byte(`{"local":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	content := "body: !include \"@here/local.json\"\n"
	resolver := filesystem.NewIncludeResolver(dir)

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		t.Fatal(err)
	}

	err := resolver.ResolveIncludes(&node, subdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIncludeResolver_TraversalRejected(t *testing.T) {
	dir := t.TempDir()

	content := "body: !include ../../etc/passwd\n"
	resolver := filesystem.NewIncludeResolver(dir)

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		t.Fatal(err)
	}

	err := resolver.ResolveIncludes(&node, dir)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestIncludeResolver_YAMLFileInclude(t *testing.T) {
	dir := t.TempDir()

	// Create a YAML file to include.
	included := "status: 200\nheaders:\n  Content-Type: application/json\n"
	if err := os.WriteFile(filepath.Join(dir, "response.yaml"), []byte(included), 0o644); err != nil {
		t.Fatal(err)
	}

	content := "response: !include response.yaml\n"
	resolver := filesystem.NewIncludeResolver(dir)

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		t.Fatal(err)
	}

	err := resolver.ResolveIncludes(&node, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The response node should be replaced by the included YAML.
	responseNode := node.Content[0].Content[1]
	if responseNode.Kind != yaml.MappingNode {
		t.Errorf("expected mapping node, got kind %d", responseNode.Kind)
	}
}

func TestIncludeResolver_MissingFile(t *testing.T) {
	dir := t.TempDir()

	content := "body: !include nonexistent.json\n"
	resolver := filesystem.NewIncludeResolver(dir)

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		t.Fatal(err)
	}

	err := resolver.ResolveIncludes(&node, dir)
	if err == nil {
		t.Error("expected error for missing included file")
	}
}

func TestIncludeResolver_InvalidYAMLInInclude(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(":\n\t\tbad"), 0o644); err != nil {
		t.Fatal(err)
	}

	content := "body: !include bad.yaml\n"
	resolver := filesystem.NewIncludeResolver(dir)

	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		t.Fatal(err)
	}

	err := resolver.ResolveIncludes(&node, dir)
	if err == nil {
		t.Error("expected error for invalid YAML in included file")
	}
}

func TestIncludeResolver_NilNode(t *testing.T) {
	dir := t.TempDir()
	resolver := filesystem.NewIncludeResolver(dir)

	err := resolver.ResolveIncludes(nil, dir)
	if err != nil {
		t.Errorf("expected nil error for nil node, got %v", err)
	}
}
