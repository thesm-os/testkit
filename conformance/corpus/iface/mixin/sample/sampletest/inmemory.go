// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sampletest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sample], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package sampletest

import (
	"context"
	"errors"
	"strings"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/sample"
)

// ErrUnprocessable is what Process refuses an input it cannot handle.
//
// The constraint is what makes the mixin worth having: a Process accepting
// every string would be satisfied by any builder at all, and the check would
// pass without saying anything about the pair.
var ErrUnprocessable = errors.New("sampletest: input is not sampled")

// inputPrefix is the shape Process requires and NewInput produces. Narrow on
// purpose: a derived string does not carry it, so the generated check reaches
// the accepting path only through the builder.
const inputPrefix = "sample:"

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu   sync.Mutex
	next int
}

var _ sample.Mixed = (*InMemory)(nil)

// NewInMemory returns a subject whose builder starts from the first sample.
func NewInMemory() *InMemory { return &InMemory{} }

// NewInput is the builder the mixin names, and produces a distinct value each
// call — sampling an input space rather than returning one fixed member of it.
func (s *InMemory) NewInput(ctx context.Context) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return inputPrefix + string(rune('a'+(s.next-1)%26)), nil
}

// Process accepts what NewInput produces and refuses what it does not.
//
// Stateless, unlike the builder beside it: what the method accepts is a fact
// about the input space, and a Process consulting the subject's own history
// would make the pair agree for reasons the mixin does not claim.
func (*InMemory) Process(ctx context.Context, input string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	if !strings.HasPrefix(input, inputPrefix) {
		return "", ErrUnprocessable
	}
	return strings.ToUpper(strings.TrimPrefix(input, inputPrefix)), nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("sampletest: nil context")
	}
	return ctx.Err()
}
