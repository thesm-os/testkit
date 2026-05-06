// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package timeaware exercises the //testkit:time-aware directive
// for clock-injected deterministic time testing. The Store interface
// has TTL-based expiry: items become unreachable after a fixed TTL
// from the time of their Put.
package timeaware

import (
	"context"
	"errors"
	"time"
)

//go:generate testkit model -o storetest/store_model.gen.go Store

// ErrNotFound is returned when a key is not present or has expired.
var ErrNotFound = errors.New("not found")

// DefaultTTL is the item expiration window.
const DefaultTTL = 10 * time.Minute

// Item is a stored entry.
type Item struct {
	ID   string
	Name string
}

// Store is a TTL-based key-value store. Items expire after [DefaultTTL].
// Implementations receive a [clock.Clock] for deterministic testing.
//
//testkit:time-aware
type Store interface {
	//testkit:errors ErrNotFound
	Get(ctx context.Context, id string) (Item, error)

	Put(ctx context.Context, item Item) error

	//testkit:deleter
	Delete(ctx context.Context, id string) error

	Count(ctx context.Context) (int, error)
}
