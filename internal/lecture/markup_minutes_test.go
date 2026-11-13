package lecture

import "testing"

// 실측(2026-11-06): 모델이 "@@section walkthrough title=... minutes=10" 순서로 써서 분량이 0, 제목이 오염됐다.
func TestSectionMinutesAfterTitleAndDefaults(t *testing.T) {
	p := &parser{}
	if err := p.openSection("walkthrough title=직접 확인하기: 프로세스, 커널, 종료 minutes=10"); err != nil {
		t.Fatal(err)
	}
	if p.section.Minutes != 10 || p.section.Title != "직접 확인하기: 프로세스, 커널, 종료" {
		t.Fatalf("got minutes=%d title=%q", p.section.Minutes, p.section.Title)
	}
	p2 := &parser{}
	if err := p2.openSection("principles title=원리"); err != nil {
		t.Fatal(err)
	}
	if p2.section.Minutes != 9 {
		t.Fatalf("default minutes not applied: %d", p2.section.Minutes)
	}
}
