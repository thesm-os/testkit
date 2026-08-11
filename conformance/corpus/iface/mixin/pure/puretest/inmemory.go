// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package puretest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pure], and the in-memory
// subject they are run against.
package puretest

import (
	"strings"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/pure"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// No fields and no mutex, which is the mixin rather than an oversight: a value
// derived from its input alone has no state to guard, and a lock here would say
// there was something to protect.
type InMemory struct{}

var _ pure.Mixed = (*InMemory)(nil)

// NewInMemory returns the subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Derive depends on its argument and nothing else — no clock, no counter, no
// receiver state. Repeated calls with one input must agree, which is what
// AUTO-PURE-DETERMINISTIC states for the model tier and what no single call can
// observe.
func (*InMemory) Derive(input string) string { return strings.ToUpper(input) }
