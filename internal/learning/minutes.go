package learning

import (
	"encoding/json"
	"math"
	"strconv"

	"github.com/forgen-team/punisher/internal/lecture"
)

// Reading-time constants (quality design v2.2 §5, Q4) [사용자 체감 기반 — 실측 데이터가 생기면 보정].
const (
	ProseCharsPerMinute = 600.0
	// ProseCharsPerMinuteEn: English prose (≈250 words/min, letters without spaces) [추정].
	ProseCharsPerMinuteEn = 1300.0
	CodeCharsPerMinute    = 500.0
	MinutesPerExercise    = 1.5
	MinutesPerPause       = 0.5
	MinutesPerTraceRow    = 0.1
)

// EstimateMinutes is the displayed lecture time: the sum of SectionMinutes, so
// the header total equals the sections' minutes; at least 1. Pure; section
// minutes written by the model are not used.
func EstimateMinutes(c lecture.Content) int {
	total := 0
	for _, m := range SectionMinutes(c) {
		total += m
	}
	return max(total, 1)
}

// SectionMinutes is the reading time of each section alone, in order: prose
// chars (no spaces)/600 + code·log chars/500 + exercises×1.5 + pauses×0.5 +
// trace rows×0.1, rounded, at least 1.
func SectionMinutes(c lecture.Content) []int {
	out := make([]int, len(c.Sections))
	for i, s := range c.Sections {
		out[i] = readingMinutes(lecture.Content{Sections: []lecture.Section{s}})
	}
	return out
}

func readingMinutes(c lecture.Content) int {
	m := lecture.MeasureContent(c)
	rate := ProseCharsPerMinute
	if m.Language() == "en" {
		rate = ProseCharsPerMinuteEn
	}
	total := float64(m.ProseChars)/rate + float64(m.CodeChars)/CodeCharsPerMinute +
		float64(m.Exercises)*MinutesPerExercise + float64(m.Pauses)*MinutesPerPause + float64(m.TraceRows)*MinutesPerTraceRow
	if n := int(math.Round(total)); n > 1 {
		return n
	}
	return 1
}

// WithSectionMinutes returns stored lecture content with each section's
// SectionMinutes added as "estimatedMinutes" and the total (EstimateMinutes)
// as the top-level "estimatedMinutes" (the model's "minutes" is kept but not
// shown), plus that total. Other fields pass through; content that does not
// decode is returned unchanged with total 0.
func WithSectionMinutes(raw json.RawMessage) (json.RawMessage, int) {
	var c lecture.Content
	var doc map[string]json.RawMessage
	var sections []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &c) != nil || json.Unmarshal(raw, &doc) != nil ||
		json.Unmarshal(doc["sections"], &sections) != nil || len(sections) != len(c.Sections) {
		return raw, 0
	}
	for i, m := range SectionMinutes(c) {
		sections[i]["estimatedMinutes"] = json.RawMessage(strconv.Itoa(m))
	}
	encoded, err := json.Marshal(sections)
	if err != nil {
		return raw, 0
	}
	total := EstimateMinutes(c)
	doc["sections"] = encoded
	doc["estimatedMinutes"] = json.RawMessage(strconv.Itoa(total))
	out, err := json.Marshal(doc)
	if err != nil {
		return raw, 0
	}
	return out, total
}
