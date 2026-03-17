package main

import (
	"context"
	"os"

	"github.com/ariel-frischer/rentalot-cli/internal/version"
	"github.com/ariel-frischer/rentalot-cli/pkg/rentalotcli"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type contextKey string

const clientKey contextKey = "client"

var rootCmd = &cobra.Command{
	Use:     "rentalot-cli",
	Short:   "CLI tool for managing Rentalot rental properties, contacts, and workflows",
	Version: version.Version,
}

func init() {
	// Disable colors when not writing to a terminal.
	if fi, err := os.Stdout.Stat(); err == nil {
		if fi.Mode()&os.ModeCharDevice == 0 {
			color.NoColor = true
		}
	}

	var noColor bool
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if noColor {
			color.NoColor = true
		}
		cfg := rentalotcli.ConfigFromEnv()
		client := rentalotcli.NewClient(cfg)
		cmd.SetContext(context.WithValue(cmd.Context(), clientKey, client))
	}

	rootCmd.SetHelpFunc(colorizedHelp)

	rootCmd.AddCommand(versionCmd)
}

// clientFromContext retrieves the API client stored by PersistentPreRun.
func clientFromContext(cmd *cobra.Command) *rentalotcli.Client {
	v := cmd.Context().Value(clientKey)
	if v == nil {
		return nil
	}
	return v.(*rentalotcli.Client)
}
