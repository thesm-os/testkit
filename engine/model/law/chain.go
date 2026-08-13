// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law

import (
	"fmt"
	"iter"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/engine/model/history"
)

// drain collects all entries from an iter.Seq2, stopping on first error.
func drain[Entry any](seq iter.Seq2[Entry, error]) ([]Entry, error) {
	var out []Entry
	for e, err := range seq {
		if err != nil {
			return out, err
		}
		out = append(out, e)
	}
	return out, nil
}

// AppendOnlyHistoryGrows checks that the chain grows monotonically:
// after each action, the replay sequence must be a prefix-extension
// of the prior replay per partition. No entries may be deleted or modified.
//
// Implements [StatefulLaw] — tracks prior snapshots across actions.
type AppendOnlyHistoryGrows[T any, K comparable, Entry any] struct {
	Replay     func(rt *rapid.T, impl T, partKey K) iter.Seq2[Entry, error]
	Partitions func() []K
	prior      map[K][]Entry
}

// ID returns the stable identifier for this law.
func (*AppendOnlyHistoryGrows[T, K, Entry]) ID() string { return lawid.AppendOnlyGrows }

// REQID returns an empty string (auto-derived).
func (*AppendOnlyHistoryGrows[T, K, Entry]) REQID() string { return "" }

// Check delegates to CheckWithStep.
func (l *AppendOnlyHistoryGrows[T, K, Entry]) Check(rt *rapid.T, sut, ref T) error {
	return l.CheckWithStep(rt, sut, ref, -1)
}

// Reset clears the prior snapshots — [Resettable]. The step-zero guard in
// CheckWithStep covers the common path; this covers a run whose first check
// lands past step zero, and keeps the stateful contract uniform.
func (l *AppendOnlyHistoryGrows[T, K, Entry]) Reset() { l.prior = nil }

// CheckWithStep verifies the chain growth invariant per partition.
func (l *AppendOnlyHistoryGrows[T, K, Entry]) CheckWithStep(rt *rapid.T, sut, _ T, step int) error {
	if step == 0 || l.prior == nil {
		l.prior = make(map[K][]Entry)
	}
	for _, partKey := range l.Partitions() {
		current, err := drain(l.Replay(rt, sut, partKey))
		if err != nil {
			//nolint:errorlint // diagnostic
			return fmt.Errorf("AppendOnlyHistoryGrows[%v]: replay error: %v", partKey, err)
		}
		prior := l.prior[partKey]
		if len(current) < len(prior) {
			return fmt.Errorf("AppendOnlyHistoryGrows[%v]: chain shrank from %d to %d",
				partKey, len(prior), len(current))
		}
		for i, p := range prior {
			if !cmp.Equal(p, current[i]) {
				return fmt.Errorf("AppendOnlyHistoryGrows[%v]: entry %d mutated", partKey, i)
			}
		}
		l.prior[partKey] = current
	}
	return nil
}

// AppendOnlyNoDrops checks that every successfully-appended entry
// appears in the SUT's replay (membership check, not exact multiplicity).
// Uses the [history.History] trace to know what was attempted.
// Catches silent drops that pure replay comparison cannot detect
// (SUT owns the read surface). Chains with idempotent retry / dedup
// semantics will not false-positive — presence is checked, not count.
type AppendOnlyNoDrops[T any, K comparable, Entry any] struct {
	Replay  func(rt *rapid.T, impl T, partKey K) iter.Seq2[Entry, error]
	History *history.History[K, Entry]
}

// ID returns the stable identifier for this law.
func (AppendOnlyNoDrops[T, K, Entry]) ID() string { return lawid.AppendOnlyNoDrops }

// REQID returns an empty string (auto-derived).
func (AppendOnlyNoDrops[T, K, Entry]) REQID() string { return "" }

// Check verifies no drops per partition (membership, not multiplicity).
func (l AppendOnlyNoDrops[T, K, Entry]) Check(rt *rapid.T, sut, _ T) error {
	// Partitions only names keys the history has recorded against, so every
	// snapshot here holds at least one attempted append.
	for _, partKey := range l.History.Partitions() {
		attempted := l.History.Snapshot(partKey)
		replayed, err := drain(l.Replay(rt, sut, partKey))
		if err != nil {
			//nolint:errorlint // diagnostic
			return fmt.Errorf("AppendOnlyNoDrops[%v]: replay error: %v", partKey, err)
		}
		seen := make(map[string]bool, len(replayed))
		for _, e := range replayed {
			seen[fmt.Sprint(e)] = true
		}
		for _, want := range attempted {
			if !seen[fmt.Sprint(want)] {
				return fmt.Errorf("AppendOnlyNoDrops[%v]: attempted append %v missing from replay",
					partKey, want)
			}
		}
	}
	return nil
}

// HashChainIntegrityViaVerify checks chain integrity by calling an
// explicit Verify method after each action.
type HashChainIntegrityViaVerify[T any] struct {
	Verify func(*rapid.T, T) error
}

// ID returns the stable identifier for this law.
//
// Distinct from [HashChainIntegrityViaErr.ID] even though both prove chain
// integrity: the identifier is what SkipByID matches, what the ran/fired
// counters index, and what a failure report prints as LawID. Sharing one
// across two laws makes a skip hit both and a report unable to say which
// fired.
func (HashChainIntegrityViaVerify[T]) ID() string { return lawid.HashChainIntegrityVerify }

// REQID returns an empty string (auto-derived).
func (HashChainIntegrityViaVerify[T]) REQID() string { return "" }

