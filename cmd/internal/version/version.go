// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package version holds build-injected version information.
package version

import "fmt"

// Set by ldflags at build time.
var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

// String returns a human-readable version string.
func String() string {
	return fmt.Sprintf("testkit %s (%s, %s)", buildVersion, buildCommit, buildDate)
}
