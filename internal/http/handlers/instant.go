package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/wooo-jin/punisher/internal/services"
)

// InstantCreate handles POST /v1/lectures/instant.
// Body: {conceptId?, query?, depth?, autoStart?}. Responses:
//   - 200 status=existing (lectureId) | needs_selection (resolution) | already_learned
//   - 202 status=started|attached (jobId, streamUrl)
func (h *LearnHandler) InstantCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body services.InstantRequest
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, err.Error())
		return
	}
	body.Query = strings.TrimSpace(body.Query)
	if (body.ConceptID == "" && body.Query == "") || utf8.RuneCountInString(body.Query) > maxQueryRunes || body.Depth < 0 || body.Depth > 3 {
		writeError(w, r, http.StatusBadRequest, ErrorBadRequest, "conceptId or query (max 500 chars) is required; depth 0..3")
		return
	}
	if !h.requireOwner(w, r, userID) {
		return
	}
	resp, err := h.Svc.StartInstant(r.Context(), userID, body)
	if err != nil {
		writeServiceError(w, r, "instant lecture", err)
		return
	}
	status := http.StatusOK
	if resp.Status == services.InstantStarted || resp.Status == services.InstantAttached {
		status = http.StatusAccepted
	}
	writeJSON(w, status, resp)
}

// InstantStream handles GET /v1/lectures/instant/{jobId}/stream as SSE.
// Events: status, delta {text}, done {lectureId,status,flagged,issues}, error {code,message}.
// Late subscribers receive a replay; for unknown/finished jobs the final state
// is read from generation_jobs.
func (h *LearnHandler) InstantStream(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	jobID := chi.URLParam(r, "jobId")
	job, err := h.Store.GetJob(r.Context(), userID, jobID)
	if err != nil {
		writeServiceError(w, r, "instant stream", err)
		return
	}

	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	send := func(e services.Event) bool {
		data, err := json.Marshal(e.Data)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.Type, data); err != nil {
			return false
		}
		return rc.Flush() == nil
	}

	replay, ch, cancel, live := h.Svc.Hub.Subscribe(jobID)
	defer cancel()
	if !live {
		send(finalEvent(job.Status, job.LectureID, job.Reason, job.Error))
		return
	}
	for _, e := range replay {
		if !send(e) {
			return
		}
	}
	ping := time.NewTicker(15 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ping.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil || rc.Flush() != nil {
				return
			}
		case e, open := <-ch:
			if !open {
				return
			}
			if !send(e) {
				return
			}
		}
	}
}

func finalEvent(status, lectureID, reason, errMsg string) services.Event {
	switch status {
	case "succeeded":
		return services.Event{Type: services.EventDone, Data: map[string]any{"lectureId": lectureID, "status": reason}}
	case "failed", "skipped":
		return services.Event{Type: services.EventError, Data: map[string]string{"code": status, "message": errMsg}}
	}
	return services.Event{Type: services.EventStatus, Data: map[string]string{"stage": status}}
}

// JobStatus handles GET /v1/jobs/{jobId}.
func (h *LearnHandler) JobStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	job, err := h.Store.GetJob(r.Context(), userID, chi.URLParam(r, "jobId"))
	if err != nil {
		writeServiceError(w, r, "job status", err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}
