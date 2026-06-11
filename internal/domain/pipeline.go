package domain

type ExecutionContext struct {
	WorkspaceRoot        string
	TestScript           string
	Dockerfile           string
	ComposeFiles         []string
	TimeoutMs            int
	RunID                string
	EnvironmentVariables map[string]string
}
