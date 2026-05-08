// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package defaults

//go:generate testkit builder -o defaultstest/convention.gen.go Request

// Request seeds its builder via the convention-based factory
// `RequestDefaults()` declared in the sibling defaultstest package
// (where the generator emits the builder). When New<Request>() is
// called, the builder dispatches to RequestDefaults() to produce
// the seed value.
type Request struct {
	RunID string
	Token int
	Data  []byte
}
