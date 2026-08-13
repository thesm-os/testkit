// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

// ReaderResult is the output for Reader-shaped operations.
type ReaderResult[V any] struct {
	Value V
	Err   error
}

// TraceOutput surfaces the read's value and its own error to the trace, so
// a trace-scanning law tells an errored read's zero from a read that
// answered zero.
func (r ReaderResult[V]) TraceOutput() (any, error) { return r.Value, r.Err }

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

// AppendResult is the output for offset-answering appends: the offset the
// subject assigned, beside the refusal that means it assigned none.
type AppendResult struct {
	Off int64
	Err error
}

// TraceOutput surfaces the assigned offset and the append's own error to
// the trace.
func (r AppendResult) TraceOutput() (any, error) { return r.Off, r.Err }

// TraceOutput surfaces the write's own error to the trace.
func (r WriterResult) TraceOutput() (any, error) { return nil, r.Err }

// AnsweringResult is the outcome of an answering write: the stored state
// the subject answered, beside the call's own error.
type AnsweringResult[V any] struct {
	Value V
	Err   error
}

// TraceOutput surfaces the answered state and the call's own error, so a
// trace-scanning law reads the version the store assigned off the value.
func (r AnsweringResult[V]) TraceOutput() (any, error) { return r.Value, r.Err }

// OpSpec defines a single operation for a linearizability model.
type OpSpec struct {
	// Name is the operation name (e.g., "Get", "Put", "Delete").
	Name string

	// PartitionKey extracts a string partition key from the input.
	PartitionKey func(input any) string

	// Step is the per-partition state transition.
	Step func(state, input, output any) (bool, any)
}
