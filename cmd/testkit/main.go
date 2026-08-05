// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Command testkit generates test doubles, fixtures, conformance suites, and
// benchmarks from Go interfaces and types — along with the tests that prove
// the generated code works.
//
// The binary is a thin shell over eidos: it hardcodes the brand "testkit",
// supplies testkit's generator set, and dispatches eidos's command kernels.
// Everything that walks Go source, orders plugins, and writes files belongs to
// eidos; see docs/adr/0003.
package main

import (
	"os"

	"go.thesmos.sh/testkit/cmd/testkit/cmds"
)

func main() {
	os.Exit(cmds.Execute())
}
