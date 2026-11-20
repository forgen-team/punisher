package services

import (
	"context"
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

func setReaderLevel(t *testing.T, s *Service, user, level string) {
	t.Helper()
	if _, err := s.Store.UpdateLearningSettings(context.Background(), user, store.LearningSettingsPatch{ReaderLevel: &level}); err != nil {
		t.Fatal(err)
	}
}

// New lectures (instant and the owner's library ahead) and regenerations take
// the user's level; a stored lecture's level is kept when none is given.
func TestTargetReaderLevelFollowsSettings(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	conceptID := englishConcept(t, s, user)
	inst, err := s.instantTarget(ctx, user, conceptID, 0, "")
	if err != nil || inst.Plan.ReaderLevel != learning.ReaderPractitioner || inst.Prompt.Reader.Level != learning.ReaderPractitioner {
		t.Fatalf("unchosen level target: %q %q %v", inst.Plan.ReaderLevel, inst.Prompt.Reader.Level, err)
	}
	setReaderLevel(t, s, user, learning.ReaderBeginner)
	if inst, err = s.instantTarget(ctx, user, conceptID, 0, ""); err != nil || inst.Plan.ReaderLevel != learning.ReaderBeginner {
		t.Fatalf("beginner target: %q %v", inst.Plan.ReaderLevel, err)
	}
	if !strings.Contains(prompts.Lecture(inst.LectureContext(), "", "", inst.Prompts), prompts.ReaderProfile(learning.ReaderBeginner, learning.LocaleKO)) {
		t.Fatal("lecture prompt lacks the beginner section")
	}
	if !strings.HasSuffix(learning.TargetKey(inst.Plan), "~beginner") {
		t.Fatalf("hub target: %s", learning.TargetKey(inst.Plan))
	}
	lib, err := s.libraryTarget(ctx, user, conceptID)
	if err != nil || lib.Plan.ReaderLevel != learning.ReaderBeginner {
		t.Fatalf("library ahead uses the owner's level: %q %v", lib.Plan.ReaderLevel, err)
	}
	old := models.Lecture{ConceptID: conceptID, ObjectiveIDs: inst.Plan.ObjectiveIDs, Depth: inst.Plan.Depth, ReaderLevel: learning.ReaderAdvanced}
	if back, err := s.TargetForLecture(ctx, user, old, "", ""); err != nil || back.Plan.ReaderLevel != learning.ReaderAdvanced {
		t.Fatalf("stored lecture keeps its level: %q %v", back.Plan.ReaderLevel, err)
	}
	if back, err := s.TargetForLecture(ctx, user, old, "", learning.ReaderNovice); err != nil || back.Prompt.Reader.Level != learning.ReaderNovice {
		t.Fatalf("regeneration takes the given level: %q %v", back.Prompt.Reader.Level, err)
	}
}

// The prompt preview shows the reader section of the caller's level.
func TestPreviewShowsReaderLevel(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	setReaderLevel(t, s, user, learning.ReaderAdvanced)
	want := prompts.ReaderProfile(learning.ReaderAdvanced, learning.LocaleKO)
	for _, block := range []string{prompts.BlockLectureFormat, prompts.BlockReviewTask, prompts.BlockQASystem, prompts.BlockExplainSystem,
		prompts.BlockGlossSystem, prompts.BlockCheckItems, prompts.BlockCheckTutor, prompts.BlockCheckPretest} {
		p, err := s.PreviewPrompt(ctx, user, block, nil)
		if err != nil || !strings.Contains(p.User, want) {
			t.Errorf("%s preview lacks the advanced section (%v)", block, err)
		}
	}
}
