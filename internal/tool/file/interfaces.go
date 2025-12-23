package file

// pathResolver defines workspace path resolution operations.
type pathResolver interface {
	Abs(path string) (string, error)
	Rel(path string) (string, error)
}
