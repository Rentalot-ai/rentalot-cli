// Package version holds rentalot-cli version information.
// Separate package to avoid import cycles.
package version

import "runtime/debug"

var (
	// Set via ldflags during build. Empty string means ldflags were not used.
	Version   string
	Commit    string
	BuildDate string
)

func init() {
	info, ok := debug.ReadBuildInfo()
	if ok {
		if Version == "" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
		if Commit == "" {
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					Commit = s.Value
				case "vcs.time":
					if BuildDate == "" {
						BuildDate = s.Value
					}
				}
			}
		}
	}
	if Version == "" {
		Version = "dev"
	}
	if Commit == "" {
		Commit = "unknown"
	}
	if BuildDate == "" {
		BuildDate = "unknown"
	}
}

func IsDevBuild() bool {
	return Version == "dev"
}
