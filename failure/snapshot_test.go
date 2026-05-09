// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package failure_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/failure"
)

func TestSnapshotIsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("nil snapshot is empty", func(t *testing.T) {
		t.Parallel()
		var s *failure.Snapshot
		testkit.True(t, s.IsEmpty(), "nil snapshot")
	})

	t.Run("zero-value snapshot is empty", func(t *testing.T) {
		t.Parallel()
		s := &failure.Snapshot{}
		testkit.True(t, s.IsEmpty(), "zero snapshot")
	})

	t.Run("any populated map is non-empty", func(t *testing.T) {
		t.Parallel()
		s := &failure.Snapshot{PerComponent: map[string]any{"Ledger": "state"}}
		testkit.False(t, s.IsEmpty(), "PerComponent populated")

		s2 := &failure.Snapshot{PerImpl: map[string]any{"v1": "state"}}
		testkit.False(t, s2.IsEmpty(), "PerImpl populated")

		s3 := &failure.Snapshot{Custom: map[string]any{"key": "value"}}
		testkit.False(t, s3.IsEmpty(), "Custom populated")
	})

	t.Run("empty maps still empty", func(t *testing.T) {
		t.Parallel()
		s := &failure.Snapshot{
			PerComponent: map[string]any{},
			PerImpl:      map[string]any{},
			Custom:       map[string]any{},
		}
		testkit.True(t, s.IsEmpty(), "empty maps are empty")
	})
}
