// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package atomic_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/atomic"
)

func TestAtomic(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Atomic)) > 0,
			"atomic consumer registered")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, atomic.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Atomic, struct{}{})
		testkit.True(t, atomic.Has(&m), "present after Set")
	})
}
