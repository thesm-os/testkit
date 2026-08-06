// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/stub"
)

// ret is the return tuple a generated double declares per method.
type ret struct {
	Value string
	Err   error
}

// arms builds the wiring a generated dispatch body supplies, with every arm
// present. Individual checks blank the ones they are not exercising.
func arms() stub.Arms[call, ret] {
	return stub.Arms[call, ret]{
		Fault: func(err error) ret { return ret{Err: err} },
		Stamp: func(c *call, r ret) { c.Err = r.Err },
	}
}

// The order the arms are consulted is the double's contract. A test that
// configures a fault and silently gets the happy path asserts nothing, so the
// precedence is pinned here rather than left to each generated body.
func TestAnswerPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("prefers an injected fault over a Func override", func(t *testing.T) {
		t.Parallel()
		s := stub.NewMethodStub[call](t, "Fake.Get")
		want := testkit.TestError("answer")
		s.Faults(want, 1)

		a := arms()
		a.Invoke = func() ret { return ret{Value: "override"} }
		got := stub.Answer(s, &call{}, a)

		testkit.ErrorIs(t, got.Err, want, "the fault must win over the override")
	})

	t.Run("prefers a Func override over a Returns fallback", func(t *testing.T) {
		t.Parallel()
		s := stub.NewMethodStub[call](t, "Fake.Get")

		a := arms()
		a.Invoke = func() ret { return ret{Value: "override"} }
		a.Fallback = &ret{Value: "fallback"}
		got := stub.Answer(s, &call{}, a)

		testkit.Equal(t, got.Value, "override", "the override must win over the fallback")
	})

	t.Run("prefers a Returns fallback over the zero value", func(t *testing.T) {
		t.Parallel()
		s := stub.NewMethodStub[call](t, "Fake.Get")

		a := arms()
		a.Fallback = &ret{Value: "fallback"}
		got := stub.Answer(s, &call{}, a)

		testkit.Equal(t, got.Value, "fallback", "the fallback must win over the zero value")
	})

	t.Run("answers with the zero value when nothing is configured", func(t *testing.T) {
		t.Parallel()
		s := stub.NewMethodStub[call](t, "Fake.Get")

		got := stub.Answer(s, &call{}, arms())

		testkit.Equal(t, got.Value, "", "an unconfigured method answers with the zero value")
	})
}

// A log reflecting only the configured arm would report what the test set up
// rather than what the code under test did, so every arm records.
func TestAnswerRecordsOnEveryArm(t *testing.T) {
	t.Parallel()

	t.Run("the fault arm", func(t *testing.T) {
		t.Parallel()
		s := stub.NewMethodStub[call](t, "Fake.Get")
		s.Faults(testkit.TestError("answer"), 1)

		stub.Answer(s, &call{}, arms())

		s.AssertCalledOnce(t, "a faulted call must be recorded")
	})

	t.Run("the override arm", func(t *testing.T) {
		t.Parallel()
		s := stub.NewMethodStub[call](t, "Fake.Get")
		a := arms()
		a.Invoke = func() ret { return ret{} }

		stub.Answer(s, &call{}, a)

		s.AssertCalledOnce(t, "an overridden call must be recorded")
	})

	t.Run("the fallback arm", func(t *testing.T) {
		t.Parallel()
		s := stub.NewMethodStub[call](t, "Fake.Get")
		a := arms()
		a.Fallback = &ret{}

		stub.Answer(s, &call{}, a)

		s.AssertCalledOnce(t, "a fallback call must be recorded")
	})

	t.Run("the zero arm", func(t *testing.T) {
		t.Parallel()
		s := stub.NewMethodStub[call](t, "Fake.Get")

		stub.Answer(s, &call{}, arms())

		s.AssertCalledOnce(t, "an unconfigured call must be recorded")
	})
}

// The recorded call is what a failure message prints, so the result has to
// reach it before the recording is taken.
func TestAnswerStampsTheResultOntoTheCall(t *testing.T) {
	t.Parallel()

	s := stub.NewMethodStub[call](t, "Fake.Get")
	want := testkit.TestError("answer")
	s.Faults(want, 1)

	stub.Answer(s, &call{}, arms())

	testkit.ErrorIs(t, s.Calls()[0].Err, want, "the recorded call must carry the injected fault")
}

// A signature with no error has nowhere to put one, so the fault arm is
// skipped rather than consulted and discarded.
func TestAnswerSkipsTheFaultArmWhenAMethodCannotFail(t *testing.T) {
	t.Parallel()

	s := stub.NewMethodStub[call](t, "Fake.Close")
	s.Faults(testkit.TestError("answer"), 1)

	a := arms()
	a.Fault = nil
	a.Fallback = &ret{Value: "fallback"}
	got := stub.Answer(s, &call{}, a)

	testkit.Equal(t, got.Value, "fallback", "a configured fault must not divert a method that cannot fail")
}

// Strict mode turns a call nobody planned for into a failure at the call site
// rather than a puzzling zero further downstream.
func TestAnswerFailsAnUnconfiguredCallInStrictMode(t *testing.T) {
	t.Parallel()

	f := testkit.NewFailableTB()
	s := stub.NewMethodStub[call](f, "Fake.Get")
	s.Strict()

	stub.Answer(s, &call{}, arms())

	testkit.True(t, f.Failed(), "strict mode must fail an unconfigured call")
}

// Stamp is optional: a method with no returns has nothing to copy onto its
// recorded call, and a nil hook must not panic on the way past.
func TestAnswerToleratesAnAbsentStamp(t *testing.T) {
	t.Parallel()

	s := stub.NewMethodStub[call](t, "Fake.Close")

	stub.Answer(s, &call{}, stub.Arms[call, struct{}]{})

	s.AssertCalledOnce(t, "a call with nothing to stamp is still recorded")
}
