package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Invalid study prefs are refused before the store is touched.
func TestPutStudyPrefsValidation(t *testing.T) {
	h := &StudyHandler{}
	for _, body := range []string{
		`{"memoryModel":"sm2"}`, `{"targetRetention":0.5}`, `{"restDays":[0,1,2,3,4,5,6]}`, `{"restDays":[8]}`, `{"reminderHour":24}`,
	} {
		req := httptest.NewRequest(http.MethodPut, "/v1/me/study-prefs", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.PutPrefs(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s → %d, want 400", body, rec.Code)
		}
	}
}
