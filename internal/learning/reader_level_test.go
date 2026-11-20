package learning

import "testing"

func TestReaderLevelValues(t *testing.T) {
	for _, l := range ReaderLevels {
		if !ValidReaderLevel(l) || EffectiveReaderLevel(l) != l {
			t.Errorf("%q must be valid and effective as itself", l)
		}
	}
	for _, l := range []string{"", "expert", "Beginner"} {
		if ValidReaderLevel(l) || EffectiveReaderLevel(l) != ReaderPractitioner {
			t.Errorf("%q must be invalid and generate as practitioner", l)
		}
	}
	if ReaderLevelSuffix("") != "" || ReaderLevelSuffix(ReaderPractitioner) != "" || ReaderLevelSuffix(ReaderBeginner) != "~beginner" {
		t.Errorf("suffix: practitioner keeps the old keys, others get ~level")
	}
	plan := LecturePlan{ObjectiveKey: "os.x#1", Depth: 1}
	if TargetKey(plan) != "os.x#1@1" {
		t.Errorf("practitioner target key must stay the pre-level key: %s", TargetKey(plan))
	}
	plan.ReaderLevel, plan.Language = ReaderAdvanced, LocaleEN
	if TargetKey(plan) != "os.x#1@1|en~advanced" || LibraryJobKey(plan, "2026-11-20") != "library:os.x#1@1|en~advanced:2026-11-20" {
		t.Errorf("level keys: %s / %s", TargetKey(plan), LibraryJobKey(plan, "2026-11-20"))
	}
}

func TestAreaFamiliarityValues(t *testing.T) {
	if !ValidAreaFamiliarity(nil) || !ValidAreaFamiliarity(map[string]string{}) {
		t.Error("empty familiarity is valid")
	}
	if !ValidAreaFamiliarity(map[string]string{"os": FamiliarityNew, "network-web": FamiliarityHeard, "database": FamiliarityCanExplain}) {
		t.Error("known areas and values are valid")
	}
	if ValidAreaFamiliarity(map[string]string{"cooking": FamiliarityNew}) || ValidAreaFamiliarity(map[string]string{"os": "expert"}) {
		t.Error("unknown area or value is invalid")
	}
}
