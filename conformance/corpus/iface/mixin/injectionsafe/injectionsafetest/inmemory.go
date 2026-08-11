// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package injectionsafetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/injectionsafe], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package injectionsafetest

import (
	"context"
	"errors"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/injectionsafe"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct{}

var _ injectionsafe.Mixed = (*InMemory)(nil)

// NewInMemory returns the subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Store stores the value as data and returns it unchanged.
func (*InMemory) Store(ctx context.Context, in string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	// Stored verbatim: interpreting the value is the defect this mixin
	// exists to name, so the subject deliberately does no parsing at all.
	return in, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("injectionsafetest: nil context")
	}
	return ctx.Err()
}
