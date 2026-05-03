// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package iterators

import (
	"context"
	"iter"
	"strings"
)

// InMemoryScanner implements [Scanner] for spec testing.
type InMemoryScanner struct {
	items []Item
}

// NewInMemoryScanner returns a scanner with sample data.
func NewInMemoryScanner() *InMemoryScanner {
	return &InMemoryScanner{
		items: []Item{
			{ID: "a-1", Data: []byte("alpha")},
			{ID: "a-2", Data: []byte("alpha2")},
			{ID: "b-1", Data: []byte("beta")},
		},
	}
}

func (s *InMemoryScanner) Keys(ctx context.Context) iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, item := range s.items {
			if ctx != nil && ctx.Err() != nil {
				return
			}
			if !yield(item.ID) {
				return
			}
		}
	}
}

func (s *InMemoryScanner) Scan(ctx context.Context, prefix string) iter.Seq2[Item, error] {
	return func(yield func(Item, error) bool) {
		for _, item := range s.items {
			if ctx != nil && ctx.Err() != nil {
				yield(Item{}, ctx.Err())
				return
			}
			if strings.HasPrefix(item.ID, prefix) {
				if !yield(item, nil) {
					return
				}
			}
		}
	}
}

func (s *InMemoryScanner) Count(ctx context.Context) (int, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}
	return len(s.items), nil
}
