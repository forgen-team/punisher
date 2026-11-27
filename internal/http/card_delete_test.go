package httpapi

import (
	"net/http"
	"testing"
)

// TestCardDeleteRestore: DELETE /v1/cards/{id} 는 표시만 하고(204) 목록에서 빠지며,
// POST /v1/cards/{id}/restore 가 되돌린다. 외부 카드도 같다 — 지운 뒤에는 externalId 로도 못 고친다.
func TestCardDeleteRestore(t *testing.T) {
	h := newCardRouter(t, nil)
	code, c := send(t, h, http.MethodPost, "/v1/cards", externalBody, nil)
	if code != http.StatusOK {
		t.Fatalf("post = %d", code)
	}
	if code, _ := send(t, h, http.MethodDelete, "/v1/cards/"+c.ID, "", nil); code != http.StatusNoContent {
		t.Fatalf("delete = %d", code)
	}
	if n := len(listCards(t, h, nil)); n != 0 {
		t.Fatalf("deleted card still listed: %d", n)
	}
	if code, _ := send(t, h, http.MethodDelete, "/v1/cards/"+c.ID, "", nil); code != http.StatusNotFound {
		t.Fatalf("delete twice = %d", code)
	}
	if code, _ := send(t, h, http.MethodPatch, "/v1/cards/external/atlas:01HCARD", `{"stale":true}`, nil); code != http.StatusNotFound {
		t.Fatalf("patch a deleted card = %d", code)
	}
	if code, _ := send(t, h, http.MethodPost, "/v1/cards/"+c.ID+"/grade", `{"rating":3}`, nil); code != http.StatusNotFound {
		t.Fatalf("grade a deleted card = %d", code)
	}
	code, back := send(t, h, http.MethodPost, "/v1/cards/"+c.ID+"/restore", "", nil)
	if code != http.StatusOK || back.ID != c.ID || back.Question != c.Question {
		t.Fatalf("restore = %d %+v", code, back)
	}
	if got := listCards(t, h, nil); len(got) != 1 || got[0].ID != c.ID {
		t.Fatalf("restored card listed: %+v", got)
	}
	if code, _ := send(t, h, http.MethodPost, "/v1/cards/nope/restore", "", nil); code != http.StatusNotFound {
		t.Fatalf("restore missing = %d", code)
	}
}
