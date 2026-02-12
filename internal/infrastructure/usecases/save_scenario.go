package usecases

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/sophialabs/proteusmock/internal/domain/scenario"
	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
)

// SaveScenarioUseCase saves a scenario's YAML content to disk.
type SaveScenarioUseCase struct {
	repo   scenario.Repository
	logger ports.Logger
}

// NewSaveScenarioUseCase creates a new use case.
func NewSaveScenarioUseCase(repo scenario.Repository, logger ports.Logger) *SaveScenarioUseCase {
	return &SaveScenarioUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute saves the YAML content for a scenario identified by id.
// For existing scenarios, it updates the file in place.
// For new scenarios (id == ""), it creates a new file.
func (uc *SaveScenarioUseCase) Execute(ctx context.Context, id string, yamlContent []byte) error {
	// Validate YAML parses correctly.
	var check yaml.Node
	if err := yaml.Unmarshal(yamlContent, &check); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	if id == "" {
		// New scenario — create with empty source file to trigger new file creation.
		// Extract the ID from the YAML content.
		var raw struct {
			ID string `yaml:"id"`
		}
		if err := yaml.Unmarshal(yamlContent, &raw); err != nil || raw.ID == "" {
			return fmt.Errorf("new scenario YAML must contain an 'id' field")
		}

		s := &scenario.Scenario{ID: raw.ID}
		if err := uc.repo.SaveScenario(ctx, s, yamlContent); err != nil {
			return fmt.Errorf("failed to create scenario: %w", err)
		}
		uc.logger.Info("scenario created", "id", raw.ID)
		return nil
	}

	// Existing scenario — look up its source file info.
	existing, err := uc.repo.LoadByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find scenario %q: %w", id, err)
	}

	if err := uc.repo.SaveScenario(ctx, existing, yamlContent); err != nil {
		return fmt.Errorf("failed to save scenario %q: %w", id, err)
	}
	uc.logger.Info("scenario updated", "id", id)
	return nil
}
