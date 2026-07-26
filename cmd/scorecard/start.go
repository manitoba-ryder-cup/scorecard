package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/manitoba-ryder-cup/scorecard/internal/app"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

var startCmd = &cli.Command{
	Name:  "start",
	Usage: "Start the HTTP API server",
	Flags: []cli.Flag{
		HTTPAddressFlag,
		JWTPublicKeyFlag,
		EnvironmentFlag,
		TrustedProxyModeFlag,
		ProxySecretFlag,
		PublicTenantIDFlag,
	},
	Action: func(c *cli.Context) error {
		// Convert CLI config to app config
		appConfig := config.ToAppConfig()

		// Create server with our API handlers
		server, err := app.NewServer(c.Context, appConfig)
		if err != nil {
			return err
		}

		httpAddr := appConfig.HTTPAddress

		ctx, cancel := signal.NotifyContext(c.Context, os.Interrupt, syscall.SIGTERM)
		defer cancel()

		group, ctx := errgroup.WithContext(ctx)

		// Start server
		group.Go(func() error {
			slog.Info("Listening for connections", "http_address", httpAddr)
			return server.Start()
		})

		// Handle shutdown
		group.Go(func() error {
			<-ctx.Done()
			slog.Info("Shutting down gracefully")

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			return server.Shutdown(shutdownCtx)
		})

		return shutdownErr(group.Wait())
	},
}

// shutdownErr reports whether a finished run actually failed. ErrServerClosed and
// Canceled are the expected outcome of a SIGTERM, so returning them would exit non-zero
// on every ordinary redeploy or scale-down.
func shutdownErr(err error) error {
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
