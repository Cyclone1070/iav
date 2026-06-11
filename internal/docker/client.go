// Package docker wraps the Docker Moby SDK to talk to the host daemon for
// container builds, runs, executions, and cleanups.
package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Cyclone1070/iav/internal/domain"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	archive "github.com/moby/go-archive"
)

type DockerClient struct {
	cli *client.Client
}

func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	// Check connection by pinging the daemon
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("docker daemon not reachable: %w", err)
	}
	return &DockerClient{cli: cli}, nil
}

func (c *DockerClient) BuildImage(ctx context.Context, opts domain.BuildOptions, out io.Writer) error {
	tarOpts := &archive.TarOptions{}
	tar, err := archive.TarWithOptions(opts.ContextDir, tarOpts)
	if err != nil {
		return fmt.Errorf("failed to create tar of context: %w", err)
	}
	defer func() { _ = tar.Close() }()

	// If a custom Dockerfile path was specified relative to context or absolute, clean it
	dockerfilePath := opts.Dockerfile
	if filepath.IsAbs(dockerfilePath) {
		rel, err := filepath.Rel(opts.ContextDir, dockerfilePath)
		if err == nil {
			dockerfilePath = rel
		}
	}

	buildOpts := build.ImageBuildOptions{
		Tags:           opts.Tags,
		Dockerfile:     dockerfilePath,
		Labels:         opts.Labels,
		Remove:         true,
		ForceRemove:    true,
		SuppressOutput: false,
	}

	resp, err := c.cli.ImageBuild(ctx, tar, buildOpts)
	if err != nil {
		return fmt.Errorf("image build failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to read build response: %w", err)
	}

	return nil
}

func convertEnvMap(env map[string]string) []string {
	var list []string
	for k, v := range env {
		list = append(list, fmt.Sprintf("%s=%s", k, v))
	}
	return list
}

func (c *DockerClient) CreateAndStartContainer(ctx context.Context, opts domain.RunOptions) (string, error) {
	// First ensure image is pulled/exists
	_, err := c.cli.ImageInspect(ctx, opts.Image)
	if err != nil {
		// Try to pull the image
		reader, err := c.cli.ImageCreate(ctx, opts.Image, image.CreateOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to pull image %q: %w", opts.Image, err)
		}
		defer func() { _ = reader.Close() }()
		// Discard pull progress output
		_, _ = io.Copy(io.Discard, reader)
	}

	config := &container.Config{
		Image:  opts.Image,
		Cmd:    opts.Cmd,
		Env:    convertEnvMap(opts.Env),
		Labels: opts.Labels,
		User:   opts.User,
	}

	hostConfig := &container.HostConfig{
		Binds:       opts.Binds,
		NetworkMode: container.NetworkMode(opts.Network),
		Privileged:  opts.Privileged,
	}

	var networkingConfig *network.NetworkingConfig
	if opts.Network != "" && opts.Network != "default" && opts.Network != "bridge" && opts.Network != "host" && opts.Network != "none" {
		var aliases []string
		if opts.Name != "" {
			aliases = []string{opts.Name}
		}
		networkingConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				opts.Network: {
					Aliases: aliases,
				},
			},
		}
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, opts.Name)
	if err != nil {
		return "", fmt.Errorf("container create failed: %w", err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Clean up the created container if startup fails
		_ = c.StopAndRemoveContainer(ctx, resp.ID)
		return "", fmt.Errorf("container start failed: %w", err)
	}

	return resp.ID, nil
}

func (c *DockerClient) RunContainerInfo(ctx context.Context, opts domain.RunOptions) (stdout string, stderr string, exitCode int, err error) {
	containerID, err := c.CreateAndStartContainer(ctx, opts)
	if err != nil {
		return "", "", -1, err
	}
	defer func() {
		_ = c.StopAndRemoveContainer(ctx, containerID)
	}()

	// Wait for container to terminate
	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", "", -1, fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		exitCode = int(status.StatusCode)
	}

	// Fetch logs
	logsOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	}
	logsReader, err := c.cli.ContainerLogs(ctx, containerID, logsOpts)
	if err != nil {
		return "", "", exitCode, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer func() { _ = logsReader.Close() }()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, logsReader)

	return stdoutBuf.String(), stderrBuf.String(), exitCode, nil
}

func (c *DockerClient) ExecInContainer(ctx context.Context, containerID string, cmd []string, env map[string]string, user string) (stdout string, stderr string, exitCode int, err error) {
	execOpts := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		Env:          convertEnvMap(env),
		User:         user,
	}

	execResp, err := c.cli.ContainerExecCreate(ctx, containerID, execOpts)
	if err != nil {
		return "", "", -1, fmt.Errorf("container exec create failed: %w", err)
	}

	attachResp, err := c.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return "", "", -1, fmt.Errorf("container exec attach failed: %w", err)
	}
	defer attachResp.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, attachResp.Reader)

	inspectResp, err := c.cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return stdoutBuf.String(), stderrBuf.String(), -1, fmt.Errorf("container exec inspect failed: %w", err)
	}

	return stdoutBuf.String(), stderrBuf.String(), inspectResp.ExitCode, nil
}

