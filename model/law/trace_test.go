// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"errors"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
	"go.thesmos.sh/testkit/model/trace"
)

func TestAfterEvery(t *testing.T) {
	t.Parallel()

	t.Run("passes when predicate holds after target action", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		l := &law.AfterEvery[int]{
			ActionName: "Put",
			Predicate:  func(_ *rapid.T, sut, _ int) error { return nil },
		}
		l.BindTrace(tr)

		tr.Record(trace.Event{ClientID: -1, OpName: "Put"})
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, 0, 0)
			if err != nil {
				rt.Fatalf("unexpected: %v", err)
			}
		})
	})

	t.Run("fires when predicate fails after target action", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		l := &law.AfterEvery[int]{
			ActionName: "Put",
			Predicate:  func(_ *rapid.T, _, _ int) error { return errors.New("count wrong") },
		}
		l.BindTrace(tr)

		tr.Record(trace.Event{ClientID: -1, OpName: "Put"})
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, 0, 0)
			if err == nil {
				rt.Fatal("should have fired")
			}
		})
	})

	t.Run("skips when last action is not the target", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		l := &law.AfterEvery[int]{
			ActionName: "Put",
			Predicate:  func(_ *rapid.T, _, _ int) error { return errors.New("should not fire") },
		}
		l.BindTrace(tr)

		tr.Record(trace.Event{ClientID: -1, OpName: "Get"})
		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, 0, 0)
			if err != nil {
				rt.Fatalf("should have skipped: %v", err)
			}
		})
	})
}

func TestEventuallyAfter(t *testing.T) {
	t.Parallel()

	t.Run("passes when response appears within budget", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		l := &law.EventuallyAfter[int]{
			Trigger:     "Put",
			Response:    "Sync",
			WithinSteps: 3,
		}
		l.BindTrace(tr)

		tr.Record(trace.Event{ClientID: -1, OpName: "Put"})
		tr.Record(trace.Event{ClientID: -1, OpName: "Get"})
		tr.Record(trace.Event{ClientID: -1, OpName: "Sync"})

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, 0, 0)
			if err != nil {
				rt.Fatalf("unexpected: %v", err)
			}
		})
	})

	t.Run("fires when response exceeds budget", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		l := &law.EventuallyAfter[int]{
			Trigger:     "Put",
			Response:    "Sync",
			WithinSteps: 2,
		}
		l.BindTrace(tr)

		tr.Record(trace.Event{ClientID: -1, OpName: "Put"})
		tr.Record(trace.Event{ClientID: -1, OpName: "Get"})
		tr.Record(trace.Event{ClientID: -1, OpName: "Get"})
		tr.Record(trace.Event{ClientID: -1, OpName: "Get"})

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, 0, 0)
			if err == nil {
				rt.Fatal("should have fired")
			}
		})
	})

	t.Run("passes when no trigger has fired", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		l := &law.EventuallyAfter[int]{
			Trigger:     "Put",
			Response:    "Sync",
			WithinSteps: 2,
		}
		l.BindTrace(tr)

		tr.Record(trace.Event{ClientID: -1, OpName: "Get"})
		tr.Record(trace.Event{ClientID: -1, OpName: "Get"})

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, 0, 0)
			if err != nil {
				rt.Fatalf("unexpected: %v", err)
			}
		})
	})
}

func TestNever(t *testing.T) {
	t.Parallel()

	t.Run("passes when forbidden action absent", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		l := &law.Never[int]{ActionName: "Panic"}
		l.BindTrace(tr)

		tr.Record(trace.Event{ClientID: -1, OpName: "Get"})
		tr.Record(trace.Event{ClientID: -1, OpName: "Put"})

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, 0, 0)
			if err != nil {
				rt.Fatalf("unexpected: %v", err)
			}
		})
	})

	t.Run("fires when forbidden action appears", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		l := &law.Never[int]{ActionName: "Panic"}
		l.BindTrace(tr)

		tr.Record(trace.Event{ClientID: -1, OpName: "Get"})
		tr.Record(trace.Event{ClientID: -1, OpName: "Panic"})

		rapid.Check(t, func(rt *rapid.T) {
			err := l.Check(rt, 0, 0)
			if err == nil {
				rt.Fatal("should have fired — forbidden action present")
			}
		})
	})
}
