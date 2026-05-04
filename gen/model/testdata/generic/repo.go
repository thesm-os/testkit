// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package generic exercises the model generator against a generic interface
// with type parameters, verifying shape detection, keyfield heuristic,
// and code generation all work through Go's type parameter instantiation.
package generic

import (
	"context"
	"errors"
)

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")

// Repository is a generic CRUD interface with type parameters.
type Repository[K comparable, V any] interface {
	//testkit:errors ErrNotFound
	Get(ctx context.Context, k K) (V, error)

	Put(ctx context.Context, v V) error

	//testkit:deleter
	Delete(ctx context.Context, k K) error

	Count(ctx context.Context) (int, error)
}

// Item is a stored value with an ID field for keyfield heuristic.
type Item struct {
	ID    string
	Name  string
	Score int
}

// ItemRepository is a concrete instantiation of [Repository] with
// string keys and [Item] values. The generator targets this type.
//
//go:generate testkit model -o repotest/repo_model.gen.go ItemRepository
type ItemRepository = Repository[string, Item]

//go:generate testkit model -o repotest/repo_generic_model.gen.go Repository
