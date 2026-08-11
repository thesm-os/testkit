// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package leasedidempotentwriter stacks the lease contract with the idempotent
// mixin on the acquire role.
//
// It earns a composite fixture because the two make opposite demands of the
// same call. Lease says every successful acquire is balanced by exactly one
// release, so the suite drives acquire and then release. Idempotent says
// repeating the call leaves observable state unchanged, so the suite drives
// acquire twice and compares.
//
// Run naively that second assertion acquires a lease it never releases, and
// the balance check the contract asks for fails on a subject that is correct.
// A generator has to reconcile them — release between the idempotence probes,
// or scope the probe to a key the balance check ignores — and getting it wrong
// produces a suite that fails against a working implementation.
package leasedidempotentwriter

import (
	"context"
	"errors"
)

// ErrHeld reports an acquire losing to a holder.
var ErrHeld = errors.New("leasedidempotentwriter: already held")

// LeasedWriter is the fixture interface.
//
//testkit:out leasedidempotentwritertest/ pkg=leasedidempotentwritertest
//testkit:stub
//testkit:suite
type LeasedWriter interface {
	// Acquire hosts the lease contract and carries the idempotent mixin.
	// Re-acquiring a lease this caller already holds is a no-op rather than a
	// conflict, which is what makes the two compatible at all.
	//testkit:contract lease role=acquire release=Release
	//testkit:mixin idempotent
	Acquire(ctx context.Context, key string) error

	// Release is the lease contract's release role.
	Release(ctx context.Context, key string) error
}
