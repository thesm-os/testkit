// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaults

//go:generate testkit builder -o defaultstest/builders.gen.go Request

// Request has a convention-based defaults function in the test package.
type Request struct {
	RunID string
	Token int
	Data  []byte
}
