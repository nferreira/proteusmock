package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// IncludeResolver resolves !include tags in YAML node trees.
type IncludeResolver struct {
	rootDir string
}

// NewIncludeResolver creates a resolver bound to rootDir for @root references.
func NewIncludeResolver(rootDir string) *IncludeResolver {
	return &IncludeResolver{rootDir: rootDir}
}

// ResolveIncludes walks a yaml.Node tree and replaces !include tagged nodes
// with the contents of the referenced files.
func (r *IncludeResolver) ResolveIncludes(node *yaml.Node, currentDir string) error {
	return r.walk(node, currentDir, 0)
}

const maxIncludeDepth = 10

func (r *IncludeResolver) walk(node *yaml.Node, currentDir string, depth int) error {
	if depth > maxIncludeDepth {
		return fmt.Errorf("!include depth exceeds maximum of %d", maxIncludeDepth)
	}
	if node == nil {
		return nil
	}

	if node.Tag == "!include" {
		return r.resolveInclude(node, currentDir, depth)
	}

	for _, child := range node.Content {
		if err := r.walk(child, currentDir, depth); err != nil {
			return err
		}
	}

	return nil
}

func (r *IncludeResolver) resolveInclude(node *yaml.Node, currentDir string, depth int) error {
	ref := node.Value
	if ref == "" {
		return fmt.Errorf("!include tag has empty value")
	}

	resolved, err := r.resolvePath(ref, currentDir)
	if err != nil {
		return fmt.Errorf("failed to resolve !include %q: %w", ref, err)
	}

	if err := r.validatePath(resolved); err != nil {
		return fmt.Errorf("!include path %q is not allowed: %w", ref, err)
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("failed to read included file %q: %w", resolved, err)
	}

	// Determine if this is a YAML file or raw content.
	ext := strings.ToLower(filepath.Ext(resolved))
	if ext == ".yaml" || ext == ".yml" {
		var included yaml.Node
		if err := yaml.Unmarshal(data, &included); err != nil {
			return fmt.Errorf("failed to parse included YAML %q: %w", resolved, err)
		}

		// Recursively resolve nested includes.
		includeDir := filepath.Dir(resolved)
		if err := r.walk(&included, includeDir, depth+1); err != nil {
			return err
		}

		// Replace the current node with the included content.
		if included.Kind == yaml.DocumentNode && len(included.Content) > 0 {
			*node = *included.Content[0]
		}
	} else {
		// Raw content: replace the node with a scalar string value.
		node.Tag = ""
		node.Kind = yaml.ScalarNode
		node.Value = string(data)
	}

	return nil
}

func (r *IncludeResolver) resolvePath(ref, currentDir string) (string, error) {
	switch {
	case strings.HasPrefix(ref, "@root/"):
		return filepath.Join(r.rootDir, ref[6:]), nil
	case strings.HasPrefix(ref, "@here/"):
		return filepath.Join(currentDir, ref[6:]), nil
	case filepath.IsAbs(ref):
		return "", fmt.Errorf("absolute paths are not allowed in !include")
	default:
		return filepath.Join(currentDir, ref), nil
	}
}

func (r *IncludeResolver) validatePath(resolved string) error {
	// Evaluate symlinks to prevent traversal attacks.
	realPath, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		// If file doesn't exist yet, validate the parent.
		realPath = resolved
	}

	absRoot, err := filepath.EvalSymlinks(r.rootDir)
	if err != nil {
		absRoot = r.rootDir
	}

	if !strings.HasPrefix(realPath, absRoot) {
		return fmt.Errorf("path escapes root directory")
	}

	return nil
}
