// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storage

import "errors"

// ErrMissing is the not-found sentinel for the storage layer.
var ErrMissing = errors.New("storage: missing")

// ErrCorrupt indicates a checksum or format failure on read.
var ErrCorrupt = errors.New("storage: corrupt")
