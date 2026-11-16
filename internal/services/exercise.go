package services

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

// ExerciseInput is one answer (or skip) to a block.
type ExerciseInput struct {
	Anchor      string `json:"anchor"`
	ContentHash string `json:"contentHash"`
	Answer      string `json:"answer"`
	Skip        bool   `json:"skip"`
}

// exerciseBlock is the located block: exercise (+ its answer) or lab.
type exerciseBlock struct {
	kind, section, problem, solution string
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
	conceptName := lec.ConceptID
	if info, err := s.Store.ConceptInfo(ctx, userID, lec.ConceptID); err == nil {
		conceptName = info.Name
	}
	runDetached("exercise", saved.ID, exerciseTimeout, func(bg context.Context) error {
		return s.gradeExercise(bg, apiKey, userID, lec.ConceptID, conceptName, saved.ID, blk, answer)
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

func (s *Service) gradeExercise(ctx context.Context, apiKey, userID, conceptID, conceptName, attemptID string, blk exerciseBlock, answer string) error {
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
	return learning.FlagSections(signals), nil
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
	return out
}
