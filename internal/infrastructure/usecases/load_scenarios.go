package usecases

import (
	"context"
	"fmt"

	"github.com/sophialabs/proteusmock/internal/domain/scenario"
	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
)

// LoadScenariosUseCase loads all scenarios, compiles them, and builds an index.
type LoadScenariosUseCase struct {
	repo          scenario.Repository
	compiler      *services.Compiler
	logger        ports.Logger
	defaultEngine string
}

// NewLoadScenariosUseCase creates a new use case.
func NewLoadScenariosUseCase(repo scenario.Repository, compiler *services.Compiler, logger ports.Logger) *LoadScenariosUseCase {
	return &LoadScenariosUseCase{
		repo:     repo,
		compiler: compiler,
		logger:   logger,
	}
}

// SetDefaultEngine sets the global default engine applied to scenarios without an explicit engine.
func (uc *LoadScenariosUseCase) SetDefaultEngine(engine string) {
	uc.defaultEngine = engine
}

// Execute loads, compiles, validates, and returns the built index.
func (uc *LoadScenariosUseCase) Execute(ctx context.Context) (*services.ScenarioIndex, error) {
	scenarios, err := uc.repo.LoadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load scenarios: %w", err)
	}

	uc.logger.Info("loaded scenarios from repository", "count", len(scenarios))

	// Apply global default engine where not overridden.
	if uc.defaultEngine != "" {
		for _, s := range scenarios {
			if s.Response.Engine == "" {
				s.Response.Engine = uc.defaultEngine
			}
		}
	}

	// Validate ID uniqueness.
	ids := make(map[string]bool, len(scenarios))
	for _, s := range scenarios {
		if ids[s.ID] {
			return nil, fmt.Errorf("duplicate scenario ID: %q", s.ID)
		}
		ids[s.ID] = true
	}

	// Compile and build index.
	index := services.NewScenarioIndex()
	var compileErrors []string

	for _, s := range scenarios {
		cs, err := uc.compiler.CompileScenario(s)
		if err != nil {
			compileErrors = append(compileErrors, err.Error())
			uc.logger.Warn("failed to compile scenario", "id", s.ID, "error", err)
			continue
		}
		index.Add(cs)
		uc.logger.Debug("compiled scenario", "id", cs.ID, "key", cs.PathKey)
	}

	if len(compileErrors) > 0 {
		uc.logger.Warn("some scenarios failed to compile", "errors", len(compileErrors))
	}

	index.Build()

	uc.logger.Info("scenario index built", "keys", len(index.Keys()), "paths", len(index.Paths()))

	return index, nil
}
