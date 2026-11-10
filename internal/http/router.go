package httpapi

import (
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/wooo-jin/punisher/internal/http/handlers"
	"github.com/wooo-jin/punisher/internal/services"
	"github.com/wooo-jin/punisher/internal/store"
)

type RouterDependencies struct {
	Store store.Store
	// Services runs lecture generation, resolution and Q&A.
	Services *services.Service
	// Claude reports the local claude CLI status for the UI banner.
	Claude handlers.ClaudeStatusSource
	// UI serves the embedded web app for every non-API path (may be nil in tests).
	UI http.Handler
	// AppVersion and DataDir are shown on the settings screen.
	AppVersion string
	DataDir    string
	// AllowedHost and AllowedLogin are the Tailscale Serve name and its owner
	// allowed besides loopback ("" = none, the default; set by `punisher phone on`).
	AllowedHost  string
	AllowedLogin string
}

func NewRouter(deps RouterDependencies) http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(localOnly(deps.AllowedHost, deps.AllowedLogin))

	healthHandler := &handlers.HealthHandler{Store: deps.Store}
	statusHandler := &handlers.StatusHandler{Claude: deps.Claude, AppVersion: deps.AppVersion, DataDir: deps.DataDir}
	conceptsHandler := &handlers.ConceptsHandler{Store: deps.Store}
	progressHandler := &handlers.ProgressHandler{Store: deps.Store}
	studyRequestsHandler := &handlers.StudyRequestsHandler{Store: deps.Store, Svc: deps.Services}
	learn := &handlers.LearnHandler{Store: deps.Store, Svc: deps.Services}

	router.Route("/v1", func(r chi.Router) {
		r.Get("/health", healthHandler.Health)
		r.Get("/status", statusHandler.Status)

		r.Get("/concepts", conceptsHandler.List)
		r.Get("/concepts/{id}", conceptsHandler.Detail)
		r.Get("/concepts/{id}/lectures", learn.ConceptLectures)
		r.Get("/graph", conceptsHandler.Graph)

		r.Get("/me/frontier", conceptsHandler.Frontier)
		r.Put("/me/concepts/{id}/state", progressHandler.SetConceptState)
		r.Get("/me/questions", learn.ListMyQuestions)
		r.Get("/me/lectures", learn.ListMyLectures)
		r.Get("/me/settings", learn.GetSettings)
		r.Put("/me/settings", learn.PutSettings)
		r.Get("/me/prompts", learn.ListPrompts)
		r.Put("/me/prompts/{id}", learn.SavePrompt)
		r.Delete("/me/prompts/{id}", learn.ResetPrompt)
		r.Get("/me/prompts/{id}/history", learn.PromptHistory)
		r.Post("/me/prompts/{id}/restore", learn.RestorePrompt)
		r.Post("/me/prompts/{id}/preview", learn.PreviewPrompt)

		r.Post("/study-requests", studyRequestsHandler.Create)
		r.Post("/study-requests/{id}/resolve", studyRequestsHandler.Resolve)
		r.Get("/study-requests", learn.ListStudyRequests)
		r.Get("/study-requests/{id}", learn.GetStudyRequest)
		r.Post("/study-requests/{id}/dismiss", learn.DismissStudyRequest)
		r.Delete("/study-requests/{id}/dismiss", learn.RestoreStudyRequest)

		r.Post("/inquiries", learn.CreateInquiry)
		r.Get("/inquiries", learn.ListInquiries)
		r.Get("/inquiries/{id}", learn.GetInquiry)
		r.Post("/inquiries/{id}/regenerate", learn.RegenerateInquiry)
		r.Post("/inquiries/{id}/read", learn.MarkInquiryRead)
		r.Delete("/inquiries/{id}/read", learn.MarkInquiryRead)
		r.Post("/inquiries/{id}/followups", learn.CreateFollowup)
		r.Post("/inquiries/{id}/followups/{followupId}/retry", learn.RetryFollowup)
		r.Get("/me/completed", learn.ListCompleted)
		r.Get("/me/completed/areas", learn.CompletedAreas)

		r.Get("/today", learn.Today)
		r.Post("/concepts/{id}/known", learn.MarkKnown)
		r.Get("/terms/{conceptId}/gloss", learn.Gloss)

		r.Post("/lectures/instant", learn.InstantCreate)
		r.Get("/lectures/instant/{jobId}/stream", learn.InstantStream)
		r.Get("/jobs/{jobId}", learn.JobStatus)
		r.Get("/lectures/{id}", learn.GetLecture)
		r.Put("/lectures/{id}/progress", learn.Progress)
		r.Post("/lectures/{id}/complete", learn.Complete)
		r.Post("/lectures/{id}/feedback", learn.Feedback)
		r.Post("/lectures/{id}/dismiss", learn.Dismiss)
		r.Delete("/lectures/{id}/dismiss", learn.Restore)
		r.Post("/lectures/{id}/explain", learn.Explain)
		r.Get("/lectures/{id}/links", learn.Links)
		r.Post("/lectures/{id}/regenerate", learn.Regenerate)
		r.Get("/lectures/{id}/questions", learn.ListQuestions)
		r.Post("/lectures/{id}/questions", learn.Ask)
		r.Get("/questions/{id}", learn.GetQuestion)

		r.Get("/notes", learn.ListNotes)
		r.Post("/notes", learn.CreateNote)
		r.Patch("/notes/{id}", learn.UpdateNote)
		r.Delete("/notes/{id}", learn.DeleteNote)

		r.Get("/chapters", learn.ListChapters)
		r.Get("/chapters/suggestions", learn.ChapterSuggestions)
		r.Post("/chapters/drafts", learn.CreateChapterDraft)
		r.Post("/chapters/drafts/{id}/apply", learn.ApplyChapterDraft)

		r.Get("/goals", learn.ListGoals)
		r.Post("/goals", learn.CreateGoal)
		r.Patch("/goals/{id}", learn.UpdateGoal)
		r.Delete("/goals/{id}", learn.DeleteGoal)
		r.Get("/goals/{id}/path", learn.GoalPath)
	})

	if deps.UI != nil {
		router.NotFound(deps.UI.ServeHTTP)
	}
	return router
}

