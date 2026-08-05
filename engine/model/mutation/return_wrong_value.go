// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import (
	"pgregory.net/rapid"
)

// ReturnWrongValue swaps a Reader's key with a different key drawn
// from the same key generator, so the SUT returns the value for the
// wrong key (or the not-found sentinel). A law suite that doesn't
// catch this is missing a per-key correctness check — typically a
// Read-after-Write or a CountEqualsReference observation.
//
// K is the key type the Reader takes. The wrapped Reader consults
// Retarget before calling the underlying impl; the operator returns
// an alternate key with the configured rate.
type ReturnWrongValue[K any] struct {
	// Rate is the per-call retarget probability in [0.0, 1.0].
	Rate float64

	// Alt draws an alternate key when the operator decides to
	// retarget. The generator and Reader's key generator typically
	// share a pool, so the alt key has the same domain.
	Alt *rapid.Generator[K]
}

// Name returns the operator's stable identifier.
func (ReturnWrongValue[K]) Name() string { return "ReturnWrongValue" }

// Retarget returns the alternate key plus true when the operator
// decides to retarget; otherwise returns the zero K plus false.
func (r ReturnWrongValue[K]) Retarget(rt *rapid.T) (K, bool) {
	if !fires(rt, "ReturnWrongValue_decision", r.Rate) {
		var zero K
		return zero, false
	}
	return r.Alt.Draw(rt, "ReturnWrongValue_alt"), true
}
