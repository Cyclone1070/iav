package tools

import (
	"github.com/Cyclone1070/deployforme/internal/tools/models"
	"github.com/Cyclone1070/deployforme/internal/tools/services"
)

// NewWorkspaceContext returns a default workspace context with system implementations.
// The workspaceRoot is canonicalised (absolute and symlink-resolved).
// Each context gets its own checksum cache instance and file size limit.
func NewWorkspaceContext(workspaceRoot string) (*models.WorkspaceContext, error) {
	return NewWorkspaceContextWithOptions(workspaceRoot, models.DefaultMaxFileSize)
}

// NewWorkspaceContextWithOptions creates a workspace context with custom max file size.
func NewWorkspaceContextWithOptions(workspaceRoot string, maxFileSize int64) (*models.WorkspaceContext, error) {
	canonicalRoot, err := services.CanonicaliseRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}

	fs := services.NewOSFileSystem(maxFileSize)

	// Initialize gitignore service (handles missing .gitignore gracefully)
	gitignoreSvc, err := services.NewGitignoreService(canonicalRoot, fs)
	if err != nil {
		// Log warning but continue with no-op service
		// In production code, you might want to use a proper logger here
		// For now, we just ignore the error as missing .gitignore is common
		gitignoreSvc = &services.NoOpGitignoreService{}
	}

	return &models.WorkspaceContext{
		FS:               fs,
		BinaryDetector:   &services.SystemBinaryDetector{},
		ChecksumManager:  services.NewChecksumManager(),
		MaxFileSize:      maxFileSize,
		WorkspaceRoot:    canonicalRoot,
		GitignoreService: gitignoreSvc,
		CommandExecutor:  &services.OSCommandExecutor{},
		TodoStore:        NewInMemoryTodoStore(),
		DockerConfig: models.DockerConfig{
			CheckCommand: []string{"docker", "info"},
			StartCommand: []string{"docker", "desktop", "start"},
		},
	}, nil
}
