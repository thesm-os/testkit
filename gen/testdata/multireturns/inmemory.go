// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multireturns

import "context"

type InMemoryService struct{}
func NewInMemoryService() *InMemoryService { return &InMemoryService{} }

func (s *InMemoryService) Checkout(ctx context.Context, _ string) (Item, Lease, error) {
	if ctx != nil { if err := ctx.Err(); err != nil { return Item{}, Lease{}, err } }
	return Item{Key: "a"}, Lease{}, nil
}

func (s *InMemoryService) Peek(ctx context.Context) (Item, Item, error) {
	if ctx != nil { if err := ctx.Err(); err != nil { return Item{}, Item{}, err } }
	return Item{Key: "first"}, Item{Key: "second"}, nil
}

func (s *InMemoryService) Stats(ctx context.Context) (int, int, string, error) {
	if ctx != nil { if err := ctx.Err(); err != nil { return 0, 0, "", err } }
	return 1, 2, "ok", nil
}
