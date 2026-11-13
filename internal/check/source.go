package check

import (
	"encoding/json"
	"strings"
	"unicode"
)

// minCopiedRunes: a predict snippet this long (normalized) found inside a
// lecture code block counts as copied even when it is only part of the block.
const minCopiedRunes = 40

// CopiesLectureCode reports whether an item's material.code is a lecture code
// block (design v2.2 §4 predict 출처): equal after normalization (whitespace
// removed, lower case), or a 40+ rune excerpt of one. A material without code
// never matches.
func CopiesLectureCode(material json.RawMessage, lectureCode []string) bool {
	var m struct {
		Code string `json:"code"`
	}
	if len(material) == 0 || json.Unmarshal(material, &m) != nil {
		return false
	}
	code := normalizeCode(m.Code)
	if code == "" {
		return false
	}
	for _, block := range lectureCode {
		b := normalizeCode(block)
		if b == code || (len([]rune(code)) >= minCopiedRunes && strings.Contains(b, code)) {
			return true
		}
	}
	return false
}

func normalizeCode(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !unicode.IsSpace(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// CoversMust reports whether the met criteria cover every must criterion: a
// must is covered by a met entry equal to it after normalization, or one
// containing the other (the tutor is told to copy must sentences into met).
func CoversMust(met, must []string) bool {
	for _, m := range must {
		want := normalizeCode(m)
		if want == "" {
			continue
		}
		covered := false
		for _, got := range met {
			g := normalizeCode(got)
			if g != "" && (g == want || strings.Contains(g, want) || strings.Contains(want, g)) {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

// reconcileVerdict lowers correct to partial when met does not cover every
// must (design v2.2 §4 판정 일관성 — the code corrects the grade).
func reconcileVerdict(verdict string, met, must []string) string {
	if verdict == "correct" && len(must) > 0 && !CoversMust(met, must) {
		return "partial"
	}
	return verdict
}
