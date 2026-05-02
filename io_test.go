// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

func TestMustMarshal(t *testing.T) {
	t.Parallel()

	t.Run("marshals valid value", func(t *testing.T) {
		t.Parallel()
		data := testkit.MustMarshal(t, map[string]int{"a": 1})
		testkit.Equal(t, string(data), `{"a":1}`, "must produce valid JSON")
	})

	t.Run("fatals on unmarshalable value", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		testkit.MustMarshal(f, make(chan int))
		if !f.Failed() {
			t.Fatal("should fail on unmarshalable value")
		}
	})
}

func TestMustUnmarshal(t *testing.T) {
	t.Parallel()

	t.Run("unmarshals valid JSON", func(t *testing.T) {
		t.Parallel()
		var m map[string]int
		testkit.MustUnmarshal(t, []byte(`{"a":1}`), &m)
		testkit.Equal(t, m, map[string]int{"a": 1}, "must unmarshal correctly")
	})

	t.Run("fatals on invalid JSON", func(t *testing.T) {
		t.Parallel()
		f := testkit.NewFailableTB()
		var m map[string]int
		testkit.MustUnmarshal(f, []byte(`not json`), &m)
		if !f.Failed() {
			t.Fatal("should fail on invalid JSON")
		}
	})
}

func TestFailingReader(t *testing.T) {
	t.Parallel()

	t.Run("fails after BeforeFail bytes", func(t *testing.T) {
		t.Parallel()
		r := &testkit.FailingReader{
			Source:     strings.NewReader("hello world"),
			BeforeFail: 5,
			Err:        io.ErrUnexpectedEOF,
		}
		data, err := io.ReadAll(r)
		testkit.ErrorIs(t, err, io.ErrUnexpectedEOF, "must return configured error")
		testkit.Equal(t, string(data), "hello", "must deliver bytes before failure point")
	})

	t.Run("zero BeforeFail fails immediately", func(t *testing.T) {
		t.Parallel()
		r := &testkit.FailingReader{
			Source:     strings.NewReader("data"),
			BeforeFail: 0,
			Err:        io.ErrClosedPipe,
		}
		_, err := r.Read(make([]byte, 10))
		testkit.ErrorIs(t, err, io.ErrClosedPipe, "must fail immediately")
	})

	t.Run("propagates source error", func(t *testing.T) {
		t.Parallel()
		// Source that errors immediately.
		source := &testkit.FailingReader{
			Source:     strings.NewReader(""),
			BeforeFail: 0,
			Err:        io.ErrUnexpectedEOF,
		}
		r := &testkit.FailingReader{
			Source:     source,
			BeforeFail: 100,
			Err:        io.ErrClosedPipe,
		}
		_, err := r.Read(make([]byte, 10))
		if err == nil {
			t.Fatal("should propagate source error")
		}
	})
}

func TestFailingWriter(t *testing.T) {
	t.Parallel()

	t.Run("fails after BeforeFail bytes", func(t *testing.T) {
		t.Parallel()
		w := &testkit.FailingWriter{BeforeFail: 5, Err: io.ErrShortWrite}
		n, err := w.Write([]byte("hello world"))
		testkit.Equal(t, n, 5, "must accept bytes up to failure point")
		testkit.ErrorIs(t, err, io.ErrShortWrite, "must return configured error")
	})

	t.Run("zero BeforeFail fails immediately", func(t *testing.T) {
		t.Parallel()
		w := &testkit.FailingWriter{BeforeFail: 0, Err: io.ErrShortWrite}
		n, err := w.Write([]byte("data"))
		testkit.Equal(t, n, 0, "must write zero bytes")
		testkit.ErrorIs(t, err, io.ErrShortWrite, "must fail immediately")
	})

	t.Run("small writes succeed until threshold", func(t *testing.T) {
		t.Parallel()
		w := &testkit.FailingWriter{BeforeFail: 10, Err: io.ErrShortWrite}
		n1, err1 := w.Write([]byte("hello"))
		testkit.Equal(t, n1, 5, "first write succeeds")
		testkit.NoError(t, err1, "first write must not error")
		n2, err2 := w.Write([]byte("world!"))
		testkit.Equal(t, n2, 5, "second write partial")
		testkit.ErrorIs(t, err2, io.ErrShortWrite, "second write must fail at threshold")
	})
}

func TestQuiet(t *testing.T) { //nolint:paralleltest // Quiet mutates process-global slog state
	t.Run("suppresses and restores slog", func(t *testing.T) {
		before := slog.Default()
		restore := testkit.Quiet(t)

		// While quiet, the handler should be DiscardHandler.
		if slog.Default().Handler().Enabled(t.Context(), slog.LevelError) {
			t.Fatal("slog should be disabled while quiet")
		}

		restore()

		if slog.Default() != before {
			t.Fatal("slog should be restored after Quiet cleanup")
		}
	})
}
