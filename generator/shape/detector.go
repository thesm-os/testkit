// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go/types"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
)

// Detector classifies a single shape. Implementations live one per
// file (reader.go, writer.go, ...) and register themselves into the
// default [Registry].
//
// Detectors are pure: same Signature → same result. They MUST NOT
// mutate the Signature or its referenced fields.
type Detector interface {
	// Name returns the canonical shape name (e.g. "Reader"). Used
	// in diagnostics and tests.
	Name() string

	// Priority orders detectors within a [Registry]. Higher fires
	// first. Conflicts are resolved by registration order (first
	// registered wins on ties).
	Priority() int

	// Detect inspects the pre-computed [Signature] view. Returns
	// (info, true) when the signature matches this shape's
	// pattern; (Info{}, false) otherwise.
	Detect(Signature) (Info, bool)
}

// Signature is the parsed view of a method's signature, computed
// once and passed to every detector. It exists so detectors can
// focus on their own pattern without re-walking params/results.
type Signature struct {
	// Method is the underlying method info.
	Method generator.MethodInfo

	// Tracker resolves type names to qualified strings.
	Tracker *generator.ImportTracker

	// Directives are the //testkit: directives on the method.
	Directives []directive.Directive

	// HasCtx is true when the first parameter is context.Context.
	HasCtx bool

	// NonCtxParams are the parameters excluding the leading ctx
	// and the variadic (if any). The variadic is held separately
	// in [Variadic].
	NonCtxParams []*types.Var

	// Variadic is the trailing variadic parameter, or nil. For
	// `func(ctx, ...string)` Variadic is the `...string` param.
	Variadic *types.Var

	// HasError is true when any result is the error interface.
	HasError bool

	// ErrIdx is the index of the error result, or -1.
	ErrIdx int

	// NonErrResults are the results excluding the error.
	NonErrResults []*types.Var

	// AllResults is the full result tuple (used when detectors
	// need to inspect type at a specific position).
	AllResults []*types.Var

	// Iter is pre-detected iter.Seq[T] / iter.Seq2[V, error]
	// info. Set when any result is an iter type.
	Iter generator.IterSeqInfo
}

// ParseSignature builds a [Signature] view for one method.
//
// Callers pass the [generator.ImportTracker] that will be used to
// render types in templates; detectors return [Info] with types
// already qualified through this tracker.
func ParseSignature(m generator.MethodInfo, tracker *generator.ImportTracker, dirs []directive.Directive) Signature {
	s := Signature{
		Method:     m,
		Tracker:    tracker,
		Directives: dirs,
		ErrIdx:     -1,
	}

	sig := m.Signature
	params := sig.Params()
	n := params.Len()

	paramEnd := n
	if sig.Variadic() && n > 0 {
		paramEnd = n - 1
		s.Variadic = params.At(n - 1)
	}
	for i := 0; i < paramEnd; i++ {
		p := params.At(i)
		if generator.IsContextType(p.Type()) {
			s.HasCtx = true
			continue
		}
		s.NonCtxParams = append(s.NonCtxParams, p)
	}

	results := sig.Results()
	rn := results.Len()
	s.AllResults = make([]*types.Var, rn)
	for i := range rn {
		r := results.At(i)
		s.AllResults[i] = r
		if generator.IsErrorType(r.Type()) {
			s.HasError = true
			s.ErrIdx = i
			continue
		}
		s.NonErrResults = append(s.NonErrResults, r)
	}

	for i := range rn {
		info := generator.AnalyzeIterReturn(results.At(i).Type(), tracker)
		if info.IsSeq || info.IsSeq2 {
			s.Iter = info
			break
		}
	}

	return s
}

// HasDirective reports whether any directive on the method has the
// given name.
func (s Signature) HasDirective(name string) bool {
	for _, d := range s.Directives {
		if d.Name == name {
			return true
		}
	}
	return false
}

// keyType renders the qualified type of the first non-ctx
// non-variadic parameter. Caller must verify NonCtxParams is
// non-empty.
func (s Signature) keyType() string {
	return generator.TypeStr(s.NonCtxParams[0].Type(), s.Tracker)
}

// valType renders the qualified type of the first non-error result.
// Caller must verify NonErrResults is non-empty.
func (s Signature) valType() string {
	return generator.TypeStr(s.NonErrResults[0].Type(), s.Tracker)
}
