package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var webhooksCmd = &cobra.Command{
	Use:   "webhooks",
	Short: "Manage webhook endpoints",
}

func init() {
	rootCmd.AddCommand(webhooksCmd)
	webhooksCmd.AddCommand(
		webhooksListCmd(),
		webhooksGetCmd(),
		webhooksCreateCmd(),
		webhooksUpdateCmd(),
		webhooksDeleteCmd(),
		webhooksTestCmd(),
	)
}

func webhooksListCmd() *cobra.Command {
	var f listFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/webhooks", f.toQueryParams())
			if err != nil {
				return fmt.Errorf("listing webhooks: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("listing webhooks: %w", decodeAPIError(resp))
			}
			if jsonOutput {
				return printRawJSON(resp.Body)
			}
			var raw any
			if err := decodeBody(resp.Body, &raw); err != nil {
				return err
			}
			rows := [][]string{}
			for _, m := range extractList(raw) {
				rows = append(rows, []string{
					str(m, "id"), str(m, "url"), str(m, "events"), str(m, "status"),
				})
			}
			printTable([]string{"ID", "URL", "EVENTS", "STATUS"}, rows)
			return nil
		},
	}
	addListFlags(cmd, &f)
	return cmd
}

func webhooksGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a webhook by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/webhooks/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("getting webhook: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("getting webhook: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
}

func webhooksCreateCmd() *cobra.Command {
	var webhookURL string
	var events []string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook",
		RunE: func(cmd *cobra.Command, args []string) error {
			if webhookURL == "" {
				return fmt.Errorf("--url is required")
			}
			body := map[string]any{
				"url":    webhookURL,
				"events": events,
			}
			client := clientFromContext(cmd)
			resp, err := client.Post(cmd.Context(), "/api/v1/webhooks", body)
			if err != nil {
				return fmt.Errorf("creating webhook: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("creating webhook: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&webhookURL, "url", "", "webhook endpoint URL (required)")
	cmd.Flags().StringSliceVar(&events, "events", nil,
		"comma-separated event types (e.g. contact.created,showing.scheduled)")
	return cmd
}

func webhooksUpdateCmd() *cobra.Command {
	var webhookURL, status string
	var events []string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if webhookURL != "" {
				body["url"] = webhookURL
			}
			if len(events) > 0 {
				body["events"] = events
			}
			if status != "" {
				body["status"] = status
			}
			client := clientFromContext(cmd)
			resp, err := client.Patch(cmd.Context(), "/api/v1/webhooks/"+args[0], body)
			if err != nil {
				return fmt.Errorf("updating webhook: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("updating webhook: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&webhookURL, "url", "", "webhook endpoint URL")
	cmd.Flags().StringSliceVar(&events, "events", nil, "event types")
	cmd.Flags().StringVar(&status, "status", "", "status (active, inactive)")
	return cmd
}

func webhooksDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Delete(cmd.Context(), "/api/v1/webhooks/"+args[0])
			if err != nil {
				return fmt.Errorf("deleting webhook: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("deleting webhook: %w", decodeAPIError(resp))
			}
			success("Webhook %s deleted.", highlight(args[0]))
			return nil
		},
	}
}

func webhooksTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test <id>",
		Short: "Send a test event to a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Post(cmd.Context(), "/api/v1/webhooks/"+args[0]+"/test",
				map[string]any{"event": "test"})
			if err != nil {
				return fmt.Errorf("testing webhook: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("testing webhook: %w", decodeAPIError(resp))
			}
			if jsonOutput {
				return printRawJSON(resp.Body)
			}
			success("Test event sent to webhook %s.", highlight(args[0]))
			return nil
		},
	}
}
