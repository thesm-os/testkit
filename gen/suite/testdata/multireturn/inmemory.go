// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multireturn

import "context"

// InMemoryService implements [Service] for spec testing.
type InMemoryService struct {
	active  int
	pending int
}

// NewInMemoryService returns a ready-to-use [InMemoryService].
func NewInMemoryService() *InMemoryService {
	return &InMemoryService{active: 3, pending: 1}
}

func (s *InMemoryService) Status(ctx context.Context) (Stats, string, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return Stats{}, "", err
		}
	}
	return Stats{Active: s.active, Pending: s.pending}, "healthy", nil
}

func (s *InMemoryService) Reset(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	s.active = 0
	s.pending = 0
	return nil
}
