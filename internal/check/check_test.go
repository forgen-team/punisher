package check

import (
	"encoding/json"
	"strings"
	"testing"
)

const answer = "페이지 캐시는 디스크 블록을 메모리에 남겨 두어 같은 파일을 다시 읽을 때 디스크 접근을 줄인다"

func goodItem(obj, purpose string) Item {
	format := FormatRecall
	if obj == "" {
		format = FormatIntegrative
	}
	return Item{ObjectiveID: obj, Purpose: purpose, Format: format,
		Prompt: "같은 파일을 두 번째 읽을 때 왜 빨라지는지 설명해 보세요.",
		Rubric: Rubric{Must: []string{"메모리에 남김", "디스크 접근 감소"}, Answer: answer, Anchor: "s1"}}
}

func goodSet() ItemSet {
	return ItemSet{Items: []Item{
		goodItem("c#1", PurposeCheck), goodItem("c#2", PurposeCheck),
		goodItem("c#1", PurposeReview), goodItem("c#2", PurposeReview),
		goodItem("", PurposeCheck),
	}}
}

var sc = SetContext{ObjectiveIDs: []string{"c#1", "c#2"}, Anchors: map[string]bool{"s1": true, "s1-b1": true}}

func TestValidateItemsAcceptsCompleteSet(t *testing.T) {
	if p := ValidateItems(goodSet(), sc); len(p) != 0 {
		t.Fatalf("problems: %v", p)
	}
	items := NormalizeItems(goodSet(), sc.ObjectiveIDs)
	last := items[len(items)-1]
	if strings.Join(last.Rubric.Covers, ",") != "c#1,c#2" || string(last.Material) != "{}" || len(items[0].Rubric.Covers) != 0 {
		t.Fatalf("normalize: covers=%v material=%s", last.Rubric.Covers, last.Material)
	}
}

func TestValidateItemsRejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ItemSet)
		want   string
	}{
		{"foreign objective", func(s *ItemSet) { s.Items[0].ObjectiveID = "other#1" }, "이 강의의 목표가 아닙니다"},
		{"missing review item", func(s *ItemSet) { s.Items = append(s.Items[:2], s.Items[3:]...) }, "review 문항은 정확히 1개"},
		{"two questions", func(s *ItemSet) { s.Items[0].Prompt = "왜 빨라지나요? 언제 느려지나요?" }, "물음이 하나만"},
		{"no must", func(s *ItemSet) { s.Items[0].Rubric.Must = []string{" "} }, "rubric.must"},
		{"five must", func(s *ItemSet) { s.Items[0].Rubric.Must = []string{"a", "b", "c", "d", "e"} }, "rubric.must"},
		{"unknown anchor", func(s *ItemSet) { s.Items[0].Rubric.Anchor = "nope" }, "앵커가 아닙니다"},
		{"prompt leaks answer", func(s *ItemSet) { s.Items[0].Prompt = answer + "에 대해 맞는지 답해 보세요." }, "모범답을 그대로"},
		{"no integrative", func(s *ItemSet) { s.Items = s.Items[:4] }, "통합 문항"},
		{"bad format", func(s *ItemSet) { s.Items[0].Format = "choice" }, "format"},
		{"material not object", func(s *ItemSet) { s.Items[0].Material = json.RawMessage(`"code"`) }, "material"},
		{"integrative with objective", func(s *ItemSet) { s.Items[4].ObjectiveID = "c#1" }, "통합 문항만"},
	}
	for _, c := range cases {
		set := goodSet()
		c.mutate(&set)
		p := ValidateItems(set, sc)
		if !strings.Contains(strings.Join(p, "\n"), c.want) {
			t.Errorf("%s: problems %v, want %q", c.name, p, c.want)
		}
	}
}

func TestStageMoves(t *testing.T) {
	cases := []struct {
		st      Stage
		verdict string
		want    string
		allowed string
	}{
		{Stage{Answers: 1}, "correct", MoveNext, "narrow,next"},
		{Stage{Answers: 1}, "partial", MoveNarrow, "narrow,next"},
		{Stage{Answers: 1}, "incorrect", MoveNarrow, "narrow,next"},
		{Stage{Answers: 1}, "held", MoveNext, "narrow,next"},
		{Stage{Answers: 1, Last: true}, "correct", MoveWrap, "narrow,wrap"},
		{Stage{Answers: 1, Integrative: true}, "correct", MoveChallenge, "challenge,narrow,next"},
		{Stage{Answers: 1, Integrative: true}, "partial", MoveNarrow, "challenge,narrow,next"},
		{Stage{Answers: 2}, "correct", MoveNext, "explain,next"},
		{Stage{Answers: 2}, "partial", MoveExplain, "explain,next"},
		{Stage{Answers: 3}, "incorrect", MoveNext, "next"},
		{Stage{Answers: 2, Challenged: true, Integrative: true, Last: true}, "incorrect", MoveWrap, "wrap"},
	}
	for _, c := range cases {
		if got := c.st.ExpectedMove(c.verdict); got != c.want {
			t.Errorf("%+v %s: move %s, want %s", c.st, c.verdict, got, c.want)
		}
		if got := strings.Join(c.st.AllowedMoves(), ","); got != c.allowed {
			t.Errorf("%+v: allowed %s, want %s", c.st, got, c.allowed)
		}
	}
	if !EndsItem(MoveNext) || !EndsItem(MoveWrap) || EndsItem(MoveNarrow) || EndsItem(MoveChallenge) {
		t.Fatal("only next/wrap end an item")
	}
}

