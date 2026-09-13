package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var propertiesCmd = &cobra.Command{
	Use:   "properties",
	Short: "Manage rental properties",
}

func init() {
	rootCmd.AddCommand(propertiesCmd)
	propertiesCmd.AddCommand(
		propertiesListCmd(),
		propertiesGetCmd(),
		propertiesCreateCmd(),
		propertiesUpdateCmd(),
		propertiesDeleteCmd(),
	)
}

func propertiesListCmd() *cobra.Command {
	var f listFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List properties",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/properties", f.toQueryParams())
			if err != nil {
				return fmt.Errorf("listing properties: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("listing properties: %w", decodeAPIError(resp))
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
					str(m, "title"),
					str(m, "address"),
					str(m, "propertyType"),
					str(m, "status"),
					str(m, "monthlyRent"),
				})
			}
			printTable([]string{"ID", "TITLE", "ADDRESS", "PROPERTY TYPE", "STATUS", "MONTHLY RENT"}, rows)
			return nil
		},
	}
	addListFlags(cmd, &f)
	return cmd
}

func propertiesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a property by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/properties/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("getting property: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("getting property: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
}

type propertyFlags struct {
	title, legacyName string
	address           string
	city              string
	state             string
	zip               string
	propertyType      string
	legacyType        string
	status            string
	description       string
	monthlyRent       int
	legacyRent        int
	bedrooms          int
	bathrooms         float64
	isPublic          bool
}

func propertiesCreateCmd() *cobra.Command {
	var f propertyFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a property with canonical API fields",
		Long:  "Create a property with canonical v1 fields. Trial writes must remain private; omit --property-type for the development-only trial first job.",
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := buildPropertyBody(cmd, &f, true)
			if err != nil {
				return err
			}
			return sendPropertyCreate(cmd, body)
		},
	}
	addPropertyFlags(cmd, &f)
	return cmd
}

