// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package idempotent_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/idempotent"
)

func TestIdempotent(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Idempotent)) > 0,
			"idempotent consumer registered")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, idempotent.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Idempotent, struct{}{})
		testkit.True(t, idempotent.Has(&m), "present after Set")
	})
}