func (c *DockerClient) InspectContainer(ctx context.Context, containerID string) (*domain.ContainerState, error) {
	json, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("container inspect failed: %w", err)
	}

	state := &domain.ContainerState{
		Running:  json.State.Running,
		ExitCode: json.State.ExitCode,
	}

	if json.State.Health != nil {
		state.Health = json.State.Health.Status
	}

	return state, nil
}

func (c *DockerClient) StopAndRemoveContainer(ctx context.Context, containerID string) error {
	stopOpts := container.StopOptions{}
	_ = c.cli.ContainerStop(ctx, containerID, stopOpts)

	removeOpts := container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}
	if err := c.cli.ContainerRemove(ctx, containerID, removeOpts); err != nil {
		return fmt.Errorf("container remove failed: %w", err)
	}

	return nil
}

func (c *DockerClient) RemoveImage(ctx context.Context, imageID string) error {
	opts := image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	}
	if _, err := c.cli.ImageRemove(ctx, imageID, opts); err != nil {
		return fmt.Errorf("image remove failed: %w", err)
	}
	return nil
}

func (c *DockerClient) CreateNetwork(ctx context.Context, name string, labels map[string]string) (string, error) {
	opts := network.CreateOptions{
		Driver: "bridge",
		Labels: labels,
	}
	resp, err := c.cli.NetworkCreate(ctx, name, opts)
	if err != nil {
		return "", fmt.Errorf("network create failed: %w", err)
	}
	return resp.ID, nil
}

func (c *DockerClient) RemoveNetwork(ctx context.Context, networkID string) error {
	if err := c.cli.NetworkRemove(ctx, networkID); err != nil {
		return fmt.Errorf("network remove failed: %w", err)
	}
	return nil
}

func (c *DockerClient) CleanupRun(ctx context.Context, runID string) error {
	c.cleanupContainers(ctx, runID)
	c.cleanupNetworks(ctx, runID)
	c.cleanupImages(ctx, runID)
	return nil
}

func (c *DockerClient) cleanupContainers(ctx context.Context, runID string) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err == nil {
		for _, ctr := range containers {
			if ctr.Labels["org.iav.run-id"] == runID {
				_ = c.StopAndRemoveContainer(ctx, ctr.ID)
			}
		}
	}
}

func (c *DockerClient) cleanupNetworks(ctx context.Context, runID string) {
	networks, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err == nil {
		for _, net := range networks {
			if net.Labels["org.iav.run-id"] == runID {
				_ = c.RemoveNetwork(ctx, net.ID)
			}
		}
	}
}

func (c *DockerClient) cleanupImages(ctx context.Context, runID string) {
	images, err := c.cli.ImageList(ctx, image.ListOptions{})
	if err == nil {
		for _, img := range images {
			if img.Labels["org.iav.run-id"] == runID {
				_ = c.RemoveImage(ctx, img.ID)
			}
		}
	}
}

func (c *DockerClient) PruneExpiredResources(ctx context.Context) error {
	now := time.Now()
	c.pruneExpiredContainers(ctx, now)
	c.pruneExpiredNetworks(ctx, now)
	c.pruneExpiredImages(ctx, now)
	return nil
}

func (c *DockerClient) pruneExpiredContainers(ctx context.Context, now time.Time) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return
	}
	for _, ctr := range containers {
		if ctr.Labels["org.iav.managed"] != "true" {
			continue
		}
		expStr, ok := ctr.Labels["org.iav.expiry"]
		if !ok {
			continue
		}
		exp, err := time.Parse(time.RFC3339, expStr)
		if err == nil && exp.Before(now) {
			_ = c.StopAndRemoveContainer(ctx, ctr.ID)
		}
	}
}

func (c *DockerClient) pruneExpiredNetworks(ctx context.Context, now time.Time) {
	networks, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return
	}
	for _, net := range networks {
		if net.Labels["org.iav.managed"] != "true" {
			continue
		}
		expStr, ok := net.Labels["org.iav.expiry"]
		if !ok {
			continue
		}
		exp, err := time.Parse(time.RFC3339, expStr)
		if err == nil && exp.Before(now) {
			_ = c.RemoveNetwork(ctx, net.ID)
		}
	}
}

func (c *DockerClient) pruneExpiredImages(ctx context.Context, now time.Time) {
	images, err := c.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return
	}
	for _, img := range images {
		if img.Labels["org.iav.managed"] != "true" {
			continue
		}
		expStr, ok := img.Labels["org.iav.expiry"]
		if !ok {
			continue
		}
		exp, err := time.Parse(time.RFC3339, expStr)
		if err == nil && exp.Before(now) {
			_ = c.RemoveImage(ctx, img.ID)
		}
	}
}

func (c *DockerClient) ComposeUp(ctx context.Context, composeFiles []string, runID string) error {
	args := []string{"compose", "-p", "iav-" + runID}
	for _, f := range composeFiles {
		args = append(args, "-f", f)
	}
	args = append(args, "up", "--build", "--exit-code-from", "sut")
	//nolint:gosec // G204: Subprocess command is parameterized by design
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up failed: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

func (c *DockerClient) ComposeDown(ctx context.Context, composeFiles []string, runID string) error {
	args := []string{"compose", "-p", "iav-" + runID}
	for _, f := range composeFiles {
		args = append(args, "-f", f)
	}
	args = append(args, "down", "-v")
	//nolint:gosec // G204: Subprocess command is parameterized by design
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose down failed: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

func (c *DockerClient) Close() error {
	return c.cli.Close()
}
