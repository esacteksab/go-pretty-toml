// SPDX-License-Identifier: MIT
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Variables set at build time using -ldflags
var (
	Version string
	Commit  string
	Date    string
	BuiltBy string
)

// GetVersionInfo builds the application version string including build details.
func GetVersionInfo() string {
	// Start with the Version set by ldflags, or provide a default
	result := Version
	if result == "" {
		result = "dev" // Default if version ldflag not set
	}

	// Append Commit if available
	commit := Commit
	if commit == "" {
		// Try reading from debug info as a fallback
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
					if len(commit) > 7 { //nolint:mnd
						commit = commit[:7]
					}
					break
				}
			}
		}
	}
	if commit != "" {
		result = fmt.Sprintf("%s\nCommit: %s", result, commit)
	}

	// Append Build Date if available
	date := Date
	if date != "" {
		result = fmt.Sprintf("%s\nBuilt at: %s", result, date)
	}

	// Append BuiltBy if available
	builtBy := BuiltBy
	if builtBy != "" {
		result = fmt.Sprintf("%s\nBuilt by: %s", result, builtBy)
	}

	// Append Go environment info
	result = fmt.Sprintf("%s\nGOOS: %s\nGOARCH: %s", result, runtime.GOOS, runtime.GOARCH)

	// Append Go module info if available
	if info, ok := debug.ReadBuildInfo(); ok {
		// info.Main contains info about the main module
		modVersion := info.Main.Version
		if modVersion != "" && modVersion != "(devel)" { // Only show if it's a real version
			result = fmt.Sprintf("%s\nModule Version: %s", result, modVersion)
		}
		if info.Main.Sum != "" {
			result = fmt.Sprintf("%s\nModule Checksum: %s", result, info.Main.Sum)
		}
	}

	return result
}
