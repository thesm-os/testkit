// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package auditchain_hash exercises the //testkit:hash directive
// for consumer-supplied hash functions.
package auditchain_hash

import (
	"context"
	"crypto/sha512"
	"iter"

	"go.thesmos.sh/testkit/model/refchain"
)

//go:generate testkit model -o hashlogtest/hashlog_model.gen.go HashLog

// Entry is a log entry.
type Entry struct {
	ID   string
	Data string
}

// SHA512Hash returns a hash function using SHA-512 instead of the
// default SHA-256. Demonstrates consumer-supplied hash override.
func SHA512Hash() refchain.HashFunc[Entry] {
	return func(prev refchain.Hash, e Entry) refchain.Hash {
		var buf [32]byte
		copy(buf[:], prev[:])
		h := sha512.Sum512(append(buf[:], []byte(e.ID+e.Data)...))
		var out refchain.Hash
		copy(out[:], h[:32])
		return out
	}
}

// HashLog is an append-only chain with custom hash.
//
//testkit:hash auditchain_hash.SHA512Hash
type HashLog interface {
	//testkit:appends
	Append(ctx context.Context, entry Entry) error

	//testkit:verifies
	Verify(ctx context.Context) error

	//testkit:replays
	Replay(ctx context.Context) iter.Seq2[Entry, error]
}
