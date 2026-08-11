// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package writertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/writer], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package writertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/writer"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]writer.Value
}

var _ writer.Writer = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]writer.Value{}} }

// Put is the shape: one value in, an error out, nothing else. It is also what
// the harness seeds every subject through — the annotator classifies it writer,
// so the interface populates itself and nothing is asked of the consumer.
func (s *InMemory) Put(ctx context.Context, v writer.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	return nil
}

// Stored reports what was written under a key, which the interface exposes no
// way to observe — a writer says only whether the write failed.
func (s *InMemory) Stored(key string) (writer.Value, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	return v, ok
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("writertest: nil context")
	}
	return ctx.Err()
}
