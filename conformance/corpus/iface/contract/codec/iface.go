// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package codec is the contract-axis fixture for the codec contract: a
// forward transform and the inverse that undoes it.
//
// `fidelity` says which equality the pair claims. This fixture declares the
// default `exact` explicitly rather than omitting it, so the fixture states
// what it means instead of relying on a reader knowing the default.
//
// Every role the contract declares is present, because a contract is a
// multi-callable protocol and a fixture missing a partner exercises the
// validator's failure path rather than the contract itself.
package codec

import (
	"context"
)

// Contract is the fixture interface.
//
//testkit:out codectest/ pkg=codectest
//testkit:stub
//testkit:suite
type Contract interface {
	// Encode is the codec contract's forward role, and hosts the directive
	// that names its partner and the fidelity it claims.
	//testkit:contract codec role=forward inverse=Decode fidelity=exact
	Encode(ctx context.Context, in string) (string, error)

	// Decode is the codec contract's inverse role.
	//testkit:contract codec role=inverse
	Decode(ctx context.Context, in string) (string, error)
}
