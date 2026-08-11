// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package embeddedforeign is the language-axis fixture for an interface
// embedding one from a package outside the run.
//
// Embedding a standard-library interface is ordinary code rather than an exotic
// case, and it used to cost this fixture everything: embeds were flattened by
// resolving each against the interfaces the run had loaded, `io.Closer` was
// never among them, and both generators declined the whole interface — a double
// missing a method does not satisfy the interface it doubles, and a harness over
// part of a contract passes an implementation that fails the rest.
//
// The frontend had the answer the whole time. It type-checks the embed to
// validate it and knows the method set; only the node graph was narrowed to the
// run's own source. eidos now carries the type-checked projection on the embed
// itself, so a foreign method set is completed from what the compiler already
// computed and a loaded declaration still wins where there is one.
//
// What this fixture proves is that the completion reaches the output: Close
// arrives from [io.Closer] with no declaration in this package, and the double,
// the harness and the harness's own checks all carry it beside Read.
//
// [embedded] holds the same shape with every embed declared locally. The two
// must agree, which is the point of having both.
package embeddedforeign

import (
	"context"
	"io"
)

// Stream embeds a standard-library interface alongside a method of its own.
// Close comes from [io.Closer]; Read is declared here.
//
//testkit:out embeddedforeigntest/ pkg=embeddedforeigntest
//testkit:stub
//testkit:suite
type Stream interface {
	io.Closer

	Read(ctx context.Context, key string) (string, error)
}
