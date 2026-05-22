// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"sort"

	"github.com/anishathalye/porcupine"

	"go.thesmos.sh/testkit/model"
)

// PartitionByDirective wraps an existing porcupine.Model, replacing
// its Partition function with one that groups by a consumer-named
// field on the input's Args. Used when //testkit:partition <Field>
// directs the model to partition by something other than the
// natural primary key.
//
// extract returns the partition key for a given operation's Args.
// Operations whose Args don't yield a partition key (extract
// returns "") share an implicit empty-string partition.
func PartitionByDirective(inner porcupine.Model, extract func(args any) string) porcupine.Model {
	out := inner
	out.Partition = func(history []porcupine.Operation) [][]porcupine.Operation {
		m := make(map[string][]porcupine.Operation)
		for _, op := range history {
			inp := op.Input.(model.OpInput)
			key := extract(inp.Args)
			m[key] = append(m[key], op)
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		result := make([][]porcupine.Operation, 0, len(keys))
		for _, k := range keys {
			result = append(result, m[k])
		}
		return result
	}
	return out
}
