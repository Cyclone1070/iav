package models

// WorkspaceContext bundles all dependencies for tool operations.
// Each context is independent and does not share state with other contexts.
type WorkspaceContext struct {
	FS               FileSystem
	BinaryDetector   BinaryDetector
	ChecksumManager  ChecksumManager
	MaxFileSize      int64
	WorkspaceRoot    string           // canonical, symlink-resolved workspace root
	GitignoreService GitignoreService // optional, can be nil
	CommandExecutor  CommandExecutor  // optional, for executing external commands
	TodoStore        TodoStore        // optional, for managing todos

	// Shell Configuration
	CommandPolicy CommandPolicy
	DockerConfig  DockerConfig
}
