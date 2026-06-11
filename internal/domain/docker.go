// Package domain defines pure data types shared across all internal packages, including Docker run options and container states.
package domain

type BuildOptions struct {
	Dockerfile string
	ContextDir string
	Tags       []string
	Labels     map[string]string
}

type RunOptions struct {
	Image      string
	Name       string
	Cmd        []string
	Env        map[string]string
	Binds      []string
	Network    string
	Labels     map[string]string
	Privileged bool
	User       string
}

type ContainerState struct {
	Running  bool
	Health   string
	ExitCode int
}
