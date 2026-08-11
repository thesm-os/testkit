// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package totaltest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/total], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package totaltest

import (
	"context"
	"errors"
	"strings"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/total"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct{}

var _ total.Mixed = (*InMemory)(nil)

// NewInMemory returns the subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Classify answers for every string, which is what the declared domain asserts.
func (*InMemory) Classify(ctx context.Context, in string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	// Total by construction: every string has a length, so there is no
	// input this can refuse. A subject with a rejecting arm would be
	// stating a narrower domain than the directive claims.
	if strings.TrimSpace(in) == "" {
		return "empty", nil
	}
	return "populated", nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("totaltest: nil context")
	}
	return ctx.Err()
}
