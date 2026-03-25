package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var showingsCmd = &cobra.Command{
	Use:   "showings",
	Short: "Manage property showings",
}

func init() {
	rootCmd.AddCommand(showingsCmd)
	showingsCmd.AddCommand(
		showingsListCmd(),
		showingsGetCmd(),
		showingsCreateCmd(),
		showingsUpdateCmd(),
		showingsCancelCmd(),
		showingsCheckAvailabilityCmd(),
	)
}

func showingsListCmd() *cobra.Command {
	var f listFlags
	var propertyID, contactID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List showings",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := f.toQueryParams()
			if propertyID != "" {
				params["property_id"] = propertyID
			}
			if contactID != "" {
				params["contact_id"] = contactID
			}
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/showings", params)
			if err != nil {
				return fmt.Errorf("listing showings: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("listing showings: %w", decodeAPIError(resp))
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
					str(m, "id"), str(m, "property_id"), str(m, "contact_id"),
					str(m, "scheduled_at"), str(m, "status"),
				})
			}
			printTable([]string{"ID", "PROPERTY", "CONTACT", "SCHEDULED AT", "STATUS"}, rows)
			return nil
		},
	}
	addListFlags(cmd, &f)
	cmd.Flags().StringVar(&propertyID, "property-id", "", "filter by property ID")
	cmd.Flags().StringVar(&contactID, "contact-id", "", "filter by contact ID")
	return cmd
}

func showingsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a showing by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/showings/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("getting showing: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("getting showing: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
}

func showingsCreateCmd() *cobra.Command {
	var propertyID, contactID, scheduledAt, notes string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Schedule a showing",
		RunE: func(cmd *cobra.Command, args []string) error {
			if propertyID == "" {
				return fmt.Errorf("--property-id is required")
			}
			if contactID == "" {
				return fmt.Errorf("--contact-id is required")
			}
			if scheduledAt == "" {
				return fmt.Errorf("--scheduled-at is required")
			}
			body := map[string]any{
				"property_id":  propertyID,
				"contact_id":   contactID,
				"scheduled_at": scheduledAt,
				"notes":        notes,
			}
			client := clientFromContext(cmd)
			resp, err := client.Post(cmd.Context(), "/api/v1/showings", body)
			if err != nil {
				return fmt.Errorf("creating showing: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("creating showing: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&propertyID, "property-id", "", "property ID (required)")
	cmd.Flags().StringVar(&contactID, "contact-id", "", "contact ID (required)")
	cmd.Flags().StringVar(&scheduledAt, "scheduled-at", "", "ISO 8601 datetime (required)")
	cmd.Flags().StringVar(&notes, "notes", "", "optional notes")
	return cmd
}

func showingsUpdateCmd() *cobra.Command {
	var scheduledAt, status, notes string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a showing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if scheduledAt != "" {
				body["scheduled_at"] = scheduledAt
			}
			if status != "" {
				body["status"] = status
			}
			if notes != "" {
				body["notes"] = notes
			}
			client := clientFromContext(cmd)
			resp, err := client.Patch(cmd.Context(), "/api/v1/showings/"+args[0], body)
			if err != nil {
				return fmt.Errorf("updating showing: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("updating showing: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&scheduledAt, "scheduled-at", "", "new datetime (ISO 8601)")
	cmd.Flags().StringVar(&status, "status", "", "status (scheduled, completed, no_show)")
	cmd.Flags().StringVar(&notes, "notes", "", "notes")
	return cmd
}

func showingsCancelCmd() *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "cancel <id>",
		Short: "Cancel a showing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{"status": "cancelled", "reason": reason}
			client := clientFromContext(cmd)
			resp, err := client.Patch(cmd.Context(), "/api/v1/showings/"+args[0], body)
			if err != nil {
				return fmt.Errorf("cancelling showing: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("cancelling showing: %w", decodeAPIError(resp))
			}
			success("Showing %s cancelled.", highlight(args[0]))
			return nil
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "cancellation reason")
	return cmd
}

func showingsCheckAvailabilityCmd() *cobra.Command {
	var propertyID, date string
	cmd := &cobra.Command{
		Use:   "check-availability",
		Short: "Check available time slots for a property",
		RunE: func(cmd *cobra.Command, args []string) error {
			if propertyID == "" {
				return fmt.Errorf("--property-id is required")
			}
			if date == "" {
				return fmt.Errorf("--date is required (YYYY-MM-DD)")
			}
			params := map[string]string{"property_id": propertyID, "date": date}
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/showings/availability", params)
			if err != nil {
				return fmt.Errorf("checking availability: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("checking availability: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&propertyID, "property-id", "", "property ID (required)")
	cmd.Flags().StringVar(&date, "date", "", "date to check (YYYY-MM-DD, required)")
	return cmd
}
