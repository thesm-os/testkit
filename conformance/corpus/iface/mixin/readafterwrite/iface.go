// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readafterwrite is the mixin-axis fixture for the readafterwrite mixin, which
// declares that a value written is immediately readable.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package readafterwrite

import (
	"context"
)

// Mixed is the fixture interface.
//
//testkit:out readafterwritetest/ pkg=readafterwritetest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Read must observe what Write stored. The write parameter names the
	// writer, so the pair has to be in one interface — this is the fixture
	// the single-method template could not express.
	//testkit:mixin readafterwrite write=Write
	Read(ctx context.Context, key string) (string, error)

	// Write is the writer the mixin names.
	Write(ctx context.Context, key, value string) error
}
