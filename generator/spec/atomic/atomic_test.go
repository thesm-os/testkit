// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package atomic_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/atomic" // self-registration
)

// Smoke: importing the package wires a consumer for directive.Atomic.
// The package is a one-line marker (init() → marker.Register); this
// test guards against accidental regression of that wiring.
func TestRegistration(t *testing.T) {
	t.Parallel()
	testkit.True(t, len(spec.Consumers(directive.Atomic)) > 0,
		"atomic consumer registered")
}
