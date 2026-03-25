package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var conversationsCmd = &cobra.Command{
	Use:   "conversations",
	Short: "Browse contact conversations",
}

func init() {
	rootCmd.AddCommand(conversationsCmd)
	conversationsCmd.AddCommand(
		conversationsListCmd(),
		conversationsGetCmd(),
		conversationsSearchCmd(),
	)
}

func conversationsListCmd() *cobra.Command {
	var f listFlags
	var contactID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List conversations",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := f.toQueryParams()
			if contactID != "" {
				params["contact_id"] = contactID
			}
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/conversations", params)
			if err != nil {
				return fmt.Errorf("listing conversations: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("listing conversations: %w", decodeAPIError(resp))
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
					str(m, "id"), str(m, "contact_id"), str(m, "channel"),
					str(m, "status"), str(m, "last_message_at"),
				})
			}
			printTable([]string{"ID", "CONTACT", "CHANNEL", "STATUS", "LAST MESSAGE"}, rows)
			return nil
		},
	}
	addListFlags(cmd, &f)
	cmd.Flags().StringVar(&contactID, "contact-id", "", "filter by contact ID")
	return cmd
}

func conversationsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a conversation by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/conversations/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("getting conversation: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("getting conversation: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
}

func conversationsSearchCmd() *cobra.Command {
	var query string
	var f listFlags
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search conversations by message content",
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" {
				return fmt.Errorf("--query is required")
			}
			params := f.toQueryParams()
			params["q"] = query
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/conversations/search", params)
			if err != nil {
				return fmt.Errorf("searching conversations: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("searching conversations: %w", decodeAPIError(resp))
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
					str(m, "id"), str(m, "contact_id"), str(m, "channel"), str(m, "status"),
				})
			}
			printTable([]string{"ID", "CONTACT", "CHANNEL", "STATUS"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "search query (required)")
	addListFlags(cmd, &f)
	return cmd
}
