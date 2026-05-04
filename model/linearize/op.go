// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

// ReaderResult is the output for Reader-shaped operations.
type ReaderResult[V any] struct {
	Value V
	Err   error
}

// ReaderBoolResult is the output for ReaderWithBool-shaped operations.
type ReaderBoolResult[V any] struct {
	Value V
	OK    bool
}

// LookupResult is the output for Lookup-shaped operations.
type LookupResult[R1, R2 any] struct {
	R1 R1
	R2 R2
	OK bool
}

// WriterResult is the output for Writer/Deleter-shaped operations.
type WriterResult struct {
	Err error
}

// OpSpec defines a single operation for a linearizability model.
type OpSpec struct {
	// Name is the operation name (e.g., "Get", "Put", "Delete").
	Name string

	// PartitionKey extracts a string partition key from the input.
	PartitionKey func(input any) string

	// Step is the per-partition state transition.
	Step func(state, input, output any) (bool, any)
}
