// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"runtime"
	"strings"
)

// Per-kind reporters add kind-specific context to the generic
// header + trace produced by [formatFailure]. Each reporter is a
// small specialization that appends to the supplied builder; the
// builder already carries the header line and (where appropriate)
// the trace dump.
//
// The reporters are observational — they read [Failure] fields and
// format them. They never mutate state.

// writeStructural adds structural-failure context: a one-line
// callout that the failure was a structural violation (chain hash
// mismatch, ordering violation, linearizability). The detailed
// chain walk lives in the wrapping Err's message; this reporter
// frames it.
func writeStructural(b *strings.Builder, f *Failure) {
	if f.LawID != "" {
		fmt.Fprintf(b, "  structural: %s\n", f.LawID) //nolint:forbidigo // strings.Builder
		return
	}
	fmt.Fprintln(b, "  structural: chain or ordering violation") //nolint:forbidigo // strings.Builder
}

// writeSemantic adds SUT-vs-reference divergence framing. The
// actual cmp.Diff lives in f.Err; this reporter cites the law (or
// action) and notes the divergence.
func writeSemantic(b *strings.Builder, f *Failure) {
	source := "action"
	if f.LawID != "" {
		source = f.LawID
	}
	fmt.Fprintf(b, "  semantic: SUT vs reference disagreement (%s)\n", source) //nolint:forbidigo // strings.Builder
}

// writeInvariant adds law-violation context: cite the law ID + REQ
// + diagnostic. Most of this is already in the header; this
// reporter adds the "law fired" framing for kind-specific tooling
// that reads per-kind sections.
func writeInvariant(b *strings.Builder, f *Failure) {
	switch {
	case f.LawID != "" && f.REQID != "":
		fmt.Fprintf(b, "  invariant: %s [%s]\n", f.LawID, f.REQID) //nolint:forbidigo // strings.Builder
	case f.LawID != "":
		fmt.Fprintf(b, "  invariant: %s\n", f.LawID) //nolint:forbidigo // strings.Builder
	default:
		fmt.Fprintln(b, "  invariant: unspecified law fired") //nolint:forbidigo // strings.Builder
	}
}

// writeLiveness adds goroutine stacks for deadlock / no-progress
// diagnoses. The full runtime.Stack output is captured at format
// time so the snapshot reflects the goroutine state at failure-
// report time. The stack is bounded to liveStackBufBytes to keep
// the report readable.
func writeLiveness(b *strings.Builder, _ *Failure) {
	//nolint:forbidigo // strings.Builder
	fmt.Fprintln(b, "  liveness: deadlock or no-progress detected; goroutine stacks follow")
	buf := make([]byte, liveStackBufBytes)
	n := runtime.Stack(buf, true /* all goroutines */)
	stack := string(buf[:n])
	for line := range strings.SplitSeq(strings.TrimRight(stack, "\n"), "\n") {
		fmt.Fprintf(b, "    %s\n", line) //nolint:forbidigo // strings.Builder
	}
}

// liveStackBufBytes caps the goroutine-stack capture so a runaway
// goroutine count doesn't blow the failure report. 64 KiB is enough
// for ~200 goroutines at typical Go frame depth.
const liveStackBufBytes = 64 * 1024
