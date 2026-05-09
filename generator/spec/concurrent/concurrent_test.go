// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrent_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/concurrent"
)

func TestConcurrent(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.Concurrent)) > 0,
			"concurrent consumer registered")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, concurrent.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.Concurrent, struct{}{})
		testkit.True(t, concurrent.Has(&m), "present after Set")
	})
}
