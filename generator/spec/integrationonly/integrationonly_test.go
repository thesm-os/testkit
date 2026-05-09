// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package integrationonly_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/integrationonly" // self-registration
)

func TestRegistration(t *testing.T) {
	t.Parallel()
	testkit.True(t, len(spec.Consumers(directive.IntegrationOnly)) > 0,
		"integration-only consumer registered")
}
