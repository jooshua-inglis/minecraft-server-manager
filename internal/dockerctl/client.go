// Package dockerctl wraps the parts of the Docker Engine API that mcm
// needs, so the fleet package can drive container lifecycles without
// spreading Docker SDK details across the CLI.
package dockerctl

import (
	"context"
	"encoding/json"
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

// PortMapping binds a container port to a host port on a specific host
// interface — e.g. "0.0.0.0" to publish the game port for LAN/internet
// players, or "127.0.0.1" to keep an administrative port (RCON) reachable
// only from the Docker host itself.
type PortMapping struct {
	ContainerPort int
	HostPort      int
	HostIP        string
}

type CreateParams struct {
	ContainerName string
	Image         string
	Env           []string
	Labels        map[string]string
	Ports         []PortMapping
	DataDir       string
}

// Create creates (but does not start) a container.
func (c *Client) Create(ctx context.Context, p CreateParams) (string, error) {
	exposed := nat.PortSet{}
	bindings := nat.PortMap{}
	for _, m := range p.Ports {
		containerPort, err := nat.NewPort("tcp", fmt.Sprintf("%d", m.ContainerPort))
		if err != nil {
			return "", err
		}
		exposed[containerPort] = struct{}{}
		bindings[containerPort] = []nat.PortBinding{{HostIP: m.HostIP, HostPort: fmt.Sprintf("%d", m.HostPort)}}
	}

	cfg := &container.Config{
		Image:        p.Image,
		Env:          p.Env,
		Labels:       p.Labels,
		ExposedPorts: exposed,
	}

	hostCfg := &container.HostConfig{
		Binds:         []string{p.DataDir + ":/data"},
		PortBindings:  bindings,
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

// Logs returns the raw (still multiplexed stdout/stderr) log stream for
// a container. Callers demultiplex it with stdcopy.StdCopy.
func (c *Client) Logs(ctx context.Context, name string, follow bool, tail string) (io.ReadCloser, error) {
	rc, err := c.cli.ContainerLogs(ctx, name, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
	})
	if err != nil {
		return nil, fmt.Errorf("getting logs for %s: %w", name, err)
	}
	return rc, nil
}

// Stats is a snapshot of a running container's CPU and memory usage.
type Stats struct {
	CPUPercent    float64
	MemUsageBytes uint64
	MemLimitBytes uint64
}

// Stats takes a resource-usage snapshot of a running container. CPU
// percent needs two samples to mean anything (it's a delta over a time
// window), and the one-shot stats endpoint doesn't prime a real previous
// sample (its precpu_stats comes back zeroed, which would compare
// CPU-seconds-since-container-start against CPU-seconds-since-host-boot
// — nonsense). So this reads two consecutive frames from the streaming
// endpoint instead, the same way `docker stats` computes CPU% itself.
func (c *Client) Stats(ctx context.Context, name string) (*Stats, error) {
	reader, err := c.cli.ContainerStats(ctx, name, true)
	if err != nil {
		return nil, fmt.Errorf("getting stats for %s: %w", name, err)
	}
	defer reader.Body.Close()

	dec := json.NewDecoder(reader.Body)
	var prev, cur container.StatsResponse
	if err := dec.Decode(&prev); err != nil {
		return nil, fmt.Errorf("decoding stats for %s: %w", name, err)
	}
	if err := dec.Decode(&cur); err != nil {
		return nil, fmt.Errorf("decoding stats for %s: %w", name, err)
	}

	return &Stats{
		CPUPercent:    cpuPercent(prev, cur),
		MemUsageBytes: cur.MemoryStats.Usage,
		MemLimitBytes: cur.MemoryStats.Limit,
	}, nil
}

// cpuPercent applies the same delta-over-delta formula the Docker CLI
// uses for `docker stats`, between two consecutive stats frames.
func cpuPercent(prev, cur container.StatsResponse) float64 {
	cpuDelta := float64(cur.CPUStats.CPUUsage.TotalUsage) - float64(prev.CPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(cur.CPUStats.SystemUsage) - float64(prev.CPUStats.SystemUsage)
	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0
	}

	onlineCPUs := float64(cur.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(cur.CPUStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0 {
		onlineCPUs = 1
	}

	return (cpuDelta / systemDelta) * onlineCPUs * 100.0
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
