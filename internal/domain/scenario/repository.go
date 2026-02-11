package scenario

import "context"

// Repository is the port for loading scenarios from storage.
type Repository interface {
	// LoadAll loads all scenarios from the configured root directory.
	LoadAll(ctx context.Context) ([]*Scenario, error)
}