// localOnly rejects requests that did not come from this machine's own UI or,
// when phone access is on, from Tailscale Serve for exactly that host and that
// owner. The server always binds 127.0.0.1; this check is the only thing
// between a proxied request and an app that has no login, so it fails closed:
//   - Funnel (public internet) traffic is refused whatever its Host says;
//   - the Host must be loopback or exactly allowedHost (blocks DNS rebinding);
//   - a loopback Host must carry no proxy headers: a proxied client picks its
//     own Host, so a loopback Host proves nothing about where it came from;
//   - allowedHost requires Tailscale-User-Login to be the owner. Serve sets it
//     in Q-encoded form (and strips clients' copies), so other tailnet users
//     and tagged devices do not pass;
//   - a browser Origin must match the host's own origin (blocks other web
//     pages driving generation).
func localOnly(allowedHost, allowedLogin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reason := refuseReason(r, allowedHost, allowedLogin); reason != "" {
				http.Error(w, reason, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func refuseReason(r *http.Request, allowedHost, allowedLogin string) string {
	if r.Header.Get("Tailscale-Funnel-Request") != "" {
		return "forbidden: public access"
	}
	if isLoopbackHost(r.Host) {
		if looksProxied(r.Header) {
			return "forbidden host"
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || !isLoopbackHost(u.Host) {
				return "forbidden origin"
			}
		}
		return ""
	}
	if allowedHost == "" || (r.Host != allowedHost && r.Host != allowedHost+":443") {
		return "forbidden host"
	}
	if allowedLogin == "" || r.Header.Get("Tailscale-User-Login") != mime.QEncoding.Encode("utf-8", allowedLogin) {
		return "forbidden: not the owner via tailscale serve"
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != "https://"+allowedHost {
		return "forbidden origin"
	}
	return ""
}

func looksProxied(h http.Header) bool {
	for name := range h {
		if strings.HasPrefix(name, "Tailscale-") || strings.HasPrefix(name, "X-Forwarded-") {
			return true
		}
	}
	return h.Get("Forwarded") != "" || h.Get("X-Real-Ip") != "" || h.Get("Via") != ""
}

func isLoopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
