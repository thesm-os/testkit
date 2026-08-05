// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal tests: the integrity check can only be exercised against a chain
// whose stored entries and hashes disagree, and neither field is reachable
// through the public API — which is the point. A caller cannot corrupt the
// chain by accident, so simulating corruption means reaching in.
package ref

import (
	"errors"
	"strings"
	"testing"
)

type tamperEntry struct {
	ID   string
	Data string
}

func TestAppendOnlyVerifyDetectsTampering(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T) *AppendOnly[tamperEntry] {
		t.Helper()
		c := NewAppendOnly[tamperEntry](nil)
		for _, e := range []tamperEntry{{ID: "a", Data: "1"}, {ID: "b", Data: "2"}} {
			if err := c.Append(t.Context(), e); err != nil {
				t.Fatalf("setup append: %v", err)
			}
		}
		return c
	}

	t.Run("an intact chain verifies", func(t *testing.T) {
		t.Parallel()
		if err := build(t).Verify(t.Context()); err != nil {
			t.Fatalf("an untouched chain must verify: %v", err)
		}
	})

	// Rewriting an entry without recomputing its hash is exactly what an
	// out-of-band edit looks like, and is what the chain exists to detect.
	t.Run("an edited entry is detected", func(t *testing.T) {
		t.Parallel()
		c := build(t)
		c.entries[0].Data = "tampered"

		err := c.Verify(t.Context())
		if err == nil {
			t.Fatal("an entry edited behind the chain's back must be detected")
		}
		if !errors.Is(err, ErrChainIntegrity) {
			t.Fatalf("the failure must carry the integrity sentinel, got: %v", err)
		}
	})

	t.Run("the diagnostic names the offending index", func(t *testing.T) {
		t.Parallel()
		c := build(t)
		c.entries[1].Data = "tampered"

		err := c.Verify(t.Context())
		if err == nil {
			t.Fatal("a tampered entry must be detected")
		}
		if got := err.Error(); !strings.Contains(got, "entry 1") {
			t.Fatalf("the diagnostic must locate the mismatch, got: %s", got)
		}
	})
}

// Replay copies under lock and yields outside it, so a consumer that stops
// early must not leave the chain wedged — and must not receive the remainder.
func TestAppendOnlyReplayStopsEarly(t *testing.T) {
	t.Parallel()

	c := NewAppendOnly[tamperEntry](nil)
	for _, e := range []tamperEntry{{ID: "a"}, {ID: "b"}, {ID: "c"}} {
		if err := c.Append(t.Context(), e); err != nil {
			t.Fatalf("setup append: %v", err)
		}
	}

	seen := 0
	for _, err := range c.Replay(t.Context()) {
		if err != nil {
			t.Fatalf("replay: %v", err)
		}
		seen++
		break
	}
	if seen != 1 {
		t.Fatalf("the consumer stopped after one entry, got %d", seen)
	}

	// The chain must still be usable after an abandoned replay.
	if err := c.Append(t.Context(), tamperEntry{ID: "d"}); err != nil {
		t.Fatalf("an abandoned replay must not wedge the chain: %v", err)
	}
	if got := c.Len(); got != 4 {
		t.Fatalf("expected 4 entries after the follow-up append, got %d", got)
	}
}

// The partitioned chain's only job on Verify is to attribute a failure to the
// partition that carries it, which means corrupting one partition and leaving
// the others intact.
func TestPartitionedVerifyNamesTheFailingPartition(t *testing.T) {
	t.Parallel()

	p := NewPartitionedAppendOnly(func(e tamperEntry) string { return e.ID }, nil)
	for _, e := range []tamperEntry{{ID: "a", Data: "1"}, {ID: "b", Data: "2"}} {
		if err := p.Append(t.Context(), e); err != nil {
			t.Fatalf("setup append: %v", err)
		}
	}
	if err := p.Verify(t.Context()); err != nil {
		t.Fatalf("an untouched partition set must verify: %v", err)
	}

	p.chains["b"].entries[0].Data = "tampered"

	err := p.Verify(t.Context())
	if err == nil {
		t.Fatal("a tampered partition must be detected")
	}
	if !errors.Is(err, ErrChainIntegrity) {
		t.Fatalf("the failure must carry the integrity sentinel, got: %v", err)
	}
	if got := err.Error(); !strings.Contains(got, "partition b") {
		t.Fatalf("the diagnostic must name the partition, got: %s", got)
	}
}
