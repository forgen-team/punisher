// Package reminder sends the standalone app's daily review reminder as a macOS
// notification (osascript). It runs only in the standalone app on darwin: the
// Witems app mode has no notification path for apps (the shell shows the
// in-app badge instead) and Linux has no common notifier.
package reminder

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/store"
)

// Interval between checks.
const Interval = 5 * time.Minute

// Source is what the reminder reads and writes.
type Source interface {
	GetStudyPrefs(ctx context.Context, userID string) (store.StudyPrefs, error)
	MarkReminded(ctx context.Context, userID, day string) error
	ReviewCounts(ctx context.Context, userID string, now time.Time) (learning.ReviewCounts, error)
	CardCounts(ctx context.Context, userID string, now time.Time) (due, doneToday int, err error)
}

// Notify shows one notification.
type Notify func(ctx context.Context, title, body string) error

// ShouldRemind: reminders on, the KST hour reached, not sent today, not a rest
// day, and something is due. No pressure: one notice a day at most, never on rest days.
func ShouldRemind(p store.StudyPrefs, now time.Time, due int) bool {
	if p.ReminderHour < 0 || due <= 0 || learning.IsRestDay(now, p.RestDays) {
		return false
	}
	if p.LastRemindedDay == learning.KSTDay(now) {
		return false
	}
	return now.Sub(learning.KSTDayStart(now)) >= time.Duration(p.ReminderHour)*time.Hour
}

// Message is the notification text (Korean or English by locale).
func Message(locale string, objectives, cards int) (string, string) {
	if locale == learning.LocaleEN {
		return "punisher", fmt.Sprintf("%d reviews are waiting (about %d min).", objectives+cards, max(1, (objectives*2+cards+1)/2))
	}
	return "punisher", fmt.Sprintf("복습 %d개가 기다려요(약 %d분).", objectives+cards, max(1, (objectives*2+cards+1)/2))
}

// Run checks every Interval until ctx ends.
func Run(ctx context.Context, src Source, userID string, locale func(ctx context.Context) string, notify Notify) {
	tick := func() {
		now := time.Now()
		p, err := src.GetStudyPrefs(ctx, userID)
		if err != nil || p.ReminderHour < 0 {
			return
		}
		counts, err := src.ReviewCounts(ctx, userID, now)
		if err != nil {
			log.Printf("[reminder] review counts: %v", err)
			return
		}
		cards, _, err := src.CardCounts(ctx, userID, now)
		if err != nil {
			log.Printf("[reminder] card counts: %v", err)
			return
		}
		if !ShouldRemind(p, now, counts.Due+cards) {
			return
		}
		title, body := Message(locale(ctx), counts.Due, cards)
		if err := notify(ctx, title, body); err != nil {
			log.Printf("[reminder] notify: %v", err)
			return
		}
		if err := src.MarkReminded(ctx, userID, learning.KSTDay(now)); err != nil {
			log.Printf("[reminder] mark: %v", err)
		}
	}
	ticker := time.NewTicker(Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick()
		}
	}
}

// MacNotify shows a macOS notification with osascript (text passed as argv, not spliced into the script).
func MacNotify(ctx context.Context, title, body string) error {
	cmd := exec.CommandContext(ctx, "osascript",
		"-e", "on run argv", "-e", "display notification (item 2 of argv) with title (item 1 of argv)", "-e", "end run", title, body)
	return cmd.Run()
}
