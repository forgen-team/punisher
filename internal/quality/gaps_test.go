package quality

import (
	"testing"

	"github.com/forgen-team/punisher/internal/lecture"
)

func TestCompareGaps(t *testing.T) {
	c := lecture.Content{Sections: []lecture.Section{
		section(lecture.SectionPrinciples, md("시퀀스는 값을 하나씩   늘린다."), md("격리 수준은 설정이다.")),
		section(lecture.SectionWalkthrough, md("WAL 을 켠다.")),
	}}
	gaps := []ReaderGap{{Section: "principles", Quote: "시퀀스는 값을 하나씩 늘린다"}, {Quote: "본문에 없는 문장"}}
	signals := map[string]int{"principles-1": 2, "principles-2": 1, "walkthrough-1": 3, "nope": 9}
	a := CompareGaps(c, gaps, signals, 2)
	if a.GapBlocks != 1 || a.Unplaced != 1 || a.LearnerBlocks != 2 || a.Both != 1 || a.Precision != 1 || a.Missed != 0.5 {
		t.Fatalf("block agreement = %+v", a)
	}
	if a.SectionBoth != 1 || a.SectionMissed != 0.5 {
		t.Fatalf("section agreement = %+v", a)
	}
	if none := CompareGaps(c, nil, nil, 1); none.Precision != -1 || none.Missed != -1 {
		t.Fatalf("nothing to measure = %+v", none)
	}
}
