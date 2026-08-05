// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

import (
	"context"
	"fmt"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model"
)

// ConcurrentReader creates a ConcurrentAction for a Reader-shaped method.
// Draws a key, calls impl.Read, records (key, ReaderResult{V, error}).
func ConcurrentReader[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) (V, error),
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return keys.Draw(rt, name+"_key")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			k := input.(K)
			v, err := read(ctx, impl, k)
			return ReaderResult[V]{Value: v, Err: err}
		},
		PartitionKey: func(input any) string {
			return fmt.Sprint(input)
		},
	}
}

// ConcurrentReaderWithBool creates a ConcurrentAction for a
// ReaderWithBool-shaped method: func(ctx, K) (V, bool).
func ConcurrentReaderWithBool[T any, K comparable, V any](
	name string,
	keys *rapid.Generator[K],
	read func(context.Context, T, K) (V, bool),
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return keys.Draw(rt, name+"_key")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			k := input.(K)
			v, ok := read(ctx, impl, k)
			return ReaderBoolResult[V]{Value: v, OK: ok}
		},
		PartitionKey: func(input any) string {
			return fmt.Sprint(input)
		},
	}
}

// ConcurrentWriter creates a ConcurrentAction for a Writer-shaped method.
// Draws a value, calls impl.Write, partitions by keyOf(value).
func ConcurrentWriter[T any, K comparable, V any](
	name string,
	values *rapid.Generator[V],
	write func(context.Context, T, V) error,
	keyOf func(V) K,
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return values.Draw(rt, name+"_value")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			v := input.(V)
			err := write(ctx, impl, v)
			return WriterResult{Err: err}
		},
		PartitionKey: func(input any) string {
			v := input.(V)
			return fmt.Sprint(keyOf(v))
		},
	}
}

// ConcurrentDeleter creates a ConcurrentAction for a Deleter-shaped method.
func ConcurrentDeleter[T any, K comparable](
	name string,
	keys *rapid.Generator[K],
	del func(context.Context, T, K) error,
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return keys.Draw(rt, name+"_key")
		},
		Apply: func(ctx context.Context, impl T, input any) any {
			k := input.(K)
			err := del(ctx, impl, k)
			return WriterResult{Err: err}
		},
		PartitionKey: func(input any) string {
			return fmt.Sprint(input)
		},
	}
}

// ConcurrentLookup creates a ConcurrentAction for a Lookup-shaped method:
// func(T, K) (R1, R2, bool).
func ConcurrentLookup[T any, K comparable, R1, R2 any](
	name string,
	keys *rapid.Generator[K],
	lookup func(T, K) (R1, R2, bool),
) model.ConcurrentAction[T] {
	return model.ConcurrentAction[T]{
		Name: name,
		Gen: func(rt *rapid.T) any {
			return keys.Draw(rt, name+"_key")
		},
		Apply: func(_ context.Context, impl T, input any) any {
			k := input.(K)
			r1, r2, ok := lookup(impl, k)
			return LookupResult[R1, R2]{R1: r1, R2: r2, OK: ok}
		},
		PartitionKey: func(input any) string {
			return fmt.Sprint(input)
		},
	}
}
