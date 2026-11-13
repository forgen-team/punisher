package learning

import (
	"encoding/json"
	"math"
	"strconv"

	"github.com/wooo-jin/punisher/internal/lecture"
)

// Reading-time constants (quality design v2.2 §5, Q4) [사용자 체감 기반 — 실측 데이터가 생기면 보정].
const (
	ProseCharsPerMinute = 600.0
	CodeCharsPerMinute  = 500.0
	MinutesPerExercise  = 1.5
	MinutesPerPause     = 0.5
	MinutesPerTraceRow  = 0.1
)

// EstimateMinutes is the displayed lecture time: prose chars (no spaces)/600
// + code·log chars/500 + exercises×1.5 + pauses×0.5 + trace rows×0.1,
// rounded, at least 1. Pure; section minutes written by the model are not used.
func EstimateMinutes(c lecture.Content) int {
	m := lecture.MeasureContent(c)
	total := float64(m.ProseChars)/ProseCharsPerMinute + float64(m.CodeChars)/CodeCharsPerMinute +
		float64(m.Exercises)*MinutesPerExercise + float64(m.Pauses)*MinutesPerPause + float64(m.TraceRows)*MinutesPerTraceRow
	if n := int(math.Round(total)); n > 1 {
		return n
	}
	return 1
}

// SectionMinutes is EstimateMinutes applied to each section alone, in order.
// Each is rounded on its own, so the sum may differ from EstimateMinutes.
func SectionMinutes(c lecture.Content) []int {
	out := make([]int, len(c.Sections))
	for i, s := range c.Sections {
		out[i] = EstimateMinutes(lecture.Content{Sections: []lecture.Section{s}})
	}
	return out
}

// WithSectionMinutes returns stored lecture content with each section's
// SectionMinutes added as "estimatedMinutes" (the model's "minutes" is kept but
// not shown). Other fields pass through; content that does not decode is
// returned unchanged.
func WithSectionMinutes(raw json.RawMessage) json.RawMessage {
	var c lecture.Content
	var doc map[string]json.RawMessage
	var sections []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &c) != nil || json.Unmarshal(raw, &doc) != nil ||
		json.Unmarshal(doc["sections"], &sections) != nil || len(sections) != len(c.Sections) {
		return raw
	}
	for i, m := range SectionMinutes(c) {
		sections[i]["estimatedMinutes"] = json.RawMessage(strconv.Itoa(m))
	}
	encoded, err := json.Marshal(sections)
	if err != nil {
		return raw
	}
	doc["sections"] = encoded
	out, err := json.Marshal(doc)
	if err != nil {
		return raw
	}
	return out
}
