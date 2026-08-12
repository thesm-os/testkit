// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package codeclossy is the contract-axis fixture for the codec contract's
// lossy form: a forward transform that discards information, paired with an
// inverse that recovers only what survived.
//
// `fidelity=lossy` selects the weaker roundtrip law: not the identity, but
// agreement on the second pass — encode, decode, encode again, and the two
// encodings match because everything the transform was going to lose is
// already gone. The exact form lives in the sibling `codec` fixture; both
// exist because the parameter selects between two laws, and a corpus stating
// only one leaves the other's selection untested.
package codeclossy

import (
	"context"
)

// Contract is the fixture interface.
//
//testkit:out codeclossytest/ pkg=codeclossytest
//testkit:stub
//testkit:suite
//testkit:model
type Contract interface {
	// Encode is the codec contract's forward role, and hosts the directive
	// that names its partner and the fidelity it claims.
	//testkit:contract codec role=forward inverse=Decode fidelity=lossy
	Encode(ctx context.Context, in string) (string, error)

	// Decode is the codec contract's inverse role.
	//testkit:contract codec role=inverse
	Decode(ctx context.Context, in string) (string, error)
}
