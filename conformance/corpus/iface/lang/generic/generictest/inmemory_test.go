// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// A generic interface's harness is generic, and the consumer instantiates it.
//
// A Go test function cannot take type parameters, so the harness cannot be one
// — but every declaration it emits can, and naming the types here is the same
// thing a consumer does when they construct the implementation. Nothing is
// derived at witnesses: the caller already knows which instantiation they run.
//
// Which is also why nothing is derived at all: a type parameter admits no
// literal, so every family the rules reached was refused and the header lists
// them. The values come from the row, which is the one place they can.
package generictest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/generic"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/generic/generictest"
)

// TestStoreContract runs the generated checks and this package's own, at the
// instantiation this package names.
func TestStoreContract(t *testing.T) {
	t.Parallel()

	generictest.RunStore[string, int](t, inMemory("in-memory"), storeChecks)
}

// TestStoreChecksCanFail drives every row against its planted defect.
func TestStoreChecksCanFail(t *testing.T) {
	t.Parallel()

	generictest.ProveStore[string, int](t, storeChecks)
}

// --- Harnesses ---------------------------------------------------------------

// The key and value the rows use. Literals rather than fixture accessors: the
// fixture's K and V are the type parameters' zeros, because no literal can be
// written for a type nobody has instantiated yet.
const (
	seededKey   = "seeded-key"
	seededValue = 7
	absentKey   = "absent-key"
)

func inMemory(
	name string,
) generictest.StoreHarness[string, int, *generictest.InMemory[string, int]] {
	return generictest.StoreHarness[string, int, *generictest.InMemory[string, int]]{
		Name: name, New: generictest.NewInMemory[string, int],
	}
}

// --- The checks: claims, bodies and defects, by name --------------------------

var storeChecks = generictest.StoreChecks[string, int]{
	{
		Method: "Get", Name: "reads-back-what-put-wrote",
		Claim: "Get returns what Put wrote",
		Run:   readsBackWhatPutWrote,
		ProvenBy: generictest.BrokenStore[string, int](
			"a store whose reads answer the value type's zero",
			planted(answersTheZero),
		),
		ProvenReason: "carries what was written",
	},

	{
		Method: "Get", Name: "miss-is-reported",
		Claim: "Get reports a key nothing wrote",
		Run:   missIsReported,
		ProvenBy: generictest.BrokenStore[string, int](
			"a store that reads a miss as the value type's zero",
			planted(readsAMissAsTheZero),
		),
		ProvenReason: "an unwritten key is a miss",
	},
}

// --- Bodies -------------------------------------------------------------------

func readsBackWhatPutWrote(
	tb testing.TB, s generic.Store[string, int],
	_ generictest.StoreFixture[string, int],
) {
	tb.Helper()
	testkit.NoError(tb, s.Put(tb.Context(), seededKey, seededValue), "the key is written")

	got, err := s.Get(tb.Context(), seededKey)
	testkit.NoError(tb, err, "a written key is found")
	testkit.Equal(tb, got, seededValue, "and carries what was written")
}

func missIsReported(
	tb testing.TB, s generic.Store[string, int],
	_ generictest.StoreFixture[string, int],
) {
	tb.Helper()
	got, err := s.Get(tb.Context(), absentKey)
	testkit.Error(tb, err, "an unwritten key is a miss")
	testkit.Equal(tb, got, 0, "and the value beside it is the zero")
}

// --- Planted defects ----------------------------------------------------------

// fault names what one planted store gets wrong.
//
// Both are the same confusion a generic store invites: V's zero is a legal
// value, so a store answering it with no error leaves a caller unable to tell a
// stored zero from a key nobody wrote.
type fault int

const (
	// answersTheZero keeps the value and hands back V's zero.
	answersTheZero fault = iota

	// readsAMissAsTheZero reports success for a key nothing wrote, which
	// for V = int is indistinguishable from a stored 0.
	readsAMissAsTheZero
)

// planted builds the constructor for one broken store, at the instantiation
// this package runs.
func planted(wrong fault) func() *plantedStore {
	return func() *plantedStore {
		return &plantedStore{wrong: wrong, held: map[string]int{}}
	}
}

type plantedStore struct {
	wrong fault
	held  map[string]int
}

func (p *plantedStore) Put(_ context.Context, key string, value int) error {
	p.held[key] = value
	return nil
}

func (p *plantedStore) Get(_ context.Context, key string) (int, error) {
	value, held := p.held[key]
	if !held {
		if p.wrong == readsAMissAsTheZero {
			return 0, nil
		}
		return 0, generictest.ErrNotFound
	}
	if p.wrong == answersTheZero {
		return 0, nil
	}
	return value, nil
}
