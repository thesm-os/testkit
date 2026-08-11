// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package xsssafetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/xsssafe], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package xsssafetest

import (
	"context"
	"errors"
	"strings"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/xsssafe"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct{}

var _ xsssafe.Mixed = (*InMemory)(nil)

// NewInMemory returns the subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Render escapes every character that could close a tag, which is the whole of the
// claim: what comes back cannot be markup.
func (*InMemory) Render(ctx context.Context, in string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	escaped := strings.NewReplacer(
		"&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;",
	).Replace(in)
	return escaped, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("xsssafetest: nil context")
	}
	return ctx.Err()
}
