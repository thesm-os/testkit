// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"testing"

	"go.thesmos.sh/testkit"
)

func TestOrderTracker(t *testing.T) {
	t.Parallel()

	t.Run("Record marks method as called", func(t *testing.T) {
		t.Parallel()
		ot := testkit.NewOrderTracker(t, false)
		testkit.False(t, ot.Called("Open"), "Open must not be called yet")
		ot.Record("Open")
		testkit.True(t, ot.Called("Open"), "Open must be called")
	})

	t.Run("AssertAfter passes when prerequisite called", func(t *testing.T) {
		t.Parallel()
		ot := testkit.NewOrderTracker(t, true)
		ot.Record("Open")
		ot.AssertAfter("Read", "Open") // must not fatal
	})

	t.Run("AssertAfter fatals when prerequisite not called", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		ot := testkit.NewOrderTracker(f, true)
		ot.AssertAfter("Read", "Open")
		testkit.True(t, f.Failed(), "must fatal when prerequisite not called")
		testkit.Assert(t, f.Msg()).
			Contains("Read", "must name the offending method").
			Contains("Open", "must name the prerequisite")
	})

	t.Run("AssertAfter is no-op in lenient mode", func(t *testing.T) {
		t.Parallel()
		ot := testkit.NewOrderTracker(t, false)
		ot.AssertAfter("Read", "Open") // must not fatal
	})

	t.Run("AssertAfter is no-op with nil tb", func(t *testing.T) {
		t.Parallel()
		ot := testkit.NewOrderTracker(nil, true)
		ot.AssertAfter("Read", "Open") // must not panic
	})

	t.Run("Reset clears call history", func(t *testing.T) {
		t.Parallel()
		ot := testkit.NewOrderTracker(t, true)
		ot.Record("Open")
		testkit.True(t, ot.Called("Open"), "must be called")
		ot.Reset()
		testkit.False(t, ot.Called("Open"), "must be cleared after Reset")
	})
}
