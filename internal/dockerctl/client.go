// Package dockerctl wraps the parts of the Docker Engine API that mcm
// needs, so the fleet package can drive container lifecycles without
// spreading Docker SDK details across the CLI.
package dockerctl

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	ManagedLabel = "mcm.managed"
	NameLabel    = "mcm.server"
)

type Client struct {
	cli *client.Client
}

func New() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connecting to docker: %w", err)
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

// EnsureImage pulls ref if it isn't already present locally.
func (c *Client) EnsureImage(ctx context.Context, ref string) error {
	images, err := c.cli.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", ref)),
	})
	if err != nil {
		return fmt.Errorf("checking for image %s: %w", ref, err)
	}
	if len(images) > 0 {
		return nil
	}

	rc, err := c.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling image %s: %w", ref, err)
	}
	defer rc.Close()
	_, err = io.Copy(io.Discard, rc)
	return err
}

type CreateParams struct {
	ContainerName string
	Image         string
	Env           []string
	Labels        map[string]string
	HostPort      int
	ContainerPort int
	DataDir       string
}

// Create creates (but does not start) a container.
func (c *Client) Create(ctx context.Context, p CreateParams) (string, error) {
	containerPort, err := nat.NewPort("tcp", fmt.Sprintf("%d", p.ContainerPort))
	if err != nil {
		return "", err
	}

	cfg := &container.Config{
		Image:  p.Image,
		Env:    p.Env,
		Labels: p.Labels,
		ExposedPorts: nat.PortSet{
			containerPort: struct{}{},
		},
	}

	hostCfg := &container.HostConfig{
		Binds: []string{p.DataDir + ":/data"},
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{{HostPort: fmt.Sprintf("%d", p.HostPort)}},
		},
		RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyUnlessStopped},
	}

	resp, err := c.cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, p.ContainerName)
	if err != nil {
		return "", fmt.Errorf("creating container %s: %w", p.ContainerName, err)
	}
	return resp.ID, nil
}

func (c *Client) Start(ctx context.Context, name string) error {
	if err := c.cli.ContainerStart(ctx, name, container.StartOptions{}); err != nil {
		return fmt.Errorf("starting %s: %w", name, err)
	}
	return nil
}

func (c *Client) Stop(ctx context.Context, name string, timeout time.Duration) error {
	secs := int(timeout.Seconds())
	if err := c.cli.ContainerStop(ctx, name, container.StopOptions{Timeout: &secs}); err != nil {
		return fmt.Errorf("stopping %s: %w", name, err)
	}
	return nil
}

func (c *Client) Remove(ctx context.Context, name string, force bool) error {
	if err := c.cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: force}); err != nil {
		return fmt.Errorf("removing %s: %w", name, err)
	}
	return nil
}

// Inspect returns nil, nil if the container does not exist.
func (c *Client) Inspect(ctx context.Context, name string) (*container.InspectResponse, error) {
	info, err := c.cli.ContainerInspect(ctx, name)
	if client.IsErrNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspecting %s: %w", name, err)
	}
	return &info, nil
}

// ListManaged returns every container carrying mcm's "managed" label.
func (c *Client) ListManaged(ctx context.Context) ([]container.Summary, error) {
	summaries, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", ManagedLabel+"=true")),
	})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}
	return summaries, nil
}

// UsedHostPorts collects every host port currently bound by an
// mcm-managed container, so the fleet package can avoid picking a
// colliding port for a new server.
func (c *Client) UsedHostPorts(ctx context.Context) (map[int]bool, error) {
	summaries, err := c.ListManaged(ctx)
	if err != nil {
		return nil, err
	}
	used := map[int]bool{}
	for _, s := range summaries {
		for _, p := range s.Ports {
			if p.PublicPort != 0 {
				used[int(p.PublicPort)] = true
			}
		}
	}
	return used, nil
}
