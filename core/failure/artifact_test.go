// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package failure_test

import (
	"encoding/json"
	"io"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/failure"
)

func TestArtifactKindString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		kind failure.ArtifactKind
		name string
	}{
		{failure.ArtifactUnclassified, "unclassified"},
		{failure.ArtifactFailfile, "failfile"},
		{failure.ArtifactPorcupineHTML, "porcupine-html"},
		{failure.ArtifactClassifiedJSON, "classified-json"},
		{failure.ArtifactTimelineHTML, "timeline-html"},
		{failure.ArtifactTraceJSON, "trace-json"},
		{failure.ArtifactSnapshotJSON, "snapshot-json"},
		{failure.ArtifactDivergenceReport, "divergence-report"},
		{failure.ArtifactTLATrace, "tla-trace"},
		{failure.ArtifactReplayCapture, "replay-capture"},
		{failure.ArtifactCertificationRecord, "certification-record"},
	}

	t.Run("every kind has a stable name", func(t *testing.T) {
		t.Parallel()
		for _, c := range cases {
			testkit.Equal(t, c.kind.String(), c.name, "kind name")
		}
	})

	t.Run("unknown kinds surface as unknown(N)", func(t *testing.T) {
		t.Parallel()
		k := failure.ArtifactKind(999)
		testkit.Equal(t, k.String(), "unknown(999)", "unknown rendering")
	})
}

func TestParseArtifactKind(t *testing.T) {
	t.Parallel()

	t.Run("round-trips every known name", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{
			"unclassified", "failfile", "porcupine-html",
			"classified-json", "timeline-html", "trace-json",
			"snapshot-json", "divergence-report", "tla-trace",
			"replay-capture", "certification-record",
		} {
			k, err := failure.ParseArtifactKind(name)
			testkit.NoError(t, err, "parse "+name)
			testkit.Equal(t, k.String(), name, "round-trip "+name)
		}
	})

	t.Run("rejects unknown names", func(t *testing.T) {
		t.Parallel()
		_, err := failure.ParseArtifactKind("not-a-kind")
		testkit.True(t, err != nil, "must error on unknown name")
	})
}

func TestArtifactKindJSONRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("marshals as the string form", func(t *testing.T) {
		t.Parallel()
		got, err := json.Marshal(failure.ArtifactPorcupineHTML)
		testkit.NoError(t, err, "marshal")
		testkit.Equal(t, string(got), `"porcupine-html"`, "string form")
	})

	t.Run("unmarshals from the string form", func(t *testing.T) {
		t.Parallel()
		var k failure.ArtifactKind
		err := json.Unmarshal([]byte(`"timeline-html"`), &k)
		testkit.NoError(t, err, "unmarshal")
		testkit.Equal(t, k, failure.ArtifactTimelineHTML, "decoded kind")
	})

	t.Run("rejects unknown names on unmarshal", func(t *testing.T) {
		t.Parallel()
		var k failure.ArtifactKind
		err := json.Unmarshal([]byte(`"not-a-thing"`), &k)
		testkit.True(t, err != nil, "must error")
	})

	t.Run("rejects non-string JSON", func(t *testing.T) {
		t.Parallel()
		var k failure.ArtifactKind
		err := json.Unmarshal([]byte(`42`), &k)
		testkit.True(t, err != nil, "must error on non-string")
	})

	t.Run("rejects non-existent paths via JSON helper", func(t *testing.T) {
		t.Parallel()
		// Causes Open to fail; JSON should propagate the error.
		a := failure.Artifact{Path: "/this/path/does/not/exist"}
		_, err := a.JSON()
		testkit.True(t, err != nil, "must error")
	})
}

func TestArtifactOpen(t *testing.T) {
	t.Parallel()

	t.Run("opens an existing file", func(t *testing.T) {
		t.Parallel()
		path := testkit.TempFile(t, "a.json", []byte(`{"x":1}`))

		a := failure.Artifact{Kind: failure.ArtifactClassifiedJSON, Path: path, Format: "json"}
		r, err := a.Open()
		testkit.NoError(t, err, "open")
		defer r.Close()
		got, err := io.ReadAll(r)
		testkit.NoError(t, err, "read")
		testkit.Equal(t, string(got), `{"x":1}`, "content")
	})

	t.Run("errors on empty path", func(t *testing.T) {
		t.Parallel()
		a := failure.Artifact{Kind: failure.ArtifactFailfile, Path: ""}
		_, err := a.Open()
		testkit.True(t, err != nil, "must error")
		testkit.Assert(t, err.Error()).Contains("empty path", "diagnostic")
	})

	t.Run("errors on missing file", func(t *testing.T) {
		t.Parallel()
		a := failure.Artifact{Path: "/nonexistent/path/that/does/not/exist"}
		_, err := a.Open()
		testkit.True(t, err != nil, "must error")
	})
}

func TestArtifactJSON(t *testing.T) {
	t.Parallel()

	t.Run("returns file contents", func(t *testing.T) {
		t.Parallel()
		path := testkit.TempFile(t, "f.json", []byte(`{"k":"v"}`))

		a := failure.Artifact{Path: path, Format: "json"}
		got, err := a.JSON()
		testkit.NoError(t, err, "json")
		testkit.Equal(t, string(got), `{"k":"v"}`, "content")
	})

	t.Run("propagates open errors", func(t *testing.T) {
		t.Parallel()
		a := failure.Artifact{Path: ""}
		_, err := a.JSON()
		testkit.True(t, err != nil, "must error")
	})
}
