// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package failure_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/failure"
)

func TestKindString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		kind failure.Kind
		name string
	}{
		{failure.KindUnclassified, "unclassified"},
		{failure.KindStructural, "structural"},
		{failure.KindSemantic, "semantic"},
		{failure.KindInvariant, "invariant"},
		{failure.KindLiveness, "liveness"},
		{failure.KindDivergence, "divergence"},
		{failure.KindReplayMismatch, "replay-mismatch"},
		{failure.KindChaosCrash, "chaos-crash"},
		{failure.KindBudgetExceeded, "budget-exceeded"},
	}

	t.Run("known kinds round-trip names", func(t *testing.T) {
		t.Parallel()
		for _, c := range cases {
			testkit.Equal(t, c.kind.String(), c.name, "kind name")
		}
	})

	t.Run("unknown kinds render as unknown(N)", func(t *testing.T) {
		t.Parallel()
		k := failure.Kind(99)
		testkit.Equal(t, k.String(), "unknown(99)", "unknown rendering")
	})
}

func TestParseKind(t *testing.T) {
	t.Parallel()

	t.Run("parses every known name", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{
			"unclassified", "structural", "semantic", "invariant",
			"liveness", "divergence", "replay-mismatch", "chaos-crash",
			"budget-exceeded",
		} {
			k, err := failure.ParseKind(name)
			testkit.NoError(t, err, "parse "+name)
			testkit.Equal(t, k.String(), name, "round-trip "+name)
		}
	})

	t.Run("rejects unknown names", func(t *testing.T) {
		t.Parallel()
		_, err := failure.ParseKind("not-a-kind")
		testkit.True(t, err != nil, "must error on unknown name")
		testkit.Assert(t, err.Error()).Contains("not-a-kind", "diagnostic cites name")
	})
}
