// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"errors"
	"go/token"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestError(t *testing.T) {
	t.Parallel()

	t.Run("Errorf with valid position formats file:line:col", func(t *testing.T) {
		t.Parallel()
		err := generator.Errorf(token.Position{Filename: "store.go", Line: 12, Column: 3}, "method %s missing", "Get")
		testkit.Assert(t, err.Error()).
			Contains("store.go:12:3:", "position prefix").
			Contains("method Get missing", "formatted message")
	})

	t.Run("Errorf with empty position emits message only", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, generator.Errorf(token.Position{}, "boom").Error(), "boom", "no position prefix")
	})

	t.Run("WrapErr preserves cause for errors.Is", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("io: closed")
		wrapped := generator.WrapErr(token.Position{Filename: "x.go", Line: 1}, cause, "while loading %s", "store")
		testkit.True(t, errors.Is(wrapped, cause), "errors.Is must find cause through Unwrap")
		testkit.Assert(t, wrapped.Error()).
			Contains("while loading store", "wrapper message").
			Contains("io: closed", "cause text")
	})

	t.Run("WrapErr without position prefixes message", func(t *testing.T) {
		t.Parallel()
		err := generator.WrapErr(token.Position{}, errors.New("oops"), "context")
		testkit.Equal(t, err.Error(), "context: oops", "no position prefix")
	})

	t.Run("TypeKind String covers every kind", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, generator.KindInterface.String(), "interface", "interface label")
		testkit.Equal(t, generator.KindStruct.String(), "struct", "struct label")
		testkit.Equal(t, generator.KindNamedType.String(), "named type", "named-type label")
		testkit.Equal(t, generator.KindAny.String(), "type", "default label")
	})
}
