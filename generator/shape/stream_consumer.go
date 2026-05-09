// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import "go/types"

// streamConsumerDetector matches `func(ctx, S) (V, error)` where
// S is interface-typed (io.Reader, io.Writer, custom interface
// params). Without this detector the same shape would be
// misclassified as Reader, treating the interface-typed param as
// a key.
//
// Detection requires ctx — naked interface-typed Reader signatures
// are too ambiguous without it.
type streamConsumerDetector struct{}

func (streamConsumerDetector) Name() string  { return "StreamConsumer" }
func (streamConsumerDetector) Priority() int { return PriorityStreamConsumer }

func (streamConsumerDetector) Detect(s Signature) (Info, bool) {
	if !s.HasCtx || s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 1 {
		return Info{}, false
	}
	if !s.HasError {
		return Info{}, false
	}
	// Exactly one non-error result (StreamConsumer always returns
	// a count or summary). Multi-result variants fall through.
	if len(s.NonErrResults) != 1 {
		return Info{}, false
	}
	if !isInterfaceTyped(s.NonCtxParams[0].Type()) {
		return Info{}, false
	}
	return Info{
		Shape:   StreamConsumer,
		KeyType: s.keyType(),
		ValType: s.valType(),
	}, true
}

// isInterfaceTyped reports whether t's underlying type is an
// interface, excluding type parameters.
//
// Type parameters report their constraint as their underlying type
// (e.g. `K comparable` has underlying type `comparable`, an
// interface). Treating them as interface-typed would route every
// generic Reader to StreamConsumer — wrong. We exclude *types.TypeParam
// so only declared interface types (io.Reader, io.Writer, custom
// interfaces) match.
//
// context.Context is filtered upstream by [ParseSignature]; only
// non-ctx params reach this check.
func isInterfaceTyped(t types.Type) bool {
	if _, ok := t.(*types.TypeParam); ok {
		return false
	}
	_, ok := t.Underlying().(*types.Interface)
	return ok
}
