// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package factory_test

import (
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/factory"
)

func TestEnvVarName(t *testing.T) {
	t.Parallel()

	t.Run("formats <PKG>_<GENERATOR>_SEED", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, factory.EnvVarName("STORETEST", "MODEL"),
			"STORETEST_MODEL_SEED", "canonical name format")
	})

	t.Run("varies across generators", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, factory.EnvVarName("LEDGERTEST", "CHAOS"),
			"LEDGERTEST_CHAOS_SEED", "per-generator naming")
	})
}

// TestSeedFromEnv covers seed resolution. The whole function is
// non-parallel because every subtest uses t.Setenv.
//
//nolint:paralleltest // t.Setenv is incompatible with t.Parallel
func TestSeedFromEnv(t *testing.T) {
	t.Run("primary env var wins", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "0xCAFEBABE")
		f := testkit.NewFailableTB()

		seed := factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.Equal(t, seed, int64(0xCAFEBABE), "primary env wins")
		testkit.False(t, f.Failed(), "no fatal expected")
		assertLogContains(t, f.Logs(), "STORETEST_MODEL_SEED")
	})

	t.Run("decimal value is parsed", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "12345")
		f := testkit.NewFailableTB()

		seed := factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.Equal(t, seed, int64(12345), "decimal accepted")
		testkit.False(t, f.Failed(), "no fatal expected")
	})

	t.Run("falls back to TESTKIT_SEED when primary unset", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "")
		t.Setenv("TESTKIT_SEED", "0xDEADBEEF")
		f := testkit.NewFailableTB()

		seed := factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.Equal(t, seed, int64(0xDEADBEEF), "fallback to TESTKIT_SEED")
		testkit.False(t, f.Failed(), "no fatal expected")
		assertLogContains(t, f.Logs(), "TESTKIT_SEED")
	})

	t.Run("falls back to wall clock when both unset", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "")
		t.Setenv("TESTKIT_SEED", "")
		f := testkit.NewFailableTB()

		_ = factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.False(t, f.Failed(), "no fatal expected")
		assertLogContains(t, f.Logs(), "rerun with STORETEST_MODEL_SEED=")
	})

	t.Run("falls back when env vars are genuinely unset", func(t *testing.T) {
		// t.Setenv keeps a var set (with empty value); to exercise
		// the truly-unset branch in nonEmptyEnv we use a prefix that
		// no developer or CI would set.
		t.Setenv("TESTKIT_SEED", "")
		f := testkit.NewFailableTB()

		_ = factory.SeedFromEnv(f, "FACTORYTEST_UNSET_XYZZY", "MODEL")

		testkit.False(t, f.Failed(), "no fatal expected")
		assertLogContains(t, f.Logs(), "rerun with FACTORYTEST_UNSET_XYZZY_MODEL_SEED=")
	})

	t.Run("primary empty falls through to fallback", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "")
		t.Setenv("TESTKIT_SEED", "0xABCD1234")
		f := testkit.NewFailableTB()

		seed := factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.Equal(t, seed, int64(0xABCD1234), "empty primary falls through")
	})

	t.Run("primary whitespace-only falls through", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "   ")
		t.Setenv("TESTKIT_SEED", "0xFEEDFACE")
		f := testkit.NewFailableTB()

		seed := factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.Equal(t, seed, int64(0xFEEDFACE), "whitespace-only primary falls through")
	})

	t.Run("invalid hex value fatals on primary", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "0xZZ")
		f := testkit.NewFailableTB()

		_ = factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.True(t, f.Failed(), "must fatal on invalid hex")
		testkit.Assert(t, f.Msg()).Contains("invalid hex seed", "diagnostic")
	})

	t.Run("invalid decimal value fatals on primary", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "not-a-number")
		f := testkit.NewFailableTB()

		_ = factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.True(t, f.Failed(), "must fatal on invalid decimal")
		testkit.Assert(t, f.Msg()).Contains("invalid decimal seed", "diagnostic")
	})

	t.Run("invalid fallback fatals citing fallback name", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "")
		t.Setenv("TESTKIT_SEED", "0xGGGG")
		f := testkit.NewFailableTB()

		_ = factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.True(t, f.Failed(), "must fatal on invalid fallback")
		testkit.Assert(t, f.Msg()).
			Contains("TESTKIT_SEED", "names fallback").
			Contains("invalid hex seed", "cites parse error")
	})

	t.Run("uppercase 0X hex prefix accepted", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "0XBEEF")
		f := testkit.NewFailableTB()
		seed := factory.SeedFromEnv(f, "STORETEST", "MODEL")
		testkit.Equal(t, seed, int64(0xBEEF), "uppercase 0X prefix accepted")
	})

	t.Run("negative decimal accepted", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "-1")
		f := testkit.NewFailableTB()
		seed := factory.SeedFromEnv(f, "STORETEST", "MODEL")
		testkit.Equal(t, seed, int64(-1), "negative decimal accepted")
	})

	t.Run("whitespace around value tolerated", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "  0xCAFE  ")
		f := testkit.NewFailableTB()
		seed := factory.SeedFromEnv(f, "STORETEST", "MODEL")
		testkit.Equal(t, seed, int64(0xCAFE), "whitespace trimmed")
	})

	t.Run("decimal overflow is rejected", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "99999999999999999999")
		f := testkit.NewFailableTB()

		_ = factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.True(t, f.Failed(), "must fatal on overflow")
		testkit.Assert(t, f.Msg()).Contains("invalid decimal seed", "diagnostic")
	})

	t.Run("hex overflow is rejected", func(t *testing.T) {
		t.Setenv("STORETEST_MODEL_SEED", "0xFFFFFFFFFFFFFFFFFF")
		f := testkit.NewFailableTB()

		_ = factory.SeedFromEnv(f, "STORETEST", "MODEL")

		testkit.True(t, f.Failed(), "must fatal on hex overflow")
		testkit.Assert(t, f.Msg()).Contains("invalid hex seed", "diagnostic")
	})
}

func assertLogContains(t *testing.T, logs []string, substr string) {
	t.Helper()
	for _, line := range logs {
		if strings.Contains(line, substr) {
			return
		}
	}
	t.Errorf("expected log containing %q; got logs: %v", substr, logs)
}
