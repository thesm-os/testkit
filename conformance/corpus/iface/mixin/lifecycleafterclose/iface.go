// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lifecycleafterclose is the mixin-axis fixture for the lifecycleafterclose mixin, which
// declares that every call after teardown reports the closed sentinel.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package lifecycleafterclose

import (
	"context"
	"errors"
)

// ErrClosed is what operations report after teardown.
var ErrClosed = errors.New("lifecycleafterclose: closed")

// Mixed is the fixture interface.
//
//testkit:out lifecycleafterclosetest/ pkg=lifecycleafterclosetest
//testkit:stub
type Mixed interface {
	// Close is the teardown the law measures from.
	//testkit:mixin lifecycleafterclose
	Close(ctx context.Context) error

	// Work must report [ErrClosed] once Close has run. Without an operation
	// to reject, closure is unobservable.
	Work(ctx context.Context) error
}
