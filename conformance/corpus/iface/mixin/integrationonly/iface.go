// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package integrationonly is the mixin-axis fixture for the integrationonly mixin, which
// declares that the method needs external infrastructure and is skipped by default.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package integrationonly

import (
	"context"
)

// Mixed is the fixture interface.
//
// Mixed is the fixture interface. The double carries Connect like any other
// method: the mixin is a law for the suite and model tiers about what a test
// runs against, not an instruction to the stub generator, and a double
// omitting the method would not satisfy the interface at all.
//
//testkit:out integrationonlytest/ pkg=integrationonlytest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Connect reaches outside the process. The mixin gates the generated
	// subtest behind a build tag rather than asserting anything at runtime.
	//testkit:mixin integrationonly
	Connect(ctx context.Context, dsn string) error
}
