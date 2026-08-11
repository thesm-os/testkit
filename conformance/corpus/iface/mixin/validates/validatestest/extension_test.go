// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// The extension slot, driven the way a generated sibling drives it.
//
// It lives in this package because that is the only place it can: the slot is
// unexported, and a sibling reaches it by being generated beside the harness. A
// hand-written option standing in for that sibling is the whole fixture — what
// is under test is the harness's half of the arrangement, not the model tier's.
package validatestest

import (
	"sync/atomic"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates"
)

// countingExtension is the shape a generated sibling emits: an option that
// closes over its own configuration and appends one named body of checks.
func countingExtension(runs *atomic.Int64) MixedOption {
	return func(c *mixedConfig) {
		c.extensions = append(c.extensions, mixedContractExtension{
			name: "counting",
			run: func(t *testing.T, subject string, factory func() validates.Mixed, cfg *mixedConfig) {
				t.Helper()
				runs.Add(1)
				testkit.NotEqual(t, subject, "", "the run is told which subject it is on")
				testkit.True(t, factory() != nil, "and can build one")
			},
		})
	}
}

// TestExtensionRunsOncePerSubject pins the two facts the slot's cost depends on.
//
// Once per subject is what makes an extension's price linear in the subject
// count. Not through the double is the other half: the wrapped pass exists to
// prove the stand-in faithful and the per-method checks already do that, so a
// second pass here would double the most expensive tier to say nothing new.
//
// The count is read from a cleanup rather than after the call. Every subject
// runs in a parallel subtest, so [AssertMixedContract] returns before any of
// them has started, and an assertion on the next line would read zero.
func TestExtensionRunsOncePerSubject(t *testing.T) {
	t.Parallel()

	var runs atomic.Int64
	t.Cleanup(func() {
		testkit.Equal(t, runs.Load(), int64(2),
			"two subjects, one run each, and none through MixedStub")
	})

	AssertMixedContract(t,
		MixedSubject("a", func() validates.Mixed { return NewInMemory() }),
		MixedSubject("b", func() validates.Mixed { return NewInMemory() }),
		countingExtension(&runs),
	)
}

// TestExtensionIsDroppedByName holds an extension to the same exit every other
// check has.
//
// An extension reports under one path and everything it runs nests beneath it,
// so dropping the path drops the tier — which is the whole recourse for a
// subject that legitimately cannot satisfy one.
func TestExtensionIsDroppedByName(t *testing.T) {
	t.Parallel()

	var runs atomic.Int64
	t.Cleanup(func() {
		testkit.Equal(t, runs.Load(), int64(0),
			"MixedWithout declined it, so nothing ran")
	})

	AssertMixedContract(t,
		MixedSubject("a", func() validates.Mixed { return NewInMemory() }),
		countingExtension(&runs),
		MixedWithout("counting"),
	)
}
