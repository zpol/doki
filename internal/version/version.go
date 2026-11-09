// Package version holds build-time version metadata for DOKI.
package version

// Version is the semantic version of the binary. Overridden at link time.
var Version = "0.4.0-dev"

// Commit is the git commit SHA. Overridden at link time.
var Commit = "unknown"

// BuildDate is the UTC build timestamp. Overridden at link time.
var BuildDate = "unknown"

// String returns a human-readable version line.
func String() string {
	return Version + " (commit " + Commit + ", built " + BuildDate + ")"
}
