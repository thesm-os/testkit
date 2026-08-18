// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law

import (
	"pgregory.net/rapid"
)

// The law combinators: adapters that lift an existing law into a new
// binding shape without touching the law's judgement. Both forward ID
// and REQID, so the wrapped law reports under its own name — a
// combinator is wiring, never a second vocabulary.
//
// Neither combinator forwards the [StatefulLaw] or [Isolated] interface
// assertions: wrapping hides them from the runner. Wrap plain
// Check-shaped laws only; a stateful or isolated law needs its own
// binding.

// Budget caps how many times inner's Check runs per property iteration;
// invocations beyond n pass without touching the pair. It exists for
// POLLING laws, whose failure cost is their timeout: bound the
// invocations and a broken subject costs at most n timeouts per
// iteration instead of one per action.
//
// The budget refills when the runner resets laws at the start of each
// iteration — deliberately, because rapid's shrinking re-runs the
// property expecting the same draws to fail the same way, and a budget
// that stayed spent across attempts would make the failure vanish
// mid-shrink and read as flaky. The trade is stated plainly: per
// iteration, the law sees n draws instead of one per action. Use it
// through the runner (WithLaw); outside the runner nothing refills it.
func Budget[T any](n int, inner Law[T]) Law[T] {
	return &budgetLaw[T]{n: n, inner: inner}
}

type budgetLaw[T any] struct {
	n     int
	spent int
	inner Law[T]
}

// ID forwards the wrapped law's identity.
func (l *budgetLaw[T]) ID() string { return l.inner.ID() }

// REQID forwards the wrapped law's requirement tag.
func (l *budgetLaw[T]) REQID() string { return l.inner.REQID() }

// Check runs inner until the iteration's budget is spent, then passes.
func (l *budgetLaw[T]) Check(rt *rapid.T, sut, ref T) error {
	if l.spent >= l.n {
		return nil // the iteration's evidence was spent on the first n
	}
	l.spent++
	return l.inner.Check(rt, sut, ref)
}

// Reset refills the budget and forwards to a Resettable inner. The
// runner calls it at the start of every property iteration, which is
// what makes the budget per-iteration and the shrinking deterministic.
func (l *budgetLaw[T]) Reset() {
	l.spent = 0
	if r, ok := l.inner.(Resettable); ok {
		r.Reset()
	}
}

// Produced lifts a law over a PRODUCED secondary — a cursor, a
// transaction handle, a subscription — into a law over its producer.
// There is no sub-harness for a produced type: the producing method is
// its constructor, so open obtains one fresh secondary from the subject
// per Check, and a refused open is [Vacuous] — a precondition this run
// supplies, counted by the census rather than passing silently.
//
// The inner law receives the produced value as BOTH sides of the pair: a
// produced secondary has no reference twin, and the laws this shape
// carries (teardown idempotence, the after-close sentinel) observe their
// subject alone. The secondary is throwaway by construction — an inner
// law that corrupts it (closing it is the point) leaves the SHARED pair
// untouched, which is what keeps this safe to register without the
// Isolated marker.
func Produced[T, S any](open func(rt *rapid.T, sut T) (S, error), inner Law[S]) Law[T] {
	return producedLaw[T, S]{open: open, inner: inner}
}

type producedLaw[T, S any] struct {
	open  func(rt *rapid.T, sut T) (S, error)
	inner Law[S]
}

// ID forwards the wrapped law's identity.
func (l producedLaw[T, S]) ID() string { return l.inner.ID() }

// REQID forwards the wrapped law's requirement tag.
func (l producedLaw[T, S]) REQID() string { return l.inner.REQID() }

// Check opens one fresh secondary and judges it with the inner law.
func (l producedLaw[T, S]) Check(rt *rapid.T, sut, _ T) error {
	s, err := l.open(rt, sut)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	return l.inner.Check(rt, s, s)
}
