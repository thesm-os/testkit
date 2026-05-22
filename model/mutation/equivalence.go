// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import (
	"sort"
)

// EquivalenceClasses groups operators by their failure profile
// across the supplied reports. Two operators are equivalent when
// they produce the same kill outcome in every supplied report —
// either both killed in report N or both survived in report N,
// for every N.
//
// Use when running a suite of properties multiple times (e.g.,
// across rate variations or different test fixtures) and wanting
// to detect operators that are observationally indistinguishable.
// The model generator's mutation harness uses this to exclude
// equivalent mutants from the kill-rate denominator.
//
// Returns a list of equivalence classes; each class is the sorted
// list of operator names that share an outcome vector. Singleton
// classes (operators with a unique profile) are included.
//
// Reports must share the same operator set; an operator appearing
// in one report but not another is treated as "survived" in the
// missing report.
func EquivalenceClasses(reports ...Report) [][]string {
	if len(reports) == 0 {
		return nil
	}

	// Build the union of operator names across all reports.
	names := make(map[string]struct{})
	for _, r := range reports {
		for _, res := range r.Results {
			names[res.Operator] = struct{}{}
		}
	}
	ordered := make([]string, 0, len(names))
	for n := range names {
		ordered = append(ordered, n)
	}
	sort.Strings(ordered)

	// Build the profile vector per operator: one bool per report,
	// true when killed in that report.
	profile := func(name string) string {
		var b []byte
		for _, r := range reports {
			killed := false
			for _, res := range r.Results {
				if res.Operator == name && res.Killed {
					killed = true
					break
				}
			}
			if killed {
				b = append(b, '1')
			} else {
				b = append(b, '0')
			}
		}
		return string(b)
	}

	// Group by profile.
	groups := make(map[string][]string)
	for _, n := range ordered {
		key := profile(n)
		groups[key] = append(groups[key], n)
	}

	// Sort group keys for deterministic output.
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([][]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, groups[k])
	}
	return out
}
