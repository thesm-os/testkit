// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package interfaces

import (
	"context"
	"io"
)

//go:generate testkit stub -o processortest/processor_stub.gen.go Processor
//go:generate testkit suite -o processortest/processor_spec.gen.go Processor
//go:generate testkit bench -o processortest/processor_bench.gen.go Processor

// Processor exercises interface-typed parameters (io.Reader, io.Writer).
type Processor interface {
	// ReadFrom reads data from a reader.
	ReadFrom(ctx context.Context, r io.Reader) (int, error)
	// WriteTo writes data to a writer.
	WriteTo(ctx context.Context, w io.Writer) error
}
