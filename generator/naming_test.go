// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestNaming(t *testing.T) {
	t.Parallel()

	t.Run("Title preserves initialisms", func(t *testing.T) {
		t.Parallel()
		cases := map[string]string{
			"":     "",
			"id":   "ID",
			"url":  "URL",
			"http": "HTTP",
			"user": "User",
			"User": "User",
			"uRL":  "URL",
			"a":    "A",
		}
		for in, want := range cases {
			testkit.Equal(t, generator.Title(in), want, "Title("+in+")")
		}
	})

	t.Run("CamelCase promotes initialisms across separators", func(t *testing.T) {
		t.Parallel()
		cases := map[string]string{
			"":              "",
			"user":          "User",
			"user_id":       "UserID",
			"user-id":       "UserID",
			"user_name":     "UserName",
			"http_method":   "HTTPMethod",
			"json-response": "JSONResponse",
			"id":            "ID",
			"a_b_c":         "ABC",
			"http":          "HTTP",
			"order_uuid":    "OrderUUID",
		}
		for in, want := range cases {
			testkit.Equal(t, generator.CamelCase(in), want, "CamelCase("+in+")")
		}
	})

	t.Run("LowerCamelCase lowers first rune of UpperCamelCase", func(t *testing.T) {
		t.Parallel()
		cases := map[string]string{
			"":            "",
			"user":        "user",
			"user_id":     "userID",
			"http_method": "hTTPMethod",
			"order_id":    "orderID",
		}
		for in, want := range cases {
			testkit.Equal(t, generator.LowerCamelCase(in), want, "LowerCamelCase("+in+")")
		}
	})

	t.Run("SnakeCase splits CamelCase and kebab-case", func(t *testing.T) {
		t.Parallel()
		cases := map[string]string{
			"":            "",
			"user":        "user",
			"User":        "user",
			"UserID":      "user_i_d",
			"userID":      "user_i_d",
			"httpMethod":  "http_method",
			"http-method": "http_method",
			"foo bar":     "foo_bar",
		}
		for in, want := range cases {
			testkit.Equal(t, generator.SnakeCase(in), want, "SnakeCase("+in+")")
		}
	})
}
