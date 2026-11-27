package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/go-chi/chi/v5"
)

// StudyHandler serves the W5a study features: study prefs (memory model,
// target retention, rest days, reminder), repeated failures, review forecast,
// daily activity, goal deadlines, review cards and search.
type StudyHandler struct {
	Store store.Store
	Study store.StudyStore
	Svc   *services.Service
	// Now is the clock (tests); nil = time.Now.
	Now func() time.Time
}

func (h *StudyHandler) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func queryInt(r *http.Request, key string, def, lo, hi int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return min(hi, max(lo, n))
}

// GetPrefs handles GET /v1/me/study-prefs.
func (h *StudyHandler) GetPrefs(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	p, err := h.Study.GetStudyPrefs(r.Context(), userID)
	if err != nil {
		writeServiceError(w, r, "study prefs", err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// PutPrefs handles PUT /v1/me/study-prefs (partial).
func (h *StudyHandler) PutPrefs(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		MemoryModel     *string  `json:"memoryModel"`
		TargetRetention *float64 `json:"targetRetention"`
		RestDays        []int    `json:"restDays"`
		ReminderHour    *int     `json:"reminderHour"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	switch {
	case body.MemoryModel != nil && !learning.ValidMemoryModel(*body.MemoryModel):
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "memoryModel must be ladder or fsrs")
		return
	case body.TargetRetention != nil && !learning.ValidTargetRetention(*body.TargetRetention):
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "targetRetention must be 0.80~0.95")
		return
	case body.RestDays != nil && !learning.ValidRestDays(body.RestDays):
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "restDays must be distinct weekdays 0~6, at most 6")
		return
	case body.ReminderHour != nil && (*body.ReminderHour < -1 || *body.ReminderHour > 23):
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "reminderHour must be -1 (off) or 0~23")
		return
	}
	p, err := h.Study.UpdateStudyPrefs(r.Context(), userID, store.StudyPrefsPatch{
		MemoryModel: body.MemoryModel, TargetRetention: body.TargetRetention, RestDays: body.RestDays, ReminderHour: body.ReminderHour,
	})
	if err != nil {
		writeServiceError(w, r, "save study prefs", err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Leeches handles GET /v1/me/leeches — repeated failures and objectives on hold.
func (h *StudyHandler) Leeches(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	leeches, suspended, err := h.Study.Leeches(r.Context(), userID, h.Store.LocaleFor(r.Context(), userID))
	if err != nil {
		writeServiceError(w, r, "leeches", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"leeches": leeches, "suspended": suspended, "threshold": learning.LeechThreshold})
}

// SetLeech handles POST /v1/objectives/{id}/leech {action: keep|suspend|resume}.
func (h *StudyHandler) SetLeech(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	if body.Action != store.LeechKeep && body.Action != store.LeechSuspend && body.Action != store.LeechResume {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "action must be keep, suspend or resume")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.Study.SetLeech(r.Context(), userID, id, body.Action); err != nil {
		writeServiceError(w, r, "leech action", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"objectiveId": id, "action": body.Action})
}

// Forecast handles GET /v1/reviews/forecast?days=14.
func (h *StudyHandler) Forecast(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	days := queryInt(r, "days", 14, 1, 60)
	list, queued, err := h.Study.ReviewForecast(r.Context(), userID, h.now(), days)
	if err != nil {
		writeServiceError(w, r, "review forecast", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"days": list, "queued": queued})
}

// Activity handles GET /v1/me/activity?days=84 — per-day activity and today's goal.
func (h *StudyHandler) Activity(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	ctx, now := r.Context(), h.now()
	days := queryInt(r, "days", 84, 7, 371)
	list, err := h.Study.Activity(ctx, userID, now, days)
	if err != nil {
		writeServiceError(w, r, "activity", err)
		return
	}
	counts, err := h.Store.ReviewCounts(ctx, userID, now)
	if err != nil {
		writeServiceError(w, r, "activity", err)
		return
	}
	cardsDue, cardsDone, err := h.Study.CardCounts(ctx, userID, now)
	if err != nil {
		writeServiceError(w, r, "activity", err)
		return
	}
	prefs, err := h.Study.GetStudyPrefs(ctx, userID)
	if err != nil {
		writeServiceError(w, r, "activity", err)
		return
	}
	settings, err := h.Store.GetLearningSettings(ctx, userID)
	if err != nil {
		writeServiceError(w, r, "activity", err)
		return
	}
	today := list[len(list)-1]
	writeJSON(w, http.StatusOK, map[string]any{
		"days": list,
		"today": map[string]int{
			"reviewsDone": today.Reviews, "reviewsDue": counts.Due, "cardsDone": cardsDone, "cardsDue": cardsDue, "lecturesDone": today.Lectures,
		},
		"restDays": prefs.RestDays, "restToday": learning.IsRestDay(now, prefs.RestDays), "dailyMinutes": settings.DailyMinutes,
	})
}

// GoalPlan handles GET /v1/goals/{id}/plan — the deadline plan of a goal.
func (h *StudyHandler) GoalPlan(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	ctx, now := r.Context(), h.now()
	g, err := h.Store.GetGoal(ctx, userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "goal plan", err)
		return
	}
	remaining := map[string]bool{}
	for _, target := range g.TargetConceptIDs {
		steps, _, err := h.Svc.GoalPath(ctx, userID, target)
		if err != nil {
			writeServiceError(w, r, "goal plan", err)
			return
		}
		for _, s := range steps {
			if !learning.IsDone(s.State) {
				remaining[s.ConceptID] = true
			}
		}
	}
	due, err := h.Study.GoalDeadline(ctx, userID, g.ID)
	if err != nil {
		writeServiceError(w, r, "goal plan", err)
		return
	}
	settings, err := h.Store.GetLearningSettings(ctx, userID)
	if err != nil {
		writeServiceError(w, r, "goal plan", err)
		return
	}
	prefs, err := h.Study.GetStudyPrefs(ctx, userID)
	if err != nil {
		writeServiceError(w, r, "goal plan", err)
		return
	}
	writeJSON(w, http.StatusOK, learning.PlanDeadline(len(remaining), due, now, settings.DailyMinutes, prefs.RestDays))
}

// Deadlines handles GET /v1/goals/deadlines — every goal's due date.
func (h *StudyHandler) Deadlines(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	m, err := h.Study.GoalDeadlines(r.Context(), userID)
	if err != nil {
		writeServiceError(w, r, "goal deadlines", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deadlines": m})
}

// SetDeadline handles PUT /v1/goals/{id}/deadline {dueOn: "YYYY-MM-DD" | ""}.
func (h *StudyHandler) SetDeadline(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		DueOn string `json:"dueOn"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	body.DueOn = strings.TrimSpace(body.DueOn)
	if body.DueOn != "" && !learning.ValidDueOn(body.DueOn, h.now()) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "dueOn must be a date from today (YYYY-MM-DD)")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.Study.SetGoalDeadline(r.Context(), userID, id, body.DueOn); err != nil {
		writeServiceError(w, r, "goal deadline", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"goalId": id, "dueOn": body.DueOn})
}

func writeCardError(w http.ResponseWriter, r *http.Request, op string, err error) {
	if errors.Is(err, store.ErrInvalidCard) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "question and answer are required (question ≤ 500, answer ≤ 2000 chars)")
		return
	}
	writeServiceError(w, r, op, err)
}

