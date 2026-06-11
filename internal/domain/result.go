package domain

type StageResult struct {
	StageName  string
	Success    bool
	DurationMs int64
	Stdout     string
	Stderr     string
}

type TestExecutionResult struct {
	Success  bool          `json:"success"`
	ExitCode int           `json:"exit_code"`
	Stages   []StageResult `json:"stages"`
	Summary  string        `json:"summary"`
}
