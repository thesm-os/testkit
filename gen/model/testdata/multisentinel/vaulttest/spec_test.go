// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package vaulttest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/model/testdata/multisentinel"
	"go.thesmos.sh/testkit/gen/model/testdata/multisentinel/vaulttest"
)

func TestInMemoryVaultModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 with multi-sentinel", func(t *testing.T) {
		t.Parallel()
		// Vault.Get has two //testkit:errors directives (ErrNotFound, ErrSealed).
		// The generator picks the first (ErrNotFound) for refmap synthesis
		// and DeleteReturnsNotFound law.
		vaulttest.AssertVaultModel(t, func() multisentinel.Vault {
			return multisentinel.NewInMemoryVault()
		})
	})
}
