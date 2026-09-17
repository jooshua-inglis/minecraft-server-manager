// Package fleet is mcm's core service layer: it's what both the CLI
// commands and (eventually) the web server call into, so business logic
// lives in exactly one place.
package fleet

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/dockerctl"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/rcon"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

const (
	defaultImage         = "itzg/minecraft-server:latest"
	containerGamePort    = 25565
	containerRCONPort    = 25575
	defaultStartPort     = 25565
	defaultRCONStartPort = 25575
	defaultStopTimeout   = 60 * time.Second
	eulaURL              = "https://www.minecraft.net/en-us/eula"
)

type Fleet struct {
	Root   string
	Docker *dockerctl.Client
}

func New(root string, docker *dockerctl.Client) *Fleet {
	return &Fleet{Root: root, Docker: docker}
}

type CreateOptions struct {
	Type       string
	Version    string
	Memory     string
	Port       int
	AcceptEULA bool
}

// EULAError is returned when Create is asked to provision a server
// without the operator explicitly accepting Mojang's EULA.
type EULAError struct{}

func (EULAError) Error() string {
	return fmt.Sprintf("the Minecraft EULA must be accepted before a server can run: %s\nre-run with --accept-eula once you've read it", eulaURL)
}

func (f *Fleet) Create(ctx context.Context, name string, opts CreateOptions) (*serverstore.Metadata, error) {
	if err := serverstore.ValidateName(name); err != nil {
		return nil, err
	}
	if serverstore.Exists(f.Root, name) {
		return nil, fmt.Errorf("server %q already exists", name)
	}
	if !opts.AcceptEULA {
		return nil, EULAError{}
	}

	if opts.Type == "" {
		opts.Type = "VANILLA"
	}
	if opts.Version == "" {
		opts.Version = "LATEST"
	}
	if opts.Memory == "" {
		opts.Memory = "2G"
	}

	used, err := f.Docker.UsedHostPorts(ctx)
	if err != nil {
		return nil, err
	}

	port := opts.Port
	if port == 0 {
		port, err = selectPort(used, defaultStartPort, 1000, isPortFree)
		if err != nil {
			return nil, err
		}
	}
	used[port] = true

	rconPort, err := selectPort(used, defaultRCONStartPort, 1000, isPortFree)
	if err != nil {
		return nil, err
	}

	rconPassword, err := randomHex(16)
	if err != nil {
		return nil, err
	}

	meta := &serverstore.Metadata{
		Name:          name,
		Type:          opts.Type,
		Version:       opts.Version,
		Memory:        opts.Memory,
		Port:          port,
		RCONPort:      rconPort,
		ContainerName: serverstore.ContainerName(name),
		RCONPassword:  rconPassword,
		CreatedAt:     time.Now().UTC(),
	}

	if err := serverstore.Save(f.Root, meta); err != nil {
		return nil, err
	}

	if err := f.createContainer(ctx, meta); err != nil {
		return nil, err
	}

	return meta, nil
}

func (f *Fleet) createContainer(ctx context.Context, meta *serverstore.Metadata) error {
	if err := f.Docker.EnsureImage(ctx, defaultImage); err != nil {
		return err
	}

	// Create data/ ourselves, as the invoking host user, before Docker
	// ever sees the bind mount — otherwise the daemon auto-creates a
	// missing bind-mount source itself (as root), leaving the host user
	// unable to write into it directly (whitelist edits, backups, etc).
	if err := os.MkdirAll(serverstore.DataDir(f.Root, meta.Name), 0o755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	_, err := f.Docker.Create(ctx, dockerctl.CreateParams{
		ContainerName: meta.ContainerName,
		Image:         defaultImage,
		Env:           f.envFor(meta),
		Labels: map[string]string{
			dockerctl.ManagedLabel: "true",
			dockerctl.NameLabel:    meta.Name,
		},
		Ports: []dockerctl.PortMapping{
			// Published on every interface so LAN/internet players can connect.
			{ContainerPort: containerGamePort, HostPort: meta.Port, HostIP: "0.0.0.0"},
			// Loopback-only: RCON is an admin channel, not for public exposure.
			{ContainerPort: containerRCONPort, HostPort: meta.RCONPort, HostIP: "127.0.0.1"},
		},
		DataDir: serverstore.DataDir(f.Root, meta.Name),
	})
	return err
}

func (f *Fleet) envFor(meta *serverstore.Metadata) []string {
	return []string{
		"EULA=TRUE",
		"TYPE=" + meta.Type,
		"VERSION=" + meta.Version,
		"MEMORY=" + meta.Memory,
		"USE_AIKAR_FLAGS=true",
		"ENABLE_RCON=TRUE",
		"RCON_PASSWORD=" + meta.RCONPassword,
	}
}

func (f *Fleet) Start(ctx context.Context, name string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}
	return f.Docker.Start(ctx, meta.ContainerName)
}

func (f *Fleet) Stop(ctx context.Context, name string) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}
	return f.Docker.Stop(ctx, meta.ContainerName, defaultStopTimeout)
}

type DestroyOptions struct {
	Purge bool
}

func (f *Fleet) Destroy(ctx context.Context, name string, opts DestroyOptions) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}

	info, err := f.Docker.Inspect(ctx, meta.ContainerName)
	if err != nil {
		return err
	}
	if info != nil {
		if err := f.Docker.Remove(ctx, meta.ContainerName, true); err != nil {
			return err
		}
	}

	if opts.Purge {
		return serverstore.Remove(f.Root, name)
	}
	return nil
}

