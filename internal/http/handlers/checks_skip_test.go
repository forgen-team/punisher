package handlers

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/learning"
	appMiddleware "github.com/forgen-team/punisher/internal/middleware"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/forgen-team/punisher/migrations"
	"github.com/go-chi/chi/v5"
)

// skipRouter: the migrated store with a two-item self-graded review of the local user's own concept.
func skipRouter(t *testing.T) (http.Handler, string, []string) {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "punisher.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx, migrations.Files); err != nil {
		t.Fatal(err)
	}
	user := appMiddleware.LocalUserID
	lectureID := store.NewID()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO concepts (id, name, area, depth, owner_user_id) VALUES ('test.skip', '건너뛰기', 'os', 1, ?)`, []any{user}},
		{`INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES ('test.skip#1', 'test.skip', '첫 목표', 1, 1)`, nil},
		{`INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES ('test.skip#2', 'test.skip', '둘째 목표', 1, 2)`, nil},
		{`INSERT INTO lectures (id, concept_id, objective_ids, objective_key, depth, kind, status, title, content, requested_by)
			VALUES (?, 'test.skip', '["test.skip#1","test.skip#2"]', 'test.skip#1,test.skip#2', 1, 'instant', 'published', '건너뛰기', '{}', ?)`,
			[]any{lectureID, user}},
	} {
		if _, err := st.DB().ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatal(err)
		}
	}
	rubric := check.Rubric{Must: []string{"기준"}, Answer: "모범답"}
	items, err := st.SaveCheckItems(ctx, store.CheckItemKey{LectureID: lectureID, Purpose: check.PurposeReview}, "test.skip", "", "m", "v",
		[]check.Item{
			{ObjectiveID: "test.skip#1", Purpose: check.PurposeReview, Format: check.FormatRecall, Prompt: "첫 질문", Rubric: rubric},
			{ObjectiveID: "test.skip#2", Purpose: check.PurposeReview, Format: check.FormatRecall, Prompt: "둘째 질문", Rubric: rubric},
		})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{items[0].ID, items[1].ID}
	sess, _, err := st.CreateCheckSession(ctx, store.NewCheckSession{UserID: user, Kind: check.PurposeReview, Grader: learning.GraderSelf,
		SessionKey: "review:skip"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.ActivateCheckSession(ctx, user, sess.ID, ids, store.TurnWrite{ItemID: ids[0], Role: "tutor", Move: check.MoveAsk, Text: "첫 질문"}); err != nil {
		t.Fatal(err)
	}
	learn := &LearnHandler{Store: st, Svc: &services.Service{Store: st, Keys: services.GatewayKeys{}, Hub: services.NewHub()}}
	r := chi.NewRouter()
	r.Post("/v1/checks/{id}/items/{itemId}/skip", learn.SkipCheckItem)
	return r, sess.ID, ids
}

// TestSkipCheckItemRoute: POST /v1/checks/{id}/items/{itemId}/skip skips the
// current item (200, next item asked); another item is 409, an unknown session 404.
func TestSkipCheckItemRoute(t *testing.T) {
	h, id, ids := skipRouter(t)
	call(t, h, http.MethodPost, "/v1/checks/"+id+"/items/"+ids[1]+"/skip", http.StatusConflict, nil)
	call(t, h, http.MethodPost, "/v1/checks/00000000-0000-0000-0000-000000000000/items/"+ids[0]+"/skip", http.StatusNotFound, nil)

	var d models.CheckDetail
	call(t, h, http.MethodPost, "/v1/checks/"+id+"/items/"+ids[0]+"/skip", http.StatusOK, &d)
	if d.Session.Cursor != 1 || d.Current == nil || d.Current.ItemID != ids[1] || !d.Current.CanSkip {
		t.Fatalf("after skip: %+v current=%+v", d.Session, d.Current)
	}
	if d.Items[0].Outcome != learning.OutcomeSkipped || d.Items[0].Answer != "" {
		t.Fatalf("skipped item = %+v", d.Items[0])
	}
	// 같은 문항을 다시 건너뛰면(이중 클릭) 409 — 지금 문항이 아니다.
	call(t, h, http.MethodPost, "/v1/checks/"+id+"/items/"+ids[0]+"/skip", http.StatusConflict, nil)
	call(t, h, http.MethodPost, "/v1/checks/"+id+"/items/"+ids[1]+"/skip", http.StatusOK, &d)
	if d.Session.Status != "finished" {
		t.Fatalf("skipping the last item finishes the session: %+v", d.Session)
	}
}
