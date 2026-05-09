// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package visualize

// htmlTemplate is the single self-contained HTML document the
// renderer emits. No external dependencies: the CSS lives in the
// <style> block, the timeline lives in an inline <svg>, and tooltips
// are SVG <title> elements that every browser renders natively. No
// JavaScript — the page is static and diffable.
//
// The template assumes data shaped by buildTemplateData. It is parsed
// via html/template, so {{.Tooltip}} et al. are HTML-escaped
// automatically. Numeric fields are formatted via the `add`/`sub`
// FuncMap helpers.
const htmlTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.Title}}</title>
<style>
:root {
  --bg: #ffffff;
  --fg: #1f2937;
  --muted: #6b7280;
  --grid: #e5e7eb;
  --row-alt: #f9fafb;
  --table-border: #e5e7eb;
  --table-header: #f3f4f6;
  --edge: #6366f1;
  --fault: #f59e0b;
  --divergence: #ef4444;
  --req: #10b981;
  --replay: #0ea5e9;
  --snapshot: #8b5cf6;
}
html.dark {
  --bg: #0f172a;
  --fg: #e2e8f0;
  --muted: #94a3b8;
  --grid: #1e293b;
  --row-alt: #111827;
  --table-border: #1e293b;
  --table-header: #1e293b;
  --edge: #818cf8;
  --fault: #fbbf24;
  --divergence: #f87171;
  --req: #34d399;
  --replay: #38bdf8;
  --snapshot: #a78bfa;
}
body {
  margin: 0;
  padding: 24px;
  background: var(--bg);
  color: var(--fg);
  font-family: ui-sans-serif, system-ui, sans-serif;
  font-size: 14px;
}
header {
  margin-bottom: 16px;
}
h1 {
  margin: 0 0 8px 0;
  font-size: 20px;
  font-weight: 600;
}
.meta {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
  color: var(--muted);
  font-size: 13px;
}
.meta strong {
  color: var(--fg);
  font-weight: 500;
}
.timeline-wrap {
  overflow-x: auto;
  border: 1px solid var(--grid);
  border-radius: 6px;
  margin-bottom: 24px;
}
svg.timeline {
  display: block;
  background: var(--bg);
}
.lane-label {
  fill: var(--fg);
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.lane-row {
  fill: var(--row-alt);
}
.lane-row.even {
  fill: var(--bg);
}
.event {
  stroke: rgba(0, 0, 0, 0.15);
  stroke-width: 1;
}
.event.err {
  stroke: var(--divergence);
  stroke-width: 2;
}
.marker-fault {
  fill: var(--fault);
  fill-opacity: 0.35;
  stroke: var(--fault);
  stroke-width: 1;
}
.marker-divergence {
  stroke: var(--divergence);
  stroke-width: 2;
  fill: none;
}
.marker-causality {
  stroke: var(--edge);
  stroke-width: 1.5;
  fill: none;
}
.marker-causality-arrow {
  fill: var(--edge);
}
.marker-req {
  fill: var(--req);
}
.marker-req-label {
  fill: var(--bg);
  font-size: 9px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.marker-replay {
  fill: var(--replay);
}
.marker-snapshot {
  fill: var(--snapshot);
}
table.events {
  border-collapse: collapse;
  width: 100%;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
}
table.events th, table.events td {
  border: 1px solid var(--table-border);
  padding: 6px 10px;
  text-align: left;
  vertical-align: top;
}
table.events th {
  background: var(--table-header);
  font-weight: 600;
}
table.events tr:nth-child(even) td {
  background: var(--row-alt);
}
.cell-err {
  color: var(--divergence);
}
.cell-req {
  color: var(--req);
}
section h2 {
  margin: 0 0 8px 0;
  font-size: 15px;
  font-weight: 600;
}
</style>
</head>
<body class="{{if eq .Theme "dark"}}dark{{end}}">
<header>
  <h1>{{.Title}}</h1>
  <div class="meta">
    <span><strong>Subject:</strong> {{.Subject}}</span>
    <span><strong>Generator:</strong> {{.Generator}}</span>
    <span><strong>Seed:</strong> {{.SeedHex}}</span>
    <span><strong>Events:</strong> {{.EventCount}}</span>
  </div>
</header>

<section>
  <h2>Timeline</h2>
  <div class="timeline-wrap">
    <svg class="timeline" width="{{.SVGWidth}}" height="{{.SVGHeight}}" xmlns="http://www.w3.org/2000/svg">
      {{- range .Lanes }}
      <rect class="lane-row{{if eq (mod .Index 2) 0}} even{{end}}" x="0" y="{{.Y}}" width="{{$.SVGWidth}}" height="{{rowHeight}}"></rect>
      <text class="lane-label" x="12" y="{{add .Y 22}}">{{.Name}}</text>
      <line x1="{{laneInset}}" y1="{{.Y}}" x2="{{$.SVGWidth}}" y2="{{.Y}}" stroke="var(--grid)" stroke-width="1"></line>
      {{- end }}

      {{- range .Events }}
      <g>
        <rect class="event{{if .Err}} err{{end}}" x="{{.X}}" y="{{.Y}}" width="{{.Width}}" height="{{.Height}}" rx="2" ry="2" fill="{{.Color}}">
          <title>{{.Tooltip}}</title>
        </rect>
      </g>
      {{- end }}

      {{- range .Markers }}
      {{- if eq .Layer "fault" }}
      <rect class="marker-fault" x="{{.X}}" y="{{.Y}}" width="{{.Width}}" height="{{eventHeight}}">
        <title>{{.Tooltip}}</title>
      </rect>
      {{- else if eq .Layer "divergence" }}
      <g>
        <line class="marker-divergence" x1="{{.X}}" y1="{{.Y}}" x2="{{add .X eventHeight}}" y2="{{add .Y eventHeight}}"></line>
        <line class="marker-divergence" x1="{{add .X eventHeight}}" y1="{{.Y}}" x2="{{.X}}" y2="{{add .Y eventHeight}}"></line>
        <title>{{.Tooltip}}</title>
      </g>
      {{- else if eq .Layer "causality" }}
      {{- if .HasEdge }}
      <g>
        <path class="marker-causality" d="M {{.EdgeFromX}} {{.EdgeFromY}} Q {{.EdgeMidX}} {{.EdgeMidY}}, {{.EdgeToX}} {{.EdgeToY}}"></path>
        <polygon class="marker-causality-arrow" points="{{.EdgeToX}},{{.EdgeToY}} {{sub .EdgeToX 6}},{{sub .EdgeToY 3}} {{sub .EdgeToX 6}},{{add .EdgeToY 3}}"></polygon>
        <title>{{.Tooltip}}</title>
      </g>
      {{- end }}
      {{- else if eq .Layer "req" }}
      <g>
        <rect class="marker-req" x="{{.X}}" y="{{sub .Y 10}}" width="20" height="10" rx="2" ry="2"></rect>
        <text class="marker-req-label" x="{{add .X 3}}" y="{{sub .Y 2}}">REQ</text>
        <title>{{.Tooltip}}</title>
      </g>
      {{- else if eq .Layer "replay" }}
      <g>
        <polygon class="marker-replay" points="{{.X}},{{sub .Y 8}} {{add .X 10}},{{sub .Y 4}} {{.X}},{{.Y}}"></polygon>
        <title>{{.Tooltip}}</title>
      </g>
      {{- else if eq .Layer "snapshot" }}
      <g>
        <rect class="marker-snapshot" x="{{.X}}" y="{{sub .Y 12}}" width="3" height="14"></rect>
        <polygon class="marker-snapshot" points="{{add .X 3}},{{sub .Y 12}} {{add .X 13}},{{sub .Y 9}} {{add .X 3}},{{sub .Y 6}}"></polygon>
        <title>{{.Tooltip}}</title>
      </g>
      {{- end }}
      {{- end }}
    </svg>
  </div>
</section>

<section>
  <h2>Events</h2>
  <table class="events">
    <thead>
      <tr>
        <th>ID</th>
        <th>Component</th>
        <th>Method</th>
        <th>Start (ns)</th>
        <th>End (ns)</th>
        <th>REQ</th>
        <th>Error</th>
      </tr>
    </thead>
    <tbody>
      {{- range .Events }}
      <tr>
        <td>{{.ID}}</td>
        <td>{{.Component}}</td>
        <td>{{.Method}}</td>
        <td>{{.StartNs}}</td>
        <td>{{.EndNs}}</td>
        <td class="cell-req">{{.REQTags}}</td>
        <td class="cell-err">{{.Err}}</td>
      </tr>
      {{- end }}
    </tbody>
  </table>
</section>
</body>
</html>
`
