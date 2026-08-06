// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

// Arms describes the ways one call can be answered, in the order a double
// consults them.
//
// R is the method's return tuple, which Go has no syntax for — a generated
// double declares a struct per method for exactly this reason, and that
// struct is what travels through here.
type Arms[C, R any] struct {
	// Invoke calls the configured Func override. Nil when none is set, which
	// is how [Answer] tells "no override" from "an override returning zero".
	Invoke func() R

	// Fallback is the result pinned by Returns, nil when none is pinned.
	Fallback *R

	// Fault builds a result carrying err in the method's error slot. Nil for
	// a method that returns no error, where an injected fault has nowhere to
	// go and the fault arm is skipped entirely.
	Fault func(err error) R

	// Stamp copies a result onto the recorded call, so the log carries what
	// the caller was actually told rather than only what it asked.
	Stamp func(call *C, result R)
}

// Answer resolves one call against a method stub and records it.
//
// The order is the double's contract and the reason this lives here rather
// than in a template: an injected fault wins over a Func override, which wins
// over a Returns fallback, which wins over the zero value. A test that
// configures a fault and gets the happy path instead is asserting nothing,
// and the failure is silent.
//
// Every arm records. A log reflecting only the configured arm would report
// what the test set up rather than what the code under test did.
//
// # Hazards
//
// call is taken by pointer because [Arms.Stamp] writes to it before the
// recording is taken; the value recorded is the stamped one. Latency is slept
// before anything else, so a test driving a virtual clock sees the call
// blocked at the point a real implementation would be.
func Answer[C, R any](s *MethodStub[C], call *C, arms Arms[C, R]) R {
	s.SleepLatency()

	if arms.Fault != nil {
		if fired, err := s.ShouldFaultFor(*call); fired {
			return answer(s, call, arms, arms.Fault(err))
		}
	}
	if arms.Invoke != nil {
		return answer(s, call, arms, arms.Invoke())
	}
	if arms.Fallback != nil {
		return answer(s, call, arms, *arms.Fallback)
	}

	s.FailUnexpectedCall(*call)
	var zero R
	return answer(s, call, arms, zero)
}

// answer stamps a result onto the call, records it, and hands the result back
// — the tail every arm shares.
func answer[C, R any](s *MethodStub[C], call *C, arms Arms[C, R], result R) R {
	if arms.Stamp != nil {
		arms.Stamp(call, result)
	}
	s.Record(*call)
	return result
}
