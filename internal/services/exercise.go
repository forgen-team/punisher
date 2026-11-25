package services

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/llmjson"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

// In-lecture exercises and labs (v3 design §2, §4): the answer is saved at
// once (the solution opens on the screen) and graded in the background. Not
// mastery evidence — records and lecture signals only.
const (
	MaxExerciseAnswerRunes = 2000
	// MaxLabOutputBytes: a pasted lab output is cut to its first 8KB.
	MaxLabOutputBytes = 8 << 10
	exerciseTimeout   = 2 * time.Minute
)

var (
	// ErrExerciseNotFound: no exercise/lab block with that anchor and content in the lecture (404).
	ErrExerciseNotFound = errors.New("그 문제를 찾지 못했어요")
	// ErrExerciseStale: the lecture changed since the page loaded (409) — reload.
	ErrExerciseStale = errors.New("강의 내용이 바뀌었어요. 새로고침해 주세요")
)

// ExerciseInput is one answer (or skip) to a block. Verdict and Confidence
// are the learner's own grade of a self-graded block (pause, recall, summary).
type ExerciseInput struct {
	Anchor      string `json:"anchor"`
	ContentHash string `json:"contentHash"`
	Answer      string `json:"answer"`
	Skip        bool   `json:"skip"`
	Verdict     string `json:"verdict,omitempty"`
	Confidence  string `json:"confidence,omitempty"`
}

// exerciseBlock is the located block: exercise (+ its answer), lab, pause
// (+ the output it hides), recall or summary.
type exerciseBlock struct {
	kind, section, problem, solution string
	// objective: the recall's objective ("" = none given).
	objective string
}

// BlockContentHash is the hash the screen sends with an attempt: the first 12
// hex of sha1(problem + "\x00" + solution) — an exercise's text and its @@answer, or a lab's text.
func BlockContentHash(problem, solution string) string {
	sum := sha1.Sum([]byte(problem + "\x00" + solution))
	return hex.EncodeToString(sum[:])[:12]
}

func findExercise(c lecture.Content, anchor string) (exerciseBlock, bool) {
	for _, s := range c.Sections {
		for i, b := range s.Blocks {
			if b.Anchor != anchor {
				continue
			}
			switch b.Type {
			case lecture.BlockExercise:
				sol := ""
				if i+1 < len(s.Blocks) && s.Blocks[i+1].Type == lecture.BlockAnswer {
					sol = s.Blocks[i+1].Text
				}
				return exerciseBlock{kind: learning.AttemptExercise, section: s.Kind, problem: b.Text, solution: sol}, true
			case lecture.BlockLab:
				return exerciseBlock{kind: learning.AttemptLab, section: s.Kind, problem: b.Text}, true
			case lecture.BlockPause:
				out := "" // the output it hides — the screen pairs only a next code/log block (blockGroups.ts)
				if i+1 < len(s.Blocks) && (s.Blocks[i+1].Type == lecture.BlockLog || s.Blocks[i+1].Type == lecture.BlockCode) {
					out = s.Blocks[i+1].Text
				}
				return exerciseBlock{kind: learning.AttemptPause, section: s.Kind, problem: b.Text, solution: out}, true
			case lecture.BlockRecall:
				return exerciseBlock{kind: learning.AttemptRecall, section: s.Kind, problem: b.Text, objective: b.Objective}, true
			case lecture.BlockSummary:
				return exerciseBlock{kind: learning.AttemptSummary, section: s.Kind, problem: b.Text}, true
			}
			return exerciseBlock{}, false
		}
	}
	return exerciseBlock{}, false
}

