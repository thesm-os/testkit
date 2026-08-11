// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package codectest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package codectest

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/codec"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct{}

var _ codec.Contract = (*InMemory)(nil)

// NewInMemory returns the subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Encode is the forward transform.
//
// Base64 rather than something bespoke: the contract claims exact fidelity,
// and a transform that lost information would make the fixture state a claim
// its own subject violates.
func (*InMemory) Encode(ctx context.Context, in string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(in)), nil
}

// Decode is the inverse, and reports an input the forward transform could not
// have produced.
func (*InMemory) Decode(ctx context.Context, in string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return "", fmt.Errorf("codectest: decode: %w", err)
	}
	return string(raw), nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("codectest: nil context")
	}
	return ctx.Err()
}
