// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"errors"
	"fmt"
	"sort"

	"github.com/anishathalye/porcupine"
	"github.com/google/go-cmp/cmp"

	"go.thesmos.sh/testkit/engine/model"
)

// casCellState is the per-partition state for a single CAS-cell key.
type casCellState[V any, Version comparable] struct {
	present bool
	value   V
	version Version
}

// CASCell builds a porcupine.Model for compare-and-swap cells
// partitioned by key. Each key holds at most one value; a write
// succeeds if its version matches the current cell's version, and an
// empty cell accepts only the zero version — the shipped VersionedCell
// oracle's own dialect, where the stamp is seen+1 and an empty cell has
// seen nothing. Step functions:
//
//   - Get:     output matches the stored value plus current version,
//     or the configured sentinel error when empty.
//   - CAS:     succeeds iff input version equals stored version; on
//     success the cell advances via nextVer. Mismatch yields
//     the configured mismatch error and leaves state unchanged.
//
// The partition key comes from model.OpInput.PartitionKey.
// versionOf extracts the version from a value; nextVer maps the
// stored version to its successor after a successful write.
func CASCell[V any, Version comparable](
	sentinel error,
	mismatch error,
	versionOf func(V) Version,
	nextVer func(Version) Version,
) porcupine.Model {
	var zeroV V
	var zeroVer Version

	return porcupine.Model{
		Partition: partitionByKey,
		Init: func() any {
			return casCellState[V, Version]{}
		},
		Step: func(state, input, output any) (bool, any) {
			s := state.(casCellState[V, Version])
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)

			switch inp.Name {
			case OpGet:
				r, ok := out.Result.(ReaderResult[V])
				if !ok {
					return false, s
				}
				if !s.present {
					if sentinel == nil {
						return r.Err != nil, s
					}
					return errors.Is(r.Err, sentinel), s
				}
				return r.Err == nil && cmp.Equal(r.Value, s.value), s

			case OpCAS:
				v, ok := inp.Args.(V)
				if !ok {
					return false, s
				}
				r, ok := out.Result.(WriterResult)
				if !ok {
					return false, s
				}
				if !s.present {
					if versionOf(v) != zeroVer {
						// Nothing has been seen, so only the zero version
						// matches — the same refusal the live oracle makes.
						return errors.Is(r.Err, mismatch), s
					}
					if r.Err != nil {
						return false, s
					}
					return true, casCellState[V, Version]{
						present: true,
						value:   v,
						version: nextVer(zeroVer),
					}
				}
				if versionOf(v) != s.version {
					return errors.Is(r.Err, mismatch), s
				}
				if r.Err != nil {
					return false, s
				}
				return true, casCellState[V, Version]{
					present: true,
					value:   v,
					version: nextVer(s.version),
				}

			default:
				_ = zeroV
				return false, s
			}
		},
		Equal: func(a, b any) bool {
			sa := a.(casCellState[V, Version])
			sb := b.(casCellState[V, Version])
			if sa.present != sb.present || sa.version != sb.version {
				return false
			}
			if !sa.present {
				return true
			}
			return cmp.Equal(sa.value, sb.value)
		},
		DescribeOperation: func(input, output any) string {
			inp := input.(model.OpInput)
			out := output.(model.OpOutput)
			return fmt.Sprintf("%s(%v) -> %v", inp.Name, inp.Args, out.Result)
		},
		DescribeState: func(state any) string {
			s := state.(casCellState[V, Version])
			if !s.present {
				return "<absent>"
			}
			return fmt.Sprintf("v=%v ver=%v", s.value, s.version)
		},
	}
}

// partitionByKey groups operations by their PartitionKey, sorting
// the resulting partitions for deterministic checking. Shared by
// every per-key linearize model.
func partitionByKey(history []porcupine.Operation) [][]porcupine.Operation {
	m := make(map[string][]porcupine.Operation)
	for _, op := range history {
		inp := op.Input.(model.OpInput)
		m[inp.PartitionKey] = append(m[inp.PartitionKey], op)
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