// SubmitExercise saves an answer to an exercise/lab block of a lecture the
// user can read and starts its grading; a skip is saved without grading.
func (s *Service) SubmitExercise(ctx context.Context, userID string, lec models.Lecture, in ExerciseInput) (models.ExerciseAttempt, error) {
	var content lecture.Content
	if err := json.Unmarshal(lec.Content, &content); err != nil {
		return models.ExerciseAttempt{}, fmt.Errorf("decode lecture content: %w", err)
	}
	blk, ok := findExercise(content, in.Anchor)
	if !ok {
		return models.ExerciseAttempt{}, ErrExerciseNotFound
	}
	if BlockContentHash(blk.problem, blk.solution) != in.ContentHash {
		return models.ExerciseAttempt{}, ErrExerciseStale
	}
	a := store.NewExerciseAttempt{UserID: userID, LectureID: lec.ID, Anchor: in.Anchor, ContentHash: in.ContentHash,
		Section: blk.section, Kind: blk.kind}
	if in.Skip {
		a.Verdict, a.Status = learning.ExerciseSkipped, store.AttemptDone
		return s.Store.InsertExerciseAttempt(ctx, a)
	}
	if learning.SelfGraded(blk.kind) {
		return s.submitSelfGraded(ctx, userID, lec, blk, a, in)
	}
	answer := strings.TrimSpace(in.Answer)
	if blk.kind == learning.AttemptLab {
		answer = truncateBytes(strings.TrimRight(in.Answer, " \n\t"), MaxLabOutputBytes)
	}
	if answer == "" {
		return models.ExerciseAttempt{}, invalidCheck(CodeAnswerEmpty, "answer is required")
	}
	if blk.kind == learning.AttemptExercise && utf8.RuneCountInString(answer) > MaxExerciseAnswerRunes {
		return models.ExerciseAttempt{}, invalidCheck(CodeAnswerTooLong, "answer is too long (max %d characters)", MaxExerciseAnswerRunes)
	}
	if !s.AllowUser(userID) {
		return models.ExerciseAttempt{}, ErrNotOwner
	}
	if err := s.CheckBudget(ctx, userID, false); err != nil {
		return models.ExerciseAttempt{}, err
	}
	apiKey, err := s.Keys.RequestKey(ctx, userID)
	if err != nil {
		return models.ExerciseAttempt{}, err
	}
	a.Answer = answer
	saved, err := s.Store.InsertExerciseAttempt(ctx, a)
	if err != nil {
		return saved, err
	}
	conceptName, lang := lec.ConceptID, learning.NormalizeLocale(lec.Language) // 연습 평·실습 읽기는 강의 언어(i18n §1.1)
	if info, err := s.Store.ConceptInfo(ctx, userID, lec.ConceptID, lang); err == nil {
		conceptName = info.Name
	}
	runDetached("exercise", saved.ID, exerciseTimeout, func(bg context.Context) error {
		return s.gradeExercise(bg, apiKey, userID, lec.ConceptID, conceptName, lang, saved.ID, blk, answer)
	}, func(bg context.Context, err error) {
		if ferr := s.Store.FinishExerciseAttempt(bg, userID, saved.ID, store.AttemptFailed, "", "", ""); ferr != nil {
			logf("fail exercise attempt %s: %v", saved.ID, ferr)
		}
	})
	return saved, nil
}

type exerciseReply struct {
	Verdict  string `json:"verdict"`
	Feedback string `json:"feedback"`
}

var exerciseVerdicts = map[string]map[string]bool{
	learning.AttemptExercise: {learning.ExerciseCorrect: true, learning.ExercisePartial: true, learning.ExerciseIncorrect: true},
	learning.AttemptLab:      {learning.LabMatched: true, learning.LabDifferent: true, learning.LabUnreadable: true},
}

func (s *Service) gradeExercise(ctx context.Context, apiKey, userID, conceptID, conceptName, lang, attemptID string, blk exerciseBlock, answer string) error {
	set := s.UserPrompts(ctx, userID)
	req := llm.MessagesRequest{Model: s.Cfg.FastModel, System: set.Text(prompts.BlockExerciseGrade, nil),
		User: prompts.ExerciseGrade(conceptName, blk.problem, blk.solution, answer), MaxTokens: 400, Timeout: 60 * time.Second,
		// 짧은 호출 줄: 강의 생성(긴 호출)이 interactive 줄을 채워도 평이 밀리지 않게(용어 팝오버와 같은 줄).
		Priority: llm.PriorityShort}
	label := "exercise"
	if blk.kind == learning.AttemptLab {
		req.Model, req.System, req.User = s.Cfg.CheckModel, set.Text(prompts.BlockLabRead, nil), prompts.LabRead(conceptName, blk.problem, answer)
		req.MaxTokens, label = 700, "lab"
	}
	req.System = prompts.WithOutputLanguage(req.System, lang)
	var u usage
	text, err := s.complete(ctx, apiKey, req, &u)
	s.recordUsage(ctx, userID, label, conceptID, u, err)
	if err != nil {
		return err
	}
	reply, err := llmjson.TryParseJSON[exerciseReply](text)
	if err != nil {
		return fmt.Errorf("parse %s reply: %w", label, err)
	}
	reply.Verdict, reply.Feedback = strings.TrimSpace(reply.Verdict), strings.TrimSpace(reply.Feedback)
	if !exerciseVerdicts[blk.kind][reply.Verdict] || reply.Feedback == "" {
		return fmt.Errorf("%s reply failed validation: verdict %q", label, reply.Verdict)
	}
	return s.Store.FinishExerciseAttempt(ctx, userID, attemptID, store.AttemptDone, reply.Verdict, reply.Feedback, u.model)
}

