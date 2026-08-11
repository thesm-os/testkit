// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package paginatedreadertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	paginatedreader "go.thesmos.sh/testkit/conformance/corpus/iface/composite/paginated-reader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/composite/paginated-reader/paginatedreadertest"
)

// corpus is what every reader in this package pages over.
//
// Five values against a page size of two, so the walk takes three pages and the
// last one is short — which is where an off-by-one either drops the tail or
// serves it twice.
var corpus = []paginatedreader.Value{
	{Key: "a", Body: "one"},
	{Key: "b", Body: "two"},
	{Key: "c", Body: "three"},
	{Key: "d", Body: "four"},
	{Key: "e", Body: "five"},
}

// paginated-reader stacks the reader detector with the pagination contract, and
// the fixture exists because the contract changes what the detector generates
// rather than adding to it: a bare reader asserts one call and one result, and
// a paginated one has to be driven as a loop.
//
// `pagination` is the model tier's under ADR-0018 — `AUTO-PAGINATOR-NO-DUPLICATES`
// and `AUTO-PAGINATOR-RESUMABLE` state it — so the generated family is the
// signature-derived one. The loop the contract implies is stated here, through
// the extension point rather than as a package test: every claim below needs
// only the interface, so every one runs against each subject a consumer
// declares and again through the double.
//
// The derived cursors are integers the reader never issued, which is what makes
// its "an error carries the zero value" check able to fail at all.
func TestPaginatedReaderContract(t *testing.T) {
	t.Parallel()

	paginatedreadertest.AssertPaginatedReaderContract(t,
		paginatedreadertest.PaginatedReaderSubject("in-memory", func() paginatedreader.PaginatedReader {
			return paginatedreadertest.NewInMemory(corpus...)
		}),
		paginatedreadertest.PaginatedReaderSubject("in-memory, empty", func() paginatedreader.PaginatedReader {
			return paginatedreadertest.NewInMemory()
		}),
		paginatedreadertest.PaginatedReaderOnPage("refuses a cursor it did not issue", func(
			tb testing.TB, subject paginatedreader.PaginatedReader, cursor int,
		) {
			tb.Helper()
			// An offset accepts any integer, so this is what separates an
			// opaque token from one — and a reader that accepts invented
			// cursors resumes somewhere nobody asked for.
			items, next, err := subject.Page(tb.Context(), cursor)
			testkit.ErrorIs(tb, err, paginatedreadertest.ErrUnknownCursor,
				"a cursor from nowhere is refused")
			testkit.Len(tb, items, 0, "with no page beside the error")
			testkit.Equal(tb, next, paginatedreadertest.End, "and no cursor to continue from")
		}),
		paginatedreadertest.PaginatedReaderOnPage("walks every value once and terminates", func(
			tb testing.TB, subject paginatedreader.PaginatedReader, _ int,
		) {
			tb.Helper()
			seen := walk(tb, subject, paginatedreadertest.Start)
			testkit.Equal(tb, len(dedupe(seen)), len(seen),
				"no value is served on two pages")
		}),
		paginatedreadertest.PaginatedReaderOnPage("resumes where a cursor was issued", func(
			tb testing.TB, subject paginatedreader.PaginatedReader, _ int,
		) {
			tb.Helper()
			// `AUTO-PAGINATOR-RESUMABLE`'s own shape: the walk from a page
			// start is the suffix of the full walk. A reader resuming from
			// anywhere else passes the no-duplicates check and loses values.
			full := walk(tb, subject, paginatedreadertest.Start)

			_, second, err := subject.Page(tb.Context(), paginatedreadertest.Start)
			testkit.NoError(tb, err, "the first page is readable")
			if second == paginatedreadertest.End {
				// Nothing to resume from, which is the empty subject. The claim
				// is vacuous rather than untested — there is no second page.
				return
			}

			resumed := walk(tb, subject, second)
			testkit.Equal(tb, resumed, full[len(full)-len(resumed):],
				"resuming yields exactly the tail of the full walk")
		}),
	)
}

// walk reads from a cursor to the end, failing on anything the reader refuses.
//
// A bounded loop rather than an open one: a reader handing back a cursor that
// never terminates would otherwise hang until the test binary's own timeout,
// which reports as a panic in whatever test happened to be running.
func walk(tb testing.TB, subject paginatedreader.PaginatedReader, from int) []paginatedreader.Value {
	tb.Helper()

	const maxPages = 16

	var seen []paginatedreader.Value
	cursor := from
	for range maxPages {
		items, next, err := subject.Page(tb.Context(), cursor)
		testkit.NoError(tb, err, "an issued cursor is readable")
		seen = append(seen, items...)
		if next == paginatedreadertest.End {
			return seen
		}
		cursor = next
	}
	tb.Fatalf("the walk did not terminate within %d pages", maxPages)
	return nil
}

// dedupe returns the distinct values, in first-seen order.
func dedupe(in []paginatedreader.Value) []paginatedreader.Value {
	seen := map[paginatedreader.Value]bool{}
	out := make([]paginatedreader.Value, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// Declining the double is separate from dropping a check.
func TestPaginatedReaderContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	paginatedreadertest.AssertPaginatedReaderContract(t,
		paginatedreadertest.PaginatedReaderSubject("in-memory", func() paginatedreader.PaginatedReader {
			return paginatedreadertest.NewInMemory(corpus...)
		}),
		paginatedreadertest.PaginatedReaderWithout("Page/smoke"),
		paginatedreadertest.PaginatedReaderWithoutDouble(),
	)
}
