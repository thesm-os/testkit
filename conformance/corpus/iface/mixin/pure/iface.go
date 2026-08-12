// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package pure is the mixin-axis fixture for the pure mixin, which
// declares that the method has no side effects and repeated calls agree.
//
// The interface carries whatever methods the law needs to be stateable.
// A mixin whose law spans two calls cannot be hosted by a single method:
// there would be nothing to compare against, and the generated subtest
// would pass by having nothing to check.
//
// There is no negated form here. eidos declares the mixin directive
// DenyNegation, because a mixin is opt-in and deleting the directive is
// the suppression (docs/adr/0016).
package pure

// Mixed is the fixture interface.
//
//testkit:out puretest/ pkg=puretest
//testkit:stub
//testkit:suite
//testkit:model
type Mixed interface {
	// Derive takes an input and returns a result computed from it alone. A
	// context parameter would suggest I/O, which is what purity excludes.
	//testkit:mixin pure
	Derive(input string) string
}
