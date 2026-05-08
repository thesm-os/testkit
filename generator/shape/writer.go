// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// writerDetector matches `func(ctx?, V) error` — single non-ctx
// parameter, error-only return.
//
// The Writer-with-result form `func(ctx?, V) (R, error)` is
// structurally identical to a [Reader] (`(ctx?, K) (V, error)`):
// the type system can't tell whether the non-error result is a
// "key for the saved object" (Writer-with-result) or a "value
// for the looked-up key" (Reader). Consumers who need the
// distinction supply a directive in their data layer; auto-
// detection routes the ambiguous case to Reader.
//
// Deleter (priority 550) claims this signature first when
// //testkit:deleter is present.
type writerDetector struct{}

func (writerDetector) Name() string  { return "Writer" }
func (writerDetector) Priority() int { return PriorityWriter }

func (writerDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 1 {
		return Info{}, false
	}
	if !s.HasError || len(s.NonErrResults) != 0 {
		return Info{}, false
	}
	return Info{
		Shape:   Writer,
		ValType: s.keyType(),
	}, true
}