func sendPropertyCreate(cmd *cobra.Command, body map[string]any) error {
	client := clientFromContext(cmd)
	resp, err := client.Post(cmd.Context(), "/api/v1/properties", body)
	if err != nil {
		return fmt.Errorf("creating property: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("creating property: %w", decodeAPIError(resp))
	}
	return printRawJSON(resp.Body)
}

func propertiesUpdateCmd() *cobra.Command {
	var f propertyFlags
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a property with canonical API fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := buildPropertyBody(cmd, &f, false)
			if err != nil {
				return err
			}
			client := clientFromContext(cmd)
			resp, err := client.Patch(cmd.Context(), "/api/v1/properties/"+args[0], body)
			if err != nil {
				return fmt.Errorf("updating property: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("updating property: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	addPropertyFlags(cmd, &f)
	return cmd
}

func addPropertyFlags(cmd *cobra.Command, f *propertyFlags) {
	flags := cmd.Flags()
	flags.StringVar(&f.title, "title", "", "human-readable property title")
	flags.StringVar(&f.legacyName, "name", "", "deprecated alias for --title")
	_ = flags.MarkDeprecated("name", "use --title instead")
	flags.StringVar(&f.address, "address", "", "street address (required when creating)")
	flags.StringVar(&f.city, "city", "", "city")
	flags.StringVar(&f.state, "state", "", "state or province")
	flags.StringVar(&f.zip, "zip", "", "ZIP or postal code")
	flags.IntVar(&f.monthlyRent, "monthly-rent", 0, "monthly rent in whole dollars (required when creating)")
	flags.IntVar(&f.legacyRent, "rent", 0, "deprecated alias for --monthly-rent")
	_ = flags.MarkDeprecated("rent", "use --monthly-rent instead")
	flags.IntVar(&f.bedrooms, "bedrooms", 0, "number of bedrooms (required when creating)")
	flags.Float64Var(&f.bathrooms, "bathrooms", 0, "number of bathrooms (required when creating)")
	flags.StringVar(&f.propertyType, "property-type", "", "canonical property type: house, apartment, condo, townhouse, room (omit for trial-safe writes)")
	flags.StringVar(&f.legacyType, "type", "", "deprecated alias for --property-type")
	_ = flags.MarkDeprecated("type", "use --property-type instead")
	flags.StringVar(&f.status, "status", "", "status: active, rented, inactive, archived")
	flags.BoolVar(&f.isPublic, "is-public", false, "whether the property is publicly listed (trial writes must remain private)")
	flags.StringVar(&f.description, "description", "", "property description")
}

func buildPropertyBody(cmd *cobra.Command, f *propertyFlags, creating bool) (map[string]any, error) {
	title, titleSet, err := resolveStringAlias(cmd, "title", f.title, "name", f.legacyName)
	if err != nil {
		return nil, err
	}
	propertyType, propertyTypeSet, err := resolveStringAlias(cmd, "property-type", f.propertyType, "type", f.legacyType)
	if err != nil {
		return nil, err
	}
	monthlyRent, monthlyRentSet, err := resolveIntAlias(cmd, "monthly-rent", f.monthlyRent, "rent", f.legacyRent)
	if err != nil {
		return nil, err
	}
	if err := validatePropertyRequired(cmd, f, monthlyRentSet, monthlyRent, creating); err != nil {
		return nil, err
	}
	if err := validatePropertyEnums(propertyType, propertyTypeSet, f.status, cmd.Flags().Changed("status")); err != nil {
		return nil, err
	}

	body := make(map[string]any)
	if titleSet {
		body["title"] = strings.TrimSpace(title)
	}
	if cmd.Flags().Changed("address") {
		body["address"] = strings.TrimSpace(f.address)
	}
	if cmd.Flags().Changed("city") && strings.TrimSpace(f.city) != "" {
		body["city"] = strings.TrimSpace(f.city)
	}
	if cmd.Flags().Changed("state") && strings.TrimSpace(f.state) != "" {
		body["state"] = strings.TrimSpace(f.state)
	}
	if cmd.Flags().Changed("zip") && strings.TrimSpace(f.zip) != "" {
		body["zip"] = strings.TrimSpace(f.zip)
	}
	if monthlyRentSet {
		body["monthlyRent"] = monthlyRent
	}
	if cmd.Flags().Changed("bedrooms") {
		body["bedrooms"] = f.bedrooms
	}
	if cmd.Flags().Changed("bathrooms") {
		body["bathrooms"] = f.bathrooms
	}
	if propertyTypeSet {
		body["propertyType"] = strings.TrimSpace(propertyType)
	}
	if cmd.Flags().Changed("status") {
		body["status"] = strings.TrimSpace(f.status)
	}
	if cmd.Flags().Changed("is-public") {
		body["isPublic"] = f.isPublic
	}
	if cmd.Flags().Changed("description") && strings.TrimSpace(f.description) != "" {
		body["description"] = f.description
	}
	if !creating && len(body) == 0 {
		return nil, fmt.Errorf("at least one property field must be provided")
	}
	return body, nil
}

func validatePropertyRequired(cmd *cobra.Command, f *propertyFlags, monthlyRentSet bool, monthlyRent int, creating bool) error {
	if creating && (!cmd.Flags().Changed("address") || strings.TrimSpace(f.address) == "") {
		return fmt.Errorf("--address is required")
	}
	if creating && !monthlyRentSet {
		return fmt.Errorf("--monthly-rent is required")
	}
	if monthlyRentSet && monthlyRent <= 0 {
		return fmt.Errorf("--monthly-rent must be a positive integer")
	}
	if creating && !cmd.Flags().Changed("bedrooms") {
		return fmt.Errorf("--bedrooms is required")
	}
	if cmd.Flags().Changed("bedrooms") && f.bedrooms < 0 {
		return fmt.Errorf("--bedrooms must be non-negative")
	}
	if creating && !cmd.Flags().Changed("bathrooms") {
		return fmt.Errorf("--bathrooms is required")
	}
	if cmd.Flags().Changed("bathrooms") && f.bathrooms <= 0 {
		return fmt.Errorf("--bathrooms must be positive")
	}
	if cmd.Flags().Changed("address") && strings.TrimSpace(f.address) == "" {
		return fmt.Errorf("--address must not be empty")
	}
	return nil
}

func validatePropertyEnums(propertyType string, propertyTypeSet bool, status string, statusSet bool) error {
	if propertyTypeSet && !oneOf(propertyType, "house", "apartment", "condo", "townhouse", "room") {
		return fmt.Errorf("invalid property type %q; use house, apartment, condo, townhouse, or room", propertyType)
	}
	if statusSet && !oneOf(status, "active", "rented", "inactive", "archived") {
		return fmt.Errorf("invalid property status %q; use active, rented, inactive, or archived", status)
	}
	return nil
}

func oneOf(value string, options ...string) bool {
	value = strings.TrimSpace(value)
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

func propertiesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a property (soft-delete)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Delete(cmd.Context(), "/api/v1/properties/"+args[0])
			if err != nil {
				return fmt.Errorf("deleting property: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("deleting property: %w", decodeAPIError(resp))
			}
			success("Property %s deleted.", highlight(args[0]))
			return nil
		},
	}
}
