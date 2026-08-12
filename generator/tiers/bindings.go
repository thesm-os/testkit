// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

import (
	"slices"
	"strings"

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
//
// Two bare spellings and four field-qualified ones. The bare pair names the
// shared pools; the qualified forms name a manifest field of the same rule
// and read a type off the method that fills it — which is the only place the
// type exists, since a law like Roundtrip is instantiated at its forward
// role's result and no pool ever draws one.
type BindArg string

// The argument vocabulary. Key and Value are the shared pool types — the same
// two every action draws from, which is what keeps a law's draws colliding
// with the sequences it runs beside. Observation is the composed whole-state
// observation's type — the same derivation the Observe handle renders.
// Partition is the replay partition key: the projection the partition mixin
// names where one is declared, the single anonymous partition otherwise.
const (
	BindKey         BindArg = "key"
	BindValue       BindArg = "value"
	BindObservation BindArg = "observation"
	BindPartition   BindArg = "partition"
)

// The field-qualified prefixes, composed by the constructors below.
const (
	bindResultPrefix = "result:"
	bindInputPrefix  = "input:"
	bindElemPrefix   = "elem:"
	bindScalarPrefix = "scalar:"
)

// ResultOf instantiates at the first non-error result type of the method the
// named manifest field resolves to.
func ResultOf(field string) BindArg { return BindArg(bindResultPrefix + field) }

// InputOf instantiates at the first non-context parameter type of the method
// the named manifest field resolves to.
func InputOf(field string) BindArg { return BindArg(bindInputPrefix + field) }

// ElemOf instantiates at the element type of the stream the named field's
// method drains — a slice's element, or an iterator's yielded value.
func ElemOf(field string) BindArg { return BindArg(bindElemPrefix + field) }

// ScalarOf instantiates at the named field's scalar observation: the method's
// numeric result where it has one, int where the observation is the length of
// a returned slice — the same adaptation the field's closure renders.
func ScalarOf(field string) BindArg { return BindArg(bindScalarPrefix + field) }

// Qualifier splits a field-qualified argument into its form and the manifest
// field it names, false for the bare pool spellings.
func (a BindArg) Qualifier() (form, field string, ok bool) {
	s := string(a)
	for _, prefix := range []string{bindResultPrefix, bindInputPrefix, bindElemPrefix, bindScalarPrefix} {
		if rest, found := strings.CutPrefix(s, prefix); found {
			return strings.TrimSuffix(prefix, ":"), rest, true
		}
	}
	return "", "", false
}

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
// A law absent here is one no generated closure can compose — a supplied
// comparator, a handle the runner does not offer, a role shape no fixture
// can declare — and the assertion gate's register carries its reason. A law
// present here can still refuse per fixture; the difference is that the
// refusal names the missing piece instead of the missing row.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var bindings = map[string]Binding{
	lawid.Cacheable:             {Type: "Cacheable", Args: []BindArg{BindKey, ResultOf(fieldRead)}},
	lawid.DefaultOnError:        {Type: "DefaultOnError", Args: []BindArg{BindKey, BindValue}},
	lawid.DeleteReturnsNotFound: {Type: "DeleteReturnsNotFound", Args: []BindArg{BindKey, BindValue}},
	lawid.PointInTime:           {Type: "PointInTime", Args: []BindArg{BindKey, BindValue}},
	lawid.ReadAfterWrite:        {Type: "ReadAfterWrite", Args: []BindArg{BindKey, BindValue}},
	lawid.Sticky:                {Type: "Sticky", Args: []BindArg{BindKey, BindValue}, Ptr: true},
	lawid.WriteObservable:       {Type: "WriteObservable", Args: []BindArg{BindValue, BindKey}},

	// The stream family. The hash argument is the value itself — the drained
	// values are comparable, so identity is the strongest fingerprint and the
	// only one nothing has to invent. The two laws over the bare drain
	// instantiate at the drained element rather than the values pool: a
	// read-only stream declares no writer, so no pool ever draws its element.
	lawid.StreamCompletion:   {Type: "StreamCompletion", Args: []BindArg{ElemOf(fieldDrain)}},
	lawid.StreamNoDuplicates: {Type: "StreamNoDuplicates", Args: []BindArg{BindValue, BindValue}},
	lawid.StreamOverMatch:    {Type: "StreamOverMatch", Args: []BindArg{BindValue, BindValue}},
	lawid.StreamPermutation:  {Type: "StreamPermutation", Args: []BindArg{BindValue, BindValue}},
	lawid.StreamReentrant:    {Type: "StreamReentrancy", Args: []BindArg{ElemOf("Collect")}},
	lawid.StreamStableOrder:  {Type: "StreamStableOrder", Args: []BindArg{BindValue}},
	lawid.StreamReflectsMutations: {
		Type: "StreamReflectsMutations",
		Args: []BindArg{ElemOf(fieldDrain), ElemOf(fieldDrain)},
	},

	// The self-contained detector laws: the method under its own claim, no
	// second role in the room.
	lawid.AggregatorBounded:        {Type: "AggregatorBounded", Args: []BindArg{ScalarOf(fieldRead)}},
	lawid.CountEqualsReference:     {Type: "CountEqualsReference", Args: []BindArg{ScalarOf(fieldCount)}},
	lawid.LifecycleRespectsContext: {Type: "LifecycleRespectsContext", Args: []BindArg{}},
	lawid.MonotonicNonDecreasing:   {Type: "MonotonicNonDecreasing", Args: []BindArg{ScalarOf(fieldRead)}, Ptr: true},
	lawid.PoisonIdempotentRead:     {Type: "PoisonIdempotentRead", Args: []BindArg{}},
	lawid.PoisonNilOnFresh:         {Type: "PoisonNilOnFresh", Args: []BindArg{}},
	lawid.PredicateConsistent:      {Type: "PredicateConsistency", Args: []BindArg{}},
	lawid.PureDeterministic:        {Type: "PureDeterminism", Args: []BindArg{ResultOf(fieldCall)}},
	lawid.TotalOver:                {Type: "TotalOver", Args: []BindArg{InputOf(fieldCall), ResultOf(fieldCall)}},

	// The write-family laws: a mutation beside the observation that makes it
	// checkable, both spelled by the manifest's own fields.
	lawid.Associative:      {Type: "Associative", Args: []BindArg{InputOf("Apply"), BindObservation}},
	lawid.AtomicWrite:      {Type: "AtomicWrite", Args: []BindArg{InputOf(fieldWrite), BindObservation}},
	lawid.CommutativeWrite: {Type: "CommutativeWrite", Args: []BindArg{InputOf(fieldWrite), BindObservation}},
	lawid.Conservative:     {Type: "Conservative", Args: []BindArg{InputOf(fieldWrite)}},
	lawid.CRDTMerge:        {Type: "CRDTMerge", Args: []BindArg{InputOf(fieldWrite), BindObservation}},
	lawid.IdempotentWrite:  {Type: "IdempotentWrite", Args: []BindArg{InputOf(fieldWrite), BindObservation}},
	lawid.InjectionSafe:    {Type: "InjectionSafe", Args: []BindArg{}},
	lawid.XSSSafe:          {Type: "XSSSafe", Args: []BindArg{}},

	// The contract laws whose roles a generated closure can call directly.
	lawid.AppenderMonotonicOffsets: {
		Type: "AppenderMonotonicOffsets",
		Args: []BindArg{InputOf("Append"), ResultOf("Append")},
		Ptr:  true,
	},
	lawid.CASAtomicOneWinner:       {Type: "CASAtomicOneWinner", Args: []BindArg{InputOf("CAS")}},
	lawid.LeaseDoubleAcquireBlocks: {Type: "LeaseDoubleAcquireBlocks", Args: []BindArg{BindKey}},
	lawid.LeakFree:                 {Type: "LeakFree", Args: []BindArg{}},
	lawid.PersisterRetrievable:     {Type: "PersisterRetrievable", Args: []BindArg{InputOf("Save"), BindKey}},
	lawid.Roundtrip:                {Type: "Roundtrip", Args: []BindArg{ResultOf("Forward")}},
	lawid.LossyRoundtrip:           {Type: "LossyRoundtrip", Args: []BindArg{ResultOf("Forward")}},
	lawid.UpdaterReplaces:          {Type: "UpdaterReplaces", Args: []BindArg{InputOf("Update"), BindKey}},
	lawid.UpserterIdempotent:       {Type: "UpserterIdempotent", Args: []BindArg{InputOf("Upsert"), BindKey}},
	lawid.ValidTransition:          {Type: "ValidTransition", Args: []BindArg{InputOf(fieldWrite), BindObservation}},

	// The chain family, over a slice-replaying log: the replay adapts to the
	// iterator the law drains, and the partition set is the anonymous single
	// partition until a partition projection is declared.
	lawid.AppendOnlyGrows: {
		Type: "AppendOnlyHistoryGrows",
		Args: []BindArg{BindPartition, ElemOf(fieldReplay)},
		Ptr:  true,
	},
	lawid.HashChainIntegrityVerify: {Type: "HashChainIntegrityViaVerify", Args: []BindArg{}},
	lawid.ReplayDeterministic: {
		Type: "ReplayDeterminism",
		Args: []BindArg{BindPartition, ElemOf(fieldReplay)},
	},
}
