// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multisentinel

import (
	"context"
	"sync"
)

// InMemoryVault implements [Vault] for model testing.
type InMemoryVault struct {
	mu   sync.Mutex
	data map[string]Secret
}

// NewInMemoryVault returns a ready-to-use [InMemoryVault].
func NewInMemoryVault() *InMemoryVault {
	return &InMemoryVault{data: make(map[string]Secret)}
}

func (v *InMemoryVault) Get(_ context.Context, id string) (Secret, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	s, ok := v.data[id]
	if !ok {
		return Secret{}, ErrNotFound
	}
	return s, nil
}

func (v *InMemoryVault) Put(_ context.Context, secret Secret) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.data[secret.ID] = secret
	return nil
}
