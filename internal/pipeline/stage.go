package pipeline

import (
	"context"

	"github.com/Cyclone1070/iav/internal/domain"
)

type Stage interface {
	Name() string
	Run(ctx context.Context, ec *domain.ExecutionContext) (*domain.StageResult, error)
	DependsOn() []string
}
