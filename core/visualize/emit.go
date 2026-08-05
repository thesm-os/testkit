// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package visualize

import (
	"fmt"
	"hash/fnv"
	"html/template"
	"io"
	"sort"
	"strings"

	"go.thesmos.sh/testkit/core/trace"
)

// Emit writes a self-contained HTML timeline for the given Timeline
// to w. The output is one HTML document with embedded CSS, no
// external dependencies, no JavaScript. Opens directly in any
// browser. Output is byte-identical for byte-identical inputs:
// lane order, event order, marker order, and color assignment are
// all deterministic from the trace contents.
//
// The HTML page contains:
//   - Header with Subject, Generator, Seed, event count.
//   - SVG timeline with one lane per component (or per goroutine
//     when no component is populated), each event a colored block
//     positioned by [trace.Event.StartNs].
//   - Overlay markers stacked on the event blocks: fault bands,
//     divergence X marks, causality arrows from predecessor to
//     successor, REQ tag labels, replay-origin indicators,
//     snapshot flags at the failure tick.
//   - Event-detail table listing every event with its full payload.
func Emit(w io.Writer, t Timeline) error {
	data := buildTemplateData(t)
	tmpl := template.Must(template.New("timeline").Funcs(templateFuncs).Parse(htmlTemplate))
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("visualize.Emit: %w", err)
	}
	return nil
}

// templateData carries everything the HTML template needs. Exposed
// fields are read by the template; unexported helpers compute them
// during buildTemplateData.
type templateData struct {
	Title      string
	Theme      string
	Subject    string
	Generator  string
	SeedHex    string
	EventCount int
	Lanes      []laneView
	Events     []eventView
	Markers    []markerView
	SVGWidth   int
	SVGHeight  int
	StartNs    int64
}

type laneView struct {
	Name  string
	Index int
	Y     int
	Color string
}

type eventView struct {
	ID        trace.EventID
	Lane      int
	X, Y      int
	Width     int
	Height    int
	Color     string
	Method    string
	Component string
	StartNs   int64
	EndNs     int64
	Tooltip   string
	REQTags   string
	Err       string
}

type markerView struct {
	Layer   string
	Style   string
	X, Y    int
	Width   int
	Tooltip string
	// EdgeFromX/Y populated for causality arrows; zero for other markers.
	HasEdge                                                    bool
	EdgeFromX, EdgeFromY, EdgeToX, EdgeToY, EdgeMidX, EdgeMidY int
}

const (
	rowHeight = 36
	rowPad    = 4
	laneInset = 180 // pixels reserved on the left for lane labels
	canvasPad = 40
	minWidth  = 6 // minimum px for fast events
)

// buildTemplateData converts the Timeline into the layout the
// template renders. Pure function; no I/O.
func buildTemplateData(t Timeline) templateData {
	events := snapshotEvents(t.Trace)

	lanes := buildLanes(events, t.Style.ComponentColors)
	laneIndex := make(map[string]int, len(lanes))
	for _, l := range lanes {
		laneIndex[l.Name] = l.Index
	}

	startNs, endNs := timespan(events)
	svgWidth, scale := computeWidth(startNs, endNs)
	svgHeight := canvasPad + len(lanes)*rowHeight + canvasPad

	eventViews := buildEventViews(events, lanes, laneIndex, startNs, scale)
	eventByID := make(map[trace.EventID]eventView, len(eventViews))
	for _, ev := range eventViews {
		eventByID[ev.ID] = ev
	}

	markers := buildMarkerViews(t.Overlays, t.Trace, eventByID)

	return templateData{
		Title:      t.title(),
		Theme:      t.Style.theme(),
		Subject:    t.Subject,
		Generator:  t.Generator,
		SeedHex:    fmt.Sprintf("0x%016X", uint64(t.Seed)),
		EventCount: len(events),
		Lanes:      lanes,
		Events:     eventViews,
		Markers:    markers,
		SVGWidth:   svgWidth,
		SVGHeight:  svgHeight,
		StartNs:    startNs,
	}
}

func snapshotEvents(t *trace.Trace) []trace.Event {
	if t == nil {
		return nil
	}
	return t.Snapshot()
}

// buildLanes groups events into rows. Component-tagged events lane
// by Component; events without a Component lane by Goroutine; the
// remainder share a default "default" lane. Lane order is
// alphabetical by name for deterministic rendering.
func buildLanes(events []trace.Event, override map[string]string) []laneView {
	names := make(map[string]struct{})
	for _, e := range events {
		names[laneNameFor(e)] = struct{}{}
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)

	out := make([]laneView, 0, len(sorted))
	for i, name := range sorted {
		out = append(out, laneView{
			Name:  name,
			Index: i,
			Y:     canvasPad + i*rowHeight,
			Color: colorFor(name, override),
		})
	}
	return out
}

func laneNameFor(e trace.Event) string {
	if e.Component != "" {
		return e.Component
	}
	if e.Goroutine != 0 {
		return fmt.Sprintf("goroutine-%d", e.Goroutine)
	}
	return "default"
}

func timespan(events []trace.Event) (start, end int64) {
	if len(events) == 0 {
		return 0, 0
	}
	start = events[0].StartNs
	end = events[0].EndNs
	for _, e := range events {
		if e.StartNs < start {
			start = e.StartNs
		}
		if e.EndNs > end {
			end = e.EndNs
		}
	}
	return start, end
}

