// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package interfaces

import (
	"context"
	"io"
)

// InMemoryProcessor implements [Processor] for testing.
type InMemoryProcessor struct{ buf []byte }

// NewInMemoryProcessor returns a ready-to-use processor.
func NewInMemoryProcessor() *InMemoryProcessor { return &InMemoryProcessor{} }

func (p *InMemoryProcessor) ReadFrom(ctx context.Context, r io.Reader) (int, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}
	if r == nil {
		return 0, nil
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	p.buf = append(p.buf, data...)
	return len(data), nil
}

func (p *InMemoryProcessor) WriteTo(ctx context.Context, w io.Writer) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if w == nil {
		return nil
	}
	_, err := w.Write(p.buf)
	return err
}
