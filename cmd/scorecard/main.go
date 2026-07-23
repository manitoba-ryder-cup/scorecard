package main

import (
	"log/slog"
	"os"

	"github.com/travisbale/knowhere/identity"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "scorecard",
		Usage: "Golf scorecard and tournament management API",
		Flags: []cli.Flag{
			DebugFlag,
			LogFormatFlag,
			DatabaseURLFlag,
		},
		Before: func(c *cli.Context) error {
			initLogger(config.LogFormat, config.Debug)
			return nil
		},
		Commands: []*cli.Command{
			startCmd,
			migrateCmd,
			seedTournamentCmd,
			versionCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}

// initLogger installs the default slog logger, wrapping the handler with
// identity.LogHandler so request-scoped fields (tenant, actor, request id, client IP)
// injected into context ride along with every *Context log call.
func initLogger(format string, debug bool) {
	var level slog.Level
	if debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(identity.LogHandler(handler)))
}
