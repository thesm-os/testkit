// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import "strings"

// PoolPlan is one role's derived pool: the three members every drawn
// position of that role cycles through, each with a named origin —
// pool[0] the field's default stamp verbatim, pool[1] the
// distinctness transform, pool[2] the hostile member. Three because
// that is the least a pool can hold and still fund a miss (a value
// nothing wrote), an overwrite (a second value), and the hostile
// coverage real drivers die on.
type PoolPlan struct {
	// Role is the //testkit:role stamp the pool serves ("key",
	// "payload").
	Role string

	// Field is the emitted config field's identifier ("KeyPool"),
	// derived from the stamped field's own name through the naming
	// policy.
	Field string

	// Members are the three rendered member expressions, in origin
	// order.
	Members [3]Expr
}

// The pool member transforms — the textual policies pool[1] and
// pool[2] are derived by. Text policies rather than value policies,
// because a default stamp is Go source and stays Go source all the
// way to the emitted file; the transforms never parse what they can
// splice.

// distinctSwap is the textual payload convention the distinctness
// transform pivots on: the corpus's defaults spell "test-*", and the
// second member swaps the word so a miss has a key nothing wrote.
const (
	distinctFrom = "test"
	distinctTo   = "other"
)

// hostilePrefix opens the hostile string member: a NUL byte and an
// invalid UTF-8 sequence, spelled as escape sequences so the emitted
// literal carries them and the generator's own source does not.
const hostilePrefix = `"\x00hostile\xff`

// DistinctMember is pool[1]: the stamp with its textual payload
// swapped "test" → "other". False when the stamp carries no swap
// point — the pool cannot fund a distinct second member, and the
// caller refuses the derivation rather than emitting two equal
// members for [suite.DistinctPool] to reject at every consumer's run.
func DistinctMember(stamp Expr) (Expr, bool) {
	out := strings.Replace(string(stamp), distinctFrom, distinctTo, 1)
	if out == string(stamp) {
		return "", false
	}
	return Expr(out), true
}

// HostileMember is pool[2]: for a quoted-string stamp, the
// NUL/invalid-UTF-8 member suffixed with the role word; for a
// composite stamp, its first string literal emptied — an empty
// payload is the hostile a struct can always hold. False where
// neither form applies, or where the literal carries escapes this
// splice would mangle.
func HostileMember(stamp Expr, role string) (Expr, bool) {
	s := string(stamp)
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) && len(s) >= 2 {
		return Expr(hostilePrefix + role + `"`), true
	}
	open := strings.Index(s, `"`)
	if open < 0 {
		return "", false
	}
	closing := strings.Index(s[open+1:], `"`)
	if closing < 0 {
		return "", false
	}
	if strings.Contains(s[open+1:open+1+closing], `\`) {
		// An escaped literal cannot be spliced by index without
		// parsing it; refusing keeps the transform honest.
		return "", false
	}
	return Expr(s[:open+1] + s[open+1+closing:]), true
}