// CardDraft handles GET /v1/cards/draft?kind=note|thread&sourceId= — the prefilled card.
func (h *StudyHandler) CardDraft(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	d, err := h.Study.CardDraftFor(r.Context(), userID, r.URL.Query().Get("kind"), r.URL.Query().Get("sourceId"))
	if err != nil {
		writeCardError(w, r, "card draft", err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// SaveCard handles POST /v1/cards {kind, sourceId, question, answer}. An
// external card (kind or sourceKind "external", sent by another app) is
// {externalId, sourceUrl, question|front, answer|back, stale?}; the same
// externalId again updates that card.
func (h *StudyHandler) SaveCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Kind       string `json:"kind"`
		SourceKind string `json:"sourceKind"`
		SourceID   string `json:"sourceId"`
		Question   string `json:"question"`
		Answer     string `json:"answer"`
		ExternalID string `json:"externalId"`
		SourceURL  string `json:"sourceUrl"`
		Front      string `json:"front"`
		Back       string `json:"back"`
		Stale      bool   `json:"stale"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	kind := firstNonEmpty(body.Kind, body.SourceKind)
	if kind == store.CardSourceExternal {
		c, err := h.Study.SaveExternalCard(r.Context(), userID, store.ExternalCard{ExternalID: body.ExternalID, SourceURL: body.SourceURL,
			Question: firstNonEmpty(body.Question, body.Front), Answer: firstNonEmpty(body.Answer, body.Back), Stale: body.Stale}, h.now())
		if err != nil {
			writeExternalCardError(w, r, "save external card", err)
			return
		}
		writeJSON(w, http.StatusOK, c)
		return
	}
	c, err := h.Study.SaveCard(r.Context(), userID, kind, body.SourceID, body.Question, body.Answer, h.now())
	if err != nil {
		writeCardError(w, r, "save card", err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// ListCards handles GET /v1/cards?due=1 (due=1: only due now, oldest first).
func (h *StudyHandler) ListCards(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var list []store.ReviewCard
	var err error
	if r.URL.Query().Get("due") == "1" {
		list, err = h.Study.DueCards(r.Context(), userID, h.now(), queryInt(r, "limit", 50, 1, 200))
	} else {
		list, err = h.Study.ListCards(r.Context(), userID)
	}
	if err != nil {
		writeServiceError(w, r, "list cards", err)
		return
	}
	if list == nil {
		list = []store.ReviewCard{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"cards": list})
}

// GradeCard handles POST /v1/cards/{id}/grade {rating: 1|2|3}.
func (h *StudyHandler) GradeCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Rating int `json:"rating"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	c, err := h.Study.GradeCard(r.Context(), userID, chi.URLParam(r, "id"), body.Rating, h.now())
	if err != nil {
		writeCardError(w, r, "grade card", err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func writeExternalCardError(w http.ResponseWriter, r *http.Request, op string, err error) {
	if errors.Is(err, store.ErrInvalidCard) {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest,
			"external card needs externalId (app:id), an http(s) sourceUrl or none, question ≤ 500 and answer ≤ 2000 chars; stale is for external cards only")
		return
	}
	writeServiceError(w, r, op, err)
}

// PatchCard handles PATCH /v1/cards/{id} and PATCH /v1/cards/external/{externalId}
// {question|front?, answer|back?, stale?} — absent fields are kept.
func (h *StudyHandler) PatchCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Question *string `json:"question"`
		Answer   *string `json:"answer"`
		Front    *string `json:"front"`
		Back     *string `json:"back"`
		Stale    *bool   `json:"stale"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	p := store.CardPatch{Question: body.Question, Answer: body.Answer, Stale: body.Stale}
	if p.Question == nil {
		p.Question = body.Front
	}
	if p.Answer == nil {
		p.Answer = body.Back
	}
	var c store.ReviewCard
	var err error
	if ext := chi.URLParam(r, "externalId"); ext != "" {
		// chi matches the raw path: an escaped id ("atlas%3A…") arrives escaped.
		if ext, err = url.PathUnescape(ext); err != nil {
			writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "invalid externalId")
			return
		}
		c, err = h.Study.PatchExternalCard(r.Context(), userID, ext, p, h.now())
	} else {
		c, err = h.Study.PatchCard(r.Context(), userID, chi.URLParam(r, "id"), p, h.now())
	}
	if err != nil {
		writeExternalCardError(w, r, "patch card", err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// DeleteCard handles DELETE /v1/cards/{id} — marks the card deleted (RestoreCard undoes it).
func (h *StudyHandler) DeleteCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	if err := h.Study.DeleteCard(r.Context(), userID, chi.URLParam(r, "id"), h.now()); err != nil {
		writeServiceError(w, r, "delete card", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RestoreCard handles POST /v1/cards/{id}/restore — brings a deleted card back with its schedule.
func (h *StudyHandler) RestoreCard(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	c, err := h.Study.RestoreCard(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, r, "restore card", err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// Search handles GET /v1/search?q=&limit=20.
func (h *StudyHandler) Search(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	if len([]rune(q)) > 100 {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "q max 100 chars")
		return
	}
	hits, err := h.Study.Search(r.Context(), userID, q, queryInt(r, "limit", 20, 1, 50))
	if err != nil {
		writeServiceError(w, r, "search", err)
		return
	}
	if hits == nil {
		hits = []store.SearchHit{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits})
}