func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// LectureFixes is the '고칠 곳' of a lecture for the user (v3 §6).
func (s *Service) LectureFixes(ctx context.Context, userID, lectureID string) ([]learning.SectionFlag, error) {
	signals, err := s.Store.LectureSignals(ctx, userID, lectureID)
	if err != nil {
		return nil, err
	}
	return learning.FlagSections(signals, s.Store.LocaleFor(ctx, userID)), nil
}

// signalLines are the lecture's '고칠 곳' as prompt lines ("섹션 제목 — 이유; 이유").
// Failure only logs: a regeneration goes on without them.
func (s *Service) signalLines(ctx context.Context, userID string, lec models.Lecture) []string {
	flags, err := s.LectureFixes(ctx, userID, lec.ID)
	if err != nil {
		logf("lecture signals for %s: %v", lec.ID, err)
		return nil
	}
	titles := map[string]string{}
	var content lecture.Content
	if json.Unmarshal(lec.Content, &content) == nil {
		for _, sec := range content.Sections {
			titles[sec.Kind] = sec.Title
		}
	}
	var out []string
	for _, f := range flags {
		name := titles[f.Section]
		if name == "" {
			name = f.Section
		}
		out = append(out, name+" — "+strings.Join(f.Reasons, "; "))
	}
	return append(out, s.anchorSignalLines(ctx, userID, lec, content, titles)...)
}

// maxAnchorSignalLines bounds the block-level lines of a regeneration prompt.
const maxAnchorSignalLines = 6

// anchorSignalLines are the blocks the learner got stuck on ("섹션 제목 · 블록 첫 구절 — 이유"):
// a confident wrong prediction, a forgotten recall, repeated questions (research #15).
func (s *Service) anchorSignalLines(ctx context.Context, userID string, lec models.Lecture, c lecture.Content, titles map[string]string) []string {
	signals, err := s.Store.LectureAnchorSignals(ctx, userID, lec.ID)
	if err != nil {
		logf("anchor signals for %s: %v", lec.ID, err)
		return nil
	}
	texts := map[string]string{}
	expected := map[string]int64{}
	for _, sec := range c.Sections {
		for _, b := range sec.Blocks {
			texts[b.Anchor] = clipRunes(squash(lecture.ProseOnly(b.Text)), 40)
			expected[b.Anchor] = blockReadingMs(b)
		}
	}
	learning.SlowReads(signals, expected)
	var out []string
	for _, f := range learning.FlagAnchors(signals, s.Store.LocaleFor(ctx, userID)) {
		if len(out) == maxAnchorSignalLines {
			break
		}
		section := learning.SectionOfAnchor(f.Anchor)
		name := titles[section]
		if name == "" {
			name = section
		}
		if t := texts[f.Anchor]; t != "" {
			name += " · \"" + t + "\""
		}
		out = append(out, name+" — "+strings.Join(f.Reasons, "; "))
	}
	return out
}

// Self-graded blocks (lecture quality v2): no LLM call, the learner's tap is the grade.
const (
	// maxPredictionRunes bounds a written pause prediction.
	maxPredictionRunes = 300
	// recallReviewDelay: a recall prompt's objective comes back in the review a day later.
	recallReviewDelay = 24 * time.Hour
)

var validConfidence = map[string]bool{"": true, learning.ConfidenceSure: true, learning.ConfidenceUnsure: true, learning.ConfidenceUnknown: true}

// submitSelfGraded saves a pause prediction (with confidence and the
// learner's correct/incorrect after seeing the output), a recall grade, or a
// summary recall (whether one was written — the text is not kept). A recall
// also puts its question into the review bank and, unless the objective is
// already scheduled, its first review a day later (queued when the backlog is
// over the limit) — never mastery evidence by itself.
func (s *Service) submitSelfGraded(ctx context.Context, userID string, lec models.Lecture, blk exerciseBlock, a store.NewExerciseAttempt, in ExerciseInput) (models.ExerciseAttempt, error) {
	verdict := strings.TrimSpace(in.Verdict)
	if blk.kind == learning.AttemptSummary {
		verdict = learning.SummaryWritten
		if strings.TrimSpace(in.Answer) == "" {
			return models.ExerciseAttempt{}, invalidCheck(CodeAnswerEmpty, "answer is required")
		}
	}
	if !learning.ValidSelfGrade(blk.kind, verdict) {
		return models.ExerciseAttempt{}, invalidCheck(CodeInvalidVerdict, "verdict %q is not valid for %s", verdict, blk.kind)
	}
	if blk.kind == learning.AttemptPause {
		if !validConfidence[in.Confidence] {
			return models.ExerciseAttempt{}, invalidCheck(CodeInvalidVerdict, "confidence must be sure|unsure|unknown")
		}
		a.Confidence, a.Answer = in.Confidence, clipRunes(strings.TrimSpace(in.Answer), maxPredictionRunes)
	}
	a.Verdict, a.Status = verdict, store.AttemptDone
	saved, err := s.Store.InsertExerciseAttempt(ctx, a)
	if err != nil || blk.kind != learning.AttemptRecall {
		return saved, err
	}
	itemID, err := s.scheduleRecall(ctx, userID, lec, blk, in.Anchor)
	if err != nil {
		logf("recall review of %s %s: %v", lec.ID, in.Anchor, err) // 회상 기록은 남는다 — 복습 연결만 빠진다
		return saved, nil
	}
	if err := s.Store.SetAttemptReviewItem(ctx, userID, saved.ID, itemID); err != nil {
		logf("recall review item of attempt %s: %v", saved.ID, err)
		return saved, nil
	}
	saved.ReviewItemID = itemID
	return saved, nil
}

