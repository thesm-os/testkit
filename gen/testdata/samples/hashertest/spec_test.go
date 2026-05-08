// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package hashertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/testdata/samples"
	"go.thesmos.sh/testkit/gen/testdata/samples/hashertest"
	"go.thesmos.sh/testkit/suite"
)

func TestHasherContract(t *testing.T) {
	t.Parallel()
	factory := func() samples.Hasher { return samples.NewInMemoryHasher() }

	hashertest.AssertHasherContract(t, factory,
		// Combine uses sample builders — this would panic without them.
		hashertest.HasherOnCombine(
			suite.AssertDeterministic[samples.Hasher, samples.Digest](3),
		),
	)
}

func BenchmarkHasherContract(b *testing.B) {
	factory := func() samples.Hasher { return samples.NewInMemoryHasher() }
	hashertest.BenchmarkHasherContract(b, factory)
}
