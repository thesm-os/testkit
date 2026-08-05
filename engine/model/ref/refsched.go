// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides the [PureScheduler] reference: given a
// DAG of nodes with dependencies, return a topological execution
// order. Pure (no side effects on the graph); same input always
// yields the same output.

package ref

import (
	"errors"
	"sort"
)

// ErrCycle is returned by [PureScheduler.Schedule] when the graph
// has a cycle and no topological order exists.
var ErrCycle = errors.New("ref: graph has a cycle")

// PureScheduler computes a topological execution order over a DAG
// of nodes G with dependencies between them. Schedule is
// deterministic: ties are broken by the supplied less function so
// the same input produces the same output across runs.
type PureScheduler[G comparable, R any] struct {
	less    func(a, b G) bool
	project func(G) R
}

// NewPureScheduler constructs a [PureScheduler].
//
// less orders nodes whose dependencies are equally satisfied —
// passes through to sort.Slice for deterministic ties.
// project maps each scheduled node to its emitted result type R.
func NewPureScheduler[G comparable, R any](
	less func(a, b G) bool,
	project func(G) R,
) *PureScheduler[G, R] {
	return &PureScheduler[G, R]{less: less, project: project}
}

// Schedule returns the topological order of nodes given their
// dependency edges (deps[node] is the list of nodes that must run
// before node). Returns ErrCycle if no topological order exists.
func (s *PureScheduler[G, R]) Schedule(nodes []G, deps map[G][]G) ([]R, error) {
	indeg := make(map[G]int, len(nodes))
	for _, n := range nodes {
		indeg[n] = 0
	}
	for n, ds := range deps {
		// Edges land at n; each dep adds one incoming edge to n.
		indeg[n] += len(ds)
	}
	ready := make([]G, 0)
	for _, n := range nodes {
		if indeg[n] == 0 {
			ready = append(ready, n)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return s.less(ready[i], ready[j]) })

	// Build reverse adjacency: who depends on n?
	dependents := make(map[G][]G, len(nodes))
	for n, ds := range deps {
		for _, d := range ds {
			dependents[d] = append(dependents[d], n)
		}
	}

	out := make([]R, 0, len(nodes))
	for len(ready) > 0 {
		n := ready[0]
		ready = ready[1:]
		out = append(out, s.project(n))
		newReady := make([]G, 0)
		for _, dep := range dependents[n] {
			indeg[dep]--
			if indeg[dep] == 0 {
				newReady = append(newReady, dep)
			}
		}
		sort.Slice(newReady, func(i, j int) bool { return s.less(newReady[i], newReady[j]) })
		ready = append(ready, newReady...)
		sort.Slice(ready, func(i, j int) bool { return s.less(ready[i], ready[j]) })
	}
	if len(out) != len(nodes) {
		return nil, ErrCycle
	}
	return out, nil
}
