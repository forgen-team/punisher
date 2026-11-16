package services

import "testing"

// The web computes the same hash (features/punisher/lecture-exercise/lib/contentHash.ts).
func TestBlockContentHashMatchesWeb(t *testing.T) {
	if got := BlockContentHash("4KB 페이지 3장은 몇 바이트?", "12288 바이트"); got != "c4ecadf8bc74" {
		t.Fatalf("hash = %s", got)
	}
}
