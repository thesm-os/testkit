// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub_test

import (
	"testing"

	"go.thesmos.sh/testkit/stub"
)

// call is the recorded-call type the fixtures below double.
type call struct {
	Key string
	Err error
}

// fake is a minimal double standing in for a generated one: one method stub,
// one method that dispatches through it. Behaviour is written against
// generated code, so what it is exercised with here has to behave the same
// way — record every call, and return an injected fault when one fires.
type fake struct {
	on       *stub.MethodStub[call]
	override func()
}

func newFake(tb testing.TB) *fake {
	tb.Helper()
	return &fake{on: stub.NewMethodStub[call](tb, "Fake.Get")}
}

// Get mirrors a generated dispatch body: fault, then record, then answer.
func (f *fake) Get(key string) error {
	c := call{Key: key}
	if fired, err := f.on.ShouldFaultFor(c); fired {
		c.Err = err
		f.on.Record(c)
		return err
	}
	if f.override != nil {
		f.override()
		f.on.Record(c)
		return nil
	}
	f.on.FailUnexpectedCall(c)
	f.on.Record(c)
	return nil
}

// subject wires a fresh fake into the shape Behaviour drives.
func subject(tb testing.TB) stub.Subject[call, ret] {
	tb.Helper()
	f := newFake(tb)
	return stub.Subject[call, ret]{
		Stub:     f.on,
		Call:     func() { _ = f.Get("k") },
		Fails:    func() error { return f.Get("k") },
		Result:   func() ret { return ret{Err: f.Get("k")} },
		Override: func(mark func()) { f.override = mark },
	}
}

// Behaviour is the one place the per-method contract is asserted, so the
// contract has to be exercised here rather than trusted through the generated
// code that calls it.
func TestBehaviour(t *testing.T) {
	t.Parallel()

	stub.Behaviour(t, "Get", subject)
}

// A method that cannot fail has no error to inject into, so the fault checks
// are skipped rather than asserting something meaningless.
func TestBehaviourSkipsFaultsWhenAMethodCannotFail(t *testing.T) {
	t.Parallel()

	stub.Behaviour(t, "Close", func(tb testing.TB) stub.Subject[call, ret] {
		tb.Helper()
		s := subject(tb)
		s.Fails = nil
		s.Result = nil
		return s
	})
}
