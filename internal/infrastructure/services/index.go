package services

import (
	"sort"

	"github.com/sophialabs/proteusmock/internal/domain/match"
)

// ScenarioIndex maps METHOD:path-pattern to sorted compiled scenarios.
type ScenarioIndex struct {
	entries map[string][]*match.CompiledScenario
	paths   []string
}

// NewScenarioIndex creates an empty index.
func NewScenarioIndex() *ScenarioIndex {
	return &ScenarioIndex{
		entries: make(map[string][]*match.CompiledScenario),
	}
}

// Add inserts a compiled scenario into the index.
func (idx *ScenarioIndex) Add(cs *match.CompiledScenario) {
	key := cs.PathKey
	idx.entries[key] = append(idx.entries[key], cs)
}

// Build sorts all entries by priority desc then ID asc, and collects unique paths.
func (idx *ScenarioIndex) Build() {
	idx.paths = nil
	seen := make(map[string]bool)

	for key, candidates := range idx.entries {
		sort.SliceStable(candidates, func(i, j int) bool {
			if candidates[i].Priority != candidates[j].Priority {
				return candidates[i].Priority > candidates[j].Priority
			}
			// More predicates = more specific = evaluated first.
			ci, cj := len(candidates[i].Predicates), len(candidates[j].Predicates)
			if ci != cj {
				return ci > cj
			}
			return candidates[i].ID < candidates[j].ID
		})
		idx.entries[key] = candidates

		// Extract path (strip METHOD: prefix).
		for _, cs := range candidates {
			path := cs.PathKey[len(cs.Method)+1:]
			if !seen[path] {
				seen[path] = true
				idx.paths = append(idx.paths, path)
			}
		}
	}

	sort.Strings(idx.paths)
}

// Lookup returns the sorted candidates for a given METHOD:path key.
func (idx *ScenarioIndex) Lookup(key string) []*match.CompiledScenario {
	return idx.entries[key]
}

// Paths returns all unique paths registered in the index.
func (idx *ScenarioIndex) Paths() []string {
	return idx.paths
}

// All returns all compiled scenarios across all keys.
func (idx *ScenarioIndex) All() []*match.CompiledScenario {
	size := 0
	for _, candidates := range idx.entries {
		size += len(candidates)
	}
	all := make([]*match.CompiledScenario, 0, size)
	for _, candidates := range idx.entries {
		all = append(all, candidates...)
	}
	return all
}

// Keys returns all index keys.
func (idx *ScenarioIndex) Keys() []string {
	keys := make([]string, 0, len(idx.entries))
	for k := range idx.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
