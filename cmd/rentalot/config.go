package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
	"github.com/spf13/cobra"
)

var configForce bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ~/.config/rentalot-cli/config.yaml",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a config file",
	RunE:  runConfigInit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show resolved configuration",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long: `Set a config value. Valid keys:
  api_key   Rentalot API key (Settings > API Keys in dashboard)
  base_url  API base URL (default: https://rentalot.ai)`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open config in $EDITOR",
	RunE:  runConfigEdit,
}

func init() {
	configInitCmd.Flags().BoolVar(&configForce, "force", false, "overwrite existing config")
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configEditCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(globalConfigFile); err == nil && !configForce {
		return fmt.Errorf("%s already exists (use --force to overwrite)", globalConfigFile)
	}
	content := "# rentalot-cli configuration\n\n" +
		"# api_key: ra_...\n" +
		"# base_url: https://rentalot.ai\n"
	if err := rentalotcli.SaveConfig(&rentalotcli.Config{}, globalConfigFile); err != nil {
		return err
	}
	if err := os.WriteFile(globalConfigFile, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	success("Created %s", fileRef(globalConfigFile))
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	file, err := rentalotcli.LoadConfig(globalConfigFile)
	if err != nil {
		return err
	}
	eff := file.Effective()

	apiKeySource := sourceLabel(file.APIKey != "")
	if os.Getenv("RENTALOT_API_KEY") != "" {
		apiKeySource = "env"
	}
	baseURLSource := sourceLabel(file.BaseURL != "")
	if os.Getenv("RENTALOT_BASE_URL") != "" {
		baseURLSource = "env"
	}
	if eff.BaseURL == "https://rentalot.ai" && file.BaseURL == "" && os.Getenv("RENTALOT_BASE_URL") == "" {
		baseURLSource = "default"
	}

	apiKeyDisplay := eff.APIKey
	if len(apiKeyDisplay) > 8 {
		apiKeyDisplay = apiKeyDisplay[:8] + "..."
	}

	printConfigRow("api_key", apiKeyDisplay, apiKeySource)
	printConfigRow("base_url", eff.BaseURL, baseURLSource)
	printConfigRow("config_file", globalConfigFile, "")
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	cfg, err := rentalotcli.LoadConfig(globalConfigFile)
	if err != nil {
		return err
	}

	switch key {
	case "api_key":
		cfg.APIKey = value
	case "base_url":
		cfg.BaseURL = value
	default:
		return fmt.Errorf("unknown key %q\nvalid keys: api_key, base_url", key)
	}

	if err := rentalotcli.SaveConfig(cfg, globalConfigFile); err != nil {
		return err
	}
	success("Set %s in %s", highlight(key), fileRef(globalConfigFile))
	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(globalConfigFile); os.IsNotExist(err) {
		warn("%s not found — run `rentalot-cli config init` first", fileRef(globalConfigFile))
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, globalConfigFile)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func printConfigRow(key, value, source string) {
	if source != "" {
		fmt.Printf("%-12s %-40s (%s)\n", highlight(key+":"), value, source)
	} else {
		fmt.Printf("%-12s %s\n", highlight(key+":"), value)
	}
}

func sourceLabel(isCustom bool) string {
	if isCustom {
		return "file"
	}
	return "default"
}
