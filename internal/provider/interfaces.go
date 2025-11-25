package provider

import (
	"context"

	"github.com/Cyclone1070/deployforme/internal/orchestrator/models"
)

// Provider represents the interface to the Language Model
type Provider interface {
	Generate(ctx context.Context, prompt string, history []models.Message) (string, error)
}
