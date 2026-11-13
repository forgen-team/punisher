package check

import (
	"fmt"
	"strings"
)

// HeldConfidence: a grade below this confidence is held (판정 보류). The model's
// confidence is not calibrated; the held share is measured in the P1 review.
const HeldConfidence = 0.5

// TutorReply is the tutor's JSON output for one learner answer (design §4.2).
type TutorReply struct {
	Verdict       string   `json:"verdict"`
	Met           []string `json:"met"`
	Missing       []string `json:"missing"`
	Quote         string   `json:"quote"`
	Misconception string   `json:"misconception"`
	Confidence    float64  `json:"confidence"`
	TutorMove     string   `json:"tutor_move"`
	TutorText     string   `json:"tutor_text"`
}

// RegradeReply is the second model's JSON output for a dispute.
type RegradeReply struct {
	Verdict    string   `json:"verdict"`
	Met        []string `json:"met"`
	Missing    []string `json:"missing"`
	Quote      string   `json:"quote"`
	Confidence float64  `json:"confidence"`
}

var gradeVerdicts = map[string]bool{"correct": true, "partial": true, "incorrect": true}

// FinalVerdict maps a low-confidence grade to held.
func FinalVerdict(verdict string, confidence float64) string {
	if confidence < HeldConfidence {
		return "held"
	}
	return verdict
}

// GradeVerdict is FinalVerdict that also holds a correct grade without a quote
// from the answer (validated quotes only): "correct" needs its evidence.
func GradeVerdict(verdict, quote string, confidence float64) string {
	v := FinalVerdict(verdict, confidence)
	if v == "correct" && strings.TrimSpace(quote) == "" {
		return "held"
	}
	return v
}

// ValidateTutorReply checks a reply against the stage and returns it cleaned
// (praise stripped, a quote not found in the answer dropped, correct lowered to
// partial when met does not cover every must) with the problems to feed back
// on a rewrite. A held grade ends the item whatever move was written.
func ValidateTutorReply(r TutorReply, st Stage, must []string, rubricAnswer, learnerAnswer string) (TutorReply, []string) {
	var problems []string
	r.Verdict = strings.ToLower(strings.TrimSpace(r.Verdict))
	r.TutorMove = strings.ToLower(strings.TrimSpace(r.TutorMove))
	if !gradeVerdicts[r.Verdict] {
		problems = append(problems, "verdict 는 correct|partial|incorrect 중 하나입니다")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		problems = append(problems, "confidence 는 0~1 입니다")
	}
	r.Met, r.Missing = nonEmpty(r.Met), nonEmpty(r.Missing)
	r.Verdict = reconcileVerdict(r.Verdict, r.Met, must)
	r.Quote = strings.TrimSpace(r.Quote)
	if r.Quote != "" && !strings.Contains(learnerAnswer, r.Quote) {
		r.Quote = "" // 학습자 답에 없는 인용은 근거가 아니다.
	}
	r.TutorText = StripPraise(r.TutorText)
	allowed := st.AllowedMoves()
	held := GradeVerdict(r.Verdict, r.Quote, r.Confidence) == "held"
	switch {
	case !contains(allowed, r.TutorMove) && !held:
		problems = append(problems, fmt.Sprintf("tutor_move %q 는 허용되지 않습니다(허용: %s)", r.TutorMove, strings.Join(allowed, " | ")))
	case len(problems) == 0 && !held && r.TutorMove != st.ExpectedMove(r.Verdict):
		problems = append(problems, fmt.Sprintf("verdict %s 이면 tutor_move 는 %s 입니다(met 가 must 전부를 덮지 않으면 correct 가 아니라 partial 입니다)",
			r.Verdict, st.ExpectedMove(r.Verdict)))
	}
	if held {
		// 보류는 문항을 닫고 화면은 모범답과 '판정 보류'를 보인다(튜터 문장은 쓰지 않음).
		r.TutorMove = st.ExpectedMove("held")
		return r, problems
	}
	problems = append(problems, validateTutorText(r, rubricAnswer)...)
	return r, problems
}

func validateTutorText(r TutorReply, rubricAnswer string) []string {
	var problems []string
	if strings.TrimSpace(r.TutorText) == "" {
		problems = append(problems, "tutor_text 가 비었습니다(칭찬 상투어만 쓰지 마세요)")
	}
	if CountQuestions(r.TutorText) > 1 {
		problems = append(problems, "tutor_text 에는 물음이 하나만 있어야 합니다")
	}
	if leakChecked[r.TutorMove] && Leaks(r.TutorText, rubricAnswer) {
		problems = append(problems, fmt.Sprintf("%s 에서는 모범답을 말하면 안 됩니다", r.TutorMove))
	}
	return problems
}

// ValidateRegrade checks a regrade reply (quote kept only when it is in the
// answer; correct lowered to partial when met does not cover every must).
func ValidateRegrade(r RegradeReply, must []string, learnerAnswer string) (RegradeReply, []string) {
	var problems []string
	r.Verdict = strings.ToLower(strings.TrimSpace(r.Verdict))
	if !gradeVerdicts[r.Verdict] {
		problems = append(problems, "verdict 는 correct|partial|incorrect 중 하나입니다")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		problems = append(problems, "confidence 는 0~1 입니다")
	}
	r.Met, r.Missing = nonEmpty(r.Met), nonEmpty(r.Missing)
	r.Verdict = reconcileVerdict(r.Verdict, r.Met, must)
	if q := strings.TrimSpace(r.Quote); q == "" || !strings.Contains(learnerAnswer, q) {
		r.Quote = ""
	}
	return r, problems
}

// DisputeAccepted: the second model graded the answer correct with confidence.
func DisputeAccepted(r RegradeReply) bool {
	return GradeVerdict(r.Verdict, r.Quote, r.Confidence) == "correct"
}
