// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package keyfield exercises the //testkit:keyfield directive override,
// proving the generator picks the directive over the ID-field heuristic.
package keyfield

import (
	"context"
	"errors"
)

//go:generate testkit model -o storetest/store_model.gen.go Store

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")

// Record uses "Key" as its primary field, not "ID".
type Record struct {
	Key   string
	Value string
}

// Store uses //testkit:keyfield to override the ID heuristic.
type Store interface {
	//testkit:errors ErrNotFound
	Get(ctx context.Context, key string) (Record, error)

	//testkit:keyfield Key
	Put(ctx context.Context, record Record) error

	Delete(ctx context.Context, key string) error
}
