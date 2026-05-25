//go:build !linux

package runner

import (
	"context"
	"errors"
	"fmt"
)

// ErrUnsupportedPlatform is returned when sandbox operations are attempted on
// a non-Linux platform. gVisor (runsc) and the Docker SDK's named-pipe code
// require Linux.
var ErrUnsupportedPlatform = errors.New("sandbox runner requires Linux")

// stubDockerClient is a no-op implementation that allows compilation on
// non-Linux platforms while returning clear errors at runtime.
type stubDockerClient struct{}

// New creates a Runner with a stub Docker client on non-Linux platforms.
// The returned Runner will compile and can be wired into the application, but
// all container operations will return ErrUnsupportedPlatform at runtime.
func New(cfg SandboxConfig) (*Runner, error) {
	return &Runner{
		docker: &stubDockerClient{},
		cfg:    cfg,
	}, nil
}

func (s *stubDockerClient) createContainer(_ context.Context, _ SandboxConfig, name string) (string, error) {
	return "", fmt.Errorf("createContainer(%s): %w", name, ErrUnsupportedPlatform)
}

func (s *stubDockerClient) copyToContainer(_ context.Context, _, _, _ string) error {
	return ErrUnsupportedPlatform
}

func (s *stubDockerClient) startContainer(_ context.Context, _ string) error {
	return ErrUnsupportedPlatform
}

func (s *stubDockerClient) inspectContainerIP(_ context.Context, _, _ string) (string, error) {
	return "", ErrUnsupportedPlatform
}

func (s *stubDockerClient) stopContainer(_ context.Context, _ string, _ int) error {
	return ErrUnsupportedPlatform
}

func (s *stubDockerClient) removeContainer(_ context.Context, _ string) error {
	return ErrUnsupportedPlatform
}

// Verify interface satisfaction at compile time.
var _ dockerClient = (*stubDockerClient)(nil)
