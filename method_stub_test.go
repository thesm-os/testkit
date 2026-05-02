// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"testing"

	"go.thesmos.sh/testkit"
)

type sampleCall struct {
	Arg    string
	Result int
	Err    error
}

func TestMethodStub(t *testing.T) {
	t.Parallel()

	t.Run("records calls via embedded Recorder", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Record(sampleCall{Arg: "a", Result: 1})
		stub.Record(sampleCall{Arg: "b", Result: 2})

		testkit.Equal(t, stub.CallCount(), 2, "must record calls")
		calls := stub.Calls()
		testkit.Equal(t, calls[0].Arg, "a", "first call arg")
		testkit.Equal(t, calls[1].Result, 2, "second call result")
	})

	t.Run("assertion helpers delegate to Recorder", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Record(sampleCall{Arg: "only"})

		call := stub.AssertCalledOnce(t, "must be called once")
		testkit.Equal(t, call.Arg, "only", "must return the call")
	})

	t.Run("Filter delegates to Recorder", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Record(sampleCall{Arg: "a"})
		stub.Record(sampleCall{Arg: "b"})
		stub.Record(sampleCall{Arg: "a"})

		filtered := stub.Filter(func(c sampleCall) bool { return c.Arg == "a" })
		testkit.Len(t, filtered, 2, "must filter correctly")
	})
}

func TestMethodStub_faults(t *testing.T) {
	t.Parallel()

	t.Run("ShouldFault returns false when not configured", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		fired, err := stub.ShouldFault()
		testkit.False(t, fired, "must not fire without configuration")
		testkit.NoError(t, err, "err must be nil")
	})

	t.Run("ShouldFault fires on configured interval", func(t *testing.T) {
		t.Parallel()
		errBoom := testkit.TestError("boom")
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Faults(errBoom, 2) // fires every 2nd call

		f1, _ := stub.ShouldFault()  // call 1: no
		f2, e2 := stub.ShouldFault() // call 2: yes
		f3, _ := stub.ShouldFault()  // call 3: no
		f4, e4 := stub.ShouldFault() // call 4: yes

		testkit.False(t, f1, "call 1 must not fault")
		testkit.True(t, f2, "call 2 must fault")
		testkit.ErrorIs(t, e2, errBoom, "call 2 fault error")
		testkit.False(t, f3, "call 3 must not fault")
		testkit.True(t, f4, "call 4 must fault")
		testkit.ErrorIs(t, e4, errBoom, "call 4 fault error")
	})

	t.Run("Reset clears fault counter", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Faults(testkit.TestError("x"), 2)
		_, _ = stub.ShouldFault() // call 1 — advance counter
		stub.Record(sampleCall{Arg: "a"})

		stub.Reset()

		testkit.Equal(t, stub.CallCount(), 0, "recordings must be cleared")
		f1, _ := stub.ShouldFault()
		testkit.False(t, f1, "fault counter must be reset")
	})
}

func TestMethodStub_strict(t *testing.T) {
	t.Parallel()

	t.Run("FailUnexpected does nothing in lenient mode", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.FailUnexpected("arg1") // should not panic or fatal
	})

	t.Run("FailUnexpected fatals in strict mode", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.Strict()

		stub.FailUnexpected("arg1")
		testkit.True(t, f.Failed(), "must fatal in strict mode")
		testkit.Assert(t, f.Msg()).
			Contains("Svc.Do", "must include method name").
			Contains("unexpected call", "must describe the failure")
	})

	t.Run("FailUnexpected without args", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.Strict()

		stub.FailUnexpected()
		testkit.True(t, f.Failed(), "must fatal in strict mode")
	})

	t.Run("IsStrict reports mode", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		testkit.False(t, stub.IsStrict(), "default is lenient")
		stub.Strict()
		testkit.True(t, stub.IsStrict(), "must be strict after Strict()")
	})
}

func TestMethodStub_verify(t *testing.T) {
	t.Parallel()

	t.Run("Times passes when count matches", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.Times(2)
		stub.Record(sampleCall{})
		stub.Record(sampleCall{})

		stub.Verify()
		testkit.False(t, f.Failed(), "must pass when count matches")
	})

	t.Run("Times fails when count does not match", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.Times(3)
		stub.Record(sampleCall{})

		stub.Verify()
		testkit.True(t, f.Failed(), "must fail when count does not match")
		testkit.Assert(t, f.Msg()).
			Contains("Svc.Do", "must include method name").
			Contains("expected 3", "must include expected count").
			Contains("got 1", "must include actual count")
	})

	t.Run("TimesAtLeast passes when count sufficient", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.TimesAtLeast(1)
		stub.Record(sampleCall{})
		stub.Record(sampleCall{})

		stub.Verify()
		testkit.False(t, f.Failed(), "must pass when count >= minimum")
	})

	t.Run("TimesAtLeast fails when count insufficient", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.TimesAtLeast(5)
		stub.Record(sampleCall{})

		stub.Verify()
		testkit.True(t, f.Failed(), "must fail when count < minimum")
		testkit.Assert(t, f.Msg()).Contains("at least 5", "must include minimum")
	})

	t.Run("Verify does nothing without tb", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		stub.Times(99)
		stub.Verify() // should not panic
	})

	t.Run("Reset clears expectations", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		stub.Times(5)
		stub.Reset()

		stub.Verify()
		testkit.False(t, f.Failed(), "reset must clear Times expectation")
	})
}

func TestMethodStub_name(t *testing.T) {
	t.Parallel()

	t.Run("Name returns configured name", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Store.Get")
		testkit.Equal(t, stub.Name(), "Store.Get", "must return name")
	})

	t.Run("TB returns configured TB", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		stub := testkit.NewMethodStub[sampleCall](f, "Svc.Do")
		testkit.True(t, stub.TB() == f, "must return configured TB")
	})

	t.Run("TB returns nil when not configured", func(t *testing.T) {
		t.Parallel()
		stub := testkit.NewMethodStub[sampleCall](nil, "Svc.Do")
		testkit.True(t, stub.TB() == nil, "must return nil")
	})
}
