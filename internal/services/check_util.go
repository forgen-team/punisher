package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/llm"
)

// runDetached runs a generation on the interactive lane, detached from the
// request. A panic is recovered (the server keeps running), logged with its
// stack and recorded by fail on the turn/job (nil = log only).
func runDetached(label, id string, timeout time.Duration, run func(context.Context) error, fail func(context.Context, error)) {
	go func() {
		ctx, cancel := context.WithTimeout(llm.WithPriority(context.Background(), llm.PriorityInteractive), timeout)
		defer cancel()
		defer recoverPanic(label, id, func(err error) {
			if fail != nil {
				fail(context.WithoutCancel(ctx), err)
			}
		})
		if err := run(ctx); err != nil {
			logf("%s %s: %v", label, id, err)
		}
	}()
}

// recoverPanic is deferred by background runs: a panic becomes a logged
// failure passed to fail instead of ending the process.
func recoverPanic(label, id string, fail func(error)) {
	r := recover()
	if r == nil {
		return
	}
	logf("%s %s: panic: %v\n%s", label, id, r, debug.Stack())
	if fail != nil {
		fail(fmt.Errorf("internal error: %v", r))
	}
}

// failTurn is the runDetached fail of a tutor / dispute turn.
func (s *Service) failTurn(userID, turnID string) func(context.Context, error) {
	return func(ctx context.Context, err error) {
		if ferr := s.Store.FailCheckTurn(ctx, userID, turnID, "", err.Error()); ferr != nil {
			logf("check turn %s: fail after panic: %v", turnID, ferr)
		}
	}
}

// failPreparing is the runDetached fail of an item generation: the job (if
// any) and the preparing session fail.
func (s *Service) failPreparing(userID, sessionID, jobID string) func(context.Context, error) {
	return func(ctx context.Context, err error) {
		if jobID != "" {
			if ferr := s.Store.FinishJob(ctx, jobID, usage{}.outcome("failed", "", err, "")); ferr != nil {
				logf("check %s: fail job after panic: %v", sessionID, ferr)
			}
		}
		if serr := s.Store.SetCheckSessionStatus(ctx, userID, sessionID, []string{"preparing"}, "failed", err.Error()); serr != nil && !errors.Is(serr, store.ErrCheckClosed) {
			logf("check %s: fail session after panic: %v", sessionID, serr)
		}
	}
}

// validID: session, turn and chapter ids are UUIDs (canonical form); anything
// else is not found — it never reaches SQL as a failing ::uuid cast (500).
func validID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, c := range id {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return false
			}
		case !strings.ContainsRune("0123456789abcdefABCDEF", c):
			return false
		}
	}
	return true
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

func validText(text string, max int) (string, bool) {
	text = strings.TrimSpace(text)
	return text, text != "" && utf8.RuneCountInString(text) <= max
}

func (s *Service) reviewIntervals(ctx context.Context, userID string) []int {
	st, err := s.Store.GetLearningSettings(ctx, userID)
	if err != nil {
		logf("review intervals for %s: %v (using defaults)", userID, err)
	}
	return st.ReviewIntervals
}

// labOS is the user's lab environment (macos | linux | none).
func (s *Service) labOS(ctx context.Context, userID string) string {
	st, err := s.Store.GetLearningSettings(ctx, userID)
	if err != nil {
		logf("lab os for %s: %v (using default)", userID, err)
		return learning.DefaultLabOS
	}
	return st.LabOS
}

// reviewSchedule is the user's ladder and whether adaptive intervals are on.
func (s *Service) reviewSchedule(ctx context.Context, userID string) ([]int, bool) {
	st, err := s.Store.GetLearningSettings(ctx, userID)
	if err != nil {
		logf("review schedule for %s: %v (using defaults)", userID, err)
		return learning.DefaultReviewIntervals, learning.DefaultAdaptiveIntervals
	}
	return st.ReviewIntervals, st.AdaptiveIntervals
}

func (s *Service) objectiveText(ctx context.Context, userID, conceptID, objectiveID string) string {
	if objectiveID == "" {
		return ""
	}
	info, err := s.Store.ConceptInfo(ctx, userID, conceptID)
	if err != nil {
		logf("objective text %s: %v", objectiveID, err)
		return ""
	}
	for _, o := range info.Objectives {
		if o.ID == objectiveID {
			return o.Text
		}
	}
	return ""
}

func materialText(item store.CheckItem) string {
	var m struct {
		Code string `json:"code"`
		Lang string `json:"lang"`
	}
	if err := json.Unmarshal(item.Material, &m); err != nil || m.Code == "" {
		return ""
	}
	return "```" + m.Lang + "\n" + m.Code + "\n```"
}

func sortTurns(list []models.CheckTurn) []models.CheckTurn {
	sort.Slice(list, func(i, j int) bool { return list[i].Seq < list[j].Seq })
	return list
}