func computeWidth(startNs, endNs int64) (width int, nsPerPixel float64) {
	span := endNs - startNs
	if span <= 0 {
		// Single-event or zero-duration trace: give it a fixed
		// canvas width so the SVG isn't degenerate.
		return 800 + laneInset, 1.0
	}
	const targetWidth = 1400
	nsPerPixel = float64(span) / float64(targetWidth-laneInset)
	width = laneInset + int(float64(span)/nsPerPixel) + canvasPad
	return width, nsPerPixel
}

func buildEventViews(
	events []trace.Event,
	lanes []laneView,
	laneIndex map[string]int,
	startNs int64,
	nsPerPixel float64,
) []eventView {
	out := make([]eventView, 0, len(events))
	for _, e := range events {
		laneName := laneNameFor(e)
		idx := laneIndex[laneName]
		lane := lanes[idx]
		x := laneInset + int(float64(e.StartNs-startNs)/nsPerPixel)
		w := max(int(float64(e.EndNs-e.StartNs)/nsPerPixel), minWidth)
		ev := eventView{
			ID:        e.ID,
			Lane:      idx,
			X:         x,
			Y:         lane.Y + rowPad,
			Width:     w,
			Height:    rowHeight - 2*rowPad,
			Color:     lane.Color,
			Method:    e.Method,
			Component: e.Component,
			StartNs:   e.StartNs,
			EndNs:     e.EndNs,
			Tooltip:   eventTooltip(e),
			REQTags:   strings.Join(e.REQTags, ","),
		}
		if e.Err != nil {
			ev.Err = e.Err.Error()
		}
		out = append(out, ev)
	}
	return out
}

func eventTooltip(e trace.Event) string {
	parts := []string{e.Method}
	if e.Component != "" {
		parts = []string{e.Component + "." + e.Method}
	}
	if e.Err != nil {
		parts = append(parts, "err="+e.Err.Error())
	}
	if len(e.REQTags) > 0 {
		parts = append(parts, "REQ="+strings.Join(e.REQTags, ","))
	}
	return strings.Join(parts, "  ")
}

func buildMarkerViews(
	overlays []Overlay,
	tr *trace.Trace,
	eventByID map[trace.EventID]eventView,
) []markerView {
	rendered := make([][]Marker, len(overlays))
	total := 0
	for i, o := range overlays {
		rendered[i] = o.Render(tr)
		total += len(rendered[i])
	}
	collected := make([]Marker, 0, total)
	for _, r := range rendered {
		collected = append(collected, r...)
	}
	// Sort for deterministic output: by layer, then EventID.
	sort.SliceStable(collected, func(i, j int) bool {
		if collected[i].Layer != collected[j].Layer {
			return collected[i].Layer < collected[j].Layer
		}
		return collected[i].EventID < collected[j].EventID
	})

	out := make([]markerView, 0, len(collected))
	for _, m := range collected {
		ev, ok := eventByID[m.EventID]
		if !ok {
			continue
		}
		mv := markerView{
			Layer:   m.Layer,
			Style:   m.Style,
			X:       ev.X,
			Y:       ev.Y,
			Width:   ev.Width,
			Tooltip: m.Tooltip,
		}
		if m.Layer == LayerCausality {
			fillCausalityEdge(&mv, m, ev, eventByID)
		}
		out = append(out, mv)
	}
	return out
}

// fillCausalityEdge populates the SVG edge coordinates for a
// causality marker — an arrow from each predecessor's center to the
// successor's center. The marker's Metadata["predecessors"] carries
// a []trace.EventID slice; the renderer draws one segment per
// predecessor. For Phase 0, a single edge is drawn per marker (the
// first predecessor); multi-predecessor causality renders as
// multiple markers via marker duplication on the overlay side.
func fillCausalityEdge(mv *markerView, m Marker, target eventView, eventByID map[trace.EventID]eventView) {
	preds, ok := m.Metadata["predecessors"].([]trace.EventID)
	if !ok || len(preds) == 0 {
		return
	}
	pred, ok := eventByID[preds[0]]
	if !ok {
		return
	}
	mv.HasEdge = true
	mv.EdgeFromX = pred.X + pred.Width/2
	mv.EdgeFromY = pred.Y + pred.Height/2
	mv.EdgeToX = target.X + target.Width/2
	mv.EdgeToY = target.Y + target.Height/2
	mv.EdgeMidX = (mv.EdgeFromX + mv.EdgeToX) / 2
	mv.EdgeMidY = (mv.EdgeFromY+mv.EdgeToY)/2 - 20 // slight upward arc
}

// colorFor returns the CSS color for the named lane. Override
// values in the caller-supplied map take precedence; fallbacks come
// from a deterministic palette indexed by FNV-1a hash of the name.
func colorFor(name string, override map[string]string) string {
	if c, ok := override[name]; ok {
		return c
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return palette[int(h.Sum32())%len(palette)]
}

// palette is the deterministic color cycle for un-overridden lanes.
// Twelve colors give enough distinction for sub-systems that
// typically have 3–8 components.
var palette = []string{
	"#5b8def", "#6cc24a", "#f5a623", "#e85d75",
	"#9b51e0", "#0fb0a8", "#ff6e40", "#3b82f6",
	"#10b981", "#facc15", "#ef4444", "#8b5cf6",
}

// templateFuncs are exported to the html/template engine so the
// template can format integers and constants without inline Go. The
// constant accessors (rowHeight, laneInset, eventHeight) let the SVG
// template reference the layout constants without re-declaring them.
var templateFuncs = template.FuncMap{
	"add":         func(a, b int) int { return a + b },
	"sub":         func(a, b int) int { return a - b },
	"mod":         func(a, b int) int { return a % b },
	"rowHeight":   func() int { return rowHeight },
	"laneInset":   func() int { return laneInset },
	"eventHeight": func() int { return rowHeight - 2*rowPad },
}
