// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import "pgregory.net/rapid"

// Operator is the shape-agnostic header every mutation operator
// implements. The generator queries Name when reporting kill
// statistics; the per-operator helper methods (defined on each
// concrete operator type) carry the actual mutation logic.
type Operator interface {
	// Name returns the stable operator identifier
	// (e.g., "DropWrites"). Used in kill-rate reports.
	Name() string
}

// fires returns true with the configured probability. Endpoints are
// short-circuited so rate ≤ 0 never fires and rate ≥ 1 always fires,
// avoiding the [0, 1]-inclusive rapid draw rounding through the
// boundaries. Every rate-driven operator routes through fires for
// consistency.
func fires(rt *rapid.T, label string, rate float64) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	return rapid.Float64Range(0, 1).Draw(rt, label) < rate
}
