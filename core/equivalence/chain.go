// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package equivalence

import (
	"github.com/google/go-cmp/cmp"
)

// Chain composes a sequence of [Relation] values into one
// comparison run. Each relation contributes [cmp.Option] values;
// go-cmp's FilterPath mechanism routes per-path options to the
// fields they apply to. Paths not covered by any relation fall
// through to go-cmp's default deep equality, matching what
// [Strict] would produce alone.
//
// The Chain is immutable post-construction; [Chain.Add] returns the
// receiver for fluent chaining but appends in place. Callers that
// want a forked variant build a new Chain.
type Chain struct {
	relations []Relation
}

// NewChain constructs an empty Chain. Relations are added with
// [Chain.Add]; the empty Chain comparison falls through to go-cmp's
// default deep equality.
func NewChain() *Chain {
	return &Chain{}
}

// Add appends r to the Chain and returns the receiver for fluent
// chaining:
//
//	chain := equivalence.NewChain().
//	    Add(equivalence.IgnoreFields(reflect.TypeOf(Entry{}), "WrittenAt")).
//	    Add(equivalence.IDField(reflect.TypeOf(Entry{}), "EntryID"))
func (c *Chain) Add(r Relation) *Chain {
	c.relations = append(c.relations, r)
	return c
}

// Relations returns the relations in chain order. The returned
// slice is a copy; callers may not mutate it to alter the Chain.
func (c *Chain) Relations() []Relation {
	out := make([]Relation, len(c.relations))
	copy(out, c.relations)
	return out
}

// Equal reports whether a and b compare equal under every relation
// in the Chain. Returns true when go-cmp finds no diff under the
// composed option set.
func (c *Chain) Equal(a, b any) bool {
	return cmp.Equal(a, b, c.options()...)
}

// Diff returns a human-readable diff between a and b under the
// composed option set, or the empty string when they compare equal.
// Output is go-cmp's standard `(-want +got)` format.
func (c *Chain) Diff(a, b any) string {
	return cmp.Diff(a, b, c.options()...)
}

// options collects the cmp.Option values from every relation in
// chain order.
func (c *Chain) options() []cmp.Option {
	var opts []cmp.Option
	for _, r := range c.relations {
		opts = append(opts, r.Options()...)
	}
	return opts
}
