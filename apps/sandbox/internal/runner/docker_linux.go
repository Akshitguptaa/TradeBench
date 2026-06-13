//go:build linux

package runner

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type realDockerClient struct {
	cli *client.Client
}

func New(cfg SandboxConfig) (*Runner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client init: %w", err)
	}
	return &Runner{
		docker: &realDockerClient{cli: cli},
		cfg:    cfg,
	}, nil
}

// creates a hardened container: all caps dropped, no privilege escalation, cgroup limits enforced
func (d *realDockerClient) createContainer(ctx context.Context, cfg SandboxConfig, containerName string) (string, error) {
	pidsLimit := cfg.PidsLimit

	resp, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Image: cfg.Image,
			Cmd:   []string{"/opt/binary"},
			ExposedPorts: nat.PortSet{
				"8080/tcp": {},
			},
			Labels: map[string]string{
				"tradebench.role": "sandbox",
			},
		},
		&container.HostConfig{
			Runtime: cfg.Runtime,
			Resources: container.Resources{
				CPUQuota:   cfg.CPUQuota,
				CPUPeriod:  cfg.CPUPeriod,
				Memory:     cfg.Memory,
				MemorySwap: cfg.MemorySwap,
				PidsLimit:  &pidsLimit,
			},
			NetworkMode: container.NetworkMode(cfg.Network),
			AutoRemove:  false,
			CapDrop:     []string{"ALL"},
			SecurityOpt: []string{"no-new-privileges"},
		},
		&network.NetworkingConfig{},
		nil,
		containerName,
	)
	if err != nil {
		return "", err
	}
	log.Printf("runner: created container %s (name=%s)", resp.ID[:12], containerName)
	return resp.ID, nil
}

// docker API requires a tar archive to copy files into a container
func (d *realDockerClient) copyToContainer(ctx context.Context, containerID, srcPath, dstDir string) error {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read binary %s: %w", srcPath, err)
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	hdr := &tar.Header{
		Name: "binary",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}

	return d.cli.CopyToContainer(ctx, containerID, dstDir, &buf, types.CopyToContainerOptions{})
}

func (d *realDockerClient) startContainer(ctx context.Context, containerID string) error {
	return d.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

// tries named network first, falls back to default bridge IP
func (d *realDockerClient) inspectContainerIP(ctx context.Context, containerID, networkName string) (string, error) {
	inspect, err := d.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}
	if nw, ok := inspect.NetworkSettings.Networks[networkName]; ok && nw.IPAddress != "" {
		return nw.IPAddress, nil
	}
	if inspect.NetworkSettings.IPAddress != "" {
		return inspect.NetworkSettings.IPAddress, nil
	}
	return "", fmt.Errorf("no IP found for container %s", containerID[:12])
}

func (d *realDockerClient) stopContainer(ctx context.Context, containerID string, timeoutSec int) error {
	stopOpts := container.StopOptions{Timeout: &timeoutSec}
	return d.cli.ContainerStop(ctx, containerID, stopOpts)
}

func (d *realDockerClient) removeContainer(ctx context.Context, containerID string) error {
	return d.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}

func (d *realDockerClient) Close() error {
	return d.cli.Close()
}

var _ dockerClient = (*realDockerClient)(nil)
var _ io.Closer = (*realDockerClient)(nil)
