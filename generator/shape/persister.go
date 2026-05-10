// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// persisterDetector promotes a Writer-with-result to Persister when
// `//testkit:persister <Reader>` names a sibling Reader-shape method.
// The contract is "the value returned by the writer is fetched by the
// named reader" — used by KV stores, document databases, and any
// system where a write returns the canonical lookup key.
//
// Detection requires an InterfaceContext: the named sibling must
// exist on the same interface and resolve to a Reader-class shape in
// the pass-1 sibling map.
type persisterDetector struct{}

func (persisterDetector) Name() string  { return "Persister" }
func (persisterDetector) Priority() int { return PriorityContractPersister }

func (persisterDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Persister)
	if !ok || d.Off || len(d.Args) == 0 {
		return Info{}, false
	}
	// Carrier must be a Writer-with-result: one non-ctx input, one
	// non-error output, error return. Same structural pattern as
	// Reader; the directive disambiguates.
	if !singleInputSingleResult(s) {
		return Info{}, false
	}
	sibling, ok := s.Interface.Shapes[d.Args[0]]
	if !ok {
		return Info{}, false
	}
	if !isReaderClass(sibling.Shape) {
		return Info{}, false
	}

	return Info{
		Shape:   Persister,
		KeyType: s.valType(), // returned ID is the lookup key
		ValType: sibling.ValType,
	}, true
}

// findDirective returns the first directive on the method with the
// given name, plus a found-flag. Helper for contract-tier detectors
// that key off a single named directive.
func findDirective(dirs []directive.Directive, name string) (directive.Directive, bool) {
	for _, d := range dirs {
		if d.Name == name {
			return d, true
		}
	}
	return directive.Directive{}, false
}

// isReaderClass reports whether the given shape returns a value
// fetchable-by-key. Contract-tier detectors that name a "Reader"
// sibling accept any of these.
func isReaderClass(s Shape) bool {
	switch s {
	case Reader, ReaderNoError, ReaderWithBool, Lookup, PointerReader:
		return true
	default:
		return false
	}
}

// singleInputSingleResult reports whether the signature is the
// canonical Reader-shaped pattern: one non-ctx, non-variadic input
// and one non-error result with an error return. Persister and
// Appender both accept this carrier shape.
func singleInputSingleResult(s Signature) bool {
	if s.Variadic != nil {
		return false
	}
	if len(s.NonCtxParams) != 1 {
		return false
	}
	if !s.HasError || len(s.NonErrResults) != 1 {
		return false
	}
	return true
}
