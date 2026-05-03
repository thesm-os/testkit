// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package multisentinel verifies that the firstSentinel template func
// picks the first sentinel when a method has multiple //testkit:errors
// directives, and that no trailing comma appears in the generated output.
package multisentinel

import (
	"context"
	"errors"
)

//go:generate testkit model -o vaulttest/vault_model.gen.go Vault

// ErrNotFound is returned when a secret is not found.
var ErrNotFound = errors.New("not found")

// ErrSealed is returned when the vault is sealed.
var ErrSealed = errors.New("sealed")

// Secret is a stored value.
type Secret struct {
	ID    string
	Value string
}

// Vault has a Reader with multiple sentinel errors.
type Vault interface {
	//testkit:errors ErrNotFound
	//testkit:errors ErrSealed
	Get(ctx context.Context, id string) (Secret, error)

	Put(ctx context.Context, secret Secret) error
}
