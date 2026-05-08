// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"go.thesmos.sh/testkit/generator"
)

// emitWireGolden is the [generator.Pipeline.PostRender] hook. It
// emits one consolidated JSON file alongside the test file whose
// top-level keys are type names and whose values are
// `{<ConstName>: <int>}` mappings. Per-type wire-compat subtests
// inside each Test<Type>Enum read just their own slice via
// [golden.AssertGoldenJSONField] — drift in any one type fails
// only that subtest, while the file stays as one PR-friendly
// artifact regardless of how many enums share the source.
//
// The basename mirrors the test file (`<file>.gen_test.go` →
// `<file>.gen_wire.json`) so a directory listing groups source +
// test + golden together.
func emitWireGolden(data *Data, opts generator.Options) ([]generator.OutputFile, error) {
	if !data.HasContent() {
		return nil, nil
	}
	doc := make(map[string]map[string]int64, len(data.Enums))
	for _, e := range data.Enums {
		mapping := make(map[string]int64, len(e.Values))
		for _, v := range e.Values {
			mapping[v.Name] = v.IntValue
		}
		doc[e.TypeName] = mapping
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal wire golden: %w", err)
	}
	body = append(body, '\n')
	return []generator.OutputFile{{
		Path:    filepath.Join(filepath.Dir(opts.Output), data.GoldenFile),
		Content: body,
	}}, nil
}
