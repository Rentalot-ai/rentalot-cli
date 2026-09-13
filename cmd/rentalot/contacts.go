package main

import (
	"fmt"
	"strings"

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
					str(m, "id"),
					str(m, "name"),
					str(m, "email"),
					str(m, "phone"),
					str(m, "role"),
				})
			}
			printTable([]string{"ID", "NAME", "EMAIL", "PHONE", "ROLE"}, rows)
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

type contactFlags struct {
	name, email, phone string
	status, role       string
	legacyType         string
	channelPreference  string
	source             string
	referralSource     string
	language           string
	notes              string
}

func contactsCreateCmd() *cobra.Command {
	var f contactFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a contact with canonical API fields",
		Long:  "Create a contact with canonical v1 fields. Single creates require --name and at least one of --email or --phone.",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := buildContactBody(cmd, &f, true)
			if err != nil {
				return err
			}
			return sendContactCreate(cmd, body)
		},
	}
	addContactFlags(cmd, &f)
	return cmd
}

func sendContactCreate(cmd *cobra.Command, body map[string]any) error {
	client := clientFromContext(cmd)
	resp, err := client.Post(cmd.Context(), "/api/v1/contacts", body)
	if err != nil {
		return fmt.Errorf("creating contact: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("creating contact: %w", decodeAPIError(resp))
	}
	return printRawJSON(resp.Body)
}

func contactsUpdateCmd() *cobra.Command {
	var f contactFlags
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a contact with canonical API fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := buildContactBody(cmd, &f, false)
			if err != nil {
				return err
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
	addContactFlags(cmd, &f)
	return cmd
}

func addContactFlags(cmd *cobra.Command, f *contactFlags) {
	flags := cmd.Flags()
	flags.StringVar(&f.name, "name", "", "contact name (required when creating)")
	flags.StringVar(&f.email, "email", "", "email address")
	flags.StringVar(&f.phone, "phone", "", "phone number")
	flags.StringVar(&f.status, "status", "", "status: prospect, scheduled, applicant, renter, archived")
	flags.StringVar(&f.role, "role", "", "role: prospect, tenant, landlord, property_manager, vendor, other")
	flags.StringVar(&f.legacyType, "type", "", "deprecated alias for --role")
	_ = flags.MarkDeprecated("type", "use --role instead")
	flags.StringVar(&f.channelPreference, "channel-preference", "", "preferred messaging channel")
	flags.StringVar(&f.source, "source", "", "contact acquisition source")
	flags.StringVar(&f.referralSource, "referral-source", "", "referral source")
	flags.StringVar(&f.language, "language", "", "ISO 639-1 language code")
	flags.StringVar(&f.notes, "notes", "", "contact notes (update only)")
}

func buildContactBody(cmd *cobra.Command, f *contactFlags, creating bool) (map[string]any, error) {
	name := strings.TrimSpace(f.name)
	if creating && name == "" {
		return nil, fmt.Errorf("--name is required")
	}
	if creating && strings.TrimSpace(f.email) == "" && strings.TrimSpace(f.phone) == "" {
		return nil, fmt.Errorf("--email or --phone is required")
	}
	role, roleSet, err := resolveStringAlias(cmd, "role", f.role, "type", f.legacyType)
	if err != nil {
		return nil, err
	}
	if err := validateContactEnums(role, roleSet, f.status, cmd.Flags().Changed("status")); err != nil {
		return nil, err
	}

	body := make(map[string]any)
	if name != "" {
		body["name"] = name
	}
	addContactString(body, "email", f.email)
	addContactString(body, "phone", f.phone)
	if cmd.Flags().Changed("status") {
		body["status"] = strings.TrimSpace(f.status)
	}
	if roleSet {
		body["role"] = strings.TrimSpace(role)
	}
	addContactString(body, "channelPreference", f.channelPreference)
	addContactString(body, "source", f.source)
	addContactString(body, "referralSource", f.referralSource)
	addContactString(body, "language", f.language)
	if cmd.Flags().Changed("notes") {
		body["notes"] = f.notes
	}
	if !creating && len(body) == 0 {
		return nil, fmt.Errorf("at least one contact field must be provided")
	}
	return body, nil
}

func addContactString(body map[string]any, key, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		body[key] = trimmed
	}
}

func validateContactEnums(role string, roleSet bool, status string, statusSet bool) error {
	if roleSet && !oneOf(role, "prospect", "tenant", "landlord", "property_manager", "vendor", "other") {
		return fmt.Errorf("invalid contact role %q; use prospect, tenant, landlord, property_manager, vendor, or other", role)
	}
	if statusSet && !oneOf(status, "prospect", "scheduled", "applicant", "renter", "archived") {
		return fmt.Errorf("invalid contact status %q; use prospect, scheduled, applicant, renter, or archived", status)
	}
	return nil
}

func contactsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a contact (soft-delete)",
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
