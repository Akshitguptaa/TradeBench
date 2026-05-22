// Package runner manages Docker container lifecycle for sandboxed submission
// execution. Containers run under gVisor (runsc) with strict cgroup limits.
//
// NOTE: The Docker SDK dependency (github.com/docker/docker) only compiles on
// Linux due to platform-specific named-pipe code. This package uses build
// constraints to keep the codebase cross-compilable. On non-Linux platforms a
// stub implementation is provided that returns an error.
package runner

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// SandboxConfig holds resource and runtime parameters for spawning containers.
type SandboxConfig struct {
	Image      string // base image name
	Runtime    string // OCI runtime (e.g. "runsc")
	Network    string // Docker network name
	CPUQuota   int64
	CPUPeriod  int64
	Memory     int64 // hard memory limit in bytes
	MemorySwap int64 // swap limit (set equal to Memory to disable swap)
	PidsLimit  int64
}

// Runner manages Docker container operations for sandboxes.
// On Linux, it wraps the real Docker SDK client.
// On other platforms, all operations return ErrUnsupportedPlatform.
type Runner struct {
	docker dockerClient
	cfg    SandboxConfig
}

// dockerClient is an internal interface abstracting the Docker SDK operations
// we need. The concrete implementation lives in docker_linux.go.
type dockerClient interface {
	createContainer(ctx context.Context, cfg SandboxConfig, containerName string) (string, error)
	copyToContainer(ctx context.Context, containerID, srcPath, dstDir string) error
	startContainer(ctx context.Context, containerID string) error
	inspectContainerIP(ctx context.Context, containerID, networkName string) (string, error)
	stopContainer(ctx context.Context, containerID string, timeoutSec int) error
	removeContainer(ctx context.Context, containerID string) error
}

// SpawnSandbox creates and starts a gVisor-sandboxed container with the
// contestant binary copied in. It waits for the container's health endpoint
// to respond before returning.
//
// Returns the container ID and the internal address (ip:8080) within the
// Docker network.
func (r *Runner) SpawnSandbox(ctx context.Context, submissionID, binaryPath string) (containerID, address string, err error) {
	containerName := fmt.Sprintf("sandbox-%s", submissionID)

	// --- Create container ---
	containerID, err = r.docker.createContainer(ctx, r.cfg, containerName)
	if err != nil {
		return "", "", fmt.Errorf("container create: %w", err)
	}

	// --- Copy binary into container ---
	if err := r.docker.copyToContainer(ctx, containerID, binaryPath, "/opt/submission"); err != nil {
		_ = r.TerminateSandbox(context.Background(), containerID)
		return "", "", fmt.Errorf("copy binary: %w", err)
	}

	// --- Start container ---
	if err := r.docker.startContainer(ctx, containerID); err != nil {
		_ = r.TerminateSandbox(context.Background(), containerID)
		return "", "", fmt.Errorf("container start: %w", err)
	}

	// --- Get container IP ---
	ip, err := r.docker.inspectContainerIP(ctx, containerID, r.cfg.Network)
	if err != nil {
		_ = r.TerminateSandbox(context.Background(), containerID)
		return "", "", fmt.Errorf("get container IP: %w", err)
	}
	address = fmt.Sprintf("%s:8080", ip)

	// --- Health-check polling ---
	if err := waitForHealthy(ctx, address, 30*time.Second); err != nil {
		_ = r.TerminateSandbox(context.Background(), containerID)
		return "", "", fmt.Errorf("health check: %w", err)
	}

	return containerID, address, nil
}

// TerminateSandbox stops and removes the container.
func (r *Runner) TerminateSandbox(ctx context.Context, containerID string) error {
	_ = r.docker.stopContainer(ctx, containerID, 10)
	return r.docker.removeContainer(ctx, containerID)
}

// waitForHealthy polls the container's /health endpoint until it returns 200
// or the timeout is reached.
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
