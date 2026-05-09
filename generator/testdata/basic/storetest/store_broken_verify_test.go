// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestBrokenFixturesAreCaught proves the suite's generated
// assertions actually catch contract violations — not just compile
// against well-formed impls. For each broken-impl fixture in this
// directory (build-tagged `broken_fixtures` so the deliberately-
// failing tests don't pollute a default `go test ./...` run), this
// test subprocesses `go test -tags=broken_fixtures -run <Name>`,
// asserts the inner test exits non-zero (proving the contract did
// flag the violation), and asserts the captured output contains the
// expected failure substring (proving the contract flagged the
// RIGHT thing, not some unrelated assertion).
//
// Without this verification a "world-class" suite is suspect — every
// contract assertion could be vacuous (always pass) or assert the
// wrong property, and the spec_test.go layer of testdata would
// never tell us. Subprocess-driven negative testing is the standard
// Go pattern for "test the test"; *testing.T isn't substitutable so
// in-process fakeT-style harnesses don't compose with the
// contract-driver API.
func TestBrokenFixturesAreCaught(t *testing.T) {
	t.Parallel()
	cases := []struct {
		// name is the human-readable subtest name reported by this
		// outer test.
		name string

		// run is the regex passed to `go test -run` to select the
		// inner deliberately-failing test.
		run string

		// wantSubstr is a substring the captured stdout/stderr must
		// contain. Each broken impl's contract-violation message is
		// distinct enough that one substring per case is sufficient.
		wantSubstr string
	}{
		{
			name:       "Reader baseline catches wrong return value",
			run:        "^TestBrokenStoreReturnsWrongValue$",
			wantSubstr: "reader must return expected value",
		},
		{
			name:       "Reader baseline catches ctx-ignoring impl",
			run:        "^TestBrokenStoreIgnoresContext$",
			wantSubstr: "reader must surface ctx.Canceled",
		},
		{
			name:       "Writer baseline catches wrong-sentinel rejection",
			run:        "^TestBrokenStoreWrongSentinel$",
			wantSubstr: "write must return expected sentinel",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.CommandContext( //nolint:gosec // test-only subprocess with fixed args from table-driven cases
				t.Context(),
				"go", "test",
				"-tags=broken_fixtures",
				"-run", c.run,
				"-count=1",
				"-v",
				".",
			)
			out, err := cmd.CombinedOutput()
			output := string(out)
			if err == nil {
				t.Fatalf(
					"expected `go test` to exit non-zero (contract violation should fail the test); output:\n%s",
					output,
				)
			}
			if !strings.Contains(output, c.wantSubstr) {
				t.Fatalf(
					"expected output to contain %q\noutput:\n%s",
					c.wantSubstr, output,
				)
			}
		})
	}
}
