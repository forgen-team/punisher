package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/wooo-jin/punisher/internal/llm"
)

// resolveNowTimeout bounds one immediate resolution (alias match + one LLM call + pre-generation kick).
const resolveNowTimeout = 3 * time.Minute

// resolving tracks study requests whose concept resolution is running right now in this process,
// so the UI can show "개념 찾는 중" instead of an unexplained waiting state.
var resolving sync.Map // request id -> start time

// IsResolving reports whether the request's concept resolution is in flight.
func IsResolving(id string) bool {
	_, ok := resolving.Load(id)
	return ok
}

// ResolveRequestNow starts concept resolution for one request immediately, in the background,
// instead of waiting for the next scheduler tick (up to 30 minutes). Returns false when the
// request is already being resolved. The user is waiting on this, so it uses the interactive lane.
func (s *Service) ResolveRequestNow(userID, requestID string) bool {
	if _, loaded := resolving.LoadOrStore(requestID, time.Now()); loaded {
		return false
	}
	go func() {
		defer resolving.Delete(requestID)
		ctx, cancel := context.WithTimeout(llm.WithPriority(context.Background(), llm.PriorityInteractive), resolveNowTimeout)
		defer cancel()
		if err := s.resolveOne(ctx, userID, requestID); err != nil {
			log.Printf("[study-request] immediate resolve %s: %v", requestID, err)
		}
	}()
	return true
}

func (s *Service) resolveOne(ctx context.Context, userID, requestID string) error {
	req, err := s.Store.GetStudyRequest(ctx, userID, requestID)
	if err != nil {
		return err
	}
	if req.Status != "unresolved" {
		return nil
	}
	res, err := s.Resolve(ctx, userID, req.Query, func() (string, error) { return s.Keys.UserKey(ctx, userID) })
	msg := ""
	if err != nil {
		msg = err.Error()
	} else if len(res.ConceptIDs) == 0 {
		msg = "no matching concept"
	}
	if rerr := s.Store.RecordResolution(ctx, requestID, res.ConceptIDs, res.StartingConceptID, msg); rerr != nil {
		return rerr
	}
	if targets := studyPregenerateTargets(res); len(targets) > 0 {
		// 강의 미리 만들기는 사용자가 기다리는 일이 아니므로 background lane.
		bg := llm.WithBackground(context.WithoutCancel(ctx))
		if _, perr := s.PregenerateStudyStarts(bg, targets); perr != nil {
			log.Printf("[study-request] pregenerate %v: %v", targets, perr)
		}
	}
	return nil
}

// studyPregenerateTargets 는 학습 요청에서 미리 만들 강의 개념이다. 사용자가 고른 주제(첫 해석 개념)를
// 먼저, 선행 개념 시작점이 다르면 그 다음으로 둔다(2026-11-09: 'systemd' 요청이 선행 '데이터 표현'만 준비돼 혼란).
func studyPregenerateTargets(res Resolution) []string {
	out := make([]string, 0, 2)
	if len(res.ConceptIDs) > 0 {
		out = append(out, res.ConceptIDs[0])
	}
	if res.StartingConceptID != "" && (len(out) == 0 || res.StartingConceptID != out[0]) {
		out = append(out, res.StartingConceptID)
	}
	return out
}
