// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package puretest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/pure], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package puretest

import (
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pure"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// No mutex, and that is the shape rather than an oversight: a value derived from
// nothing but the receiver has no state to guard, and adding a lock would say
// there was something to protect.
type InMemory struct{ label string }

var _ pure.Pure = (*InMemory)(nil)

// NewInMemory returns a subject describing itself as label.
func NewInMemory(label string) *InMemory { return &InMemory{label: label} }

// Describe derives its answer from the receiver alone — no arguments, no
// context, no error, and nothing observed. Repeated calls must agree, which is
// the law the shape carries and no single call can check.
func (s *InMemory) Describe() string { return "in-memory: " + s.label }
