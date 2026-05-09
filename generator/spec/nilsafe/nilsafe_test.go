// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nilsafe_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/nilsafe"
)

func TestNilSafe(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.NilSafe)) > 0,
			"nilsafe consumer registered")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, nilsafe.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.NilSafe, struct{}{})
		testkit.True(t, nilsafe.Has(&m), "present after Set")
	})
}
