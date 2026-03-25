package main

import (
	"fmt"
	"strings"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
	"github.com/spf13/cobra"
)

// listFlags holds the common flags for list commands.
type listFlags struct {
	limit  int
	page   int
	filter string
}

// addListFlags registers --limit, --page, and --filter on a command.
func addListFlags(cmd *cobra.Command, f *listFlags) {
	cmd.Flags().IntVar(&f.limit, "limit", 0, "max results to return")
	cmd.Flags().IntVar(&f.page, "page", 0, "page number (1-based)")
	cmd.Flags().StringVar(&f.filter, "filter", "", "filter as key=value (e.g. status=active)")
}

// toQueryParams converts listFlags to QueryParams, skipping zero values.
func (f *listFlags) toQueryParams() rentalotcli.QueryParams {
	params := rentalotcli.QueryParams{}
	if f.limit > 0 {
		params["limit"] = fmt.Sprintf("%d", f.limit)
	}
	if f.page > 0 {
		params["page"] = fmt.Sprintf("%d", f.page)
	}
	if f.filter != "" {
		parts := strings.SplitN(f.filter, "=", 2)
		if len(parts) == 2 {
			params[parts[0]] = parts[1]
		} else {
			params["filter"] = f.filter
		}
	}
	return params
}
