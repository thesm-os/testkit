// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package unknown

import (
	"context"
	"sync"
)

// InMemoryProcessor implements [Processor] for model testing.
type InMemoryProcessor struct {
	mu   sync.Mutex
	data map[string]Item
}

// NewInMemoryProcessor returns a ready-to-use [InMemoryProcessor].
func NewInMemoryProcessor() *InMemoryProcessor {
	return &InMemoryProcessor{data: make(map[string]Item)}
}

func (p *InMemoryProcessor) Get(_ context.Context, id string) (Item, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.data[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return v, nil
}

func (p *InMemoryProcessor) Put(_ context.Context, item Item) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.data[item.ID] = item
	return nil
}

func (p *InMemoryProcessor) Process(_ context.Context, input string, _ int) (string, bool, error) {
	return "processed:" + input, true, nil
}
