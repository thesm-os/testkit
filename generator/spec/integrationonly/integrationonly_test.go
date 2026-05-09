// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package integrationonly_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/integrationonly"
)

func TestIntegrationOnly(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.IntegrationOnly)) > 0,
			"integration-only consumer registered")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, integrationonly.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.IntegrationOnly, struct{}{})
		testkit.True(t, integrationonly.Has(&m), "present after Set")
	})
}
