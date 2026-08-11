// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package streamconsumertest holds the generated harnesses and doubles for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamconsumer], and
// the in-memory subjects they are run against — scaffolding for the run, so they
// live beside the harnesses rather than in the package stating the shape.
package streamconsumertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamconsumer"
)

// ErrNoSource is what Ingest reports when handed nothing to read.
//
// The failure path the generated zero-value check reaches. A nil Source is the
// one input a derivation can write down for an interface parameter, and it is
// also the one a caller reaches by forgetting an argument.
var ErrNoSource = errors.New("streamconsumertest: nil source")

// ErrSourceFailed is what a source reports when it breaks mid-stream.
//
// Declared here rather than in the test beside it so it carries this package's
// name: a partial read is a case only a deliberately broken source reaches, and
// the sentinel is the test-support this package exists to provide.
var ErrSourceFailed = errors.New("streamconsumertest: source failed")

// InMemory consumes a stream and reports how many elements it took.
type InMemory struct {
	mu       sync.Mutex
	ingested []streamconsumer.Value
}

var _ streamconsumer.StreamConsumer = (*InMemory)(nil)

// NewInMemory returns a consumer that has taken nothing.
func NewInMemory() *InMemory { return &InMemory{} }

// Ingest drains the source and returns a count beside its error, which is what
// puts it in the shape: one non-context parameter, one value, one error, and the
// parameter is an interface rather than a func.
//
// It returns the zero count with every error, including a partial read. A
// consumer that reported what it managed before failing would leave the caller
// unable to tell a complete ingest from an interrupted one.
func (s *InMemory) Ingest(ctx context.Context, src streamconsumer.Source) (int, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	if src == nil {
		return 0, ErrNoSource
	}

	var taken []streamconsumer.Value
	for {
		v, ok, err := src.Next(ctx)
		if err != nil {
			return 0, err
		}
		if !ok {
			break
		}
		taken = append(taken, v)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ingested = append(s.ingested, taken...)
	return len(taken), nil
}

// Ingested reports what was taken, which the interface exposes no way to
// observe — Ingest answers how many, never which.
func (s *InMemory) Ingested() []streamconsumer.Value {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]streamconsumer.Value, len(s.ingested))
	copy(out, s.ingested)
	return out
}

// SliceSource is the subject the Source contract is run against: a stream over
// a fixed slice.
type SliceSource struct {
	mu     sync.Mutex
	values []streamconsumer.Value
	at     int
}

var _ streamconsumer.Source = (*SliceSource)(nil)

// NewSliceSource returns a source yielding values in order.
func NewSliceSource(values ...streamconsumer.Value) *SliceSource {
	return &SliceSource{values: values}
}

// Next reports exhaustion through its flag and zeroes every slot beside an
// error, which is the property the generated check on Source is about.
func (s *SliceSource) Next(ctx context.Context) (streamconsumer.Value, bool, error) {
	if err := contextErr(ctx); err != nil {
		return streamconsumer.Value{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.at >= len(s.values) {
		return streamconsumer.Value{}, false, nil
	}
	v := s.values[s.at]
	s.at++
	return v, true, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("streamconsumertest: nil context")
	}
	return ctx.Err()
}
