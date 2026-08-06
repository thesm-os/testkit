// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package crdtmerge is the mixin-axis fixture for the crdtmerge mixin, which
// declares that replicas converge to the same state regardless of merge order.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package crdtmerge

import (
	"context"
)

// Replica is the peer Merge folds in. The law compares two replicas merged in
// opposite orders, so the type has to be nameable in the signature.
//
//testkit:out crdtmergetest/ pkg=crdtmergetest
//testkit:stub
type Replica interface {
	Items(ctx context.Context) ([]string, error)
}

// Mixed is the fixture interface.
//
//testkit:out crdtmergetest/ pkg=crdtmergetest
//testkit:stub
type Mixed interface {
	// Merge folds another replica in. Convergence is a statement about two
	// merges in opposite orders, so the method has to take a peer rather than
	// a value.
	//testkit:mixin crdtmerge
	Merge(ctx context.Context, peer Replica) error

	// Add introduces divergence for the merge to reconcile.
	Add(ctx context.Context, item string) error

	// Items observes convergence.
	Items(ctx context.Context) ([]string, error)
}
