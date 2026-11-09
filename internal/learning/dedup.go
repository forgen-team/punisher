package learning

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// MaxObjectivesPerLecture bounds one 30-minute lecture.
const MaxObjectivesPerLecture = 4

// ErrNothingToLearn means every objective at the requested depth is already
// completed/known — no lecture may be generated or recommended (design §3.4).
var ErrNothingToLearn = errors.New("all objectives at this depth are completed or known")

// ObjectiveRef is an active objective of a concept.
type ObjectiveRef struct {
	ID      string
	Depth   int
	Ordinal int
}

// DoneSet holds (objective, depth) pairs the user completed or marked known.
// Keys come from DoneKey. It is scoped to one user.
type DoneSet map[string]bool

// DoneKey is the dedup key for an objective at a depth. Together with the
// user and concept it forms (user, concept_id, objective_id, depth).
func DoneKey(objectiveID string, depth int) string {
	return fmt.Sprintf("%s@%d", objectiveID, depth)
}

// IsDone reports whether the objective is done at its depth.
func (d DoneSet) IsDone(o ObjectiveRef) bool { return d[DoneKey(o.ID, o.Depth)] }

// LecturePlan is what to generate/recommend for one concept.
type LecturePlan struct {
	ConceptID    string
	ObjectiveIDs []string
	ObjectiveKey string
	Depth        int
}

// PendingObjectives returns not-done objectives, ordered by depth then ordinal.
func PendingObjectives(objs []ObjectiveRef, done DoneSet) []ObjectiveRef {
	out := []ObjectiveRef{}
	for _, o := range objs {
		if !done.IsDone(o) {
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Depth != out[j].Depth {
			return out[i].Depth < out[j].Depth
		}
		return out[i].Ordinal < out[j].Ordinal
	})
	return out
}

// PlanLecture picks the objectives for the next lecture on a concept.
// depth 0 = the shallowest depth that still has a pending objective;
// depth N = only pending objectives at exactly N ("deeper lecture only if an
// unlearned objective/depth exists"). Done objectives are never included.
func PlanLecture(conceptID string, objs []ObjectiveRef, done DoneSet, depth int) (LecturePlan, error) {
	pending := PendingObjectives(objs, done)
	if len(pending) == 0 {
		return LecturePlan{}, ErrNothingToLearn
	}
	if depth == 0 {
		depth = pending[0].Depth
	}
	ids := []string{}
	for _, o := range pending {
		if o.Depth == depth && len(ids) < MaxObjectivesPerLecture {
			ids = append(ids, o.ID)
		}
	}
	if len(ids) == 0 {
		return LecturePlan{}, ErrNothingToLearn
	}
	return LecturePlan{ConceptID: conceptID, ObjectiveIDs: ids, ObjectiveKey: ObjectiveKey(ids), Depth: depth}, nil
}

// HasPending reports whether any objective is not done.
func HasPending(objs []ObjectiveRef, done DoneSet) bool {
	return len(PendingObjectives(objs, done)) > 0
}

// Generation job kinds.
const (
	JobLibrary = "library"
	JobInstant = "instant"
)

var kst = mustKST()

func mustKST() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return time.FixedZone("KST", 9*60*60)
	}
	return loc
}

// KSTDay returns the KST calendar day (YYYY-MM-DD) of t.
func KSTDay(t time.Time) string { return t.In(kst).Format("2006-01-02") }

// KSTDayStart returns 00:00 KST of t's KST day.
func KSTDayStart(t time.Time) time.Time {
	k := t.In(kst)
	return time.Date(k.Year(), k.Month(), k.Day(), 0, 0, 0, 0, kst)
}

// LibraryJobKey is the idempotency key for a library generation: one attempt
// per (objective set, depth) per KST day. It depends only on objective ids, so
// a concept reached via any alias/name yields the same key. Concurrent
// generation of the same objective set is additionally blocked by the
// generation_jobs partial unique index on active jobs.
func LibraryJobKey(plan LecturePlan, day string) string {
	return fmt.Sprintf("%s:%s@%d:%s", JobLibrary, plan.ObjectiveKey, plan.Depth, day)
}

// TargetKey identifies a generation target independent of job kind.
func TargetKey(plan LecturePlan) string {
	return fmt.Sprintf("%s@%d", plan.ObjectiveKey, plan.Depth)
}
