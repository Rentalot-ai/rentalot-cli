package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workflowsCmd = &cobra.Command{
	Use:   "workflows",
	Short: "Manage automation workflows",
}

func init() {
	rootCmd.AddCommand(workflowsCmd)
	workflowsCmd.AddCommand(
		workflowsListCmd(),
		workflowsGetCmd(),
		workflowsCreateCmd(),
		workflowsUpdateCmd(),
		workflowsDeleteCmd(),
	)
}

func workflowsListCmd() *cobra.Command {
	var f listFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/workflows", f.toQueryParams())
			if err != nil {
				return fmt.Errorf("listing workflows: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("listing workflows: %w", decodeAPIError(resp))
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
					str(m, "id"), str(m, "name"), str(m, "type"), str(m, "trigger"), str(m, "status"),
				})
			}
			printTable([]string{"ID", "NAME", "TYPE", "TRIGGER", "STATUS"}, rows)
			return nil
		},
	}
	addListFlags(cmd, &f)
	return cmd
}

func workflowsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a workflow by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Get(cmd.Context(), "/api/v1/workflows/"+args[0], nil)
			if err != nil {
				return fmt.Errorf("getting workflow: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("getting workflow: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
}

func workflowsCreateCmd() *cobra.Command {
	var name, wfType, trigger, status string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			body := map[string]any{
				"name":    name,
				"type":    wfType,
				"trigger": trigger,
				"status":  status,
			}
			client := clientFromContext(cmd)
			resp, err := client.Post(cmd.Context(), "/api/v1/workflows", body)
			if err != nil {
				return fmt.Errorf("creating workflow: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("creating workflow: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "workflow name (required)")
	cmd.Flags().StringVar(&wfType, "type", "", "workflow type")
	cmd.Flags().StringVar(&trigger, "trigger", "", "trigger event")
	cmd.Flags().StringVar(&status, "status", "active", "status (active, inactive)")
	return cmd
}

func workflowsUpdateCmd() *cobra.Command {
	var name, wfType, trigger, status string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if name != "" {
				body["name"] = name
			}
			if wfType != "" {
				body["type"] = wfType
			}
			if trigger != "" {
				body["trigger"] = trigger
			}
			if status != "" {
				body["status"] = status
			}
			client := clientFromContext(cmd)
			resp, err := client.Patch(cmd.Context(), "/api/v1/workflows/"+args[0], body)
			if err != nil {
				return fmt.Errorf("updating workflow: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("updating workflow: %w", decodeAPIError(resp))
			}
			return printRawJSON(resp.Body)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "workflow name")
	cmd.Flags().StringVar(&wfType, "type", "", "workflow type")
	cmd.Flags().StringVar(&trigger, "trigger", "", "trigger event")
	cmd.Flags().StringVar(&status, "status", "", "status (active, inactive)")
	return cmd
}

func workflowsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := clientFromContext(cmd)
			resp, err := client.Delete(cmd.Context(), "/api/v1/workflows/"+args[0])
			if err != nil {
				return fmt.Errorf("deleting workflow: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode >= 400 {
				return fmt.Errorf("deleting workflow: %w", decodeAPIError(resp))
			}
			success("Workflow %s deleted.", highlight(args[0]))
			return nil
		},
	}
}
