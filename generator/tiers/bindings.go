// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import (
	"slices"

	"go.thesmos.sh/testkit/core/lawid"
)

// Binding is how a generated file instantiates a rule's law: the exported
// struct in `engine/model/law`, and its type arguments after the subject.
//
// A column rather than a derivation because neither half is derivable. The
// type name carries word boundaries and renames the identifier flattened —
// AUTO-CURSOR-NEXT-AFTER-CLOSE is CursorNextAfterCloseSentinel — and the
// argument order is each struct's own: WriteObservable is [T, V, K] where
// ReadAfterWrite is [T, K, V], and a generator that guessed would produce a
// file that fails to compile in whichever corpus package armed it. The
// conformance gate holds every filled row to the shipped struct by reflection.
//
// The column fills as fixtures arm: a rule without one selects and is reported
// unbound by the generated header, never silently dropped.
type Binding struct {
	// Type is the law struct's exported identifier.
	Type string

	// Args are the type arguments after the subject, in the struct's own
	// declaration order.
	Args []BindArg
}

// BindArg names one type argument, resolved by the generator against the
// interface it is emitting for.
type BindArg string

// The argument vocabulary. Key and Value are the shared pool types — the same
// two every action draws from, which is what keeps a law's draws colliding
// with the sequences it runs beside.
const (
	BindKey   BindArg = "key"
	BindValue BindArg = "value"
)

// BindingFor returns the named law's instantiation spec, and whether the
// column carries one yet.
func BindingFor(law string) (Binding, bool) {
	b, ok := bindings[law]
	return b, ok
}

// Bound returns every law the column covers, sorted, for the gate that holds
// each row to the shipped struct.
func Bound() []string {
	out := make([]string, 0, len(bindings))
	for id := range bindings {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// bindings is the column, keyed by law identifier.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var bindings = map[string]Binding{
	lawid.WriteObservable: {Type: "WriteObservable", Args: []BindArg{BindValue, BindKey}},
}
