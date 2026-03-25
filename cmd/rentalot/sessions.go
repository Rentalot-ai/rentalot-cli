package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Manage AI-driven chat sessions",
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
	sessionsCmd.AddCommand(
		sessionsListCmd(),
		sessionsGetCmd(),
		sessionsReviewCmd(),
	)
}

func sessionsListCmd() *cobra.Command {
	var f listFlags
	var contactID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := f.toQueryParams()
			if contactID != "" {
				params["contact_id"] = contactID
			}
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/sessions", params)
			if err != nil {
				return fmt.Errorf("listing sessions: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("listing sessions: %w", decodeAPIError(resp))
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
					str(m, "id"), str(m, "contact_id"), str(m, "property_id"), str(m, "status"), str(m, "started_at"),
				})
			}
			printTable([]string{"ID", "CONTACT", "PROPERTY", "STATUS", "STARTED AT"}, rows)
			return nil
		},
	}
	addListFlags(cmd, &f)
	cmd.Flags().StringVar(&contactID, "contact-id", "", "filter by contact ID")
	return cmd
}

func sessionsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a session by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/sessions/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("getting session: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("getting session: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
}

func sessionsReviewCmd() *cobra.Command {
	var notes string
	var rating int
	cmd := &cobra.Command{
		Use:   "review <id>",
		Short: "Submit a review for a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{
				"notes":  notes,
				"rating": rating,
			}
			client := clientFromContext(cmd)
			resp, err := client.Post(cmd.Context(), "/api/v1/sessions/"+args[0]+"/review", body)
			if err != nil {
				return fmt.Errorf("reviewing session: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("reviewing session: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&notes, "notes", "", "review notes")
	cmd.Flags().IntVar(&rating, "rating", 0, "rating (1-5)")
	return cmd
}
