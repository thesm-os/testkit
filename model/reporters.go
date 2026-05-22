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
// action) and notes the divergence. When a Porcupine linearizability
// visualization is attached, it surfaces the path as the primary
// debugging artifact.
func writeSemantic(b *strings.Builder, f *Failure) {
	source := "action"
	if f.LawID != "" {
		source = f.LawID
	}
	fmt.Fprintf(b, "  semantic: SUT vs reference disagreement (%s)\n", source) //nolint:forbidigo // strings.Builder
	for _, p := range f.ArtifactPaths {
		if path, ok := strings.CutPrefix(p, "viz: "); ok {
			fmt.Fprintf(b, "    porcupine: %s\n", path) //nolint:forbidigo // strings.Builder
		}
	}
}

// writeInvariant adds law-violation context: cite the law ID + REQ
// then the state at violation when the runner captured one. The
// header already carries the law's err message; this reporter adds
// the structured snapshot for per-kind tooling that reads sections.
func writeInvariant(b *strings.Builder, f *Failure) {
	switch {
	case f.LawID != "" && f.REQID != "":
		fmt.Fprintf(b, "  invariant: %s [%s]\n", f.LawID, f.REQID) //nolint:forbidigo // strings.Builder
	case f.LawID != "":
		fmt.Fprintf(b, "  invariant: %s\n", f.LawID) //nolint:forbidigo // strings.Builder
	default:
		fmt.Fprintln(b, "  invariant: unspecified law fired") //nolint:forbidigo // strings.Builder
	}
	if f.SUTState != "" {
		fmt.Fprintf(b, "    sut: %s\n", f.SUTState) //nolint:forbidigo // strings.Builder
	}
	if f.RefState != "" {
		fmt.Fprintf(b, "    ref: %s\n", f.RefState) //nolint:forbidigo // strings.Builder
	}
}

// writeLiveness adds goroutine stacks for deadlock / no-progress
// diagnoses. The full runtime.Stack output is captured at format
// time so the snapshot reflects the goroutine state at failure-
// report time. The stack is bounded to liveStackBufBytes to keep
// the report readable. A "suspected blocker:" line cites the first
// goroutine in a known blocking state when one is detected.
func writeLiveness(b *strings.Builder, _ *Failure) {
	//nolint:forbidigo // strings.Builder
	fmt.Fprintln(b, "  liveness: deadlock or no-progress detected; goroutine stacks follow")
	buf := make([]byte, liveStackBufBytes)
	n := runtime.Stack(buf, true /* all goroutines */)
	stack := string(buf[:n])
	if id, state := suspectedBlocker(stack); id != "" {
		fmt.Fprintf(b, "    suspected blocker: goroutine %s [%s]\n", id, state) //nolint:forbidigo // strings.Builder
	}
	for line := range strings.SplitSeq(strings.TrimRight(stack, "\n"), "\n") {
		fmt.Fprintf(b, "    %s\n", line) //nolint:forbidigo // strings.Builder
	}
}

// suspectedBlocker scans a runtime.Stack-formatted dump for the
// first goroutine header in a blocking state. The dump's goroutine
// headers look like `goroutine 47 [chan receive]:`; blocking states
// are the ones that imply waiting on a synchronization primitive.
// Returns goroutine ID and trimmed state, or empty strings if no
// blocked goroutine is found.
func suspectedBlocker(stack string) (id, state string) {
	for line := range strings.SplitSeq(stack, "\n") {
		if !strings.HasPrefix(line, "goroutine ") {
			continue
		}
		open := strings.IndexByte(line, '[')
		closeIdx := strings.LastIndexByte(line, ']')
		if open < 0 || closeIdx <= open {
			continue
		}
		st := line[open+1 : closeIdx]
		// "chan receive, 5 minutes" → match the leading word.
		head := st
		if c := strings.IndexByte(head, ','); c >= 0 {
			head = head[:c]
		}
		if !isBlockingState(head) {
			continue
		}
		idEnd := strings.IndexByte(line[len("goroutine "):], ' ')
		if idEnd < 0 {
			continue
		}
		return line[len("goroutine "):][:idEnd], st
	}
	return "", ""
}

// stateChanReceive is the runtime goroutine state for a channel
// receive — the canonical deadlock signature. Named so test fixtures
// and the matcher reference one literal.
const stateChanReceive = "chan receive"

// isBlockingState reports whether a Go-runtime goroutine state is
// one that implies waiting on a synchronization primitive (and is
// therefore worth flagging as a deadlock candidate). Running,
// runnable, and syscall states are excluded; they're not waits.
func isBlockingState(s string) bool {
	switch s {
	case stateChanReceive, "chan send", "select",
		"semacquire", "sync.Cond.Wait", "sync.WaitGroup.Wait",
		"IO wait", "sleep":
		return true
	}
	return false
}

// liveStackBufBytes caps the goroutine-stack capture so a runaway
// goroutine count doesn't blow the failure report. 64 KiB is enough
// for ~200 goroutines at typical Go frame depth.
const liveStackBufBytes = 64 * 1024
