/*
Copyright © 2025 NVIDIA Corporation
SPDX-License-Identifier: Apache-2.0
*/
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/logging"
)

const (
	name           = "eidos"
	versionDefault = "dev"
)

var (
	// overridden during build with ldflags
	version = versionDefault
	commit  = "unknown"
	date    = "unknown"
)

// Execute starts the CLI application.
// This is called by main.main().
func Execute() {
	cmd := &cli.Command{
		Name:                  name,
		Usage:                 "Cloud Native Stack CLI",
		Version:               fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
		EnableShellCompletion: true,
		HideHelpCommand:       true,
		Metadata: map[string]interface{}{
			"git-commit": commit,
			"build-date": date,
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "enable debug logging",
				Sources: cli.EnvVars("EIDOS_DEBUG"),
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			isDebug := c.Bool("debug")
			logLevel := "info"
			if isDebug {
				logLevel = "debug"
			}
			logging.SetDefaultStructuredLoggerWithLevel(name, version, logLevel)
			slog.Debug("starting",
				"name", name,
				"version", version,
				"commit", commit,
				"date", date,
				"logLevel", logLevel)
			return ctx, nil
		},
		Commands: []*cli.Command{
			snapshotCmd(),
			recipeCmd(),
		},
		ShellComplete: commandLister,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}

func commandLister(_ context.Context, cmd *cli.Command) {
	if cmd == nil || cmd.Root() == nil {
		return
	}
	for _, c := range cmd.Root().Commands {
		if c.Hidden {
			continue
		}
		fmt.Println(c.Name)
	}
}
