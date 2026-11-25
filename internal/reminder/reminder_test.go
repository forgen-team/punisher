package reminder

import (
	"strings"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/store"
)

func TestShouldRemind(t *testing.T) {
	kst := time.FixedZone("KST", 9*3600)
	wed9 := time.Date(2026, 11, 25, 9, 30, 0, 0, kst)
	p := store.DefaultStudyPrefs()
	if ShouldRemind(p, wed9, 5) {
		t.Fatal("off by default")
	}
	p.ReminderHour = 9
	if !ShouldRemind(p, wed9, 5) {
		t.Fatal("on at the hour with reviews due")
	}
	if ShouldRemind(p, wed9.Add(-time.Hour), 5) || ShouldRemind(p, wed9, 0) {
		t.Fatal("not before the hour, not with nothing due")
	}
	p.LastRemindedDay = learning.KSTDay(wed9)
	if ShouldRemind(p, wed9, 5) {
		t.Fatal("once a day")
	}
	p.LastRemindedDay, p.RestDays = "", []int{3}
	if ShouldRemind(p, wed9, 5) {
		t.Fatal("never on a rest day")
	}
}

func TestMessage(t *testing.T) {
	if _, body := Message("ko", 4, 2); !strings.Contains(body, "복습 6개") {
		t.Fatalf("ko body %q", body)
	}
	if _, body := Message("en", 1, 0); !strings.Contains(body, "1 reviews") && !strings.Contains(body, "1 review") {
		t.Fatalf("en body %q", body)
	}
}
