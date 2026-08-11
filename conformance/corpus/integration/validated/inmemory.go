// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package validated

import (
	"context"
	"errors"
	"strings"
	"sync"
)

// The errors the store reports, named so a caller can tell a refused value from
// a missing one.
var (
	ErrNotFound = errors.New("validated: not found")
	ErrInvalid  = errors.New("validated: invalid account")
)

// uuidLen is the length of the canonical UUID spelling, which is all this
// fixture checks — the point is that *something* is checked, not that the
// checking is thorough.
const uuidLen = 36

// InMemory validates before it stores, which is what makes this fixture worth
// having: a derived sample fails here, and the generated harness must pass
// anyway because the source declares [AccountDefaults].
type InMemory struct {
	mu       sync.Mutex
	accounts map[string]Account
}

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{accounts: map[string]Account{}} }

// Put refuses an Account whose fields do not validate.
func (s *InMemory) Put(ctx context.Context, a Account) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if len(a.ID) != uuidLen || !strings.Contains(a.Email, "@") {
		return ErrInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts[a.ID] = a
	return nil
}

// Get returns the zero value alongside every error it reports.
func (s *InMemory) Get(ctx context.Context, id string) (Account, error) {
	if err := contextErr(ctx); err != nil {
		return Account{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.accounts[id]
	if !ok {
		return Account{}, ErrNotFound
	}
	return a, nil
}

// contextErr reports a cancelled or expired context and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this not panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("validated: nil context")
	}
	return ctx.Err()
}
