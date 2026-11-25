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
	// Self-graded, no LLM (lecture quality v2): a pause prediction, a section-end
	// recall prompt, the summary recalled before it opens.
	AttemptPause   = "pause"
	AttemptRecall  = "recall"
	AttemptSummary = "summary"

	ExerciseCorrect   = "correct"
	ExercisePartial   = "partial"
	ExerciseIncorrect = "incorrect"
	ExerciseSkipped   = "skipped"
	LabMatched        = "matched"
	LabDifferent      = "different"
	LabUnreadable     = "unreadable"
	// Recall self-grades (partial = ExercisePartial); a pause is graded correct | incorrect.
	RecallRemembered = "remembered"
	RecallForgot     = "forgot"
	// SummaryWritten: the learner wrote a recall before opening the summary (the text is not kept).
	SummaryWritten = "written"
)

// ValidAttemptKind reports a known attempt kind.
func ValidAttemptKind(k string) bool {
	switch k {
	case AttemptExercise, AttemptLab, AttemptPause, AttemptRecall, AttemptSummary:
		return true
	}
	return false
}

// SelfGraded: the kind is graded by the learner's one tap (no LLM call).
func SelfGraded(kind string) bool {
	return kind == AttemptPause || kind == AttemptRecall || kind == AttemptSummary
}

// ValidSelfGrade reports a verdict the learner may give a self-graded kind.
func ValidSelfGrade(kind, verdict string) bool {
	switch kind {
	case AttemptPause:
		return verdict == ExerciseCorrect || verdict == ExerciseIncorrect
	case AttemptRecall:
		return verdict == RecallRemembered || verdict == ExercisePartial || verdict == RecallForgot
	case AttemptSummary:
		return verdict == SummaryWritten
	}
	return false
}

// AttemptMiss: the attempt shows the section did not land.
func AttemptMiss(verdict string) bool {
	switch verdict {
	case ExerciseIncorrect, ExerciseSkipped, LabDifferent, LabUnreadable, RecallForgot:
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
	// Pauses / PauseMisses: self-graded predictions (wrong or skipped); Recalls / RecallMisses: forgot or skipped.
	Pauses       int `json:"pauses"`
	PauseMisses  int `json:"pauseMisses"`
	Recalls      int `json:"recalls"`
	RecallMisses int `json:"recallMisses"`
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
		attempts, misses := s.Exercises+s.Labs+s.Pauses+s.Recalls, s.ExerciseMs+s.LabMisses+s.PauseMisses+s.RecallMisses
		if misses >= 1 && misses*2 >= attempts {
			if s.ExerciseMs > 0 {
				reasons = append(reasons, fmt.Sprintf(inLocale(locale, "직접 해 보기 %d번 중 %d번 틀리거나 건너뜀",
					"missed or skipped %[2]d of %[1]d exercises"), s.Exercises, s.ExerciseMs))
			}
			if s.LabMisses > 0 {
				reasons = append(reasons, fmt.Sprintf(inLocale(locale, "실습 %d번 중 %d번 결과가 예상과 다름",
					"%[2]d of %[1]d labs differed from the expected result"), s.Labs, s.LabMisses))
			}
			if s.PauseMisses > 0 {
				reasons = append(reasons, fmt.Sprintf(inLocale(locale, "출력 예측 %d번 중 %d번 빗나가거나 건너뜀",
					"missed or skipped %[2]d of %[1]d output predictions"), s.Pauses, s.PauseMisses))
			}
			if s.RecallMisses > 0 {
				reasons = append(reasons, fmt.Sprintf(inLocale(locale, "떠올려 보기 %d번 중 %d번 못 떠올리거나 건너뜀",
					"couldn't recall or skipped %[2]d of %[1]d recall prompts"), s.Recalls, s.RecallMisses))
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

// AnchorSignal is one block's signals (research #15: block-level instead of section-level).
type AnchorSignal struct {
	Anchor string `json:"anchor"`
	// ConfidentMiss: a pause predicted with "sure" that turned out wrong (착각).
	ConfidentMiss bool `json:"confidentMiss"`
	// Forgot: a recall the learner could not recall.
	Forgot bool `json:"forgot"`
	// Questions: lecture questions and passage explanations asked on the block.
	Questions int `json:"questions"`
	// TermOpens: term popovers opened from the block; DwellMs: time it was the block in view.
	TermOpens int   `json:"termOpens"`
	DwellMs   int64 `json:"dwellMs"`
	// SlowRead: the block was in view far longer than its reading time (set by SlowReads).
	SlowRead bool `json:"slowRead"`
}

// Reading-signal thresholds (research #15) [추정 — 기록이 쌓이면 조정].
const (
	// FlagMinTermOpens: term popovers opened from one block that flag it.
	FlagMinTermOpens = 3
	// SlowReadFactor × the block's reading time (and at least SlowReadMinMs) flags a block.
	SlowReadFactor = 3
	SlowReadMinMs  = 60_000
)

// SlowReads marks the signals whose dwell is SlowReadFactor× the block's
// reading time (expectedMs by anchor) and at least SlowReadMinMs.
func SlowReads(signals []AnchorSignal, expectedMs map[string]int64) {
	for i := range signals {
		exp, ok := expectedMs[signals[i].Anchor]
		signals[i].SlowRead = ok && signals[i].DwellMs >= SlowReadMinMs && signals[i].DwellMs >= SlowReadFactor*exp
	}
}

// AnchorFlag is a block marked to rework, with its reasons.
type AnchorFlag struct {
	Anchor  string   `json:"anchor"`
	Reasons []string `json:"reasons"`
}

// FlagAnchors marks blocks with a confident miss, a forgotten recall, or
// FlagMinQuestions+ questions, worded in locale.
func FlagAnchors(signals []AnchorSignal, locale string) []AnchorFlag {
	out := []AnchorFlag{}
	for _, s := range signals {
		var reasons []string
		if s.ConfidentMiss {
			reasons = append(reasons, inLocale(locale, "확신하고 예측했는데 틀림", "predicted with confidence and was wrong"))
		}
		if s.Forgot {
			reasons = append(reasons, inLocale(locale, "읽은 직후 떠올리지 못함", "could not recall it right after reading"))
		}
		if s.Questions >= FlagMinQuestions {
			reasons = append(reasons, fmt.Sprintf(inLocale(locale, "이 블록에 질문·설명 요청 %d개", "%d questions or explanations on this block"), s.Questions))
		}
		if s.TermOpens >= FlagMinTermOpens {
			reasons = append(reasons, fmt.Sprintf(inLocale(locale, "용어 설명을 %d번 열어 봄", "opened term explanations %d times"), s.TermOpens))
		}
		if s.SlowRead {
			reasons = append(reasons, fmt.Sprintf(inLocale(locale, "읽기 시간보다 훨씬 오래 머묾(%d초)", "stayed far longer than its reading time (%ds)"), s.DwellMs/1000))
		}
		if len(reasons) > 0 {
			out = append(out, AnchorFlag{Anchor: s.Anchor, Reasons: reasons})
		}
	}
	return out
}
