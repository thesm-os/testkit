// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package embedded is the errors-kind fixture for an error contract reached
// through embedding rather than declared outright.
//
// A struct's declarations are not its method set. `type NotFoundError struct {
// BaseError; Key string }` is the dominant Go idiom for a family of custom
// errors, and a generator reading only the declarations finds no Error method
// on it at all — so the package's directive says its errors are a contract and
// the generated file covers half of them, without reporting that it did.
//
// Every half is reached the same way here: Error, Is and Unwrap are all
// promoted, so a generator reading declarations sees a plain struct and emits
// nothing about it.
//
// The embedder adds no field of its own, and that is deliberate rather than
// minimal. A promoted Error cannot mention a field the embedder declared, so a
// type that both embeds its contract and adds a field is asserting two things
// at once — and the generated "every field reaches the message" check would
// fail for a reason that has nothing to do with promotion.
//
// [go.thesmos.sh/testkit/conformance/corpus/errors/store] is the same contract
// declared directly, which is what makes the pair able to tell "promotion is
// handled" from "the type happened to declare everything".
//
//testkit:sentinel
package embedded

import (
	"errors"
	"fmt"
)

// ErrNotFound is the sentinel the embedded contract wraps.
var ErrNotFound = errors.New("embedded: not found")

// BaseError carries the whole contract, and is embedded rather than used.
type BaseError struct {
	Op    string
	Cause error
}

// Error satisfies the error interface for every type embedding this one.
func (e BaseError) Error() string { return fmt.Sprintf("embedded: %s: %v", e.Op, e.Cause) }

// Is lets errors.Is reach the wrapped sentinel through the embedder.
func (e BaseError) Is(target error) bool { return errors.Is(e.Cause, target) }

// Unwrap exposes the cause to errors.Unwrap through the embedder.
func (e BaseError) Unwrap() error { return e.Cause }

// NotFoundError declares nothing and inherits the entire error contract. It is
// a distinct type errors.As can single out, which is what the idiom buys, and
// nothing about it is visible to a generator reading declarations alone.
type NotFoundError struct {
	BaseError
}
