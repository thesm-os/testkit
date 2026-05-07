// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package samples exercises the //testkit:sample directive, proving
// that sample builder functions replace zero-value parameters in
// smoke tests and plug-in dispatch Call closures.
package samples

//go:generate testkit suite -o hashertest/hasher_spec.gen.go Hasher
//go:generate testkit bench -o hashertest/hasher_bench.gen.go Hasher

// Digest is a fixed-size hash that panics on zero-value operations.
type Digest [32]byte

// SampleDigest returns a Digest suitable for testing the given Hasher.
// The builder takes the SUT so it can produce impl-aware values.
func SampleDigest(_ Hasher) Digest {
	var d Digest
	d[0] = 0x42
	return d
}

// Hasher combines digests. Combine panics if either argument is
// the zero-value Digest — this exercises the sample directive.
type Hasher interface {
	// Combine merges two digests into one. Panics on zero-value input.
	// SampleDigest lives in the source package — resolved with qualifier.
	//testkit:sample SampleDigest SampleDigest
	Combine(a, b Digest) Digest

	// Verify checks a digest against data. Panics on zero-value digest.
	// TestSampleDigest lives in the output package — resolved unqualified.
	//testkit:sample TestSampleDigest
	Verify(d Digest) bool

	// Name returns the hash algorithm name — no params, no sample needed.
	Name() string
}
