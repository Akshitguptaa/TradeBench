package runner

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type SandboxConfig struct {
	Image      string
	Runtime    string
	Network    string
	CPUQuota   int64
	CPUPeriod  int64
	Memory     int64
	MemorySwap int64
	PidsLimit  int64
}

type Runner struct {
	docker dockerClient
	cfg    SandboxConfig
}

// abstracts docker operations so we can stub it on non-linux
type dockerClient interface {
	createContainer(ctx context.Context, cfg SandboxConfig, containerName string) (string, error)
	copyToContainer(ctx context.Context, containerID, srcPath, dstDir string) error
	startContainer(ctx context.Context, containerID string) error
	inspectContainerIP(ctx context.Context, containerID, networkName string) (string, error)
	stopContainer(ctx context.Context, containerID string, timeoutSec int) error
	removeContainer(ctx context.Context, containerID string) error
}

// SpawnSandbox creates a container, copies the binary in, starts it, and waits
// until the process inside is healthy. If any step fails, it tears down whatever was created.
func (r *Runner) SpawnSandbox(ctx context.Context, submissionID, binaryPath string) (containerID, address string, err error) {
	containerName := fmt.Sprintf("sandbox-%s", submissionID)

	containerID, err = r.docker.createContainer(ctx, r.cfg, containerName)
	if err != nil {
		return "", "", fmt.Errorf("container create: %w", err)
	}

	if err := r.docker.copyToContainer(ctx, containerID, binaryPath, "/opt"); err != nil {
		_ = r.TerminateSandbox(context.Background(), containerID)
		return "", "", fmt.Errorf("copy binary: %w", err)
	}

	if err := r.docker.startContainer(ctx, containerID); err != nil {
		_ = r.TerminateSandbox(context.Background(), containerID)
		return "", "", fmt.Errorf("container start: %w", err)
	}

	ip, err := r.docker.inspectContainerIP(ctx, containerID, r.cfg.Network)
	if err != nil {
		_ = r.TerminateSandbox(context.Background(), containerID)
		return "", "", fmt.Errorf("get container IP: %w", err)
	}
	address = fmt.Sprintf("%s:8080", ip) // submissions always listen on 8080 inside the container

	if err := waitForHealthy(ctx, address, 30*time.Second); err != nil {
		_ = r.TerminateSandbox(context.Background(), containerID)
		return "", "", fmt.Errorf("health check: %w", err)
	}

	return containerID, address, nil
}

func (r *Runner) TerminateSandbox(ctx context.Context, containerID string) error {
	_ = r.docker.stopContainer(ctx, containerID, 10)
	return r.docker.removeContainer(ctx, containerID)
}

// polls /health every 200ms until the container responds or we hit the deadline
func waitForHealthy(ctx context.Context, address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	healthURL := fmt.Sprintf("http://%s/health", address)
	httpClient := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := httpClient.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}

	return fmt.Errorf("container at %s did not become healthy within %v", address, timeout)
}
