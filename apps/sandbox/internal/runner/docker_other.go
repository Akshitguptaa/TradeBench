//go:build !linux

package runner

import (
	"context"
	"errors"
	"fmt"
)

var ErrUnsupportedPlatform = errors.New("sandbox runner requires Linux")

// stub so the project compiles on non-linux (macOS, CI, etc) — all operations return an error
type stubDockerClient struct{}

func New(cfg SandboxConfig) (*Runner, error) {
	return &Runner{
		docker: &stubDockerClient{},
		cfg:    cfg,
	}, nil
}

func (s *stubDockerClient) createContainer(_ context.Context, _ SandboxConfig, name, _ string) (string, error) {
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

var _ dockerClient = (*stubDockerClient)(nil)
