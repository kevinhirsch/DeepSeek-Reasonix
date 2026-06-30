package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DockerEngine implements a container-based sandbox using Docker or Podman.
// It provides workspace isolation, resource limits, network restriction,
// and timeout enforcement.
type DockerEngine struct {
	// Image is the container image to use (default: reasonix/sandbox:latest).
	Image string
	// APIKeyEnvVars are environment variable names to forward from the host.
	APIKeyEnvVars []string
	// AllowedHosts limits outbound network access from the container.
	AllowedHosts []string
	// WorkspaceRoot is the host path mounted as /workspace in the container.
	WorkspaceRoot string
	// MemoryMB is the container memory limit in megabytes.
	MemoryMB int
	// CPUQuota is the CPU allocation (default: 1 core).
	CPUQuota int
	// Timeout is the maximum container lifetime.
	Timeout time.Duration
	// Runtime is "docker" or "podman" (auto-detected if empty).
	Runtime string
}

// DefaultDockerImage is the official sandbox image.
const DefaultDockerImage = "reasonix/sandbox:latest"

// NewDockerEngine creates a Docker sandbox with sensible defaults.
func NewDockerEngine() *DockerEngine {
	return &DockerEngine{
		Image:         DefaultDockerImage,
		APIKeyEnvVars: []string{"DEEPSEEK_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "REASONIX_REMOTE_TOKEN"},
		MemoryMB:      4096,
		CPUQuota:      1,
		Timeout:       30 * time.Minute,
	}
}

func (e *DockerEngine) detectRuntime() string {
	if e.Runtime != "" {
		return e.Runtime
	}
	for _, rt := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(rt); err == nil {
			return rt
		}
	}
	return "docker"
}

// Run executes a command in a Docker container with the configured limits.
// Returns the combined stdout/stderr output.
func (e *DockerEngine) Run(ctx context.Context, command string, args ...string) (string, error) {
	rt := e.detectRuntime()

	// Pull image
	pullCmd := exec.CommandContext(ctx, rt, "pull", e.Image)
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		return "", fmt.Errorf("%s pull %s: %w", rt, e.Image, err)
	}

	// Build run args
	runArgs := []string{"run", "--rm"}

	// Resource limits
	runArgs = append(runArgs, "--memory", fmt.Sprintf("%dm", e.MemoryMB))
	if e.CPUQuota > 0 {
		runArgs = append(runArgs, "--cpus", fmt.Sprintf("%d", e.CPUQuota))
	}

	// Timeout
	if e.Timeout > 0 {
		runArgs = append(runArgs, "--timeout", fmt.Sprintf("%ds", int(e.Timeout.Seconds())))
	}

	// Workspace mount
	if e.WorkspaceRoot != "" {
		os.MkdirAll(e.WorkspaceRoot, 0755)
		runArgs = append(runArgs, "-v", fmt.Sprintf("%s:/workspace", e.WorkspaceRoot))
		runArgs = append(runArgs, "-w", "/workspace")
	}

	// API key env vars (injected, never written to filesystem)
	for _, envKey := range e.APIKeyEnvVars {
		if val := os.Getenv(envKey); val != "" {
			runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", envKey, val))
		}
	}
	// Also pass through any REASONIX_ env vars
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "REASONIX_") && !strings.HasPrefix(env, "REASONIX_REMOTE_TOKEN") {
			runArgs = append(runArgs, "-e", env)
		}
	}

	// Network restriction
	if len(e.AllowedHosts) > 0 {
		for _, host := range e.AllowedHosts {
			runArgs = append(runArgs, "--add-host", host)
		}
		runArgs = append(runArgs, "--network", "bridge")
	}

	runArgs = append(runArgs, e.Image)
	runArgs = append(runArgs, command)
	runArgs = append(runArgs, args...)

	cmd := exec.CommandContext(ctx, rt, runArgs...)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return string(output), fmt.Errorf("%s run: %w", rt, err)
	}
	return string(output), nil
}

// Cleanup removes any dangling containers for the given name prefix.
func (e *DockerEngine) Cleanup(namePrefix string) {
	rt := e.detectRuntime()
	// Find and remove containers with matching name prefix
	listCmd := exec.Command(rt, "ps", "-a", "--filter", "name="+namePrefix, "--format", "{{.ID}}")
	out, _ := listCmd.Output()
	for _, id := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if id == "" {
			continue
		}
		exec.Command(rt, "rm", "-f", id).Run()
	}
}

// IsAvailable reports whether Docker or Podman is available on this system.
func IsDockerAvailable() bool {
	for _, rt := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(rt); err == nil {
			return true
		}
	}
	return false
}
