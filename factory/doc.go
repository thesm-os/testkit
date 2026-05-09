// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package factory ships small primitives shared across testkit's
// generators: [Named] for multi-impl conformance harnesses and
// [SeedFromEnv] for the canonical seed-resolution contract used by
// every generator-emitted test.
//
// [Named] is a tagged constructor closure. Generated code threads
// it through `*AcrossImpls` entry points so failures cite a stable
// implementation name rather than a positional index. The model,
// sim, and differential-rollout generators all consume it.
//
// [SeedFromEnv] resolves a deterministic seed for property-based
// runs. Each generator emits a call with its own (pkgPrefix,
// generator) tuple — `STORETEST_MODEL_SEED`, `LEDGERTEST_SIM_SEED`,
// `LEDGERTEST_CHAOS_SEED`, etc. The function checks the named env
// var first, then the global `TESTKIT_SEED`, then falls back to a
// wall-clock-derived seed that is logged via `t.Logf` so reruns
// can pin it.
//
// Invalid env-var values fail loud via `t.Fatalf` rather than
// silent fallback, matching the always-err policy that runs across
// the testkit substrate.
package factory
