// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mixed

import "context"

// InMemoryProcessor implements [Processor] for spec testing.
type InMemoryProcessor struct{}

// NewInMemoryProcessor returns a ready-to-use [InMemoryProcessor].
func NewInMemoryProcessor() *InMemoryProcessor { return &InMemoryProcessor{} }

func (*InMemoryProcessor) Run(ctx context.Context) error {
	if ctx != nil {
		return ctx.Err()
	}
	return nil
}

func (*InMemoryProcessor) Process(ctx context.Context, data []byte) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if len(data) == 0 {
		return ErrInvalidInput
	}
	return nil
}

func (*InMemoryProcessor) Describe() string {
	return "in-memory processor"
}

func (p *InMemoryProcessor) LegacyProcess(ctx context.Context, data []byte) error {
	return p.Process(ctx, data)
}
