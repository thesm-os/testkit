// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package factory

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"go.thesmos.sh/testkit/clock"
)

// EnvSeedSuffix is the trailing fragment every generator-specific
// seed env var ends with. Combined with a package prefix and a
// generator name to form names like `STORETEST_MODEL_SEED`.
const EnvSeedSuffix = "_SEED"

// EnvFallbackSeed names the global env var consulted when the
// generator-specific seed env var is unset. Letting the consumer
// pin one seed for an entire CI run.
const EnvFallbackSeed = "TESTKIT_SEED"

// SeedFromEnv resolves a deterministic seed for a generator-emitted
// property-based test. Resolution order:
//
//  1. The generator-specific env var <PKG>_<GENERATOR>_SEED if set.
//     Example: pkgPrefix="STORETEST", generator="MODEL" reads
//     STORETEST_MODEL_SEED.
//  2. The global TESTKIT_SEED if set.
//  3. A clock-derived seed (UnixNano from [clock.RealClock]). The
//     chosen seed is logged via tb.Logf so a failing run can be
//     pinned by setting either env var on rerun.
//
// pkgPrefix and generator are passed verbatim — the function does
// not uppercase or transform them. Generator-emitted code passes
// already-uppercased values to keep the env-var name a stable string
// constant rather than a runtime computation.
//
// Both decimal and hex (with "0x" prefix) are accepted in env-var
// values. Invalid values fail loud via tb.Fatalf rather than silent
// fallback, matching the always-err policy.
func SeedFromEnv(tb testing.TB, pkgPrefix, generator string) int64 {
	tb.Helper()

	envName := pkgPrefix + "_" + generator + EnvSeedSuffix
	if v, ok := nonEmptyEnv(envName); ok {
		seed, err := parseSeed(v)
		if err != nil {
			tb.Fatalf("factory.SeedFromEnv: %s=%q: %v", envName, v, err)
		}
		tb.Logf("seed: 0x%016X (from %s)", uint64(seed), envName)
		return seed
	}

	if v, ok := nonEmptyEnv(EnvFallbackSeed); ok {
		seed, err := parseSeed(v)
		if err != nil {
			tb.Fatalf("factory.SeedFromEnv: %s=%q: %v", EnvFallbackSeed, v, err)
		}
		tb.Logf("seed: 0x%016X (from %s)", uint64(seed), EnvFallbackSeed)
		return seed
	}

	seed := clockSeed()
	tb.Logf("seed: 0x%016X (rerun with %s=0x%016X)", uint64(seed), envName, uint64(seed))
	return seed
}

// EnvVarName returns the canonical seed env-var name for a
// (pkgPrefix, generator) pair, formatted per the
// `<PKG>_<GENERATOR>_SEED` convention. Exported so generator
// templates can emit the name as a string constant alongside the
// SeedFromEnv call without duplicating the formatting rule.
func EnvVarName(pkgPrefix, generator string) string {
	return pkgPrefix + "_" + generator + EnvSeedSuffix
}

func parseSeed(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		v, err := strconv.ParseUint(s[2:], 16, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid hex seed: %w", err)
		}
		return int64(v), nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid decimal seed: %w", err)
	}
	return v, nil
}

// nonEmptyEnv returns the env var's value only when it is set AND
// not empty (after whitespace trimming). Empty strings are treated
// as unset so callers can clear an inherited env var with
// `t.Setenv(name, "")` and have the resolver fall through to the
// next layer rather than fatal on an empty parse.
func nonEmptyEnv(name string) (string, bool) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return "", false
	}
	if strings.TrimSpace(v) == "" {
		return "", false
	}
	return v, true
}

// clockSeed derives a fallback seed from the wall clock via the
// testkit clock package. Two CI runs at different moments produce
// different seeds; consecutive calls within one run produce
// monotonic-but-distinct values. The seed is reproducible after the
// fact by reading the value logged in tb.Logf.
func clockSeed() int64 {
	return clock.RealClock().Now().UnixNano()
}
