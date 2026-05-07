// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package samples

// InMemoryHasher implements [Hasher] for testing.
type InMemoryHasher struct{}

func NewInMemoryHasher() *InMemoryHasher { return &InMemoryHasher{} }

func (h *InMemoryHasher) Combine(a, b Digest) Digest {
	if a == (Digest{}) || b == (Digest{}) {
		panic("Combine: zero-value Digest not allowed")
	}
	var out Digest
	for i := range out {
		out[i] = a[i] ^ b[i]
	}
	return out
}

func (h *InMemoryHasher) Verify(d Digest) bool {
	if d == (Digest{}) {
		panic("Verify: zero-value Digest not allowed")
	}
	return d[0] != 0
}

func (h *InMemoryHasher) Name() string { return "xor" }
