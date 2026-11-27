package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/forgen-team/punisher/internal/store"
)

// POST /v1/jobs/{id}/cancel: 200 with the cancelled job, 409 once it has
// ended, 404 for an unknown job.
func TestJobCancelRoute(t *testing.T) {
	f := newThreadsFixture(t)
	f.h.(interface {
		Post(string, http.HandlerFunc)
	}).Post("/v1/jobs/{jobId}/cancel", (&LearnHandler{Store: f.st, Svc: f.svc}).JobCancel)
	job, _, err := f.st.CreateJob(context.Background(), store.NewJob{IdempotencyKey: "cancel-route", Kind: "instant",
		UserID: "local", ObjectiveKey: "cancel-route", Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	send(t, f.h, http.MethodPost, "/v1/jobs/"+job.ID+"/cancel", "", http.StatusOK, &got)
	if got.Status != "failed" || got.Reason != store.JobReasonCancelled {
		t.Fatalf("cancelled job %+v", got)
	}
	send(t, f.h, http.MethodPost, "/v1/jobs/"+job.ID+"/cancel", "", http.StatusConflict, nil)
	send(t, f.h, http.MethodPost, "/v1/jobs/no-such-job/cancel", "", http.StatusNotFound, nil)
}

// POST /v1/checks/{id}/cancel: a preparing check session ends abandoned (200),
// then 409 (not preparing), 404 for an unknown session.
func TestCheckCancelRoute(t *testing.T) {
	f := newThreadsFixture(t)
	f.h.(interface {
		Post(string, http.HandlerFunc)
	}).Post("/v1/checks/{id}/cancel", (&LearnHandler{Store: f.st, Svc: f.svc}).CancelCheck)
	sess, _, err := f.st.CreateCheckSession(context.Background(), store.NewCheckSession{UserID: "local", Kind: "check",
		Grader: "llm", ConceptID: "test.thread-a", LectureID: f.lectureID, SessionKey: "lecture:" + f.lectureID})
	if err != nil {
		t.Fatal(err)
	}
	if sess.Status != "preparing" {
		t.Fatalf("new session %q", sess.Status)
	}
	var got struct {
		Session struct {
			Status string `json:"status"`
		} `json:"session"`
	}
	send(t, f.h, http.MethodPost, "/v1/checks/"+sess.ID+"/cancel", "", http.StatusOK, &got)
	if got.Session.Status != "abandoned" {
		t.Fatalf("cancelled session %q", got.Session.Status)
	}
	send(t, f.h, http.MethodPost, "/v1/checks/"+sess.ID+"/cancel", "", http.StatusConflict, nil)
	send(t, f.h, http.MethodPost, "/v1/checks/00000000-0000-0000-0000-000000000000/cancel", "", http.StatusNotFound, nil)
}
