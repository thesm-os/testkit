// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/stub"
)

func TestOrderTracker(t *testing.T) {
	t.Parallel()

	t.Run("Record marks method as called in strict mode", func(t *testing.T) {
		t.Parallel()
		ot := stub.NewOrderTracker(t, true)
		testkit.False(t, ot.Called("Open"), "Open must not be called yet")
		ot.Record("Open")
		testkit.True(t, ot.Called("Open"), "Open must be called")
	})

	t.Run("Record is no-op in lenient mode", func(t *testing.T) {
		t.Parallel()
		ot := stub.NewOrderTracker(t, false)
		ot.Record("Open")
		testkit.False(t, ot.Called("Open"), "lenient mode must not record")
	})

	t.Run("AssertAfter passes when prerequisite called", func(t *testing.T) {
		t.Parallel()
		ot := stub.NewOrderTracker(t, true)
		ot.Record("Open")
		ot.AssertAfter("Read", "Open") // must not fatal
	})

	t.Run("AssertAfter fatals when prerequisite not called", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		ot := stub.NewOrderTracker(f, true)
		ot.AssertAfter("Read", "Open")
		testkit.True(t, f.Failed(), "must fatal when prerequisite not called")
		testkit.Assert(t, f.Msg()).
			Contains("Read", "must name the offending method").
			Contains("Open", "must name the prerequisite")
	})

	t.Run("AssertAfter is no-op in lenient mode", func(t *testing.T) {
		t.Parallel()
		ot := stub.NewOrderTracker(t, false)
		ot.AssertAfter("Read", "Open") // must not fatal
	})

	t.Run("AssertAfter is no-op with nil tb", func(t *testing.T) {
		t.Parallel()
		ot := stub.NewOrderTracker(nil, true)
		ot.AssertAfter("Read", "Open") // must not panic
	})

	t.Run("Reset clears call history", func(t *testing.T) {
		t.Parallel()
		ot := stub.NewOrderTracker(t, true)
		ot.Record("Open")
		testkit.True(t, ot.Called("Open"), "must be called")
		ot.Reset()
		testkit.False(t, ot.Called("Open"), "must be cleared after Reset")
	})
}

// String is a debugging aid: when an ordering assertion fails, the tracker's
// contents are what tell the reader which calls actually arrived.
func TestOrderTrackerString(t *testing.T) {
	t.Parallel()

	t.Run("names every recorded method", func(t *testing.T) {
		t.Parallel()
		// strict=true: Record is a documented no-op in lenient mode, so a
		// lenient tracker has nothing to render.
		ot := stub.NewOrderTracker(t, true)
		ot.Record("Open")
		ot.Record("Close")
		got := ot.String()
		testkit.HasPrefix(t, got, "OrderTracker{", "must be identifiable in a dump")
		testkit.Contains(t, got, "Open", "must name recorded methods")
		testkit.Contains(t, got, "Close", "must name recorded methods")
	})

	t.Run("renders empty when nothing recorded", func(t *testing.T) {
		t.Parallel()
		ot := stub.NewOrderTracker(t, true)
		testkit.HasPrefix(t, ot.String(), "OrderTracker{", "must render without panicking")
	})
}
