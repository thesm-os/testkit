// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package failure

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ArtifactKind identifies the kind of on-disk artifact a [Failure]
// references. Listed kinds match the artifacts every layer is
// expected to produce; unknown values surface via [ArtifactKind.String]
// as `unknown(N)` rather than silently coercing.
type ArtifactKind int

const (
	// ArtifactUnclassified is the zero value.
	ArtifactUnclassified ArtifactKind = iota

	// ArtifactFailfile — rapid failfile (model). Auto-replayed on
	// rerun by rapid's testdata harness.
	ArtifactFailfile

	// ArtifactPorcupineHTML — Porcupine linearizability
	// visualization (model concurrent + sim cross-interface).
	ArtifactPorcupineHTML

	// ArtifactClassifiedJSON — the full [Failure] envelope itself,
	// serialized for CI ingestion. Self-referential: a Failure's
	// Artifacts list typically includes one entry pointing to its
	// own JSON dump.
	ArtifactClassifiedJSON

	// ArtifactTimelineHTML — the unified visualize timeline.
	ArtifactTimelineHTML

	// ArtifactTraceJSON — the captured [*trace.Trace] serialized
	// to JSON.
	ArtifactTraceJSON

	// ArtifactSnapshotJSON — per-component or per-impl state
	// captured at failure tick.
	ArtifactSnapshotJSON

	// ArtifactDivergenceReport — diff-rollout's first-divergence
	// report with per-impl outputs and the equivalence chain that
	// rejected them.
	ArtifactDivergenceReport

	// ArtifactTLATrace — TLC counter-example trace, either consumed
	// from TLC's output or emitted by the sim TLA+ bridge.
	ArtifactTLATrace

	// ArtifactReplayCapture — replay's output trace recording the
	// SUT's responses to a replayed input trace; itself replayable.
	ArtifactReplayCapture

	// ArtifactCertificationRecord — diff-rollout's NDJSON-format
	// certification record append for a candidate impl.
	ArtifactCertificationRecord
)

// String returns the artifact-kind name in lowercase-hyphen form.
func (k ArtifactKind) String() string {
	switch k {
	case ArtifactUnclassified:
		return "unclassified" //nolint:goconst // duplication with Kind.String is intentional; both enums independently expose an unclassified zero

	case ArtifactFailfile:
		return "failfile"
	case ArtifactPorcupineHTML:
		return "porcupine-html"
	case ArtifactClassifiedJSON:
		return "classified-json"
	case ArtifactTimelineHTML:
		return "timeline-html"
	case ArtifactTraceJSON:
		return "trace-json"
	case ArtifactSnapshotJSON:
		return "snapshot-json"
	case ArtifactDivergenceReport:
		return "divergence-report"
	case ArtifactTLATrace:
		return "tla-trace"
	case ArtifactReplayCapture:
		return "replay-capture"
	case ArtifactCertificationRecord:
		return "certification-record"
	default:
		return fmt.Sprintf("unknown(%d)", int(k))
	}
}

// ParseArtifactKind decodes a name produced by [ArtifactKind.String].
// Unknown names error rather than coerce.
func ParseArtifactKind(s string) (ArtifactKind, error) {
	switch s {
	case "unclassified":
		return ArtifactUnclassified, nil
	case "failfile":
		return ArtifactFailfile, nil
	case "porcupine-html":
		return ArtifactPorcupineHTML, nil
	case "classified-json":
		return ArtifactClassifiedJSON, nil
	case "timeline-html":
		return ArtifactTimelineHTML, nil
	case "trace-json":
		return ArtifactTraceJSON, nil
	case "snapshot-json":
		return ArtifactSnapshotJSON, nil
	case "divergence-report":
		return ArtifactDivergenceReport, nil
	case "tla-trace":
		return ArtifactTLATrace, nil
	case "replay-capture":
		return ArtifactReplayCapture, nil
	case "certification-record":
		return ArtifactCertificationRecord, nil
	default:
		return ArtifactUnclassified, fmt.Errorf("failure: unknown ArtifactKind %q", s)
	}
}

// Artifact references one on-disk file that a [Failure] envelope
// points at. CI tooling iterates [Failure.Artifacts] and reads what
// it cares about — a PR bot opens the timeline HTML; a REQ coverage
// matrix reads the classified-failure JSON; a regression test
// reader replays the failfile.
type Artifact struct {
	// Kind classifies the artifact for routing.
	Kind ArtifactKind `json:"kind"`

	// Path is the file path, typically relative to the test
	// package's testdata directory. Absolute paths are valid but
	// not portable across CI runners.
	Path string `json:"path"`

	// Format is the artifact's encoding ("json", "ndjson", "html",
	// "tla", "porcupine", "txt"). Stable across versions; unknown
	// formats are passed through unchanged.
	Format string `json:"format"`

	// Metadata carries artifact-specific structured data: a
	// classified-JSON artifact might carry the seed it was
	// produced under; a Porcupine HTML might carry the
	// linearizability check result.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Open returns a reader for the artifact's file. Caller closes the
// returned [io.ReadCloser]. Errors when [Artifact.Path] is empty or
// the file is missing.
func (a Artifact) Open() (io.ReadCloser, error) {
	if a.Path == "" {
		return nil, fmt.Errorf("failure.Artifact.Open: empty path (kind=%s)", a.Kind)
	}
	f, err := os.Open(a.Path) //nolint:gosec // caller-supplied artifact path; CI tooling reads it
	if err != nil {
		return nil, fmt.Errorf("failure.Artifact.Open: %w", err)
	}
	return f, nil
}

// JSON reads the artifact's contents and returns them as a byte
// slice. Convenience for artifacts whose Format is "json" or
// "ndjson"; for other formats, callers use [Artifact.Open] directly.
// Does not validate the content shape — returns whatever the file
// contains.
//
// The io.ReadAll error wrap below is unreachable from blackbox
// tests using regular files (os.File on a finite path doesn't
// surface read errors short of a disk failure mid-read). The wrap
// is retained for diagnostic clarity if such a failure does occur
// in production.
func (a Artifact) JSON() ([]byte, error) {
	r, err := a.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r) //nolint:wrapcheck // unreachable from blackbox test surface; mid-read disk errors only
}

// MarshalJSON encodes [ArtifactKind] as its string form. Pairs with
// [ArtifactKind] field-tagged JSON on [Artifact] — the human-readable
// name round-trips through CI artifacts rather than the integer
// constant.
//
// json.Marshal of a string cannot fail with the inputs this method
// produces ([ArtifactKind.String] returns a fixed set of ASCII
// names plus the "unknown(N)" fallback); the error return is
// retained to satisfy [json.Marshaler] but is always nil.
func (k ArtifactKind) MarshalJSON() ([]byte, error) {
	// json.Marshal of a string with no special characters never
	// fails; the unreachable error is acceptable here.
	return json.Marshal(k.String()) //nolint:wrapcheck
}

// UnmarshalJSON decodes the string form. Errors on unknown names.
func (k *ArtifactKind) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("failure.ArtifactKind.UnmarshalJSON: %w", err)
	}
	parsed, err := ParseArtifactKind(s)
	if err != nil {
		return err
	}
	*k = parsed
	return nil
}
