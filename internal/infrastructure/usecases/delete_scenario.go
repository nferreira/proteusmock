package usecases

import (
	"context"
	"fmt"

	"github.com/sophialabs/proteusmock/internal/domain/scenario"
	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
)

// DeleteScenarioUseCase removes a scenario from its source file.
type DeleteScenarioUseCase struct {
	repo   scenario.Repository
	logger ports.Logger
}

// NewDeleteScenarioUseCase creates a new use case.
func NewDeleteScenarioUseCase(repo scenario.Repository, logger ports.Logger) *DeleteScenarioUseCase {
	return &DeleteScenarioUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute removes the scenario with the given ID.
func (uc *DeleteScenarioUseCase) Execute(ctx context.Context, id string) error {
	existing, err := uc.repo.LoadByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find scenario %q: %w", id, err)
	}

	if err := uc.repo.DeleteScenario(ctx, existing.SourceFile, existing.SourceIndex); err != nil {
		return fmt.Errorf("failed to delete scenario %q: %w", id, err)
	}

	uc.logger.Info("scenario deleted", "id", id)
	return nil
}
