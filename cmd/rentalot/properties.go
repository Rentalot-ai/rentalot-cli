package main

import (
	"fmt"

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
					str(m, "id"), str(m, "name"), str(m, "address"), str(m, "type"), str(m, "status"), str(m, "rent"),
				})
			}
			printTable([]string{"ID", "NAME", "ADDRESS", "TYPE", "STATUS", "RENT"}, rows)
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

func propertiesCreateCmd() *cobra.Command {
	var name, address, propType, status, rent string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a property",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			body := map[string]any{
				"name":    name,
				"address": address,
				"type":    propType,
				"status":  status,
				"rent":    rent,
			}
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
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "property name (required)")
	cmd.Flags().StringVar(&address, "address", "", "property address")
	cmd.Flags().StringVar(&propType, "type", "", "property type (apartment, house, condo)")
	cmd.Flags().StringVar(&status, "status", "", "status (available, rented, maintenance)")
	cmd.Flags().StringVar(&rent, "rent", "", "monthly rent amount")
	return cmd
}

func propertiesUpdateCmd() *cobra.Command {
	var name, address, propType, status, rent string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a property",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if name != "" {
				body["name"] = name
			}
			if address != "" {
				body["address"] = address
			}
			if propType != "" {
				body["type"] = propType
			}
			if status != "" {
				body["status"] = status
			}
			if rent != "" {
				body["rent"] = rent
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
	cmd.Flags().StringVar(&name, "name", "", "property name")
	cmd.Flags().StringVar(&address, "address", "", "property address")
	cmd.Flags().StringVar(&propType, "type", "", "property type")
	cmd.Flags().StringVar(&status, "status", "", "status")
	cmd.Flags().StringVar(&rent, "rent", "", "monthly rent")
	return cmd
}

func propertiesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a property",
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
