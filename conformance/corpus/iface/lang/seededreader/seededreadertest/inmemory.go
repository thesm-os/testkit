// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package seededreadertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/seededreader], and the
// in-memory subject they are run against.
package seededreadertest

import (
	"context"
	"errors"
	"maps"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/seededreader"
)

// InMemory is the seed seam's subject: a catalog loaded once at
// construction and read-only thereafter, which is the shape that makes
// the harness receive its corpus rather than write one.
type InMemory struct {
	docs map[seededreader.Key]seededreader.Body
}

// NewInMemory loads the given corpus. The suite decides what is in it;
// this decides only how it is held.
func NewInMemory(docs map[seededreader.Key]seededreader.Body) *InMemory {
	loaded := make(map[seededreader.Key]seededreader.Body, len(docs))
	maps.Copy(loaded, docs)
	return &InMemory{docs: loaded}
}

// Lookup answers for a seeded key and reports the zero for any other.
//
// The context is honoured before the map is touched, which is what the
// generated cancel, deadline and nil-context checks ask of every
// context-taking method — a read that answers from memory can still be
// asked not to.
func (s *InMemory) Lookup(ctx context.Context, key seededreader.Key) (seededreader.Body, error) {
	if err := alive(ctx); err != nil {
		return "", err
	}
	return s.docs[key], nil
}

// Len reports how many documents were loaded.
func (s *InMemory) Len(ctx context.Context) (int, error) {
	if err := alive(ctx); err != nil {
		return 0, err
	}
	return len(s.docs), nil
}

// alive reports the context's own refusal, and refuses a nil one rather
// than dereferencing it — an errant caller's nil is a failed request,
// not an outage.
func alive(ctx context.Context) error {
	if ctx == nil {
		return errors.New("seededreadertest: nil context")
	}
	return ctx.Err()
}
