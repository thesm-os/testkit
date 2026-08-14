// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"cmp"
	"fmt"
	"hash/fnv"
	"iter"
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

// EmptySeq answers an empty sequence shaped like the one handed in.
//
// For a generated defect worn on a method that streams. The zero value of an
// iterator is a nil function, and ranging over one panics — so a mutant that
// "answers nothing" by returning the zero takes the run down instead of
// testing anything, and the law it was worn for never gets a verdict.
//
// The argument is read for its type and nothing else: an empty sequence
// cannot be written without naming the element types, and inference supplies
// them from the real call the mutant is standing in for. Passing the
// subject's own sequence keeps the generated line honest about what it
// replaced.
func EmptySeq[V any](iter.Seq[V]) iter.Seq[V] {
	return func(func(V) bool) {}
}

// EmptySeq2 answers an empty two-value sequence shaped like the one handed
// in — [EmptySeq] for the `iter.Seq2` half, and for the same reason.
func EmptySeq2[K, V any](iter.Seq2[K, V]) iter.Seq2[K, V] {
	return func(func(K, V) bool) {}
}

// FadedSeq answers the sequence reversed and one element short — the drain
// that lies about what it already showed.
//
// Collected first: the order and the length are properties of the whole
// sequence, so there is nothing to reverse until it has ended. A defect that
// streamed lazily could not be one, which is why this shape is the collected
// one.
func FadedSeq[V any](seq iter.Seq[V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		out := slices.Collect(seq)
		slices.Reverse(out)
		if len(out) > 0 {
			out = out[:len(out)-1]
		}
		for _, v := range out {
			if !yield(v) {
				return
			}
		}
	}
}

// FadedSeq2 is [FadedSeq] for the two-value form.
func FadedSeq2[K, V any](seq iter.Seq2[K, V]) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		type pair struct {
			k K
			v V
		}
		var out []pair
		for k, v := range seq {
			out = append(out, pair{k, v})
		}
		slices.Reverse(out)
		if len(out) > 0 {
			out = out[:len(out)-1]
		}
		for _, p := range out {
			if !yield(p.k, p.v) {
				return
			}
		}
	}
}

// DoubledSeq answers every element twice — the drain that repeats what a
// no-duplicates claim says it never will.
func DoubledSeq[V any](seq iter.Seq[V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			// Twice, in two statements: the repeat is the defect, and one
			// short-circuiting condition reads as a copied typo.
			if !yield(v) {
				return
			}
			if !yield(v) {
				return
			}
		}
	}
}

// DoubledSeq2 is [DoubledSeq] for the two-value form.
func DoubledSeq2[K, V any](seq iter.Seq2[K, V]) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range seq {
			if !yield(k, v) {
				return
			}
			if !yield(k, v) {
				return
			}
		}
	}
}

// FloodLimit is how many elements the flooding drains yield.
//
// It matches the completion law's own default ceiling, which is the only
// number in play: nothing binds that field, so a drain has not "failed to
// terminate" until it reaches this. One more than the ceiling would be
// tidier arithmetic and a worse defect — a stream that stops at all is a
// stream that terminated, and the law is entitled to say so.
const FloodLimit = 10000

// FloodedSeq is the drain that will not end within any budget a test can
// wait for: the subject's own elements, then [FloodLimit] repeats of the
// last one.
//
// It stops. An endless version is the obvious way to write this and the
// wrong one — every drain the generator emits ranges to exhaustion and
// appends, and the completion law reads its limit only after the drain
// returns, so an unbounded yield takes the machine rather than the test.
// That is not hypothetical: it cost 30 GB and a session.
//
// The yield's verdict is honoured on every send, so a consumer that stops
// early stops this too.
func FloodedSeq[V any](seq iter.Seq[V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		var last V
		for v := range seq {
			last = v
			if !yield(v) {
				return
			}
		}
		for range FloodLimit {
			if !yield(last) {
				return
			}
		}
	}
}

// FloodedSeq2 is [FloodedSeq] for the two-value form.
func FloodedSeq2[K, V any](seq iter.Seq2[K, V]) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		var (
			lastK K
			lastV V
		)
		for k, v := range seq {
			lastK, lastV = k, v
			if !yield(k, v) {
				return
			}
		}
		for range FloodLimit {
			if !yield(lastK, lastV) {
				return
			}
		}
	}
}
