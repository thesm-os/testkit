// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"testing"

	"go.thesmos.sh/testkit"
)

// Subject is one method of a test double, described well enough to exercise
// it without knowing its signature.
//
// C is the recorded-call type and R the return tuple — the struct a generated
// double declares per method because Go has no tuple syntax. Between them
// they let the contract below be written once rather than regenerated for
// every method of every interface in every consuming repository.
//
// A generated companion supplies the closures. Each is optional except Call:
// a method that cannot fail has no Fails, one that returns nothing has no
// Result, and the checks needing them are skipped rather than asserting
// something meaningless.
type Subject[C, R any] struct {
	// Stub is the method's configuration point on the double under test.
	Stub *MethodStub[C]

	// Call invokes the method with zero-valued arguments and discards
	// whatever it returns. The values are incidental: what these checks
	// assert is that a call was recorded, cleared, faulted, or refused.
	Call func()

	// Fails invokes the method and returns its error. Nil for a method that
	// cannot fail.
	Fails func() error

	// Result invokes the method and returns what it answered, boxed into the
	// return tuple. Nil for a method that returns nothing.
	Result func() R

	// Override installs a Func override that calls mark when it runs and then
	// answers with zero values. The closure is generated because installing
	// one needs the method's signature; what it proves — that an override
	// takes precedence over the zero value — is the same for every method.
	Override func(mark func())
}

// Behaviour runs the per-method contract every generated double owes,
// independent of the method's signature.
//
// newSubject must build a *fresh* double bound to the supplied TB and return
// the subject for one of its methods. It is called once per check: several
// assert on failure, which needs a double bound to a [testkit.FailableTB]
// rather than to the running test, and a shared instance would carry recorded
// calls between checks.
//
// What is deliberately not here: any check needing a value the caller can
// tell apart from a zero one. "Returns what Returns pinned" is that case —
// pinning a zero result is indistinguishable from configuring nothing, and
// building a distinguishable value of an arbitrary type needs the type. That
// one stays generated.
//
// # Hazards
//
// Failures report against this file rather than against generated code. The
// subtests are scoped by the caller's test function, which names the method,
// and name is carried into every failure message; that is a weaker signal
// than an inlined assertion and is the price of testing this logic once
// instead of trusting it everywhere.
func Behaviour[C, R any](t *testing.T, name string, newSubject func(tb testing.TB) Subject[C, R]) {
	t.Helper()

	t.Run("records the call", func(t *testing.T) {
		t.Parallel()
		s := newSubject(t)
		s.Call()
		s.Stub.AssertCalledOnce(t, name+" must record every call")
	})

	t.Run("clears the record on reset", func(t *testing.T) {
		t.Parallel()
		s := newSubject(t)
		s.Call()
		s.Stub.Reset()
		s.Stub.AssertNotCalled(t, name+" must have no recorded calls after a reset")
	})

	t.Run("reports an unmet call count", func(t *testing.T) {
		t.Parallel()
		// Verify is called directly rather than through the TB's cleanups. A
		// generated constructor registers it as one, but that is the double's
		// wiring rather than this contract, and depending on it would make
		// the check pass for the wrong reason against a double that never
		// registered anything.
		f := testkit.NewFailableTB()
		s := newSubject(f)
		s.Stub.Times(2)
		s.Call()
		s.Stub.Verify()
		assertFailed(t, f, name+" must report a call count short of Times")
	})

	t.Run("reports an unmet minimum call count", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		s := newSubject(f)
		s.Stub.TimesAtLeast(3)
		s.Call()
		s.Stub.Verify()
		assertFailed(t, f, name+" must report a call count short of TimesAtLeast")
	})

	t.Run("refuses an unconfigured call in strict mode", func(t *testing.T) {
		t.Parallel()
		// Strict mode turns a call nobody planned for into a failure at the
		// call site rather than a puzzling zero further downstream. It is
		// reachable per method, so no construction option is needed.
		f := testkit.NewFailableTB()
		s := newSubject(f)
		s.Stub.Strict()
		s.Call()
		assertFailed(t, f, "strict mode must refuse an unconfigured "+name)
	})

	behaviourAnswers(t, name, newSubject)
	behaviourFaults(t, name, newSubject)
}

// behaviourAnswers runs the checks about what a call answers with, each
// skipped when the subject cannot supply what it needs.
func behaviourAnswers[C, R any](t *testing.T, name string, newSubject func(tb testing.TB) Subject[C, R]) {
	t.Helper()

	probe := newSubject(t)

	if probe.Result != nil {
		t.Run("answers with the zero value when unconfigured", func(t *testing.T) {
			t.Parallel()
			s := newSubject(t)
			var want R
			testkit.Equal(t, s.Result(), want, "an unconfigured "+name+" answers with the zero value")
		})
	}

	if probe.Override != nil {
		t.Run("dispatches to the Func override", func(t *testing.T) {
			t.Parallel()
			called := false
			s := newSubject(t)
			s.Override(func() { called = true })
			s.Call()
			testkit.True(t, called, name+" must dispatch to the Func override")
		})
	}
}

// behaviourFaults runs the checks that need an error to inject into, skipped
// for a method that cannot fail.
func behaviourFaults[C, R any](t *testing.T, name string, newSubject func(tb testing.TB) Subject[C, R]) {
	t.Helper()

	if newSubject(t).Fails == nil {
		return
	}

	t.Run("returns an injected fault", func(t *testing.T) {
		t.Parallel()
		s := newSubject(t)
		want := testkit.TestError(name + "-behaviour")
		s.Stub.Faults(want, 1)
		if got := s.Fails(); got == nil {
			t.Fatalf("%s: want the injected fault, got nil", name)
		}
	})

	t.Run("fires a counted fault on the nth call", func(t *testing.T) {
		t.Parallel()
		// A fault that fires immediately cannot distinguish a retry loop that
		// works from one that never runs, which is the case counted faults
		// exist for.
		s := newSubject(t)
		s.Stub.Faults(testkit.TestError(name+"-behaviour"), 3)
		if got := s.Fails(); got != nil {
			t.Fatalf("%s: call 1 must succeed, got %v", name, got)
		}
		if got := s.Fails(); got != nil {
			t.Fatalf("%s: call 2 must succeed, got %v", name, got)
		}
		if got := s.Fails(); got == nil {
			t.Fatalf("%s: call 3 must fault", name)
		}
	})

	t.Run("records the injected fault on the call", func(t *testing.T) {
		t.Parallel()
		// The recorded call is what a failure message prints, so a fault that
		// fired without being recorded leaves the reader with a passing log
		// and a failing test.
		s := newSubject(t)
		s.Stub.Faults(testkit.TestError(name+"-behaviour"), 1)
		_ = s.Fails()
		s.Stub.AssertCalledOnce(t, name+" must record a faulted call")
	})
}

// assertFailed reports when f did not fail, naming what was expected.
func assertFailed(t *testing.T, f *testkit.FailableTB, msg string) {
	t.Helper()
	if !f.Failed() {
		t.Fatal(msg)
	}
}
