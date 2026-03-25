package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage contacts (tenants, owners, prospects)",
}

func init() {
	rootCmd.AddCommand(contactsCmd)
	contactsCmd.AddCommand(
		contactsListCmd(),
		contactsGetCmd(),
		contactsCreateCmd(),
		contactsUpdateCmd(),
		contactsDeleteCmd(),
	)
}

func contactsListCmd() *cobra.Command {
	var f listFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List contacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/contacts", f.toQueryParams())
			if err != nil {
				return fmt.Errorf("listing contacts: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("listing contacts: %w", decodeAPIError(resp))
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
					str(m, "id"), str(m, "name"), str(m, "email"), str(m, "phone"), str(m, "type"),
				})
			}
			printTable([]string{"ID", "NAME", "EMAIL", "PHONE", "TYPE"}, rows)
			return nil
		},
	}
	addListFlags(cmd, &f)
	return cmd
}

func contactsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a contact by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/contacts/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("getting contact: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("getting contact: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
}

func contactsCreateCmd() *cobra.Command {
	var name, email, phone, contactType string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a contact",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			client := clientFromContext(cmd)
			body := map[string]any{
				"name":  name,
				"email": email,
				"phone": phone,
				"type":  contactType,
			}
			resp, err := client.Post(cmd.Context(), "/api/v1/contacts", body)
			if err != nil {
				return fmt.Errorf("creating contact: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("creating contact: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "contact name (required)")
	cmd.Flags().StringVar(&email, "email", "", "email address")
	cmd.Flags().StringVar(&phone, "phone", "", "phone number")
	cmd.Flags().StringVar(&contactType, "type", "", "contact type (tenant, owner, prospect)")
	return cmd
}

func contactsUpdateCmd() *cobra.Command {
	var name, email, phone, contactType string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if name != "" {
				body["name"] = name
			}
			if email != "" {
				body["email"] = email
			}
			if phone != "" {
				body["phone"] = phone
			}
			if contactType != "" {
				body["type"] = contactType
			}
			client := clientFromContext(cmd)
			resp, err := client.Patch(cmd.Context(), "/api/v1/contacts/"+args[0], body)
			if err != nil {
				return fmt.Errorf("updating contact: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("updating contact: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "contact name")
	cmd.Flags().StringVar(&email, "email", "", "email address")
	cmd.Flags().StringVar(&phone, "phone", "", "phone number")
	cmd.Flags().StringVar(&contactType, "type", "", "contact type")
	return cmd
}

func contactsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Delete(cmd.Context(), "/api/v1/contacts/"+args[0])
			if err != nil {
				return fmt.Errorf("deleting contact: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("deleting contact: %w", decodeAPIError(resp))
			}
			success("Contact %s deleted.", highlight(args[0]))
			return nil
		},
	}
}
