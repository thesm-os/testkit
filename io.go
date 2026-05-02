// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
)

// MustMarshal marshals v to JSON and calls tb.Fatalf if marshaling fails.
// Use this for fixture construction — not for testing marshal behavior itself.
//
//	body := testkit.MustMarshal(t, request)
func MustMarshal(tb testing.TB, v any) []byte {
	tb.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		tb.Fatalf("MustMarshal: %v", err)
	}
	return data
}

// MustUnmarshal unmarshals data from JSON into v and calls tb.Fatalf if
// unmarshaling fails. Use this for fixture construction — not for testing
// unmarshal behavior itself.
//
//	var resp Response
//	testkit.MustUnmarshal(t, body, &resp)
func MustUnmarshal(tb testing.TB, data []byte, v any) {
	tb.Helper()
	err := json.Unmarshal(data, v)
	if err != nil {
		tb.Fatalf("MustUnmarshal: %v", err)
	}
}

// FailingReader is an [io.Reader] that succeeds for the first BeforeFail
// bytes, then returns Err on the next read. Use it to exercise mid-stream
// error handling.
//
//	r := &testkit.FailingReader{Source: strings.NewReader(data), BeforeFail: 10, Err: io.ErrUnexpectedEOF}
//	_, err := io.ReadAll(r)
//	testkit.ErrorIs(t, err, io.ErrUnexpectedEOF, "must surface read error")
type FailingReader struct {
	Source     io.Reader
	BeforeFail int
	Err        error
	readSoFar  int
}

// Read implements [io.Reader]. It delegates to Source until BeforeFail bytes
// have been read, then returns Err.
func (r *FailingReader) Read(p []byte) (int, error) {
	remaining := r.BeforeFail - r.readSoFar
	if remaining <= 0 {
		return 0, r.Err
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	n, err := r.Source.Read(p)
	r.readSoFar += n
	if err != nil {
		return n, fmt.Errorf("FailingReader: %w", err)
	}
	if r.readSoFar >= r.BeforeFail {
		return n, r.Err
	}
	return n, nil
}

// FailingWriter is an [io.Writer] that succeeds for the first BeforeFail
// bytes, then returns Err on the next write. Use it to exercise mid-stream
// write error handling.
//
//	w := &testkit.FailingWriter{BeforeFail: 5, Err: io.ErrShortWrite}
//	_, err := w.Write([]byte("hello world"))
//	testkit.ErrorIs(t, err, io.ErrShortWrite, "must surface write error")
type FailingWriter struct {
	BeforeFail   int
	Err          error
	writtenSoFar int
}

// Write implements [io.Writer]. It accepts bytes until BeforeFail bytes
// have been written, then returns Err.
func (w *FailingWriter) Write(p []byte) (int, error) {
	remaining := w.BeforeFail - w.writtenSoFar
	if remaining <= 0 {
		return 0, w.Err
	}
	if len(p) > remaining {
		w.writtenSoFar += remaining
		return remaining, w.Err
	}
	w.writtenSoFar += len(p)
	return len(p), nil
}

// Quiet suppresses [log/slog] output for the duration of the test by
// replacing the default handler with a discard handler. Returns a cleanup
// function that restores the previous handler. Call it with defer.
//
// NOT safe for use with [testing.T.Parallel] — slog.SetDefault is
// process-global.
//
//	restore := testkit.Quiet(t)
//	defer restore()
//	// code that logs will not produce output
func Quiet(tb testing.TB) func() {
	tb.Helper()
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.DiscardHandler))
	return func() {
		slog.SetDefault(prev)
	}
}
