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
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/NVIDIA/cloud-native-stack/pkg/logging"
	"github.com/NVIDIA/cloud-native-stack/pkg/serializer"
)

const (
	name           = "cnsctl"
	versionDefault = "dev"
)

var (
	// overridden during build with ldflags
	version = versionDefault
	commit  = "unknown"
	date    = "unknown"

	outputFlag = &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   fmt.Sprintf("output destination: file path, ConfigMap URI (%snamespace/name), or stdout (default)", serializer.ConfigMapURIScheme),
	}

	formatFlag = &cli.StringFlag{
		Name:    "format",
		Aliases: []string{"t"},
		Value:   string(serializer.FormatYAML),
		Usage:   fmt.Sprintf("output format (%s)", strings.Join(serializer.SupportedFormats(), ", ")),
	}

	kubeconfigFlag = &cli.StringFlag{
		Name:    "kubeconfig",
		Aliases: []string{"k"},
		Usage:   "Path to kubeconfig file (overrides KUBECONFIG env and default ~/.kube/config)",
	}
)

// Execute starts the CLI application.
// This is called by main.main().
func Execute() {
	// Set version for ConfigMap labels
	serializer.SetVersion(version)

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
				Sources: cli.EnvVars("CNS_DEBUG"),
			},
			&cli.BoolFlag{
				Name:    "log-json",
				Usage:   "enable structured logging",
				Sources: cli.EnvVars("CNS_LOG_JSON"),
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			isDebug := c.Bool("debug")
			logLevel := "info"
			if isDebug {
				logLevel = "debug"
			}

			// Configure logger based on flags
			switch {
			case c.Bool("log-json"):
				logging.SetDefaultStructuredLoggerWithLevel(name, version, logLevel)
			case isDebug:
				// In debug mode, use text logger with full metadata
				logging.SetDefaultLoggerWithLevel(name, version, logLevel)
			default:
				// Default mode: use CLI logger for clean, user-friendly output
				logging.SetDefaultCLILogger(logLevel)
			}

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
			bundleCmd(),
			validateCmd(),
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
