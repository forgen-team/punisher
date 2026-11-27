package httpapi

import (
	"github.com/forgen-team/punisher/internal/http/handlers"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/go-chi/chi/v5"
)

// registerStudyRoutes adds the W5a study-feature API under /v1 (a store
// without store.StudyStore — test fakes — gets none).
func registerStudyRoutes(r chi.Router, deps RouterDependencies) {
	st, ok := deps.Store.(store.StudyStore)
	if !ok {
		return
	}
	h := &handlers.StudyHandler{Store: deps.Store, Study: st, Svc: deps.Services}
	r.Get("/me/study-prefs", h.GetPrefs)
	r.Put("/me/study-prefs", h.PutPrefs)
	r.Get("/me/leeches", h.Leeches)
	r.Post("/objectives/{id}/leech", h.SetLeech)
	r.Get("/reviews/forecast", h.Forecast)
	r.Get("/me/activity", h.Activity)
	r.Get("/goals/deadlines", h.Deadlines)
	r.Get("/goals/{id}/plan", h.GoalPlan)
	r.Put("/goals/{id}/deadline", h.SetDeadline)
	r.Get("/cards/draft", h.CardDraft)
	r.Get("/cards", h.ListCards)
	r.Post("/cards", h.SaveCard)
	r.Post("/cards/{id}/grade", h.GradeCard)
	r.Patch("/cards/{id}", h.PatchCard)
	r.Patch("/cards/external/{externalId}", h.PatchCard)
	r.Delete("/cards/{id}", h.DeleteCard)
	r.Post("/cards/{id}/restore", h.RestoreCard)
	r.Get("/search", h.Search)
}
