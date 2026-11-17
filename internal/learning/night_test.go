package learning

import (
	"testing"
	"time"
)

func TestInKSTHours(t *testing.T) {
	at := func(h int) time.Time { return time.Date(2026, 11, 16, h, 30, 0, 0, kst) }
	cases := []struct {
		h, start, end int
		want          bool
	}{
		{2, 2, 6, true}, {5, 2, 6, true}, {6, 2, 6, false}, {1, 2, 6, false},
		{23, 23, 5, true}, {0, 23, 5, true}, {4, 23, 5, true}, {5, 23, 5, false}, {12, 23, 5, false},
		{3, 3, 3, false},
	}
	for _, c := range cases {
		if got := InKSTHours(at(c.h), c.start, c.end); got != c.want {
			t.Errorf("%02d:30 in [%d,%d) = %v, want %v", c.h, c.start, c.end, got, c.want)
		}
	}
	// UTC 18:30 = KST 03:30 (다음 날).
	if !InKSTHours(time.Date(2026, 11, 15, 18, 30, 0, 0, time.UTC), 2, 6) {
		t.Error("UTC time must be judged in KST")
	}
}
