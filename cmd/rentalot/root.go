package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Rentalot-ai/rentalot-cli/internal/version"
	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type contextKey string

const clientKey contextKey = "client"

var globalConfigFile string

var rootCmd = &cobra.Command{
	Use:     "rentalot",
	Short:   "CLI tool for managing Rentalot rental properties, contacts, and workflows",
	Version: version.Version,
}

func init() {
	// fatih/color automatically handles NO_COLOR env var, TTY detection,
	// and TERM=dumb at package init time. The --no-color flag below is
	// an explicit CLI override for users who want to force no-color.

	defaultCfgPath, _ := rentalotcli.ConfigPath()

	var noColor bool
	var timeout time.Duration
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().StringVar(&globalConfigFile, "config", defaultCfgPath, "config file")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", rentalotcli.DefaultTimeout, "HTTP request timeout, including response body (must be positive)")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if timeout <= 0 {
			return fmt.Errorf("--timeout must be positive")
		}
		if noColor {
			color.NoColor = true
		}
		fileCfg, _ := rentalotcli.LoadConfig(globalConfigFile)
		cfg := fileCfg.Effective()
		cfg.Timeout = timeout
		client := rentalotcli.NewClient(cfg)
		cmd.SetContext(context.WithValue(cmd.Context(), clientKey, client))
		return nil
	}

	rootCmd.SetHelpFunc(colorizedHelp)

	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

// clientFromContext retrieves the API client stored by PersistentPreRunE.
func clientFromContext(cmd *cobra.Command) *rentalotcli.Client {
	v := cmd.Context().Value(clientKey)
	if v == nil {
		return nil
	}
	return v.(*rentalotcli.Client)
}
