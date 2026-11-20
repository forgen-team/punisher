package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// New lectures take the user's locale; a stored lecture's target keeps its own
// language (review backfill), and its check items follow it.
func TestTargetLanguageFollowsLocale(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	var conceptID string
	if err := testDB(s).QueryRowContext(ctx, `
		SELECT c.id FROM concepts c WHERE c.status = 'active'
		  AND EXISTS (SELECT 1 FROM objectives o WHERE o.concept_id = c.id AND o.status = 'active')
		ORDER BY c.id LIMIT 1`).Scan(&conceptID); err != nil {
		t.Skipf("no concept: %v", err)
	}
	inst, err := s.instantTarget(ctx, user, conceptID, 0, "")
	if err != nil || inst.Plan.Language != learning.LocaleKO {
		t.Fatalf("default locale target: %q %v", inst.Plan.Language, err)
	}
	en := learning.LocaleEN
	if _, err := s.Store.UpdateLearningSettings(ctx, user, store.LearningSettingsPatch{Locale: &en}); err != nil {
		t.Fatal(err)
	}
	if inst, err = s.instantTarget(ctx, user, conceptID, 0, ""); err != nil || inst.Plan.Language != learning.LocaleEN {
		t.Fatalf("en user target: %q %v", inst.Plan.Language, err)
	}
	if learning.TargetKey(inst.Plan) != fmt.Sprintf("%s@%d|en", inst.Plan.ObjectiveKey, inst.Plan.Depth) {
		t.Fatalf("en hub target: %s", learning.TargetKey(inst.Plan))
	}
	ko := models.Lecture{ID: "00000000-0000-0000-0000-000000000001", ConceptID: conceptID,
		ObjectiveIDs: inst.Plan.ObjectiveIDs, Depth: inst.Plan.Depth, Language: learning.LocaleKO}
	back, err := s.TargetForLecture(ctx, user, ko, "", "")
	if err != nil || back.Plan.Language != learning.LocaleKO {
		t.Fatalf("stored lecture keeps its language: %q %v", back.Plan.Language, err)
	}
	if k := s.checkItemKey(ctx, user, ko); k.Language != learning.LocaleKO {
		t.Fatalf("check items follow the lecture: %q", k.Language)
	}
}
