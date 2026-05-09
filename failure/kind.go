// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package failure

import "fmt"

// Kind classifies a [Failure] for routing to the right per-kind
// reporter (Pillar 5 of the model design). Reporters render
// structural failures as chain walks, semantic failures as cmp.Diffs,
// invariant failures with cited law IDs and REQ tags, etc.
type Kind int

const (
	// KindUnclassified is the zero value. A Failure constructed
	// without explicit Kind surfaces as `[unclassified]` in CI
	// output — visible-but-unhelpful so producers fix the omission.
	KindUnclassified Kind = iota

	// KindStructural — chain hash mismatch, append-ordering
	// violation, linearizability check failure. Reporter walks the
	// causal chain to the first inconsistent point.
	KindStructural

	// KindSemantic — SUT vs reference mismatch detected in
	// per-iteration comparison. Reporter emits cmp.Diff and the
	// action sequence leading to divergence.
	KindSemantic

	// KindInvariant — auto-derived law or cross-method invariant
	// fired. Reporter cites the law ID, REQ tag, and state at
	// violation.
	KindInvariant

	// KindLiveness — deadlock, no-progress, goroutine leak. Reporter
	// emits goroutine stacks and identifies the suspected blocker.
	KindLiveness

	// KindDivergence — diff-rollout: control vs candidate output
	// differed beyond the equivalence chain. Reporter emits the
	// per-impl divergence report and snapshots.
	KindDivergence

	// KindReplayMismatch — replay: SUT output differed from the
	// recorded trace under the configured tolerance. Reporter
	// emits expected vs actual and the trace event where the
	// divergence first appeared.
	KindReplayMismatch

	// KindChaosCrash — chaos: panic, crash, or unrecoverable
	// failure under fault injection. Reporter emits the active
	// fault set and the load-bearing-fault extraction result.
	KindChaosCrash

	// KindBudgetExceeded — any layer: tick/event/fault budget
	// reached without progress. Reporter emits the budget
	// configuration and what consumed it.
	KindBudgetExceeded
)

// String returns the kind name. Returns "unknown(N)" for
// unrecognized values so JSON unmarshaling of a future-version
// Failure surfaces unrecognized kinds rather than silently
// remapping them.
func (k Kind) String() string {
	switch k {
	case KindUnclassified:
		return "unclassified" //nolint:goconst // duplication with ArtifactKind.String is intentional; both enums independently expose an unclassified zero

	case KindStructural:
		return "structural"
	case KindSemantic:
		return "semantic"
	case KindInvariant:
		return "invariant"
	case KindLiveness:
		return "liveness"
	case KindDivergence:
		return "divergence"
	case KindReplayMismatch:
		return "replay-mismatch"
	case KindChaosCrash:
		return "chaos-crash"
	case KindBudgetExceeded:
		return "budget-exceeded"
	default:
		return fmt.Sprintf("unknown(%d)", int(k))
	}
}

// ParseKind decodes a [Kind] name produced by [Kind.String]. Returns
// an error for unrecognized names — JSON unmarshaling of a Failure
// from a newer producer surfaces the mismatch rather than coercing
// to KindUnclassified.
func ParseKind(s string) (Kind, error) {
	switch s {
	case "unclassified":
		return KindUnclassified, nil
	case "structural":
		return KindStructural, nil
	case "semantic":
		return KindSemantic, nil
	case "invariant":
		return KindInvariant, nil
	case "liveness":
		return KindLiveness, nil
	case "divergence":
		return KindDivergence, nil
	case "replay-mismatch":
		return KindReplayMismatch, nil
	case "chaos-crash":
		return KindChaosCrash, nil
	case "budget-exceeded":
		return KindBudgetExceeded, nil
	default:
		return KindUnclassified, fmt.Errorf("failure: unknown Kind %q", s)
	}
}