func reply(verdict, move, text string) TutorReply {
	return TutorReply{Verdict: verdict, Confidence: 0.9, TutorMove: move, TutorText: text, Met: []string{"a"}}
}

func TestValidateTutorReply(t *testing.T) {
	learner := "메모리에 남겨서 빨라져요"
	cases := []struct {
		name string
		r    TutorReply
		st   Stage
		want string // "" = valid
	}{
		{"narrow ok", reply("partial", MoveNarrow, "디스크 대신 어디에서 읽게 될까요?"), Stage{Answers: 1}, ""},
		{"move not allowed", reply("partial", MoveExplain, "설명입니다."), Stage{Answers: 1}, "허용되지 않습니다"},
		{"move inconsistent with verdict", reply("correct", MoveNarrow, "다시 볼까요?"), Stage{Answers: 1}, "tutor_move 는 next"},
		{"two questions", reply("partial", MoveNarrow, "어디서 읽나요? 왜죠?"), Stage{Answers: 1}, "물음이 하나만"},
		{"narrow leaks answer", reply("partial", MoveNarrow, "힌트: "+answer+". 이제 답해 볼까요?"), Stage{Answers: 1}, "모범답을 말하면 안 됩니다"},
		{"explain may state the answer", reply("incorrect", MoveExplain, answer+". 그러면 쓰기는 어떨까요?"), Stage{Answers: 2}, ""},
		{"wrap may state the answer", reply("incorrect", MoveWrap, answer+"."), Stage{Answers: 3, Last: true}, ""},
		{"challenge leaks", reply("correct", MoveChallenge, answer+" 라는 주장에 반론해 보면?"), Stage{Answers: 1, Integrative: true}, "모범답을 말하면 안 됩니다"},
		{"only praise", reply("correct", MoveNext, "훌륭해요!"), Stage{Answers: 1}, "tutor_text 가 비었습니다"},
		{"bad verdict", reply("great", MoveNext, "다음으로"), Stage{Answers: 1}, "verdict"},
	}
	for _, c := range cases {
		_, p := ValidateTutorReply(c.r, c.st, answer, learner)
		joined := strings.Join(p, "\n")
		if (c.want == "") != (len(p) == 0) || (c.want != "" && !strings.Contains(joined, c.want)) {
			t.Errorf("%s: problems %v, want %q", c.name, p, c.want)
		}
	}
}

func TestTutorReplyCleaning(t *testing.T) {
	r := reply("correct", MoveNext, "좋은 답변이에요! 맞아요, 메모리에서 읽습니다.")
	r.Quote = "지어낸 인용"
	got, p := ValidateTutorReply(r, Stage{Answers: 1}, answer, "메모리에서 읽어요")
	if len(p) != 0 || got.TutorText != "맞아요, 메모리에서 읽습니다." || got.Quote != "" {
		t.Fatalf("cleaned = %+v problems=%v", got, p)
	}
	low := reply("correct", MoveNarrow, "")
	low.Confidence = 0.3
	got, p = ValidateTutorReply(low, Stage{Answers: 1}, answer, "x")
	if len(p) != 0 || got.TutorMove != MoveNext || FinalVerdict(got.Verdict, got.Confidence) != "held" {
		t.Fatalf("low confidence must be held and end the item: %+v %v", got, p)
	}
}

func TestLeakRatioNormalizes(t *testing.T) {
	if LeakRatio("캐시가 디스크를 대신한다", "") != 0 || LeakRatio("x", "두 단어") != 0 {
		t.Fatal("answers under three words never leak (no division by zero)")
	}
	a := "페이지 캐시는 메모리에 블록을 둔다"
	if r := LeakRatio("페이지 캐시가 메모리에 블록을 둔다고 하죠", a); r < 0.99 {
		t.Fatalf("particles must not hide a copy: ratio %.2f", r)
	}
	if Leaks("어디에서 다시 읽게 될까요?", a) {
		t.Fatal("an unrelated question does not leak")
	}
	if CountQuestions("왜？ 그리고?") != 2 {
		t.Fatal("full-width question marks count")
	}
}

func TestRegrade(t *testing.T) {
	r, p := ValidateRegrade(RegradeReply{Verdict: "Correct", Confidence: 0.8, Quote: "메모리"}, "메모리에서 읽어요")
	if len(p) != 0 || !DisputeAccepted(r) || r.Quote != "메모리" {
		t.Fatalf("regrade = %+v %v", r, p)
	}
	if DisputeAccepted(RegradeReply{Verdict: "correct", Confidence: 0.4}) || DisputeAccepted(RegradeReply{Verdict: "partial", Confidence: 0.9}) {
		t.Fatal("only a confident correct regrade is accepted")
	}
}