// scheduleRecall adds the recall prompt to the review bank of its objective
// (the block's, else the lecture's first) with the inflow control of a new check.
func (s *Service) scheduleRecall(ctx context.Context, userID string, lec models.Lecture, blk exerciseBlock, anchor string) (string, error) {
	objective := blk.objective
	if !slices.Contains(lec.ObjectiveIDs, objective) {
		if len(lec.ObjectiveIDs) == 0 {
			return "", fmt.Errorf("lecture has no objectives")
		}
		objective = lec.ObjectiveIDs[0]
	}
	question, key := lecture.RecallParts(blk.problem)
	itemID, _, err := s.Store.AddRecallReview(ctx, store.RecallReview{UserID: userID, LectureID: lec.ID, ConceptID: lec.ConceptID,
		ObjectiveID: objective, Anchor: anchor, Question: question, Key: key, Due: s.now().Add(recallReviewDelay),
		Hold: s.reviewHold(ctx, userID), PromptVersion: lec.PromptVersion})
	return itemID, err
}

// blockReadingMs is the reading time of one block (learning.SectionMinutes rates, in ms).
func blockReadingMs(b lecture.Block) int64 {
	m := lecture.MeasureContent(lecture.Content{Sections: []lecture.Section{{Blocks: []lecture.Block{b}}}})
	rate := learning.ProseCharsPerMinute
	if m.Language() == "en" {
		rate = learning.ProseCharsPerMinuteEn
	}
	minutes := float64(m.ProseChars)/rate + float64(m.CodeChars)/learning.CodeCharsPerMinute + float64(m.TraceRows)*learning.MinutesPerTraceRow
	return int64(minutes * 60_000)
}

// Reading signal bounds of one POST (the screen flushes about every 30 seconds).
const (
	maxReadingBatch     = 200
	maxReadingDwellMs   = 10 * 60_000
	maxReadingTermOpens = 50
)

// RecordReadingSignals adds the screen's per-block dwell times and term
// popover opens of a lecture the user can read (anchors must be the
// lecture's blocks; values are clipped).
func (s *Service) RecordReadingSignals(ctx context.Context, userID string, lec models.Lecture, batch []store.ReadingSignal) error {
	if len(batch) > maxReadingBatch {
		return invalidCheck(CodeInvalidVerdict, "at most %d signals per request", maxReadingBatch)
	}
	var c lecture.Content
	if err := json.Unmarshal(lec.Content, &c); err != nil {
		return fmt.Errorf("decode lecture content: %w", err)
	}
	anchors := map[string]bool{}
	for _, sec := range c.Sections {
		for _, b := range sec.Blocks {
			anchors[b.Anchor] = true
		}
	}
	var keep []store.ReadingSignal
	for _, r := range batch {
		if !anchors[r.Anchor] {
			continue // 강의가 바뀌었거나 블록이 아닌 자리
		}
		r.DwellMs = min(max(r.DwellMs, 0), maxReadingDwellMs)
		terms := map[string]int{}
		total := 0
		for id, n := range r.Terms {
			if n > 0 && id != "" && total < maxReadingTermOpens {
				n = min(n, maxReadingTermOpens-total)
				terms[id], total = n, total+n
			}
		}
		r.Terms = terms
		if r.DwellMs > 0 || len(terms) > 0 {
			keep = append(keep, r)
		}
	}
	if len(keep) == 0 {
		return nil
	}
	return s.Store.AddReadingSignals(ctx, userID, lec.ID, keep)
}