type ServerView struct {
	Name    string
	Type    string
	Version string
	Port    int
	Status  string
}

func (f *Fleet) List(ctx context.Context) ([]ServerView, error) {
	names, err := serverstore.List(f.Root)
	if err != nil {
		return nil, err
	}

	var views []ServerView
	for _, name := range names {
		meta, err := serverstore.Load(f.Root, name)
		if err != nil {
			return nil, err
		}
		status := "not created"
		info, err := f.Docker.Inspect(ctx, meta.ContainerName)
		if err != nil {
			return nil, err
		}
		if info != nil && info.State != nil {
			status = string(info.State.Status)
		}
		views = append(views, ServerView{
			Name:    meta.Name,
			Type:    meta.Type,
			Version: meta.Version,
			Port:    meta.Port,
			Status:  status,
		})
	}
	return views, nil
}

type Status struct {
	Metadata *serverstore.Metadata
	Info     *container.InspectResponse
	DataDir  string
}

func (f *Fleet) Status(ctx context.Context, name string) (*Status, error) {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return nil, err
	}
	info, err := f.Docker.Inspect(ctx, meta.ContainerName)
	if err != nil {
		return nil, err
	}
	return &Status{
		Metadata: meta,
		Info:     info,
		DataDir:  serverstore.DataDir(f.Root, name),
	}, nil
}

func (f *Fleet) Rename(ctx context.Context, oldName, newName string) error {
	if err := serverstore.ValidateName(newName); err != nil {
		return err
	}
	if serverstore.Exists(f.Root, newName) {
		return fmt.Errorf("server %q already exists", newName)
	}

	meta, err := serverstore.Load(f.Root, oldName)
	if err != nil {
		return err
	}

	wasRunning, err := f.removeContainerIfExists(ctx, meta)
	if err != nil {
		return err
	}

	if err := serverstore.Rename(f.Root, meta, newName); err != nil {
		return err
	}

	if err := f.createContainer(ctx, meta); err != nil {
		return err
	}
	if wasRunning {
		return f.Docker.Start(ctx, meta.ContainerName)
	}
	return nil
}

type EditOptions struct {
	Type    string
	Version string
}

// TypeOrVersionChanging reports whether opts would change the server's
// software or Minecraft version, so the CLI can require an explicit
// confirmation before a possibly plugin/mod-breaking change is applied.
func (opts EditOptions) TypeOrVersionChanging(meta *serverstore.Metadata) bool {
	return (opts.Type != "" && opts.Type != meta.Type) ||
		(opts.Version != "" && opts.Version != meta.Version)
}

func (f *Fleet) Edit(ctx context.Context, name string, opts EditOptions) error {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return err
	}

	if opts.Type != "" {
		meta.Type = opts.Type
	}
	if opts.Version != "" {
		meta.Version = opts.Version
	}

	wasRunning, err := f.removeContainerIfExists(ctx, meta)
	if err != nil {
		return err
	}

	if err := serverstore.Save(f.Root, meta); err != nil {
		return err
	}

	if err := f.createContainer(ctx, meta); err != nil {
		return err
	}
	if wasRunning {
		return f.Docker.Start(ctx, meta.ContainerName)
	}
	return nil
}

// Exec sends a single console command to a running server via RCON and
// returns its response text.
func (f *Fleet) Exec(ctx context.Context, name, command string) (string, error) {
	meta, err := serverstore.Load(f.Root, name)
	if err != nil {
		return "", err
	}

	info, err := f.Docker.Inspect(ctx, meta.ContainerName)
	if err != nil {
		return "", err
	}
	if info == nil || info.State == nil || !info.State.Running {
		return "", fmt.Errorf("server %q is not running", name)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", meta.RCONPort)
	client, err := rcon.Dial(addr, meta.RCONPassword)
	if err != nil {
		return "", err
	}
	defer client.Close()

	return client.Execute(command)
}

// isRunning reports whether meta's container currently exists and is
// running, so callers (whitelist/op/ban management, in particular) can
// choose between a live RCON command and a direct on-disk edit.
func (f *Fleet) isRunning(ctx context.Context, meta *serverstore.Metadata) (bool, error) {
	info, err := f.Docker.Inspect(ctx, meta.ContainerName)
	if err != nil {
		return false, err
	}
	return info != nil && info.State != nil && info.State.Running, nil
}

// removeContainerIfExists tears down meta's container (if any) ahead of a
// recreate, reporting whether it was running so the caller can restart it
// afterward.
func (f *Fleet) removeContainerIfExists(ctx context.Context, meta *serverstore.Metadata) (wasRunning bool, err error) {
	info, err := f.Docker.Inspect(ctx, meta.ContainerName)
	if err != nil {
		return false, err
	}
	if info == nil {
		return false, nil
	}
	if info.State != nil {
		wasRunning = info.State.Running
	}
	if err := f.Docker.Remove(ctx, meta.ContainerName, true); err != nil {
		return false, err
	}
	return wasRunning, nil
}

// selectPort finds the first port in [start, start+count) that's neither
// in used nor already bound on the host, per isFree. Split out from
// pickPort so it's testable without a Docker daemon.
func selectPort(used map[int]bool, start, count int, isFree func(int) bool) (int, error) {
	for p := start; p < start+count; p++ {
		if used[p] {
			continue
		}
		if isFree(p) {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free port found starting at %d", start)
}

func isPortFree(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
