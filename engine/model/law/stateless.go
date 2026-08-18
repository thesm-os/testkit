// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//nolint:errorlint // Law errors are diagnostic, not wrapped.
package law

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
)

// Roundtrip verifies Inverse(F(x)) == x for the consumer-supplied
// forward and inverse functions. Auto-emitted for Pure/Mutator
// methods carrying //testkit:contract codec role=forward inverse=<M> fidelity=exact.
type Roundtrip[T any, X any] struct {
	Forward func(*rapid.T, T, X) (X, error)
	Inverse func(*rapid.T, T, X) (X, error)
	Values  *rapid.Generator[X]
}

// ID returns the stable identifier for this law.
func (Roundtrip[T, X]) ID() string { return lawid.Roundtrip }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (Roundtrip[T, X]) REQID() string { return "" }

// Check verifies Inverse(Forward(x)) == x.
func (l Roundtrip[T, X]) Check(rt *rapid.T, sut, _ T) error {
	x := l.Values.Draw(rt, "Roundtrip_x")
	y, err := l.Forward(rt, sut, x)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	back, err := l.Inverse(rt, sut, y)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	if diff := cmp.Diff(x, back); diff != "" {
		return fmt.Errorf("roundtrip law: Inverse(Forward(%v)) != %v:\n%s", x, x, diff)
	}
	return nil
}

// LossyRoundtrip verifies F(Inverse(F(x))) == F(x) for the consumer-
// supplied lossy forward and inverse functions. Auto-emitted for
// Pure/Mutator methods carrying //testkit:contract codec role=forward inverse=<M> fidelity=lossy.
type LossyRoundtrip[T any, X any] struct {
	Forward func(*rapid.T, T, X) (X, error)
	Inverse func(*rapid.T, T, X) (X, error)
	Values  *rapid.Generator[X]
}

// ID returns the stable identifier for this law.
func (LossyRoundtrip[T, X]) ID() string { return lawid.LossyRoundtrip }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (LossyRoundtrip[T, X]) REQID() string { return "" }

// Check verifies F(Inverse(F(x))) == F(x).
func (l LossyRoundtrip[T, X]) Check(rt *rapid.T, sut, _ T) error {
	x := l.Values.Draw(rt, "LossyRoundtrip_x")
	y1, err := l.Forward(rt, sut, x)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	back, err := l.Inverse(rt, sut, y1)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	y2, err := l.Forward(rt, sut, back)
	if err != nil {
		return Vacuous // a precondition this run supplies was refused
	}
	if diff := cmp.Diff(y1, y2); diff != "" {
		return fmt.Errorf("LossyRoundtrip: F(Inverse(F(%v))) != F(%v):\n%s", x, x, diff)
	}
	return nil
}

// TotalOver verifies the SUT returns a non-zero-value result for
// every input drawn from the consumer-supplied domain generator.
// Auto-emitted for Pure/Aggregator methods carrying
// //testkit:mixin total domain=<D>.
type TotalOver[T any, X any, R comparable] struct {
	Call  func(*rapid.T, T, X) (R, error)
	Input *rapid.Generator[X]
}

// ID returns the stable identifier for this law.
func (TotalOver[T, X, R]) ID() string { return lawid.TotalOver }

// REQID returns an empty string (auto-derived laws have no REQ tag).
func (TotalOver[T, X, R]) REQID() string { return "" }

// Check verifies the function is total (no error) over the domain.
func (l TotalOver[T, X, R]) Check(rt *rapid.T, sut, _ T) error {
	x := l.Input.Draw(rt, "TotalOver_x")
	_, err := l.Call(rt, sut, x)
	if err != nil {
		return fmt.Errorf("TotalOver: input %v: errored %v but domain claims totality", x, err)
	}
	return nil
}
