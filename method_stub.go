// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import "testing"

// MethodStub is a generic per-method test double that combines recording,
// fault injection, strict mode, and call count expectations. Generated
// stub code embeds one MethodStub per interface method and adds type-specific
// dispatch (Func, Returns).
//
//	type GetStub struct {
//	    *testkit.MethodStub[GetCall]
//	    fn       func(context.Context, string) (Item, error)
//	    fallback *getReturn
//	}
type MethodStub[C any] struct {
	*Recorder[C]

	fault   *FaultInjector
	tb      testing.TB
	name    string // "Store.Get" — for error messages
	strict  bool
	times   *int
	atLeast *int
}

// NewMethodStub creates a [MethodStub] for the named method. Pass tb for
// auto-verification via [testing.TB.Cleanup]; pass nil for a pure stub
// without test integration.
//
//nolint:thelper // constructor, not a test helper — tb may be nil
func NewMethodStub[C any](tb testing.TB, name string) *MethodStub[C] {
	return &MethodStub[C]{
		Recorder: NewRecorder[C](),
		tb:       tb,
		name:     name,
	}
}

// Strict enables strict mode. In strict mode, [MethodStub.FailUnexpected]
// fatals the test when called — generated stubs call FailUnexpected when
// no behavior (func, returns, matcher) is configured.
func (s *MethodStub[C]) Strict() {
	s.strict = true
}

// IsStrict reports whether strict mode is enabled.
func (s *MethodStub[C]) IsStrict() bool {
	return s.strict
}

// Faults configures fault injection on this method. The fault fires on
// every nth call. Returns the receiver for chaining.
func (s *MethodStub[C]) Faults(err error, failEveryN int) *MethodStub[C] {
	fi := NewFaultInjector(err, failEveryN)
	s.fault = &fi
	return s
}

// ShouldFault checks whether the fault should fire on this call.
// Returns (true, faultErr) if the fault fires, (false, nil) otherwise.
// Generated stubs call this at the start of every method dispatch.
func (s *MethodStub[C]) ShouldFault() (bool, error) {
	if s.fault != nil && s.fault.FaultShouldFire() {
		return true, s.fault.FaultErr
	}
	return false, nil
}

// FailUnexpected fatals the test if strict mode is enabled. Generated
// stubs call this when no behavior is configured for a method call.
func (s *MethodStub[C]) FailUnexpected(args ...any) {
	if s.strict && s.tb != nil {
		if len(args) > 0 {
			s.tb.Fatalf("%s: unexpected call (strict mode)\n  args: %v", s.name, args)
		} else {
			s.tb.Fatalf("%s: unexpected call (strict mode)", s.name)
		}
	}
}

// Times sets the exact expected number of calls. Checked by [MethodStub.Verify].
func (s *MethodStub[C]) Times(n int) *MethodStub[C] {
	s.times = &n
	return s
}

// TimesAtLeast sets the minimum expected number of calls. Checked by
// [MethodStub.Verify].
func (s *MethodStub[C]) TimesAtLeast(n int) *MethodStub[C] {
	s.atLeast = &n
	return s
}

// Verify checks Times and TimesAtLeast expectations against the actual
// call count. Called automatically via [testing.TB.Cleanup] when tb was
// provided to [NewMethodStub].
func (s *MethodStub[C]) Verify() {
	if s.tb == nil {
		return
	}
	count := s.CallCount()
	if s.times != nil && count != *s.times {
		s.tb.Errorf("%s: expected %d call(s), got %d", s.name, *s.times, count)
	}
	if s.atLeast != nil && count < *s.atLeast {
		s.tb.Errorf("%s: expected at least %d call(s), got %d", s.name, *s.atLeast, count)
	}
}

// Reset clears recorded calls, resets fault counters, and clears
// Times/TimesAtLeast expectations. It does NOT clear Func, Returns,
// or Faults configuration — behavior is preserved, only observations
// are rewound. This matches the pattern: config sticks, counters rewind.
func (s *MethodStub[C]) Reset() {
	s.Recorder.Reset()
	if s.fault != nil {
		s.fault.FaultReset()
	}
	s.times = nil
	s.atLeast = nil
}

// Name returns the method name (e.g. "Store.Get").
func (s *MethodStub[C]) Name() string {
	return s.name
}

// TB returns the associated testing.TB, or nil.
func (s *MethodStub[C]) TB() testing.TB {
	return s.tb
}
