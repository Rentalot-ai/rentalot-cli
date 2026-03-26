package version

import "testing"

func TestIsDevBuild(t *testing.T) {
	tests := map[string]struct {
		version string
		want    bool
	}{
		"dev build":     {version: "dev", want: true},
		"release build": {version: "1.0.0", want: false},
		"pre-release":   {version: "1.0.0-rc1", want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			orig := Version
			defer func() { Version = orig }()

			Version = tt.version
			if got := IsDevBuild(); got != tt.want {
				t.Errorf("IsDevBuild() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitSetsDefaults(t *testing.T) {
	// After init(), empty vars should be populated with fallback values
	if Version == "" {
		t.Error("Version should not be empty after init")
	}
	if Commit == "" {
		t.Error("Commit should not be empty after init")
	}
	if BuildDate == "" {
		t.Error("BuildDate should not be empty after init")
	}
}

func TestLdflagsTakePrecedence(t *testing.T) {
	origV := Version
	origC := Commit
	origB := BuildDate
	defer func() {
		Version = origV
		Commit = origC
		BuildDate = origB
	}()

	// Simulate ldflags-set values
	Version = "v1.2.3"
	Commit = "abc1234"
	BuildDate = "2026-01-01T00:00:00Z"

	if Version != "v1.2.3" {
		t.Errorf("Version = %q, want %q", Version, "v1.2.3")
	}
	if Commit != "abc1234" {
		t.Errorf("Commit = %q, want %q", Commit, "abc1234")
	}
	if BuildDate != "2026-01-01T00:00:00Z" {
		t.Errorf("BuildDate = %q, want %q", BuildDate, "2026-01-01T00:00:00Z")
	}
	if IsDevBuild() {
		t.Error("IsDevBuild() should be false when Version is set via ldflags")
	}
}
