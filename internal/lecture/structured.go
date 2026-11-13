package lecture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// bulletRe matches a top-level markdown list item.
var bulletRe = regexp.MustCompile(`^(?:[-*+]|\d+[.)])\s+\S`)

// Structured block bodies (quality design v2.2 §6). The LLM writes data only;
// the web renders it (no HTML/JS from the model).

// TraceData is the body of @@trace: a hand-calculation table shown row by row.
type TraceData struct {
	Caption string     `json:"caption"`
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
	Note    string     `json:"note,omitempty"`
}

// ChartAxisX is the x axis of a chart (values are labels: strings or numbers).
type ChartAxisX struct {
	Label  string            `json:"label"`
	Values []json.RawMessage `json:"values"`
}

// ChartAxisY is the y axis of a chart.
type ChartAxisY struct {
	Label string `json:"label"`
	Scale string `json:"scale"` // linear | log
}

// ChartSeries is one line/bar series.
type ChartSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
}

// ChartData is the body of @@chart.
type ChartData struct {
	Kind   string        `json:"kind"` // line | bar
	Title  string        `json:"title"`
	X      ChartAxisX    `json:"x"`
	Y      ChartAxisY    `json:"y"`
	Series []ChartSeries `json:"series"`
	Note   string        `json:"note,omitempty"`
}

// DiffData is the body of @@diff: a before/after change (prescription).
type DiffData struct {
	Before string `json:"before"`
	After  string `json:"after"`
	Note   string `json:"note,omitempty"`
}

// Structured block bounds (§6 검증 열).
const (
	TraceMinColumns  = 2
	TraceMaxColumns  = 6
	TraceMinRows     = 2
	TraceMaxRows     = 16
	ChartMinSeries   = 1
	ChartMaxSeries   = 5
	ChartMinPoints   = 2
	ChartMaxPoints   = 20
	DiffMaxLines     = 60
	SummaryMinBullet = 3
	SummaryMaxBullet = 6
)

// decodeStrict decodes a structured body, rejecting unknown top-level keys so
// a misspelled field is reported instead of silently dropped.
func decodeStrict(raw json.RawMessage, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// structuredProblems lists the schema problems of a trace/chart/diff block ("" free = valid).
func structuredProblems(b Block) []string {
	if len(b.Data) == 0 {
		return []string{fmt.Sprintf("%s block body must be one JSON object (no prose around it)", b.Type)}
	}
	switch b.Type {
	case BlockTrace:
		var d TraceData
		if err := decodeStrict(b.Data, &d); err != nil {
			return []string{"trace JSON does not match {caption, columns[], rows[][] of strings, note}: " + err.Error()}
		}
		return traceProblems(d)
	case BlockChart:
		var d ChartData
		if err := decodeStrict(b.Data, &d); err != nil {
			return []string{"chart JSON does not match {kind, title, x{label, values}, y{label, scale}, series[{name, values[] numbers}], note}: " + err.Error()}
		}
		return chartProblems(d)
	case BlockDiff:
		var d DiffData
		if err := decodeStrict(b.Data, &d); err != nil {
			return []string{"diff JSON does not match {before, after, note}: " + err.Error()}
		}
		return diffProblems(d)
	}
	return nil
}

func traceProblems(d TraceData) []string {
	var p []string
	if n := len(d.Columns); n < TraceMinColumns || n > TraceMaxColumns {
		p = append(p, fmt.Sprintf("trace needs %d..%d columns, got %d", TraceMinColumns, TraceMaxColumns, n))
	}
	if n := len(d.Rows); n < TraceMinRows || n > TraceMaxRows {
		p = append(p, fmt.Sprintf("trace needs %d..%d rows, got %d", TraceMinRows, TraceMaxRows, n))
	}
	for i, row := range d.Rows {
		if len(row) != len(d.Columns) {
			p = append(p, fmt.Sprintf("trace row %d has %d cells, columns has %d", i+1, len(row), len(d.Columns)))
		}
	}
	return p
}

func chartProblems(d ChartData) []string {
	var p []string
	if d.Kind != "line" && d.Kind != "bar" {
		p = append(p, fmt.Sprintf("chart kind must be line or bar, got %q", d.Kind))
	}
	if d.Y.Scale != "linear" && d.Y.Scale != "log" {
		p = append(p, fmt.Sprintf("chart y.scale must be linear or log, got %q", d.Y.Scale))
	}
	points := len(d.X.Values)
	if points < ChartMinPoints || points > ChartMaxPoints {
		p = append(p, fmt.Sprintf("chart needs %d..%d x values, got %d", ChartMinPoints, ChartMaxPoints, points))
	}
	for i, v := range d.X.Values {
		if !scalarLabel(v) {
			p = append(p, fmt.Sprintf("chart x.values[%d] must be a string or a number", i))
		}
	}
	if n := len(d.Series); n < ChartMinSeries || n > ChartMaxSeries {
		p = append(p, fmt.Sprintf("chart needs %d..%d series, got %d", ChartMinSeries, ChartMaxSeries, n))
	}
	for i, s := range d.Series {
		if len(s.Values) != points {
			p = append(p, fmt.Sprintf("chart series %d has %d values, x has %d", i+1, len(s.Values), points))
		}
		for _, v := range s.Values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				p = append(p, fmt.Sprintf("chart series %d has a non-finite value", i+1))
				break
			}
			if d.Y.Scale == "log" && v <= 0 {
				p = append(p, fmt.Sprintf("chart series %d has a value <= 0 on a log scale", i+1))
				break
			}
		}
	}
	return p
}

func scalarLabel(raw json.RawMessage) bool {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return true
	}
	var f float64
	return json.Unmarshal(raw, &f) == nil
}

func diffProblems(d DiffData) []string {
	var p []string
	if strings.TrimSpace(d.Before) == "" || strings.TrimSpace(d.After) == "" {
		p = append(p, "diff needs both before and after")
	} else if strings.TrimSpace(d.Before) == strings.TrimSpace(d.After) {
		p = append(p, "diff before and after are the same")
	}
	for _, side := range []struct{ name, text string }{{"before", d.Before}, {"after", d.After}} {
		if n := len(strings.Split(strings.TrimRight(side.text, "\n"), "\n")); n > DiffMaxLines {
			p = append(p, fmt.Sprintf("diff %s has %d lines (max %d)", side.name, n, DiffMaxLines))
		}
	}
	return p
}

// TraceRows is the number of rows of a trace block (0 when its data does not parse).
func TraceRows(b Block) int {
	var d TraceData
	if b.Type != BlockTrace || json.Unmarshal(b.Data, &d) != nil {
		return 0
	}
	return len(d.Rows)
}

// countBullets counts top-level markdown list items ("- ", "* ", "1. ").
func countBullets(text string) int {
	n := 0
	for _, line := range strings.Split(text, "\n") {
		if bulletRe.MatchString(line) {
			n++
		}
	}
	return n
}
