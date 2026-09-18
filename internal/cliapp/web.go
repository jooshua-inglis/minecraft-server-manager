package cliapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/config"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/webapi"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/webui"
)

const webShutdownTimeout = 5 * time.Second

func newWebCmd() *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Start the web dashboard and its API",
		Long: "Starts mcm's embedded HTTP server: the SvelteKit dashboard, and\n" +
			"under /api/ the JSON endpoints mirroring the CLI's commands plus\n" +
			"SSE streams for live logs and stats. Reads are open; every write\n" +
			"action (start/stop/console/mods/backup/...) needs the bearer token\n" +
			"printed below, which mcm generates once and remembers. Binds to\n" +
			"localhost only by default — treat anything else as unsafe to\n" +
			"expose without also locking down who can reach it.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8080", "address for the web server to listen on")

	return cmd
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
