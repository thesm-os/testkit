// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import "testing"

// Rejects drives fn against an implementation it is meant to reject, and
// asserts that fn failed. It returns the failure message, empty if there was
// none.
//
// The assertion that an assertion can fail. A check whose every statement is
// [NoError] passes against a subject whose methods return nil and do nothing —
// it reads as coverage and asserts nothing — so the wrong implementation is
// named, driven, and the rejection observed.
//
//	testkit.Rejects(t, "a store that overwrites must not satisfy the check",
//	    func(tb testing.TB) { refusesADuplicate(tb, overwritingStore{}) })
//
// The returned message is what makes the proof specific, and it is worth using.
// A wrong subject that panics on a nil map satisfies this while the check's own
// assertion never ran — the guard passes and proves nothing, which is the
// defect it exists to catch, one level up:
//
//	got := testkit.Rejects(t, "an unbounded pool must not satisfy the check",
//	    func(tb testing.TB) { handsOutWhatItHolds(tb, unboundedPool{}) })
//	testkit.Assert(t, got).Contains("the pool it came from is then empty",
//	    "and rejects it for the reason the check is about")
//
// # Concurrency
//
// fn runs on a goroutine of its own and this call blocks until it finishes. A
// [FailableTB] in Goexit mode implements Fatal as [runtime.Goexit] does, which
// is what stops a check running past an assertion it already failed — and
// Goexit needs a goroutine to exit. The goroutine does not outlive the call, so
// nothing leaks and a leak check around it stays quiet.
//
// A panic inside fn is not recovered. It crosses the goroutine boundary and
// takes the process down, which is right: a check that panics is a defect in
// the check or in the stand-in, and reporting it as "rejected" would hide it.
func Rejects(tb testing.TB, msg string, fn func(tb testing.TB)) string {
	tb.Helper()

	f := NewFailableTB().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(f)
	}()
	<-done

	if !f.Failed() {
		tb.Fatalf("%s: the check passed against an implementation it must reject", msg)
	}
	return f.Msg()
}
