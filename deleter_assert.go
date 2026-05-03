// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// DeleterBindings holds the reusable shape wiring for a Deleter-shaped method.
// Shared by suite (via DeleterContext) and future generators (bench, model).
type DeleterBindings[T any, K comparable] struct {
	Factory func() T
	Call    func(context.Context, T, K) error
}

// DeleterContext provides a typed factory and call function to Deleter-shape
// primitives. A Deleter-shaped method has the signature func(ctx, K) error
// and requires the //testkit:deleter directive.
type DeleterContext[T any, K comparable] struct {
	T *testing.T
	DeleterBindings[T, K]
}

// DeleterAssertion is a typed conformance primitive for Deleter-shaped methods.
type DeleterAssertion[T any, K comparable] func(DeleterContext[T, K])

// AssertDeleteSucceeds deletes the given key and asserts no error.
// The consumer should pre-populate the key via the factory.
func AssertDeleteSucceeds[T any, K comparable](key K) DeleterAssertion[T, K] {
	return func(ctx DeleterContext[T, K]) {
		ctx.T.Run(fmt.Sprintf("delete succeeds %v", key), func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, key)
			NoError(t, err, "delete must succeed for existing key")
		})
	}
}

// AssertDeleteIdempotent deletes the given key twice and asserts neither
// call returns an error (or both return the same error).
func AssertDeleteIdempotent[T any, K comparable](key K) DeleterAssertion[T, K] {
	return func(ctx DeleterContext[T, K]) {
		ctx.T.Run("delete idempotent", func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err1 := ctx.Call(t.Context(), impl, key)
			err2 := ctx.Call(t.Context(), impl, key)
			if err1 == nil && err2 == nil {
				return
			}
			if err1 != nil && err2 != nil && (errors.Is(err1, err2) || errors.Is(err2, err1)) {
				return
			}
			t.Errorf("delete must be idempotent: first=%v, second=%v", err1, err2)
		})
	}
}

// AssertDeleteReturnsNotFound deletes an unknown key and asserts the
// given not-found sentinel is returned.
func AssertDeleteReturnsNotFound[T any, K comparable](unknown K, sentinel error) DeleterAssertion[T, K] {
	return func(ctx DeleterContext[T, K]) {
		ctx.T.Run(fmt.Sprintf("delete returns not-found for %v", unknown), func(t *testing.T) {
			t.Parallel()
			impl := ctx.Factory()
			err := ctx.Call(t.Context(), impl, unknown)
			ErrorIs(t, err, sentinel, "delete of unknown key must return sentinel")
		})
	}
}
