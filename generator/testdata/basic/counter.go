// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

// Counter is a struct fixture for builder-style tests.
type Counter struct {
	N    int
	Name string `testkit:"required"`
}

// Reset clears the counter.
//
//testkit:idempotent
func (c *Counter) Reset() { c.N = 0 }
