package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Executor runs sub-agent tasks in Docker containers with resource limits
// and workspace isolation.
type Executor struct {
	config *Config
}

// NewExecutor creates a container executor.
func NewExecutor(cfg *Config) *Executor {
	return &Executor{config: cfg}
}

// RunTask executes a work item in a Docker container. Returns the combined
// stdout/stderr output.
func (e *Executor) RunTask(ctx context.Context, item *WorkItem) (string, error) {
	image := e.config.DockerImage
	if image == "" {
		image = "reasonix/sandbox:latest"
	}

	// Ensure image is pulled
	if err := e.pullImage(ctx, image); err != nil {
		return "", err
	}

	// Build docker run arguments
	args := []string{
		"run",
		"--rm",
		"--name", fmt.Sprintf("reasonix-worker-%s", item.ID),
		"--memory", fmt.Sprintf("%dm", e.config.MemoryLimitMB),
		"--cpus", fmt.Sprintf("%d", e.config.MaxConcurrent),
		"--timeout", fmt.Sprintf("%dm", e.config.TimeoutMinutes),
		"--network", "bridge",
	}

	// Workspace mount
	workspaceDir := fmt.Sprintf("/var/lib/reasonix/workspaces/%s", item.ID)
	os.MkdirAll(workspaceDir, 0755)
	args = append(args, "-v", fmt.Sprintf("%s:/workspace", workspaceDir))

	// API key env vars
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "REASONIX_") || strings.HasPrefix(env, "DEEPSEEK_") ||
			strings.HasPrefix(env, "ANTHROPIC_") || strings.HasPrefix(env, "OPENAI_") {
			args = append(args, "-e", env)
		}
	}

	args = append(args, image)
	args = append(args, "reasonix", "run", "--prompt", item.Prompt)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("docker run: %w\n%s", err, string(output))
	}
	return string(output), nil
}

func (e *Executor) pullImage(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, "docker", "pull", image)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker pull %s: %w\n%s", image, err, string(out))
	}
	return nil
}

// Cleanup removes any dangling Docker resources for a work item.
func (e *Executor) Cleanup(id string) {
	name := fmt.Sprintf("reasonix-worker-%s", id)
	exec.Command("docker", "rm", "-f", name).Run()
	os.RemoveAll(fmt.Sprintf("/var/lib/reasonix/workspaces/%s", id))
}
