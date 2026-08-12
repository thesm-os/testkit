// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package injectionsafe is the mixin-axis fixture for the injectionsafe mixin, which
// declares that a value carrying a control sequence is stored and returned
// as data rather than interpreted.
//
// The store-and-load pair is the claim's own shape: a hostile value goes in
// under a key, and what the key answers afterwards is the data that went in —
// never an interpretation of it. AUTO-INJECTION-SAFE probes exactly that
// round trip with adversarial payloads on both slots.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package injectionsafe

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out injectionsafetest/ pkg=injectionsafetest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Store keeps the value as data, under the key as data.
	//testkit:mixin injectionsafe
	Store(ctx context.Context, key, value string) error

	// Load answers what Store kept, uninterpreted.
	Load(ctx context.Context, key string) (string, error)
}
