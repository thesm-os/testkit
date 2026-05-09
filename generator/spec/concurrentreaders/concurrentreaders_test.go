// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrentreaders_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/concurrentreaders"
)

func TestConcurrentReaders(t *testing.T) {
	t.Parallel()

	t.Run("importing the package registers a consumer", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, len(spec.Consumers(directive.ConcurrentReaders)) > 0,
			"concurrent-readers consumer registered")
	})

	t.Run("Has reflects presence in Attachments", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.False(t, concurrentreaders.Has(&m), "absent without attachment")
		spec.Set(&m.Attachments, directive.ConcurrentReaders, struct{}{})
		testkit.True(t, concurrentreaders.Has(&m), "present after Set")
	})
}
