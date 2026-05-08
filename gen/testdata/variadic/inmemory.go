// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package variadic

import "context"

type InMemoryFinder struct{ data map[string]string }

func NewInMemoryFinder() *InMemoryFinder {
	return &InMemoryFinder{data: map[string]string{"a": "alpha", "b": "beta"}}
}

func (f *InMemoryFinder) Find(ctx context.Context, ids ...string) ([]string, error) {
	if ctx != nil { if err := ctx.Err(); err != nil { return nil, err } }
	var r []string
	for _, id := range ids { r = append(r, f.data[id]) }
	return r, nil
}

func (f *InMemoryFinder) Merge(ctx context.Context, values ...int) (int, error) {
	if ctx != nil { if err := ctx.Err(); err != nil { return 0, err } }
	sum := 0
	for _, v := range values { sum += v }
	return sum, nil
}
