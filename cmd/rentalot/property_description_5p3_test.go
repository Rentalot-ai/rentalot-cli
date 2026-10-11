package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
	"github.com/spf13/cobra"
)

type propertyDescription5p3Fixture struct {
	Components struct {
		Schemas map[string]struct {
			Properties map[string]struct {
				Type        string
				Description string
			}
		}
		Examples map[string]struct {
			Value json.RawMessage
		}
	}
}

func loadPropertyDescription5p3Fixture(t *testing.T) propertyDescription5p3Fixture {
	t.Helper()
	data, err := os.ReadFile("testdata/property-description.openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture propertyDescription5p3Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ValidationEnvelope", "ValidationProblem"} {
		if len(fixture.Components.Examples[name].Value) == 0 {
			t.Fatalf("missing validation response fixture %s", name)
		}
	}
	return fixture
}

func propertyDescription5p3Command(creating bool) (cmd *cobra.Command, args []string) {
	if creating {
		return propertiesCreateCmd(), []string{"--address", "123 Main St", "--monthly-rent", "1800", "--bedrooms", "2", "--bathrooms", "1"}
	}
	return propertiesUpdateCmd(), []string{"p1"}
}

func TestPropertyDescription5p3Help(t *testing.T) {
	fixture := loadPropertyDescription5p3Fixture(t)
	for _, creating := range []bool{true, false} {
		cmd, _ := propertyDescription5p3Command(creating)
		schemaName := "UpdatePropertyRequest"
		if creating {
			schemaName = "CreatePropertyRequest"
		}
		t.Run(cmd.Name(), func(t *testing.T) {
			field := fixture.Components.Schemas[schemaName].Properties["description"]
			if field.Type != "string" || field.Description == "" {
				t.Fatalf("missing description-field contract in %s", schemaName)
			}
			if !strings.Contains(cmd.Long, field.Description) {
				t.Errorf("command help = %q, want contract rule %q", cmd.Long, field.Description)
			}
			cmd.SetHelpFunc(colorizedHelp)
			out, err := captureStdout(t, cmd.Help)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, field.Description) || !strings.Contains(out, "--description") {
				t.Errorf("rendered help = %q, want description flag and contact-info rule", out)
			}
			usage := cmd.Flags().Lookup("description").Usage
			for _, term := range []string{"no phone numbers", "email addresses", "website links", "listing contact form"} {
				if !strings.Contains(usage, term) {
					t.Errorf("description flag help = %q, missing %q", usage, term)
				}
			}
		})
	}
}

func TestPropertyDescription5p3ServerValidation(t *testing.T) {
	fixture := loadPropertyDescription5p3Fixture(t)
	for _, creating := range []bool{true, false} {
		for _, description := range []string{
			"Sunny apartment. Call (415) 555-0123 for a tour.",
			"Sunny apartment. Email leasing@example.com for details.",
			"Sunny apartment. View tours at https://example.com/listings/123.",
		} {
			for name, example := range fixture.Components.Examples {
				cmd, args := propertyDescription5p3Command(creating)
				t.Run(cmd.Name()+"/"+description+"/"+name, func(t *testing.T) {
					wantMethod, wantPath, wantContext := http.MethodPatch, "/api/v1/properties/p1", "updating property"
					if creating {
						wantMethod, wantPath, wantContext = http.MethodPost, "/api/v1/properties", "creating property"
					}
					requests := 0
					handler := func(w http.ResponseWriter, r *http.Request) {
						requests++
						if r.Method != wantMethod || r.URL.Path != wantPath {
							t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, wantMethod, wantPath)
						}
						var body map[string]any
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Error(err)
						}
						if body["description"] != description {
							t.Errorf("description = %v, want unchanged %q", body["description"], description)
						}
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write(example.Value)
					}
					_, err := executeCommand(t, handler, cmd, append(args, "--description", description)...)
					if err == nil {
						t.Fatal("expected server validation error")
					}
					var apiErr *rentalotcli.APIError
					if !errors.As(err, &apiErr) {
						t.Fatalf("error = %v, want wrapped APIError", err)
					}
					if apiErr.StatusCode != http.StatusBadRequest || apiErr.Retryable() {
						t.Errorf("API error = %+v, want terminal HTTP 400", apiErr)
					}
					wantCode := "validation_error"
					if name == "ValidationProblem" {
						wantCode = "urn:problem:validation"
					}
					if apiErr.Code != wantCode {
						t.Errorf("API error code = %q, want %q", apiErr.Code, wantCode)
					}
					wantMessage := fixture.Components.Schemas["CreatePropertyRequest"].Properties["description"].Description
					if apiErr.Message != wantMessage || !strings.Contains(err.Error(), wantContext) {
						t.Errorf("error = %v, want operation context and unchanged validation message %q", err, wantMessage)
					}
					if requests != 1 {
						t.Errorf("requests = %d, want one request without retries", requests)
					}
				})
			}
		}
	}
}

func TestPropertyDescription5p3AllowsOrdinaryNumbers(t *testing.T) {
	const description = "  Sunny 2-bedroom apartment at 123 Main St. 850 sq ft, built in 2020, $1,800/month.\nAvailable June 1.  "
	for _, creating := range []bool{true, false} {
		cmd, args := propertyDescription5p3Command(creating)
		t.Run(cmd.Name(), func(t *testing.T) {
			requests := 0
			handler := func(w http.ResponseWriter, r *http.Request) {
				requests++
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["description"] != description {
					t.Errorf("description = %v, want unchanged %q", body["description"], description)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "p1", "description": description}})
			}
			out, err := executeCommand(t, handler, cmd, append(args, "--description", description)...)
			if err != nil {
				t.Fatalf("ordinary property description rejected: %v", err)
			}
			if requests != 1 {
				t.Errorf("requests = %d, want 1", requests)
			}
			var response struct {
				Data struct{ Description string }
			}
			if err := json.Unmarshal([]byte(out), &response); err != nil {
				t.Fatal(err)
			}
			if response.Data.Description != description {
				t.Errorf("output description = %q, want %q", response.Data.Description, description)
			}
		})
	}
}
