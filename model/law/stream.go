// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law

import (
	"fmt"
	"sort"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"
)

// StreamReentrancy checks that iterating a StreamReader-shaped method
// twice produces the same items. Catches one-shot iterators or
// iterators that mutate state during iteration.
//
// Iteration is observational — the stream should not modify state.
type StreamReentrancy[T any, V any] struct {
	// Collect drains the stream iterator into a slice.
	Collect func(*rapid.T, T) ([]V, error)
}

// ID returns the stable identifier for this law.
func (StreamReentrancy[T, V]) ID() string { return "AUTO-STREAM-REENTRANT" }

// REQID returns an empty string (auto-derived law).
func (StreamReentrancy[T, V]) REQID() string { return "" }

// Check verifies reentrancy by collecting the stream twice and
// comparing the results (order-insensitive).
func (l StreamReentrancy[T, V]) Check(rt *rapid.T, sut, _ T) error {
	first, err1 := l.Collect(rt, sut)
	if err1 != nil {
		return fmt.Errorf("StreamReentrancy: first iteration error: %w", err1)
	}
	second, err2 := l.Collect(rt, sut)
	if err2 != nil {
		return fmt.Errorf("StreamReentrancy: second iteration error: %w", err2)
	}
	sortByString(first)
	sortByString(second)
	if diff := cmp.Diff(first, second); diff != "" {
		return fmt.Errorf("StreamReentrancy: iterations differ (-first +second):\n%s", diff)
	}
	return nil
}

// sortByString sorts a slice by the Sprint representation of each element.
func sortByString[V any](s []V) {
	sort.Slice(s, func(i, j int) bool {
		return fmt.Sprint(s[i]) < fmt.Sprint(s[j])
	})
}
