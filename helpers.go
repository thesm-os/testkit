// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"cmp"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"net"
	"os"
	"slices"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
)

// TestError returns a deterministic sentinel error whose message is based on
// name. Two calls with the same name return errors that satisfy [errors.Is];
// two calls with different names do not. Use this to create distinguishable
// error values in tests without polluting the production error namespace.
//
//	errBoom := testkit.TestError("boom")
//	testkit.ErrorIs(t, err, errBoom, "must return boom")
func TestError(name string) error {
	return &testError{name: name}
}

type testError struct{ name string }

func (e *testError) Error() string { return "testkit: " + e.name }

func (e *testError) Is(target error) bool {
	other, ok := target.(*testError)
	return ok && other.name == e.name
}

// RequireEnv skips the test if the environment variable key is not set.
// Returns the value if present.
//
//	dsn := testkit.RequireEnv(t, "DATABASE_URL")
func RequireEnv(tb testing.TB, key string) string {
	tb.Helper()
	v, ok := os.LookupEnv(key)
	if !ok {
		tb.Skipf("skipping: %s not set", key)
	}
	return v
}

// SeededRand returns a deterministic [rand.Rand] seeded from the FNV-1a hash
// of tb.Name(). The same test name produces the same byte sequence across
// runs, making flaky-test reproduction possible without logging seeds.
//
//	r := testkit.SeededRand(t)
//	id := r.IntN(1000)
func SeededRand(tb testing.TB) *rand.Rand {
	tb.Helper()
	h := fnv.New64a()
	_, _ = h.Write([]byte(tb.Name()))
	return rand.New(rand.NewPCG(h.Sum64(), 0)) //nolint:gosec // deterministic seed is intentional
}

// TempFile creates a temporary file in tb.TempDir() with the given name and
// content. The file is automatically cleaned up when the test finishes.
// Returns the absolute path to the file.
//
//	path := testkit.TempFile(t, "config.json", []byte(`{"key":"value"}`))
func TempFile(tb testing.TB, name string, content []byte) string {
	tb.Helper()
	path := tb.TempDir() + "/" + name
	err := os.WriteFile(path, content, 0o644) //nolint:gosec // test file permissions are fine
	if err != nil {
		tb.Fatalf("TempFile: write %s: %v", name, err)
	}
	return path
}

// FreePort returns an available TCP port on localhost. It binds to port 0,
// reads the assigned port, and closes the listener before returning. There is
// a small window where another process could claim the port — this is
// acceptable for test setup.
//
//	port := testkit.FreePort(t)
//	srv := httptest.NewServer(handler) // use port
func FreePort(tb testing.TB) int {
	tb.Helper()
	var lc net.ListenConfig
	l, err := lc.Listen(tb.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		tb.Fatalf("FreePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	// Closing a listener that was just opened has no buffered state to flush
	// and a known-good descriptor, so the error is discarded rather than
	// guarded by a branch no test can reach.
	_ = l.Close()
	return port
}

// SortedKeys returns the keys of m in sorted order. Use this to iterate maps
// deterministically in test output.
//
//	for _, k := range testkit.SortedKeys(counts) {
//	    t.Logf("%s: %d", k, counts[k])
//	}
func SortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// MapDiff describes the difference between two maps with the same key type.
type MapDiff[K comparable, V any] struct {
	Added   map[K]V    // keys in after but not before
	Removed map[K]V    // keys in before but not after
	Changed map[K][2]V // keys in both with different values (before, after)
}

// DiffMap computes the structural difference between before and after. Values
// are compared using [cmp.Equal] from go-cmp.
//
//	diff := testkit.DiffMap(beforeCounts, afterCounts)
//	testkit.Len(t, diff.Added, 1, "must add exactly one key")
func DiffMap[K comparable, V any](before, after map[K]V) MapDiff[K, V] {
	d := MapDiff[K, V]{
		Added:   make(map[K]V),
		Removed: make(map[K]V),
		Changed: make(map[K][2]V),
	}
	for k, bv := range before {
		av, ok := after[k]
		if !ok {
			d.Removed[k] = bv
			continue
		}
		if !gocmp.Equal(bv, av) {
			d.Changed[k] = [2]V{bv, av}
		}
	}
	for k, av := range after {
		if _, ok := before[k]; !ok {
			d.Added[k] = av
		}
	}
	return d
}

// TableTest runs a table-driven test using the provided cases. Each case's
// name is derived from its Name field if it implements the [interface{ Name() string }]
// interface, otherwise it uses the index. The run function receives the test
// and the case value.
//
//	testkit.TableTest(t, []myCase{
//	    {name: "empty", input: "", want: 0},
//	    {name: "single", input: "a", want: 1},
//	}, func(t *testing.T, tc myCase) {
//	    got := len(tc.input)
//	    testkit.Equal(t, got, tc.want, tc.name)
//	})
func TableTest[T any](t *testing.T, cases []T, run func(t *testing.T, tc T)) {
	t.Helper()
	for i, tc := range cases {
		name := fmt.Sprintf("#%d", i)
		if n, ok := any(tc).(interface{ Name() string }); ok {
			name = n.Name()
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			run(t, tc)
		})
	}
}
