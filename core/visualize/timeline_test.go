// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package visualize_test

import (
	"bytes"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/trace"
	"go.thesmos.sh/testkit/core/visualize"
)

func TestStyleTheme(t *testing.T) {
	t.Parallel()

	t.Run("empty theme defaults to light", func(t *testing.T) {
		t.Parallel()
		tl := visualize.Timeline{
			Subject:   "basic.Store",
			Generator: "model",
			Trace:     trace.New(),
		}
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		testkit.Assert(t, buf.String()).
			NotContains("class=\"dark\"", "default theme is light").
			Contains("<body class=\"\">", "body has empty class for light theme")
	})

	t.Run("explicit dark theme renders dark body class", func(t *testing.T) {
		t.Parallel()
		tl := visualize.Timeline{
			Subject:   "basic.Store",
			Generator: "model",
			Trace:     trace.New(),
			Style:     visualize.Style{Theme: "dark"},
		}
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		testkit.Assert(t, buf.String()).Contains(`<body class="dark">`, "dark theme")
	})
}

func TestTimelineTitle(t *testing.T) {
	t.Parallel()

	t.Run("title override wins", func(t *testing.T) {
		t.Parallel()
		tl := visualize.Timeline{
			Subject:   "basic.Store",
			Generator: "model",
			Trace:     trace.New(),
			Style:     visualize.Style{Title: "Custom Title"},
		}
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		testkit.Assert(t, buf.String()).Contains("<title>Custom Title</title>", "override title")
	})

	t.Run("default title joins subject and generator", func(t *testing.T) {
		t.Parallel()
		tl := visualize.Timeline{
			Subject:   "basic.Store",
			Generator: "model",
			Trace:     trace.New(),
		}
		var buf bytes.Buffer
		testkit.NoError(t, visualize.Emit(&buf, tl), "emit")
		testkit.Assert(t, buf.String()).Contains(
			"<title>basic.Store — model timeline</title>",
			"default-derived title",
		)
	})
}

func TestEmitWriterError(t *testing.T) {
	t.Parallel()

	tr := trace.New()
	tr.Record(trace.Event{StartNs: 0, EndNs: 1, Method: "Get"})
	tl := visualize.Timeline{
		Subject:   "basic.Store",
		Generator: "model",
		Trace:     tr,
	}
	err := visualize.Emit(failingWriter{}, tl)
	testkit.True(t, err != nil, "writer error must propagate")
	testkit.Assert(t, err.Error()).
		Contains("visualize.Emit:", "wraps with package prefix").
		Contains("disk full", "preserves underlying message")
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }
