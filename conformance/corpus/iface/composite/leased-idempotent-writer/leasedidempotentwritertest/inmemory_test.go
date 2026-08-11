// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package leasedidempotentwritertest_test

import (
	"context"
	"sync"
	"testing"

	"go.thesmos.sh/testkit"
	leasedidempotentwriter "go.thesmos.sh/testkit/conformance/corpus/iface/composite/leased-idempotent-writer"
	"go.thesmos.sh/testkit/conformance/corpus/iface/composite/leased-idempotent-writer/leasedidempotentwritertest"
)

// The two classifications make opposite demands of Acquire, and the fixture
// exists because a generator reading them independently produces a suite that
// fails a correct implementation.
//
// `lease` is the model tier's under ADR-0018 and so is `idempotent`, so neither
// generates a check today — which means nothing here would notice the conflict.
// What the extension point can carry is what a single subject can be asked, and
// [TestEveryCheckRejectsAWrongLease] names the implementations each check is
// there to reject. The rest — that a release frees one key and not the others —
// needs a second holder and is a package test, because a lease is refused only
// when somebody else has it.
func TestLeasedWriterContract(t *testing.T) {
	t.Parallel()

	leasedidempotentwritertest.AssertLeasedWriterContract(
		t,
		leasedidempotentwritertest.LeasedWriterSubject(
			"in-memory",
			func() leasedidempotentwriter.LeasedWriter {
				return leasedidempotentwritertest.NewInMemory()
			},
		),
		leasedidempotentwritertest.LeasedWriterSubject(
			"in-memory, contended",
			func() leasedidempotentwriter.LeasedWriter {
				// A lease is refused only when somebody else holds it, and a check
				// receives one subject — so the refusal is reached by handing the
				// run a holder whose registry another already has the key in. The
				// seed then fails for this subject, which LeasedWriterSeed answers.
				r := leasedidempotentwritertest.NewRegistry()
				incumbent := r.Holder()
				if err := incumbent.Acquire(t.Context(), contendedKey); err != nil {
					panic("leasedidempotentwritertest_test: seating the incumbent: " + err.Error())
				}
				return r.Holder()
			},
		),
		leasedidempotentwritertest.LeasedWriterOnAcquire("loses a key another holder took", func(
			tb testing.TB, subject leasedidempotentwriter.LeasedWriter, _ string,
		) {
			tb.Helper()
			// True of the contended subject and vacuous for the lone one, which
			// is the shape a two-subject claim takes when only one of them can
			// be in the losing state.
			if err := subject.Acquire(tb.Context(), contendedKey); err != nil {
				testkit.ErrorIs(tb, err, leasedidempotentwriter.ErrHeld,
					"an acquire that loses says who to")
				return
			}
			testkit.NoError(tb, subject.Release(tb.Context(), contendedKey),
				"and one that wins can give it back")
		}),
		leasedidempotentwritertest.LeasedWriterOnAcquire(
			"repeats without unbalancing the lease", repeatsWithoutUnbalancing,
		),
		leasedidempotentwritertest.LeasedWriterOnRelease(
			"tolerates a key nobody holds", toleratesAnUnheldKey,
		),
	)
}

// contendedKey is the key the contended subject's registry already holds.
//
// Distinct from the fixture's derived key, so the harness can still seed every
// subject through Acquire: a contended subject that refused the seed would fail
// every check before reaching the one it exists for.
const contendedKey = "held-by-another"

// repeatsWithoutUnbalancing is the whole of the composite.
//
// The seed already acquired this key, so these are the repeats `idempotent`
// asks for — and one release still has to settle them, which is what `lease`
// asks for. The implementation it rejects is a plain lease: one that refuses
// the second acquire, which is correct for the contract alone and wrong for the
// pair.
func repeatsWithoutUnbalancing(
	tb testing.TB, subject leasedidempotentwriter.LeasedWriter, key string,
) {
	tb.Helper()
	testkit.NoError(tb, subject.Acquire(tb.Context(), key), "re-acquiring is a no-op")
	testkit.NoError(tb, subject.Acquire(tb.Context(), key), "however often it happens")

	testkit.NoError(tb, subject.Release(tb.Context(), key), "one release settles them")
	testkit.NoError(tb, subject.Acquire(tb.Context(), key),
		"and the key is free again rather than still held")
}

// toleratesAnUnheldKey holds the shutdown path usable.
//
// A caller deferring Release and returning early on a failed Acquire is
// ordinary Go, and the implementation this rejects is one strict about it —
// which reports a failure to give up something never taken, on every path that
// did not get the lease.
func toleratesAnUnheldKey(
	tb testing.TB, subject leasedidempotentwriter.LeasedWriter, key string,
) {
	tb.Helper()
	testkit.NoError(tb, subject.Release(tb.Context(), key+"-untaken"),
		"releasing what was never held is not a failure")
}

// plainLease is the lease contract without the mixin: correct on its own, and
// wrong for a method carrying both.
type plainLease struct {
	mu   sync.Mutex
	held map[string]bool
}

func newPlainLease() *plainLease { return &plainLease{held: map[string]bool{}} }

func (s *plainLease) Acquire(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held[key] {
		return leasedidempotentwriter.ErrHeld
	}
	s.held[key] = true
	return nil
}

func (s *plainLease) Release(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.held, key)
	return nil
}

// strictLease refuses to release a key it does not hold, which takes down every
// caller whose deferred shutdown runs after a failed acquire.
type strictLease struct{ plainLease }

func (s *strictLease) Release(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.held[key] {
		return leasedidempotentwriter.ErrHeld
	}
	delete(s.held, key)
	return nil
}

// Each check rejects the implementation it exists to catch.
//
// Without this the two above are three NoErrors each, which a subject whose
// methods return nil satisfies — and the composite's whole question would be
// asked of nothing.
func TestEveryCheckRejectsAWrongLease(t *testing.T) {
	t.Parallel()

	t.Run("a plain lease refuses the repeat", func(t *testing.T) {
		t.Parallel()
		// The seed is the harness's, so it is restated here: the check assumes
		// the key is already held by this caller.
		subject := newPlainLease()
		testkit.NoError(t, subject.Acquire(t.Context(), "k"), "the seed takes the lease")

		got := testkit.Rejects(t, "a lease with no idempotence", func(tb testing.TB) {
			tb.Helper()
			repeatsWithoutUnbalancing(tb, subject, "k")
		})
		testkit.Assert(t, got).Contains("re-acquiring is a no-op",
			"rejected for the reason the composite is about")
	})

	t.Run("a strict lease refuses the release", func(t *testing.T) {
		t.Parallel()
		got := testkit.Rejects(t, "a lease strict about releasing", func(tb testing.TB) {
			tb.Helper()
			toleratesAnUnheldKey(tb, &strictLease{plainLease: *newPlainLease()}, "k")
		})
		testkit.Assert(t, got).Contains("releasing what was never held",
			"rejected for the reason the check is about")
	})
}

// Declining the double is separate from dropping a check.
func TestLeasedWriterContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	leasedidempotentwritertest.AssertLeasedWriterContract(
		t,
		leasedidempotentwritertest.LeasedWriterSubject(
			"in-memory",
			func() leasedidempotentwriter.LeasedWriter {
				return leasedidempotentwritertest.NewInMemory()
			},
		),
		leasedidempotentwritertest.LeasedWriterWithout("Acquire/smoke"),
		leasedidempotentwritertest.LeasedWriterWithoutDouble(),
	)
}
