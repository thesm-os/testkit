// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestTitle(t *testing.T) {
	t.Parallel()

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.Title(""), "", "empty must return empty")
	})

	t.Run("lowercase word", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.Title("hello"), "Hello", "must capitalize first letter")
	})

	t.Run("initialism id", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.Title("id"), "ID", "must uppercase initialism")
	})

	t.Run("initialism url", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.Title("url"), "URL", "must uppercase initialism")
	})

	t.Run("initialism http", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.Title("http"), "HTTP", "must uppercase initialism")
	})

	t.Run("initialism uuid", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.Title("uuid"), "UUID", "must uppercase initialism")
	})

	t.Run("non-initialism", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.Title("name"), "Name", "non-initialism just capitalizes")
	})

	t.Run("already capitalized", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.Title("Name"), "Name", "already capitalized unchanged")
	})
}

func TestQualifyType(t *testing.T) {
	t.Parallel()

	t.Run("with qualifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.QualifyType("store", "Item"), "store.Item", "must prefix")
	})

	t.Run("empty qualifier", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.QualifyType("", "Item"), "Item", "empty must return bare name")
	})
}

func TestFormatDocComment(t *testing.T) {
	t.Parallel()

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.FormatDocComment(""), "", "empty must return empty")
	})

	t.Run("single line", func(t *testing.T) {
		t.Parallel()
		got := gen.FormatDocComment("Get retrieves an item.")
		testkit.Equal(t, got, "// Get retrieves an item.", "must prefix with //")
	})

	t.Run("multi line", func(t *testing.T) {
		t.Parallel()
		got := gen.FormatDocComment("Get retrieves an item.\nReturns ErrNotFound if missing.")
		testkit.Equal(t, got, "// Get retrieves an item.\n// Returns ErrNotFound if missing.", "must prefix each line")
	})

	t.Run("blank line in middle", func(t *testing.T) {
		t.Parallel()
		got := gen.FormatDocComment("First paragraph.\n\nSecond paragraph.")
		testkit.Equal(t, got, "// First paragraph.\n//\n// Second paragraph.", "blank line gets bare //")
	})
}

func TestParamName(t *testing.T) {
	t.Parallel()

	t.Run("index 0", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.ParamName(0), "p0", "must be p0")
	})

	t.Run("index 3", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.ParamName(3), "p3", "must be p3")
	})
}

func TestCamelCase(t *testing.T) {
	t.Parallel()

	t.Run("underscore separated", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.CamelCase("hello_world"), "HelloWorld", "underscore split")
	})

	t.Run("hyphen separated", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.CamelCase("hello-world"), "HelloWorld", "hyphen split")
	})

	t.Run("already camel", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.CamelCase("helloWorld"), "HelloWorld", "camel boundary")
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.CamelCase(""), "", "empty")
	})
}

func TestLowerCamelCase(t *testing.T) {
	t.Parallel()

	t.Run("underscore separated", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.LowerCamelCase("hello_world"), "helloWorld", "underscore split")
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.LowerCamelCase(""), "", "empty")
	})
}

func TestSnakeCase(t *testing.T) {
	t.Parallel()

	t.Run("camel to snake", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.SnakeCase("HelloWorld"), "hello_world", "camel to snake")
	})

	t.Run("hyphen to snake", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.SnakeCase("hello-world"), "hello_world", "hyphen to snake")
	})
}

func TestSplitWords(t *testing.T) {
	t.Parallel()

	t.Run("underscore", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.SplitWords("hello_world"), []string{"hello", "world"}, "underscore split")
	})

	t.Run("camelCase", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gen.SplitWords("helloWorld"), []string{"hello", "World"}, "camel split")
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, gen.SplitWords(""), 0, "empty")
	})
}
