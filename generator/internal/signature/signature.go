// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package signature projects a source interface method into the form a
// template renders.
//
// Four generators read interface methods — stub, suite, bench, model — and
// all four need the same answers: what a parameter is called when the source
// did not name it, which field of a recorded call a return maps to, whether
// the generated signature may carry the source's return names. Deriving those
// once means a change to the convention lands in one place rather than in
// four generators that were meant to agree.
//
// Types arrive as [emit.Ref] values rather than rendered strings. Import
// resolution belongs to the backend, and a projection that produced type
// strings would name packages the rendered file never imports.
//
// It is internal deliberately. These are conventions between testkit's own
// generators, not an API a consumer writes plugins against — publishing them
// would owe stability under docs/adr/0002 to a surface that exists to change
// as the generators learn.
package signature

import (
	"strconv"

	"go.thesmos.sh/eidos/core/naming"
	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
)

// ReceiverIdent is the identifier generated methods bind their receiver to.
// Exported because [NamedReturnsUsable] reasons about collisions against it,
// and a template that chose a different receiver would silently invalidate
// that guard.
const ReceiverIdent = "s"

// Param is one rendered parameter: the in-method identifier and its type,
// already lifted to an [emit.Ref] so `renderType` consumes it.
type Param struct {
	// Name is the identifier the generated body references.
	Name string

	// Type is the parameter's type in emit form.
	Type emit.Ref

	// Field is the exported field name a recorded-call struct uses.
	Field string
}

// Return is one rendered return slot.
//
// Name is the source's declared return name, empty when the signature did not
// name it. Field is always populated — a recorded-call struct needs a field
// name whether or not the source supplied one, so unnamed returns fall back
// to positional.
type Return struct {
	Name  string
	Type  emit.Ref
	Field string

	// Local is the identifier a generated body binds this return to when
	// capturing a delegate's result. It equals Name when the signature
	// declares named results, and is positional otherwise. Computed here
	// rather than in a template so the collision guard lives next to the rule
	// it enforces.
	Local string

	// Error reports whether this slot is the builtin error. Fault injection
	// writes to it and the generated suite asserts on it, so a double has to
	// know which slot it is rather than assuming the last one — a signature
	// returning `(error, bool)` is unusual but legal.
	Error bool
}

// Iterator names the two range-over-func shapes a generator treats specially,
// so a caller can branch on the shape without matching package and name.
type Iterator string

// The iterator shapes. NotIterator is the zero value, so a projection that
// never looked reads as "not one" rather than as an unhandled case.
const (
	NotIterator Iterator = ""

	// SeqIterator is `iter.Seq[V]` — a sequence of values that cannot fail
	// mid-iteration.
	SeqIterator Iterator = "seq"

	// Seq2Iterator is `iter.Seq2[K, V]`, including the `iter.Seq2[V, error]`
	// spelling Go uses for a sequence that can fail partway through.
	Seq2Iterator Iterator = "seq2"
)

// IteratorOf classifies a return type as a range-over-func sequence.
//
// Matched on the stdlib package path and name rather than on structure: a
// consumer's own two-parameter generic type is not a sequence, and treating
// it as one would emit helpers that do not compile.
func IteratorOf(t *node.TypeRef) Iterator {
	if t == nil || t.Package != "iter" {
		return NotIterator
	}
	switch {
	case t.Name == "Seq" && len(t.TypeArgs) == 1:
		return SeqIterator
	case t.Name == "Seq2" && len(t.TypeArgs) == 2:
		return Seq2Iterator
	}
	return NotIterator
}

// IteratorYieldsError reports whether a sequence's second type argument is
// the builtin error — the `iter.Seq2[V, error]` shape, which is the one where
// a generated helper can usefully append a terminal failure.
func IteratorYieldsError(t *node.TypeRef) bool {
	return IteratorOf(t) == Seq2Iterator && IsError(t.TypeArgs[1])
}

// IteratorElem returns the sequence's element type in emit form, or nil when
// the reference is not a sequence. For Seq2 this is the first type argument,
// which is the value a caller collects.
func IteratorElem(t *node.TypeRef) emit.Ref {
	if IteratorOf(t) == NotIterator {
		return nil
	}
	return golang.FromNode(t.TypeArgs[0])
}

// IteratorSecond returns a Seq2's second type argument in emit form — the
// error slot in the `iter.Seq2[V, error]` spelling — or nil for anything else.
func IteratorSecond(t *node.TypeRef) emit.Ref {
	if IteratorOf(t) != Seq2Iterator {
		return nil
	}
	return golang.FromNode(t.TypeArgs[1])
}

// IsError reports whether a source type reference is the builtin error.
//
// Matched on the unqualified name with no package, which is what a frontend
// records for a builtin. A consumer's own type named `error` in some package
// is a different type and correctly does not match.
func IsError(t *node.TypeRef) bool {
	return t != nil && t.Package == "" && t.Name == "error"
}

