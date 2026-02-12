package scenario

import (
	"context"
	"errors"
)

// ErrNotFound indicates a scenario was not found.
var ErrNotFound = errors.New("scenario not found")

// Repository is the port for loading and persisting scenarios.
type Repository interface {
	// LoadAll loads all scenarios from the configured root directory.
	LoadAll(ctx context.Context) ([]*Scenario, error)

	// LoadByID loads a single scenario by its unique ID.
	// Returns ErrNotFound if no scenario with the given ID exists.
	LoadByID(ctx context.Context, id string) (*Scenario, error)

	// SaveScenario writes scenario YAML content to disk.
	// If the scenario has a SourceFile, it updates the existing file.
	// If SourceFile is empty, it creates a new file.
	SaveScenario(ctx context.Context, s *Scenario, yamlContent []byte) error

	// DeleteScenario removes a scenario from its source file.
	// For single-scenario files, the file is deleted.
	// For multi-scenario files, the entry is removed from the sequence.
	DeleteScenario(ctx context.Context, sourceFile string, sourceIndex int) error

	// ReadSourceYAML reads the raw YAML content for a specific scenario
	// from its source file.
	ReadSourceYAML(ctx context.Context, s *Scenario) ([]byte, error)
}
