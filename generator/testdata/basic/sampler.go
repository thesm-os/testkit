// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

//go:generate testkit bench -o storetest/sampler_bench.gen.go Sampler

import "context"

// Sampler exercises the //testkit:sample directive: each method
// declares one sample function per non-ctx parameter, and the
// generator's sample consumer resolves them against this package.
//
// Used by generator/spec/sample tests to validate consumer behavior
// (happy path, count mismatch, missing func, wrong signature).
type Sampler interface {
	// Lookup takes one non-ctx param and gets one sample func.
	//
	//testkit:sample SampleKey
	Lookup(ctx context.Context, key string) (Item, error)

	// Apply takes two non-ctx params and gets two sample funcs.
	//
	//testkit:sample SampleKey SampleItem
	Apply(ctx context.Context, key string, item Item) error
}

// SampleKey returns a non-zero string suitable for Lookup's key
// param. Resolved by the sample consumer at generation time.
func SampleKey() string { return "test-key" }

// SampleItem returns a non-zero Item suitable for Apply's item
// param.
func SampleItem() Item { return Item{ID: "test-id", Name: "test-name"} }
