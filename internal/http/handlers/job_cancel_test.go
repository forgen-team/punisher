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
