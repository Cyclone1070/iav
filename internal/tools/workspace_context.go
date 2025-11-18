package tools

// WorkspaceContext bundles all dependencies for tool operations.
// Each context is independent and does not share state with other contexts.
type WorkspaceContext struct {
	FS               FileSystem
	BinaryDetector   BinaryDetector
	ChecksumComputer ChecksumComputer
	ChecksumCache    ChecksumStore
	MaxFileSize      int64
	WorkspaceRoot    string // canonical, symlink-resolved workspace root
}

// NewWorkspaceContext returns a default workspace context with system implementations.
// The workspaceRoot is canonicalised (absolute and symlink-resolved).
// Each context gets its own checksum cache instance and file size limit.
func NewWorkspaceContext(workspaceRoot string) (*WorkspaceContext, error) {
	return NewWorkspaceContextWithOptions(workspaceRoot, DefaultMaxFileSize)
}

// NewWorkspaceContextWithOptions creates a workspace context with custom max file size.
func NewWorkspaceContextWithOptions(workspaceRoot string, maxFileSize int64) (*WorkspaceContext, error) {
	canonicalRoot, err := CanonicaliseRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}

	return &WorkspaceContext{
		FS:               NewOSFileSystem(maxFileSize),
		BinaryDetector:   &SystemBinaryDetector{},
		ChecksumComputer: &SHA256Checksum{},
		ChecksumCache:    NewChecksumCache(),
		MaxFileSize:      maxFileSize,
		WorkspaceRoot:    canonicalRoot,
	}, nil
}

// NewWorkspaceContextWithDI creates a workspace context with dependency-injected root canonicaliser.
// This version is designed for testing with mocks and follows DI principles.
func NewWorkspaceContextWithDI(workspaceRoot string, canonicaliser RootCanonicaliser) (*WorkspaceContext, error) {
	return NewWorkspaceContextWithOptionsDI(workspaceRoot, DefaultMaxFileSize, canonicaliser)
}

// NewWorkspaceContextWithOptionsDI creates a workspace context with custom max file size and dependency-injected root canonicaliser.
// This version is designed for testing with mocks and follows DI principles.
func NewWorkspaceContextWithOptionsDI(workspaceRoot string, maxFileSize int64, canonicaliser RootCanonicaliser) (*WorkspaceContext, error) {
	canonicalRoot, err := canonicaliser.CanonicaliseRoot(workspaceRoot)
	if err != nil {
		return nil, err
	}

	return &WorkspaceContext{
		FS:               NewOSFileSystem(maxFileSize),
		BinaryDetector:   &SystemBinaryDetector{},
		ChecksumComputer: &SHA256Checksum{},
		ChecksumCache:    NewChecksumCache(),
		MaxFileSize:      maxFileSize,
		WorkspaceRoot:    canonicalRoot,
	}, nil
}
