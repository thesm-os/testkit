// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package hashlogtest_test

import (
	"testing"

	ah "go.thesmos.sh/testkit/gen/model/testdata/auditchain_hash"
	"go.thesmos.sh/testkit/gen/model/testdata/auditchain_hash/hashlogtest"
)

func TestHashLogModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 chain with SHA-512 hash override", func(t *testing.T) {
		t.Parallel()
		// Tier 0: framework auto-synthesizes a refchain.AppendOnly[Entry]
		// using the consumer-supplied SHA512Hash via //testkit:hash directive.
		// All chain laws fire with the custom hash function.
		hashlogtest.AssertHashLogModel(t,
			func() ah.HashLog { return ah.NewInMemoryHashLog() },
		)
	})
}
