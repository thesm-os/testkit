// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// testkit generates test infrastructure for Go projects.
package main

import (
	"fmt"
	"os"

	"go.thesmos.sh/testkit/cmd/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version.String())
		return
	}
	fmt.Fprintln(os.Stderr, "testkit: not yet implemented")
	os.Exit(1)
}
