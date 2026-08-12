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

	// Ptr marks a stateful law — one whose Check keeps memory across calls
	// behind a pointer receiver, so the composite literal must be addressed.
	// The gate holds it to the struct's method set by reflection: a value
	// type with no Check method is a law that must be bound by pointer.
	Ptr bool
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
	lawid.Cacheable:             {Type: "Cacheable", Args: []BindArg{BindKey, BindValue}},
	lawid.DefaultOnError:        {Type: "DefaultOnError", Args: []BindArg{BindKey, BindValue}},
	lawid.DeleteReturnsNotFound: {Type: "DeleteReturnsNotFound", Args: []BindArg{BindKey, BindValue}},
	lawid.PointInTime:           {Type: "PointInTime", Args: []BindArg{BindKey, BindValue}},
	lawid.ReadAfterWrite:        {Type: "ReadAfterWrite", Args: []BindArg{BindKey, BindValue}},
	lawid.Sticky:                {Type: "Sticky", Args: []BindArg{BindKey, BindValue}, Ptr: true},
	lawid.WriteObservable:       {Type: "WriteObservable", Args: []BindArg{BindValue, BindKey}},

	// The stream family. The hash argument is the value itself — the drained
	// values are comparable, so identity is the strongest fingerprint and the
	// only one nothing has to invent.
	lawid.StreamCompletion:   {Type: "StreamCompletion", Args: []BindArg{BindValue}},
	lawid.StreamNoDuplicates: {Type: "StreamNoDuplicates", Args: []BindArg{BindValue, BindValue}},
	lawid.StreamOverMatch:    {Type: "StreamOverMatch", Args: []BindArg{BindValue, BindValue}},
	lawid.StreamPermutation:  {Type: "StreamPermutation", Args: []BindArg{BindValue, BindValue}},
	lawid.StreamReentrant:    {Type: "StreamReentrancy", Args: []BindArg{BindValue}},
	lawid.StreamStableOrder:  {Type: "StreamStableOrder", Args: []BindArg{BindValue}},
}
