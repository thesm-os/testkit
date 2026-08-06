// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package embeddedforeign is the language-axis fixture for an embedded
// interface the generator cannot reach.
//
// Embedded method sets are flattened by resolving each embed against the
// interfaces the run loaded. A standard-library interface is not among them,
// and neither is one from any package outside the run's scope — which is the
// ordinary case in real code rather than an exotic one.
//
// No double is generated. One missing a method does not satisfy the interface
// it doubles, so it cannot be passed anywhere that interface is expected —
// which is the whole of what a double is for, and recording faithfully is
// worth nothing if nothing can accept it. A warning names the embed so the
// absence is attributable rather than mysterious.
//
// The frontend type-checks the embedded interface and knows its method set;
// it is the node graph that carries only the run's own source. Closing that
// gap upstream is what would let this fixture generate.
//
// [embedded] holds the resolvable path; this holds the other one.
package embeddedforeign

import (
	"context"
	"io"
)

// Stream embeds a standard-library interface alongside a method of its own.
// Close comes from [io.Closer] and cannot be projected; Read can.
//
//testkit:out embeddedforeigntest/ pkg=embeddedforeigntest
//testkit:stub
type Stream interface {
	io.Closer

	Read(ctx context.Context, key string) (string, error)
}
