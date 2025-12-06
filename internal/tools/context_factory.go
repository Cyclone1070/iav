package tools

import (
	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/tools/models"
	"github.com/Cyclone1070/iav/internal/tools/services"
)

// NewWorkspaceContext returns a default workspace context with system implementations.
// The workspaceRoot is canonicalised (absolute and symlink-resolved).
// Each context gets its own checksum cache instance and file size limit from config.
func NewWorkspaceContext(cfg *config.Config, workspaceRoot string) (*models.WorkspaceContext, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return NewWorkspaceContextWithOptions(cfg, workspaceRoot, cfg.Tools.MaxFileSize)
}

// NewWorkspaceContextWithOptions creates a workspace context with custom max file size.
func NewWorkspaceContextWithOptions(cfg *config.Config, workspaceRoot string, maxFileSize int64) (*models.WorkspaceContext, error) {
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
		Config:           cfg,
		FS:               fs,
		BinaryDetector:   &services.SystemBinaryDetector{SampleSize: cfg.Tools.BinaryDetectionSampleSize},
		ChecksumManager:  services.NewChecksumManager(),
		MaxFileSize:      maxFileSize,
		WorkspaceRoot:    canonicalRoot,
		GitignoreService: gitignoreSvc,
		CommandExecutor:  &services.OSCommandExecutor{MaxOutputSize: cfg.Tools.DefaultMaxCommandOutputSize},
		TodoStore:        NewInMemoryTodoStore(),
		DockerConfig: models.DockerConfig{
			CheckCommand: []string{"docker", "info"},
			// TODO(MVP): MacOS-specific. Linux uses systemd, Windows uses different command.
			StartCommand: []string{"docker", "desktop", "start"},
		},
	}, nil
}
