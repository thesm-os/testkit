// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package partnernaming is the language-axis fixture for a relational mixin
// whose partner spells the shared parameter differently from the method that
// names it.
//
// Every other relational fixture in the corpus spells both halves alike, so the
// derivation that matches them had one arm exercised and two dead. That is the
// shape this fixture exists to vary: the classification, the types and the
// arity are all ordinary, and only the identifiers differ.
//
// The three methods are the three answers the derivation can give, one each:
//
//   - Touch/Seen — one candidate of the type, so the correspondence is derivable
//     and the check is generated. Under the old rule it was not, because the
//     rule was "spelled identically" rather than "unambiguous".
//   - Move/At — two candidates of the type and nothing saying which, which is
//     the ambiguity `partition` settles with `axis=`. Declined, and the run
//     says so.
//   - Emit/Count — no candidate at all, so a check would have to invent a value
//     the method never takes. Declined, and for a different reason.
//
// This axis varies Go's naming rather than the classification, so it breaks
// generators independently of any directive — which is why it sits here beside
// [receivercollision] rather than under `mixin/sideeffect`. That fixture is
// about what the mixin claims; this one is about whether the generator can
// write the claim down.
package partnernaming

import (
	"context"
)

// Store carries one relational pair per outcome of the partner-argument
// derivation.
//
//testkit:out partnernamingtest/ pkg=partnernamingtest
//testkit:stub
//testkit:suite
type Store interface {
	// Touch mutates state the return value does not carry, and Seen is what
	// makes the effect visible.
	//
	// The observer spells the key `k` where this spells it `key`, and takes
	// nothing of the other parameter's type — so exactly one of Touch's
	// parameters can be it and no guess is involved.
	//testkit:mixin sideeffect observe=Seen
	Touch(ctx context.Context, key string, weight int) error

	// Seen reports what Touch left behind.
	Seen(ctx context.Context, k string) (int, error)

	// Move mutates state under one of two identifiers of the same type, and At
	// asks about one of them without saying which.
	//
	// Nothing in the source decides it: `from` and `to` are both strings, and
	// a generator picking one would write a check about the wrong end of the
	// move half the time. The corresponding fix is to spell the observer's
	// parameter as one of them.
	//testkit:mixin sideeffect observe=At
	Move(ctx context.Context, from, to string) error

	// At reports what lives at one end of a move.
	At(ctx context.Context, where string) (int, error)

	// Emit mutates state keyed on a string, and Count asks about a bucket the
	// method never takes.
	//
	// Not an ambiguity but an absence: a check receives Emit's arguments and
	// nothing else, so there is no int to hand Count. The fix is a fixture
	// field, not a rename.
	//testkit:mixin sideeffect observe=Count
	Emit(ctx context.Context, id string) error

	// Count reports how many records a bucket holds.
	Count(ctx context.Context, bucket int) (int, error)
}
