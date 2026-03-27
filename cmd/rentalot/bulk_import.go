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
	Short: "Import properties from a CSV/JSON file",
	Long: `Import properties from a local CSV or JSON file.

Field names are flexible — Zillow, AppFolio, and other common aliases
are auto-normalized server-side.

Examples:
  rentalot-cli bulk-import --file properties.csv
  rentalot-cli bulk-import --file properties.json
  rentalot-cli bulk-import --file properties.csv --poll=false`,
	RunE: bulkImportRun,
}

func init() {
	rootCmd.AddCommand(bulkImportCmd)
	bulkImportCmd.Flags().String("file", "", "path to CSV or JSON file (required)")
	bulkImportCmd.Flags().Bool("poll", true, "poll job status until complete")
	_ = bulkImportCmd.MarkFlagRequired("file")
}

func bulkImportRun(cmd *cobra.Command, args []string) error {
	filePath, _ := cmd.Flags().GetString("file")
	poll, _ := cmd.Flags().GetBool("poll")

	records, err := loadImportFile(filePath)
	if err != nil {
		return fmt.Errorf("loading file: %w", err)
	}
	if len(records) == 0 {
		return fmt.Errorf("no records found in %s", filePath)
	}

	body := map[string]any{
		"properties": records,
	}

	client := clientFromContext(cmd)
	resp, err := client.Post(cmd.Context(), "/api/v1/properties/bulk", body)
	if err != nil {
		return fmt.Errorf("submitting import: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("submitting import: %w", decodeAPIError(resp))
	}

	var envelope struct {
		Data struct {
			JobID  string `json:"jobId"`
			Status string `json:"status"`
			Total  int    `json:"total"`
		} `json:"data"`
	}
	if err := decodeBody(resp.Body, &envelope); err != nil {
		return err
	}

	success("Import job %s submitted (%d properties).", highlight(envelope.Data.JobID), envelope.Data.Total)

	if !poll || envelope.Data.JobID == "" {
		return nil
	}

	return pollJobStatus(cmd, envelope.Data.JobID)
}

func pollJobStatus(cmd *cobra.Command, jobID string) error {
	client := clientFromContext(cmd)
	for {
		resp, err := client.Get(cmd.Context(), "/api/v1/properties/bulk/"+jobID, nil)
		if err != nil {
			return fmt.Errorf("polling job status: %w", err)
		}
		var envelope struct {
			Data struct {
				JobID   string `json:"jobId"`
				Status  string `json:"status"`
				Total   int    `json:"total"`
				Created int    `json:"created"`
				Failed  int    `json:"failed"`
				Errors  []struct {
					Row     int    `json:"row"`
					Field   string `json:"field"`
					Message string `json:"message"`
				} `json:"errors"`
			} `json:"data"`
		}
		if err := decodeBody(resp.Body, &envelope); err != nil {
			_ = resp.Body.Close()
			return err
		}
		_ = resp.Body.Close()

		job := envelope.Data
		switch job.Status {
		case "completed":
			success("Import complete: %d created, %d failed.", job.Created, job.Failed)
			return nil
		case "failed":
			msg := "import job failed"
			if len(job.Errors) > 0 {
				msg = fmt.Sprintf("import job failed: %s", job.Errors[0].Message)
			}
			return fmt.Errorf("%s", msg)
		default:
			fmt.Printf("  status: %s (%d/%d)\n", job.Status, job.Created, job.Total)
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
