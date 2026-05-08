// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go/types"

	"go.thesmos.sh/testkit/generator"
)

// lookupDetector matches `func(ctx?, K) (R1, R2, bool)` — exactly
// three results with bool last and no error. The bool position
// distinguishes Lookup from MultiReader.
type lookupDetector struct{}

func (lookupDetector) Name() string  { return "Lookup" }
func (lookupDetector) Priority() int { return PriorityLookup }

func (lookupDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 1 {
		return Info{}, false
	}
	if s.HasError || len(s.AllResults) != 3 {
		return Info{}, false
	}
	if !isBool(s.AllResults[2].Type()) {
		return Info{}, false
	}
	return Info{
		Shape:   Lookup,
		KeyType: s.keyType(),
		ValType: generator.TypeStr(s.AllResults[0].Type(), s.Tracker),
		RetType: generator.TypeStr(s.AllResults[1].Type(), s.Tracker),
	}, true
}

func isBool(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.Bool
}
