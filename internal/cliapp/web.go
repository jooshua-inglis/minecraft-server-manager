package cliapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/config"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/webapi"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/webui"
)

const (
	webShutdownTimeout  = 5 * time.Second
	webStopPollInterval = 100 * time.Millisecond
	webStopTimeout      = 10 * time.Second
	defaultWebAddr      = "127.0.0.1:8080"
)

func newWebCmd() *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Run the web dashboard and its API in the foreground",
		Long: "Starts mcm's embedded HTTP server: the SvelteKit dashboard, and\n" +
			"under /api/ the JSON endpoints mirroring the CLI's commands plus\n" +
			"SSE streams for live logs and stats. Reads are open; every write\n" +
			"action (start/stop/console/mods/backup/...) needs the bearer token\n" +
			"printed below, which mcm generates once and remembers. Binds to\n" +
			"localhost only by default — treat anything else as unsafe to\n" +
			"expose without also locking down who can reach it.\n\n" +
			"Runs in the foreground until interrupted — this is what a\n" +
			"container's entrypoint should run. For host/bare-metal use where\n" +
			"you want it backgrounded, see `mcm web start`/`mcm web stop`.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWebForeground(cmd, addr)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", defaultWebAddr, "address for the web server to listen on")

	cmd.AddCommand(newWebStartCmd(), newWebStopCmd(), newWebStatusCmd())
	return cmd
}

// runWebForeground is mcm web's actual implementation: build the
// fleet, ensure an auth token exists, and serve until ctx is canceled
// (SIGINT/SIGTERM) or the listener fails.
func runWebForeground(cmd *cobra.Command, addr string) error {
	f, err := newFleet()
	if err != nil {
		return err
	}
	defer f.Docker.Close()

	token, err := ensureWebToken()
	if err != nil {
		return fmt.Errorf("preparing web auth token: %w", err)
	}

	if host, _, splitErr := net.SplitHostPort(addr); splitErr == nil && !isLoopback(host) {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: binding to %q exposes mcm's dashboard beyond localhost — reads still need no auth, so anyone who can reach it can see every server's status and logs\n", addr)
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ui, err := webui.Handler()
	if err != nil {
		return fmt.Errorf("loading embedded web UI: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", webapi.NewHandler(f, token))
	mux.Handle("/", ui)

	srv := &http.Server{Addr: addr, Handler: mux}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ListenAndServe() }()

	fmt.Fprintf(cmd.OutOrStdout(), "mcm web listening on http://%s\n", addr)
	fmt.Fprintf(cmd.OutOrStdout(), "web token (paste into the dashboard to unlock write actions): %s\n", token)

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), webShutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ensureWebToken returns mcm's persisted web auth token, generating and
// saving one on first use so it's stable across `mcm web` restarts.
func ensureWebToken() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if cfg.WebToken != "" {
		return cfg.WebToken, nil
	}

	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	cfg.WebToken = hex.EncodeToString(b)

	if err := config.Save(cfg); err != nil {
		return "", err
	}
	return cfg.WebToken, nil
}

// webState is what `mcm web start` records about the background
// process it launched, so `stop`/`status` can find it again.
type webState struct {
	PID  int    `json:"pid"`
	Addr string `json:"addr"`
}

func webStateDir() (string, error) {
	cfgPath, err := config.Path()
	if err != nil {
		return "", err
	}
	return filepath.Dir(cfgPath), nil
}

func webStatePath() (string, error) {
	dir, err := webStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "web.pid.json"), nil
}

func webLogPath() (string, error) {
	dir, err := webStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "web.log"), nil
}

func readWebState() (*webState, error) {
	path, err := webStatePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var st webState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func writeWebState(st *webState) error {
	path, err := webStatePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func removeWebState() error {
	path, err := webStatePath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// processAlive reports whether pid identifies a live process this user
// can signal. Sending signal 0 doesn't actually deliver anything — the
// kernel just validates the pid/permissions.
func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func newWebStartCmd() *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the web dashboard in the background",
		Long: "Launches `mcm web` as a detached background process (re-execing\n" +
			"this same binary), so you get your shell back immediately — the\n" +
			"host/bare-metal equivalent of `mcm start <server>`. Use `mcm web\n" +
			"stop` to stop it and `mcm web status` to check on it. Inside a\n" +
			"container, run `mcm web` directly instead — let the container\n" +
			"runtime own the process lifecycle.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if st, err := readWebState(); err != nil {
				return err
			} else if st != nil && processAlive(st.PID) {
				return fmt.Errorf("mcm web is already running (pid %d, %s) — stop it first with `mcm web stop`", st.PID, st.Addr)
			}

			exe, err := os.Executable()
			if err != nil {
				return err
			}

			logPath, err := webLogPath()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
				return err
			}
			logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return fmt.Errorf("opening %s: %w", logPath, err)
			}
			defer logFile.Close()

			proc := exec.Command(exe, "web", "--addr", addr)
			proc.Stdout = logFile
			proc.Stderr = logFile
			proc.Stdin = nil
			// Detach from this process's session so it survives the
			// terminal/SSH session that ran `mcm web start` closing.
			proc.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

			if err := proc.Start(); err != nil {
				return fmt.Errorf("starting mcm web: %w", err)
			}
			// Release() invalidates the Process handle (on Unix it sets
			// Pid to -1), so grab the pid first.
			pid := proc.Process.Pid
			if err := proc.Process.Release(); err != nil {
				return err
			}

			if err := writeWebState(&webState{PID: pid, Addr: addr}); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "mcm web started (pid %d) on http://%s\n", pid, addr)
			fmt.Fprintf(cmd.OutOrStdout(), "logs: %s\n", logPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", defaultWebAddr, "address for the web server to listen on")
	return cmd
}

func newWebStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the background web dashboard started by `mcm web start`",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			st, err := readWebState()
			if err != nil {
				return err
			}
			if st == nil || !processAlive(st.PID) {
				fmt.Fprintln(out, "mcm web is not running")
				return removeWebState()
			}

			proc, err := os.FindProcess(st.PID)
			if err != nil {
				return err
			}
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				return fmt.Errorf("stopping pid %d: %w", st.PID, err)
			}

			deadline := time.Now().Add(webStopTimeout)
			for processAlive(st.PID) && time.Now().Before(deadline) {
				time.Sleep(webStopPollInterval)
			}
			if processAlive(st.PID) {
				_ = proc.Signal(syscall.SIGKILL)
			}

			if err := removeWebState(); err != nil {
				return err
			}
			fmt.Fprintf(out, "mcm web stopped (pid %d)\n", st.PID)
			return nil
		},
	}
}

func newWebStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether the background web dashboard is running",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			st, err := readWebState()
			if err != nil {
				return err
			}
			if st == nil || !processAlive(st.PID) {
				fmt.Fprintln(out, "mcm web is not running")
				return nil
			}
			fmt.Fprintf(out, "mcm web is running (pid %d) on http://%s\n", st.PID, st.Addr)
			return nil
		},
	}
}
