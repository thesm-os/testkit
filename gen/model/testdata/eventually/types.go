// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package eventually exercises the EventuallyAfter trace combinator
// with a minimal 2-method interface where the trigger→response budget
// is reliable under random action selection.
package eventually

import (
	"context"
	"errors"
)

//go:generate testkit model -o buffertest/buffer_model.gen.go Buffer

// ErrNotFound is returned when the buffer is empty.
var ErrNotFound = errors.New("not found")

// Buffer is a minimal interface with Write (trigger) and Read (response).
type Buffer interface {
	//testkit:errors ErrNotFound
	Read(ctx context.Context, key string) (string, error)

	Write(ctx context.Context, value string) error
}