// Check calls Verify on both SUT and ref; both must return nil.
//
// # What this does not check
//
// Detection. Nothing here corrupts the chain, so a run proves only that the
// subject reports intact through the operations the sequences drove — which
// is a real claim, and a real failure for a chain that breaks its own links
// under append, but it is not evidence that tampering would be noticed.
//
// That claim is [TamperEvident]'s, which carries the corruption step this
// one deliberately lacks: a law that both tampers and verifies would make
// the two indistinguishable in a report, and the tamper is the half a
// consumer must supply because only they know what corrupting their storage
// looks like.
func (l HashChainIntegrityViaVerify[T]) Check(rt *rapid.T, sut, ref T) error {
	sutErr := l.Verify(rt, sut)
	refErr := l.Verify(rt, ref)
	if sutErr != nil {
		//nolint:errorlint // diagnostic
		return fmt.Errorf("HashChainIntegrity: SUT verify failed: %v", sutErr)
	}
	if refErr != nil {
		//nolint:errorlint // diagnostic
		return fmt.Errorf("HashChainIntegrity: ref verify failed: %v", refErr)
	}
	return nil
}

// HashChainIntegrityViaErr checks chain integrity using the
// PoisonAccessor Err() method.
type HashChainIntegrityViaErr[T any] struct {
	Err func(T) error
}

// ID returns the stable identifier for this law. See
// [HashChainIntegrityViaVerify.ID] for why the two are not the same string.
func (HashChainIntegrityViaErr[T]) ID() string { return lawid.HashChainIntegrityErr }

// REQID returns an empty string (auto-derived).
func (HashChainIntegrityViaErr[T]) REQID() string { return "" }

// Check calls Err() on both SUT and ref.
func (l HashChainIntegrityViaErr[T]) Check(_ *rapid.T, sut, ref T) error {
	sutErr := l.Err(sut)
	refErr := l.Err(ref)
	if sutErr != nil {
		//nolint:errorlint // diagnostic
		return fmt.Errorf("HashChainIntegrity: SUT poisoned: %v", sutErr)
	}
	if refErr != nil {
		//nolint:errorlint // diagnostic
		return fmt.Errorf("HashChainIntegrity: ref poisoned: %v", refErr)
	}
	return nil
}

// ReplayDeterminism checks that two consecutive Replay calls on the
// same chain state return identical sequences per partition.
type ReplayDeterminism[T any, K comparable, Entry any] struct {
	Replay     func(rt *rapid.T, impl T, partKey K) iter.Seq2[Entry, error]
	Partitions func() []K
}

// ID returns the stable identifier for this law.
func (ReplayDeterminism[T, K, Entry]) ID() string { return lawid.ReplayDeterministic }

// REQID returns an empty string (auto-derived).
func (ReplayDeterminism[T, K, Entry]) REQID() string { return "" }

// Check replays twice per partition on the SUT and compares.
func (l ReplayDeterminism[T, K, Entry]) Check(rt *rapid.T, sut, _ T) error {
	for _, partKey := range l.Partitions() {
		first, err1 := drain(l.Replay(rt, sut, partKey))
		if err1 != nil {
			//nolint:errorlint // diagnostic
			return fmt.Errorf("ReplayDeterminism[%v]: first replay error: %v", partKey, err1)
		}
		second, err2 := drain(l.Replay(rt, sut, partKey))
		if err2 != nil {
			//nolint:errorlint // diagnostic
			return fmt.Errorf("ReplayDeterminism[%v]: second replay error: %v", partKey, err2)
		}
		if len(first) != len(second) {
			return fmt.Errorf("ReplayDeterminism[%v]: first has %d, second has %d",
				partKey, len(first), len(second))
		}
		for i := range first {
			if !cmp.Equal(first[i], second[i]) {
				return fmt.Errorf("ReplayDeterminism[%v]: entry %d differs", partKey, i)
			}
		}
	}
	return nil
}

// ReplayRespectsCausality checks that the SUT's replay order respects
// happens-before: for each entry in the replay, all entries it depends
// on must appear earlier. Walks entries across all partitions in
// Partitions() enumeration order (which is deterministic via History).
type ReplayRespectsCausality[T any, K comparable, Entry any] struct {
	Replay     func(rt *rapid.T, impl T, partKey K) iter.Seq2[Entry, error]
	Partitions func() []K
	EntryID    func(Entry) string
	DependsOn  func(Entry) []string
}

// ID returns the stable identifier for this law.
func (ReplayRespectsCausality[T, K, Entry]) ID() string { return lawid.ReplayCausalOrdering }

// REQID returns an empty string (auto-derived).
func (ReplayRespectsCausality[T, K, Entry]) REQID() string { return "" }

// Check walks the SUT's replay and verifies deps appear before referents.
func (l ReplayRespectsCausality[T, K, Entry]) Check(rt *rapid.T, sut, _ T) error {
	seen := make(map[string]bool)
	for _, partKey := range l.Partitions() {
		entries, err := drain(l.Replay(rt, sut, partKey))
		if err != nil {
			//nolint:errorlint // diagnostic
			return fmt.Errorf("ReplayRespectsCausality[%v]: replay error: %v",
				partKey, err)
		}
		for _, e := range entries {
			for _, dep := range l.DependsOn(e) {
				if !seen[dep] {
					return fmt.Errorf(
						"ReplayRespectsCausality[%v]: entry %s depends on %s"+
							" which is missing or appears later in replay",
						partKey, l.EntryID(e), dep,
					)
				}
			}
			seen[l.EntryID(e)] = true
		}
	}
	return nil
}
