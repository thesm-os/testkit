// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/sentinel"
)

func TestData_HasContent(t *testing.T) {
	t.Parallel()

	t.Run("empty data has no content", func(t *testing.T) {
		t.Parallel()
		var d sentinel.Data
		testkit.False(t, d.HasContent(), "zero-value Data is empty")
	})

	t.Run("any sentinel makes content non-empty", func(t *testing.T) {
		t.Parallel()
		d := sentinel.Data{Sentinels: []sentinel.ErrorVar{{Name: "ErrFoo"}}}
		testkit.True(t, d.HasContent(), "has sentinel")
	})

	t.Run("any error type makes content non-empty", func(t *testing.T) {
		t.Parallel()
		d := sentinel.Data{ErrorTypes: []sentinel.ErrorType{{Name: "FooError"}}}
		testkit.True(t, d.HasContent(), "has error type")
	})
}
