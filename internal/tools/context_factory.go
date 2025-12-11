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

	canonicalRoot, err := services.CanonicaliseRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}

	fs := services.NewOSFileSystem()

	// Initialize gitignore service (handles missing .gitignore gracefully)
	gitignoreSvc, err := services.NewGitignoreService(canonicalRoot, fs)
	if err != nil {
		// MVP DEFERRAL: Intentionally silent fallback for now.
		// TODO(logging): Add slog.Warn("gitignore initialization failed", "error", err) when logging is set up.
		gitignoreSvc = &services.NoOpGitignoreService{}
	}

	return &models.WorkspaceContext{
		Config:          *cfg,
		FS:              fs,
		BinaryDetector:  &services.SystemBinaryDetector{SampleSize: cfg.Tools.BinaryDetectionSampleSize},
		ChecksumManager: services.NewChecksumManager(),

		WorkspaceRoot:    canonicalRoot,
		GitignoreService: gitignoreSvc,
		CommandExecutor:  &services.OSCommandExecutor{},

		TodoStore: NewInMemoryTodoStore(),
		DockerConfig: models.DockerConfig{
			CheckCommand: []string{"docker", "info"},
			// TODO(cross-platform): MacOS-specific Docker commands. Linux uses systemctl, Windows uses Start-Service.
			StartCommand: []string{"docker", "desktop", "start"},
		},
	}, nil
}
