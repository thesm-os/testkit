// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package leakcheck exercises goroutine leak detection via the
// NoGoroutineLeaks FinalLaw.
package leakcheck

import (
	"context"
	"errors"
)

//go:generate testkit model -o storetest/store_model.gen.go Store

// ErrNotFound is returned when a key is not present.
var ErrNotFound = errors.New("not found")

// Item is a stored entry.
type Item struct {
	ID   string
	Name string
}

// Store is a simple key-value store.
type Store interface {
	//testkit:errors ErrNotFound
	Get(ctx context.Context, id string) (Item, error)

	Put(ctx context.Context, item Item) error

	//testkit:deleter
	Delete(ctx context.Context, id string) error

	Count(ctx context.Context) (int, error)
}
