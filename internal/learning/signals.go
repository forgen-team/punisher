package learning

import (
	"fmt"
	"strings"
)

// Lecture signals (v3 design §6): where readers got stuck in a lecture —
// in-lecture exercises, labs and lecture questions. Check failures are the
// learner's state, not the lecture's, and are left out.

// Exercise attempt kinds and verdicts (lecture_exercise_attempts).
const (
	AttemptExercise = "exercise"
	AttemptLab      = "lab"

	ExerciseCorrect   = "correct"
	ExercisePartial   = "partial"
	ExerciseIncorrect = "incorrect"
	ExerciseSkipped   = "skipped"
	LabMatched        = "matched"
	LabDifferent      = "different"
	LabUnreadable     = "unreadable"
)

// ValidAttemptKind reports a known attempt kind.
func ValidAttemptKind(k string) bool { return k == AttemptExercise || k == AttemptLab }

// AttemptMiss: the attempt shows the section did not land.
func AttemptMiss(verdict string) bool {
	switch verdict {
	case ExerciseIncorrect, ExerciseSkipped, LabDifferent, LabUnreadable:
		return true
	}
	return false
}

// SectionSignal is one section's counts.
type SectionSignal struct {
	Section    string `json:"section"`
	Exercises  int    `json:"exercises"`
	ExerciseMs int    `json:"exerciseMisses"`
	Labs       int    `json:"labs"`
	LabMisses  int    `json:"labMisses"`
	Questions  int    `json:"questions"`
}

// SectionFlag is a section marked '고칠 곳' with its reasons.
type SectionFlag struct {
	Section string   `json:"section"`
	Reasons []string `json:"reasons"`
}

// FlagMinQuestions: questions on one section that flag it on their own.
const FlagMinQuestions = 2

// FlagSections marks a section when at least one exercise or lab missed and
// misses are half or more of its attempts, or when it drew FlagMinQuestions
// questions. Reasons are worded in locale (ko|en; they are shown and fed to a regeneration).
func FlagSections(signals []SectionSignal, locale string) []SectionFlag {
	out := []SectionFlag{}
	for _, s := range signals {
		var reasons []string
		attempts, misses := s.Exercises+s.Labs, s.ExerciseMs+s.LabMisses
		if misses >= 1 && misses*2 >= attempts {
			if s.ExerciseMs > 0 {
				reasons = append(reasons, fmt.Sprintf(inLocale(locale, "직접 해 보기 %d번 중 %d번 틀리거나 건너뜀",
					"missed or skipped %[2]d of %[1]d exercises"), s.Exercises, s.ExerciseMs))
			}
			if s.LabMisses > 0 {
				reasons = append(reasons, fmt.Sprintf(inLocale(locale, "실습 %d번 중 %d번 결과가 예상과 다름",
					"%[2]d of %[1]d labs differed from the expected result"), s.Labs, s.LabMisses))
			}
		}
		if s.Questions >= FlagMinQuestions {
			reasons = append(reasons, fmt.Sprintf(inLocale(locale, "질문 %d개", "%d questions"), s.Questions))
		}
		if len(reasons) > 0 {
			out = append(out, SectionFlag{Section: s.Section, Reasons: reasons})
		}
	}
	return out
}

// SectionOfAnchor is the section kind of a block anchor ("principles-3" → "principles").
func SectionOfAnchor(anchor string) string {
	if i := strings.LastIndexByte(anchor, '-'); i > 0 {
		return anchor[:i]
	}
	return anchor
}
