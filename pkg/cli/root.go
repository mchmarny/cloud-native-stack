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
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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

	cfgFile  string
	logLevel string

	output string
	format string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   name,
	Short: "eidos - Cloud Native Stack CLI",
	Long: fmt.Sprintf(`eidos - Cloud Native Stack CLI

Version: %s
Commit:  %s
Built:   %s

Tooling to provide system optimization and verification capabilities: 

snapshot - captures system configuration snapshots including kernel modules,
           systemd services, GRUB parameters, and sysctl settings.`, version, commit, date),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	// Define command groups
	rootCmd.AddGroup(
		&cobra.Group{
			ID:    "functional",
			Title: "Functional Commands:",
		},
	)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.eidos.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)

		// Fail fast if user-specified config doesn't exist
		if err := viper.ReadInConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading config file %s: %v\n", cfgFile, err)
			os.Exit(1)
		}
		return
	}

	// Auto-discover config
	home, err := os.UserHomeDir()
	if err != nil {
		// Gracefully degrade if home directory not available
		return
	}

	// Search config in home directory and current directory
	viper.AddConfigPath(home)
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetConfigName(".eidos")

	// Automatic environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvPrefix("EIDOS")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// If a config file is found, read it in (optional)
	_ = viper.ReadInConfig()
}

// initLogger configures slog after Cobra parses flags/config so overrides like
// --log-level take effect before any command executes.
func initLogger() {
	logging.SetDefaultStructuredLoggerWithLevel(name, version, logLevel)
	slog.Info("starting",
		"name", name,
		"version", version,
		"commit", commit,
		"date", date,
		"logLevel", logLevel)
}
