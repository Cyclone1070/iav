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

	return &models.WorkspaceContext{
		FS:              services.NewOSFileSystem(maxFileSize),
		BinaryDetector:  &services.SystemBinaryDetector{},
		ChecksumManager: services.NewChecksumManager(),
		MaxFileSize:     maxFileSize,
		WorkspaceRoot:   canonicalRoot,
		// CommandPolicy and DockerConfig are zero-valued by default
	}, nil
}
