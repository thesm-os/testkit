// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package namedreturns

import (
	"context"
	"time"
)

type InMemoryService struct{ data map[string]string }

func NewInMemoryService() *InMemoryService {
	return &InMemoryService{data: make(map[string]string)}
}

func (s *InMemoryService) Swap(ctx context.Context, key, value string) (string, string, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return "", "", err
		}
	}
	prev := s.data[key]
	s.data[key] = value
	return prev, value, nil
}

func (s *InMemoryService) Timestamps(ctx context.Context) (time.Time, time.Time, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	now := time.Now()
	return now.Add(-time.Hour), now, nil
}
