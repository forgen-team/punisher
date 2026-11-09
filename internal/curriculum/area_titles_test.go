package curriculum

import "testing"

// 모든 영역에 표시 이름이 있어야 사용자 문구에 slug 가 새지 않는다.
func TestAreaTitlesCoverAllAreas(t *testing.T) {
	for _, a := range Areas {
		if AreaTitle(a) == a {
			t.Fatalf("area %q has no display title", a)
		}
	}
	if AreaTitle("unknown-area") != "unknown-area" {
		t.Fatal("unknown area must fall back to its slug")
	}
}
