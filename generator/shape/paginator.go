// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// paginatorDetector promotes a Reader to Paginator when
// `//testkit:pagination <CursorField>` is present. The contract is
// "drain terminates, no duplicates, resumable from any cursor" —
// classic API pagination.
//
// The directive name is shared with the suite-side polling assertion
// (see directive.Pagination); the model side reads the same payload.
type paginatorDetector struct{}

func (paginatorDetector) Name() string  { return "Paginator" }
func (paginatorDetector) Priority() int { return PriorityContractPaginator }

func (paginatorDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Pagination)
	if !ok || d.Off || len(d.Args) == 0 {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	if !isReaderClass(carrier.Shape) && carrier.Shape != StreamReader {
		return Info{}, false
	}
	return Info{
		Shape:   Paginator,
		KeyType: carrier.KeyType,
		ValType: carrier.ValType,
	}, true
}
