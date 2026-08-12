// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package codeclossytest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec-lossy], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package codeclossytest

import (
	"context"
	"errors"
	"strings"

	codeclossy "go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec-lossy"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct{}

var _ codeclossy.Contract = (*InMemory)(nil)

// NewInMemory returns the subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Encode is the forward transform: case folding, which is honestly lossy —
// two inputs differing only in case collapse to one encoding, and no inverse
// can tell them apart afterwards. What the law demands survives it: a second
// pass over the decoded value changes nothing, because everything the fold
// was going to lose is already gone.
func (*InMemory) Encode(ctx context.Context, in string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	return strings.ToLower(in), nil
}

// Decode is the inverse of what survived: the fold keeps every character it
// kept, so recovery is the identity over the encoded form.
func (*InMemory) Decode(ctx context.Context, in string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	return in, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("codeclossytest: nil context")
	}
	return ctx.Err()
}