// ParamsOf lifts a method's parameters into rendered form.
//
// A parameter with no declared name gets a positional identifier so the
// generated body can reference it when recording the call. The recorded-call
// field name is the exported form of whichever identifier ends up in use.
func ParamsOf(m *node.Method) []Param {
	out := make([]Param, 0, len(m.Params))
	for i, p := range m.Params {
		name := p.Name
		if name == "" {
			name = "arg" + strconv.Itoa(i)
		}
		out = append(out, Param{
			Name:  name,
			Type:  golang.FromNode(p.Type),
			Field: naming.Pascal(name),
		})
	}
	return out
}

// ReturnsOf lifts a method's return slots into rendered form.
//
// Field is populated for every slot regardless of whether the source named
// it: a recorded-call struct needs one field per return. This is deliberately
// independent of [NamedReturnsUsable] — a signature that cannot carry names
// on the generated method still records every slot under a readable name.
//
// The fallback for an unnamed slot is chosen to read well in a failure
// message rather than to be mechanically positional:
//
//   - The error slot is `Err`, because that is what it is. Fault injection
//     and every generated assertion name it, and `Result1` would say nothing.
//   - A lone non-error slot is `Result`, since an index distinguishes it from
//     nothing.
//   - Several non-error slots are `Result0`, `Result1`, … indexed across the
//     non-error slots only, so adding an error return to a signature does not
//     renumber the fields beside it.
func ReturnsOf(m *node.Method) []Return {
	values := 0
	for _, r := range m.Returns {
		if !IsError(r.Type) {
			values++
		}
	}

	out := make([]Return, 0, len(m.Returns))
	valueIdx, errIdx := 0, 0
	for _, r := range m.Returns {
		isErr := IsError(r.Type)
		out = append(out, Return{
			Name:  r.Name,
			Type:  golang.FromNode(r.Type),
			Field: returnField(r.Name, isErr, values, &valueIdx, &errIdx),
			Error: isErr,
		})
	}
	return out
}

// returnField picks the recorded-call field name for one return slot.
//
// The index counters advance only for the slot class they name, which is what
// keeps value numbering independent of the error slot's presence.
func returnField(declared string, isErr bool, values int, valueIdx, errIdx *int) string {
	if declared != "" {
		return naming.Pascal(declared)
	}
	if isErr {
		// A second error return is legal and vanishingly rare; index it
		// rather than emitting a duplicate field name that would not compile.
		defer func() { *errIdx++ }()
		if *errIdx == 0 {
			return "Err"
		}
		return "Err" + strconv.Itoa(*errIdx)
	}
	defer func() { *valueIdx++ }()
	if values == 1 {
		return "Result"
	}
	return "Result" + strconv.Itoa(*valueIdx)
}

// NamedReturnsUsable reports whether a generated method may carry the
// source's return names on its own signature.
//
// Propagation is all-or-nothing for two independent reasons, either of which
// forces the unnamed form:
//
//   - Go requires a signature's results to be all named or all anonymous, and
//     the emit layer enforces it — a mixed slice fails the render with
//     [emit.ErrMixedNamedReturns]. A source signature reaches that state
//     legitimately: `(_ User, err error)` is valid Go, and the blank
//     identifier normalises to unnamed, so the model holds one named and one
//     unnamed slot.
//   - A return name colliding with the receiver identifier or with a
//     parameter name does not compile. Renaming around it would break the
//     correspondence the names exist to carry, so the whole signature drops
//     back to anonymous results.
//
// Falling back costs documentation on the generated signature and nothing
// else; the recorded-call struct keeps its field names either way.
func NamedReturnsUsable(m *node.Method) bool {
	if len(m.Returns) == 0 {
		return false
	}
	taken := map[string]struct{}{ReceiverIdent: {}}
	for i, p := range m.Params {
		name := p.Name
		if name == "" {
			name = "arg" + strconv.Itoa(i)
		}
		taken[name] = struct{}{}
	}
	for _, r := range m.Returns {
		if r.Name == "" {
			return false
		}
		if _, clash := taken[r.Name]; clash {
			return false
		}
	}
	return true
}

// WithLocals assigns each return slot the identifier a generated body binds
// it to.
//
// Named results are already declared by the signature, so the local is the
// declared name. Anonymous results need a fresh local for the capture, which
// is positional — and prefixed with an underscore on the rare occasion a
// parameter already holds that identifier, since shadowing a parameter would
// record the wrong value.
func WithLocals(returns []Return, params []Param, named bool) []Return {
	taken := make(map[string]struct{}, len(params))
	for _, p := range params {
		taken[p.Name] = struct{}{}
	}
	for i := range returns {
		if named {
			returns[i].Local = returns[i].Name
			continue
		}
		local := "r" + strconv.Itoa(i)
		if _, clash := taken[local]; clash {
			local = "_" + local
		}
		returns[i].Local = local
	}
	return returns
}
