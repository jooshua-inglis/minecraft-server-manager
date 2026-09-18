package cliapp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/webapi"
)

const webShutdownTimeout = 5 * time.Second

func newWebCmd() *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Start the read-only web API (and, once built, the web UI)",
		Long: "Starts mcm's embedded HTTP server: JSON endpoints mirroring `mcm\n" +
			"list`/`status`/`logs`, plus SSE streams for live logs and stats.\n" +
			"Binds to localhost only by default — there's no write access or\n" +
			"authentication yet, so treat anything else as unsafe to expose.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			if host, _, splitErr := net.SplitHostPort(addr); splitErr == nil && !isLoopback(host) {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: binding to %q exposes mcm's API beyond localhost with no authentication yet — anyone who can reach it can see every server's status and logs\n", addr)
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			srv := &http.Server{Addr: addr, Handler: webapi.NewHandler(f)}

			serveErr := make(chan error, 1)
			go func() { serveErr <- srv.ListenAndServe() }()

			fmt.Fprintf(cmd.OutOrStdout(), "mcm web listening on http://%s\n", addr)

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
