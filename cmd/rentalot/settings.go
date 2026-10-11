package main

import (
	"fmt"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage account settings",
}

func init() {
	rootCmd.AddCommand(settingsCmd)
	settingsCmd.AddCommand(
		settingsGetCmd(),
		settingsUpdateCmd(),
	)
}

func settingsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get current settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/settings", nil)
			if err != nil {
				return fmt.Errorf("getting settings: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("getting settings: %w", decodeAPIError(resp))
			}
			if jsonOutput {
				return printRawJSON(resp.Body)
			}
			var raw any
			if err := decodeBody(resp.Body, &raw); err != nil {
				return err
			}
			// Flatten settings as key/value table.
			rows := [][]string{}
			if m, ok := raw.(map[string]any); ok {
				for k, v := range m {
					rows = append(rows, []string{k, fmt.Sprintf("%v", v)})
				}
			}
			printTable([]string{"SETTING", "VALUE"}, rows)
			return nil
		},
	}
}

func settingsUpdateCmd() *cobra.Command {
	var businessName string
	var followupEnabled string
	var followupDelay int
	var dailyLeasingBriefEnabled, dailyLeasingBriefNoActivityEnabled bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update account settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if cmd.Flags().Changed("business-name") {
				if utf8.RuneCountInString(businessName) > 100 {
					return fmt.Errorf("--business-name must be at most 100 characters")
				}
				body["businessName"] = businessName
			}
			if cmd.Flags().Changed("followup-enabled") {
				body["followup_enabled"] = followupEnabled == "true"
			}
			if cmd.Flags().Changed("followup-delay") {
				body["followup_delay_hours"] = followupDelay
			}
			emailPreferences := map[string]any{}
			if cmd.Flags().Changed("daily-leasing-brief-enabled") {
				emailPreferences["dailyLeasingBriefEnabled"] = dailyLeasingBriefEnabled
			}
			if cmd.Flags().Changed("daily-leasing-brief-no-activity-enabled") {
				emailPreferences["dailyLeasingBriefNoActivityEnabled"] = dailyLeasingBriefNoActivityEnabled
			}
			if len(emailPreferences) > 0 {
				body["emailPreferences"] = emailPreferences
			}
			if len(body) == 0 {
				return fmt.Errorf("no settings specified — use --business-name, --followup-enabled, --followup-delay, --daily-leasing-brief-enabled or --daily-leasing-brief-no-activity-enabled")
			}
			client := clientFromContext(cmd)
			resp, err := client.Patch(cmd.Context(), "/api/v1/settings", body)
			if err != nil {
				return fmt.Errorf("updating settings: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("updating settings: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&businessName, "business-name", "", "business name (max 100 characters; empty clears)")
	cmd.Flags().StringVar(&followupEnabled, "followup-enabled", "", "enable followup messages (true/false)")
	cmd.Flags().IntVar(&followupDelay, "followup-delay", 0, "followup delay in hours")
	cmd.Flags().BoolVar(&dailyLeasingBriefEnabled, "daily-leasing-brief-enabled", false, "enable daily leasing brief emails (use =false to disable)")
	cmd.Flags().BoolVar(&dailyLeasingBriefNoActivityEnabled, "daily-leasing-brief-no-activity-enabled", false, "enable daily leasing brief emails when there is no activity (use =false to disable)")
	return cmd
}
