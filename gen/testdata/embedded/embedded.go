// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package embedded

import "io"

// ReadWriter embeds io.Reader and io.Writer.
type ReadWriter interface {
	io.Reader
	io.Writer
}

// TripleReader embeds ReadWriter and adds Close.
type TripleReader interface {
	ReadWriter
	Close() error
}
