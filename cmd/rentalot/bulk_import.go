package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var bulkImportCmd = &cobra.Command{
	Use:   "bulk-import",
	Short: "Import properties or contacts from a CSV/JSON file",
	Long: `Import records from a local CSV or JSON file.

Examples:
  rentalot-cli bulk-import --file properties.csv --type properties
  rentalot-cli bulk-import --file contacts.json --type contacts`,
	RunE: bulkImportRun,
}

func init() {
	rootCmd.AddCommand(bulkImportCmd)
	bulkImportCmd.Flags().String("file", "", "path to CSV or JSON file (required)")
	bulkImportCmd.Flags().String("type", "properties", "record type: properties or contacts")
	bulkImportCmd.Flags().Bool("poll", true, "poll job status until complete")
	_ = bulkImportCmd.MarkFlagRequired("file")
}

func bulkImportRun(cmd *cobra.Command, args []string) error {
	filePath, _ := cmd.Flags().GetString("file")
	recordType, _ := cmd.Flags().GetString("type")
	poll, _ := cmd.Flags().GetBool("poll")

	records, err := loadImportFile(filePath)
	if err != nil {
		return fmt.Errorf("loading file: %w", err)
	}
	if len(records) == 0 {
		return fmt.Errorf("no records found in %s", filePath)
	}

	body := map[string]any{
		"type":    recordType,
		"records": records,
	}

	client := clientFromContext(cmd)
	resp, err := client.Post(cmd.Context(), "/api/v1/bulk-import", body)
	if err != nil {
		return fmt.Errorf("submitting import: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("submitting import: %w", decodeAPIError(resp))
	}

	var job struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := decodeBody(resp.Body, &job); err != nil {
		return err
	}

	success("Import job %s submitted (%d records).", highlight(job.ID), len(records))

	if !poll || job.ID == "" {
		return nil
	}

	return pollJobStatus(cmd, job.ID)
}

func pollJobStatus(cmd *cobra.Command, jobID string) error {
	client := clientFromContext(cmd)
	for {
		resp, err := client.Get(cmd.Context(), "/api/v1/bulk-import/"+jobID, nil)
		if err != nil {
			return fmt.Errorf("polling job status: %w", err)
		}
		var job struct {
			ID       string `json:"id"`
			Status   string `json:"status"`
			Total    int    `json:"total"`
			Imported int    `json:"imported"`
			Failed   int    `json:"failed"`
			Error    string `json:"error"`
		}
		if err := decodeBody(resp.Body, &job); err != nil {
			_ = resp.Body.Close()
			return err
		}
		_ = resp.Body.Close()

		switch job.Status {
		case "completed":
			success("Import complete: %d imported, %d failed.", job.Imported, job.Failed)
			return nil
		case "failed":
			return fmt.Errorf("import job failed: %s", job.Error)
		default:
			fmt.Printf("  status: %s (%d/%d)\n", job.Status, job.Imported, job.Total)
			time.Sleep(2 * time.Second)
		}
	}
}

// loadImportFile reads CSV or JSON (detected by extension) and returns records.
func loadImportFile(path string) ([]map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	if strings.HasSuffix(strings.ToLower(path), ".json") {
		var records []map[string]any
		if err := json.NewDecoder(f).Decode(&records); err != nil {
			return nil, fmt.Errorf("parsing JSON: %w", err)
		}
		return records, nil
	}

	// Default: CSV
	r := csv.NewReader(f)
	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV headers: %w", err)
	}

	var records []map[string]any
	for {
		row, err := r.Read()
		if err != nil {
			break
		}
		m := make(map[string]any, len(headers))
		for i, h := range headers {
			if i < len(row) {
				m[strings.TrimSpace(h)] = row[i]
			}
		}
		records = append(records, m)
	}
	return records, nil
}
