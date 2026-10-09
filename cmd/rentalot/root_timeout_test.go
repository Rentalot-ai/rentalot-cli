package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestRootTimeoutFlag(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  string
	}{
		{value: "50ms"},
		{value: "0", want: "--timeout must be positive"},
		{value: "-1s", want: "--timeout must be positive"},
		{value: "invalid", want: "invalid argument"},
	} {
		t.Run(tc.value, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.want != "" {
					t.Error("invalid timeout must not send a request")
				}
				<-r.Context().Done()
			}))
			defer srv.Close()
			t.Setenv("RENTALOT_BASE_URL", srv.URL)
			t.Setenv("RENTALOT_API_KEY", "test-key")

			// Reuse the real root flags and pre-run hook without mutating its
			// command tree or the other command tests' contexts.
			cmd := &cobra.Command{
				Use:               "timeout-test",
				PersistentPreRunE: rootCmd.PersistentPreRunE,
				RunE: func(cmd *cobra.Command, args []string) error {
					resp, err := clientFromContext(cmd).Get(cmd.Context(), "/stalled", nil)
					if resp != nil {
						_ = resp.Body.Close()
					}
					return err
				},
			}
			for _, name := range []string{"config", "timeout"} {
				flag := rootCmd.PersistentFlags().Lookup(name)
				value, changed := flag.Value.String(), flag.Changed
				t.Cleanup(func() {
					if err := flag.Value.Set(value); err != nil {
						t.Errorf("restoring %s: %v", name, err)
					}
					flag.Changed = changed
				})
			}
			cmd.PersistentFlags().AddFlagSet(rootCmd.PersistentFlags())
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"--config", filepath.Join(t.TempDir(), "config.yaml"), "--timeout", tc.value})
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			cmd.SetContext(ctx)
			err := cmd.Execute()
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %v, want %q", err, tc.want)
				}
				return
			}
			if ctx.Err() != nil {
				t.Fatal("request reached caller backstop instead of --timeout")
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("error = %v, want deadline exceeded", err)
			}
		})
	}
}
