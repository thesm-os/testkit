// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"strconv"
	"testing"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/linearize"
)

func TestPartitionByDirective(t *testing.T) {
	t.Parallel()

	t.Run("partitions Counter ops by the supplied field", func(t *testing.T) {
		t.Parallel()
		// Wrap Counter with a partition fn that buckets by int %2 of args.
		// Counter is normally single-partition; the override makes
		// even-args one partition and odd-args another, so concurrent
		// Inc(1) and Inc(2) run independently.
		base := linearize.Counter()
		m := linearize.PartitionByDirective(base, func(args any) string {
			n, ok := args.(int)
			if !ok {
				return ""
			}
			return strconv.Itoa(n % 2)
		})

		// Same partition (both odd): chain. Counter advances by args,
		// so Inc(1) from 0 → 1, Inc(3) from 1 → 4.
		history := []porcupine.Operation{
			opCAS(0, "Inc", "ignored", 1, int64(1)),
			opCAS(1, "Inc", "ignored", 3, int64(4)),
		}
		testkit.True(t, porcupine.CheckOperations(m, history),
			"same-partition chain accepted")
	})

	t.Run("operations in different partitions run independently", func(t *testing.T) {
		t.Parallel()
		base := linearize.Counter()
		m := linearize.PartitionByDirective(base, func(args any) string {
			n, ok := args.(int)
			if !ok {
				return ""
			}
			if n%2 == 0 {
				return "even"
			}
			return "odd"
		})
		// Each partition has its own Counter state starting at 0.
		// "odd" sees Inc(1) → 1; "even" sees Inc(2) → 2.
		history := []porcupine.Operation{
			opCAS(0, "Inc", "ignored", 1, int64(1)),
			opCAS(1, "Inc", "ignored", 2, int64(2)),
		}
		testkit.True(t, porcupine.CheckOperations(m, history),
			"independent partitions")
	})

	t.Run("invalid op in any partition still rejects", func(t *testing.T) {
		t.Parallel()
		base := linearize.Counter()
		m := linearize.PartitionByDirective(base, func(_ any) string { return "p1" })
		history := []porcupine.Operation{
			opCAS(0, "Inc", "ignored", nil, int64(99)), // wrong result
		}
		testkit.False(t, porcupine.CheckOperations(m, history),
			"per-partition invariants still enforced")
	})
}
