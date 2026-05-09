// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package idempotent_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/idempotent" // self-registration
)

func TestRegistration(t *testing.T) {
	t.Parallel()
	testkit.True(t, len(spec.Consumers(directive.Idempotent)) > 0,
		"idempotent consumer registered")
}
