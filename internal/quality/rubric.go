// Package quality scores a lecture against the deterministic rubric of the
// lecture quality research (§4.1): reading time, interaction spacing,
// definition runs, wording, scene numbers, misconceptions, exercise kinds,
// sources, block lengths, recall prompts, and in-process checks of SQL
// examples and arithmetic. Every finding is advisory — it feeds a rewrite and
// the stored quality record, never blocks publishing (factual findings go to
// the reviewer as evidence instead). Nothing here runs an external command.
package quality

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/lecture"
)

// ReportVersion is bumped when rules or thresholds change (stored with the report).
const ReportVersion = 1

// Rule statuses.
const (
	Pass = "pass"
	Fail = "fail"
	NA   = "na" // the rule does not apply (e.g. no exercise kinds in an older lecture)
)

// Thresholds [실측: curriculum/samples 3편 기준선 → 추정 임계값, 샘플이 늘면 조정].
const (
	MinMinutes            = 15
	MaxMinutes            = 25
	MaxQuietProse         = 1800 // prose chars without an interaction block in between (×2 for English)
	MaxDefinitions        = 4    // definition keypoints in principles
	MaxDefinitionRun      = 2    // definitions in a row with no interaction between them
	MaxFirstInteraction   = 2000 // prose chars from the principles start to its first interaction
	MaxParagraphSentences = 5
	MinSceneNumbers       = 2
	MaxMisconceptions     = 2
	MaxWhyExercises       = 2
	MaxCodeLines          = 30
	MinSceneLogLines      = 6 // prompt: 8~10 lines (±2)
	MaxSceneLogLines      = 12
	MinRecallsPerSection  = 1
	MaxRecallsPerSection  = 2
)

// RuleResult is one rule's outcome on a lecture.
type RuleResult struct {
	ID     string  `json:"id"`
	Status string  `json:"status"`
	Value  float64 `json:"value"`
	Limit  string  `json:"limit,omitempty"`
	Detail string  `json:"detail,omitempty"`
	Path   string  `json:"path,omitempty"`
}

// Report is the deterministic rubric of one lecture.
type Report struct {
	Version    int           `json:"version"`
	Rules      []RuleResult  `json:"rules"`
	SQL        []SQLResult   `json:"sql,omitempty"`
	Arithmetic []ArithResult `json:"arithmetic,omitempty"`
}

// Context is what the rules check against.
type Context struct {
	// AllowedHosts: source hosts accepted besides OfficialHosts (the curriculum's doc_links hosts).
	AllowedHosts map[string]bool
	// SkipSQL turns the in-memory SQL run off (G21).
	SkipSQL bool
}

// ruleTitles are short Korean rule names (CLI table, revision notes).
var ruleTitles = map[string]string{
	"G07": "읽기 시간", "G08": "상호작용 없는 구간", "G09": "정의 연속·개수", "G10": "첫 상호작용 위치",
	"G11": "전제·과장 표현", "G12": "용어 링크 중복", "G13": "문단 문장 수", "G14": "장면 숫자 회수",
	"G15": "오개념 형식", "G16": "연습 종류 구성", "G17": "walkthrough 예측·수정", "G18": "출처 도메인",
	"G20": "코드·로그 길이", "G21": "SQL 실행 대조", "G22": "산술 재계산", "G23": "회상 프롬프트",
}

// RuleTitle is the Korean name of a rule id ("" when unknown).
func RuleTitle(id string) string { return ruleTitles[id] }

// Check runs every rule on c.
func Check(c lecture.Content, ctx Context) Report {
	r := Report{Version: ReportVersion}
	add := func(res RuleResult) { r.Rules = append(r.Rules, res) }
	add(ruleMinutes(c))
	scale := 1 // English needs about twice the letters for the same content (lecture.ProseBounds)
	if lecture.MeasureContent(c).Language() == "en" {
		scale = 2
	}
	add(ruleQuietProse(c, MaxQuietProse*scale))
	add(ruleDefinitions(c))
	add(ruleFirstInteraction(c, MaxFirstInteraction*scale))
	add(rulePhrases(c))
	add(ruleTermLinks(c))
	add(ruleParagraphs(c))
	add(ruleSceneNumbers(c))
	add(ruleMisconceptions(c))
	add(ruleExerciseKinds(c))
	add(ruleWalkthrough(c))
	add(ruleSources(c, ctx.AllowedHosts))
	add(ruleLengths(c))
	if !ctx.SkipSQL {
		r.SQL = CheckSQL(c)
	}
	add(ruleSQL(r.SQL))
	var checked int
	r.Arithmetic, checked = CheckArithmetic(c)
	add(ruleArithmetic(r.Arithmetic, checked))
	add(ruleRecalls(c))
	return r
}

// Rule returns the result of rule id (zero value when absent).
func (r Report) Rule(id string) RuleResult {
	for _, x := range r.Rules {
		if x.ID == id {
			return x
		}
	}
	return RuleResult{}
}

// Failed lists the failing rules.
func (r Report) Failed() []RuleResult {
	var out []RuleResult
	for _, x := range r.Rules {
		if x.Status == Fail {
			out = append(out, x)
		}
	}
	return out
}

// Issues are the failing rules as advisory issues for a rewrite (Path
// "quality.<id>"; Message = rule name, measured value, limit and detail).
func (r Report) Issues() []lecture.Issue { return r.IssuesFor(nil) }

// IssuesFor is Issues of the failing rules keep accepts (nil = all).
func (r Report) IssuesFor(keep map[string]bool) []lecture.Issue {
	var out []lecture.Issue
	for _, x := range r.Failed() {
		if keep != nil && !keep[x.ID] {
			continue
		}
		msg := fmt.Sprintf("%s: %s", RuleTitle(x.ID), x.Detail)
		if x.Limit != "" {
			msg += " (기준 " + x.Limit + ")"
		}
		path := "quality." + x.ID
		if x.Path != "" {
			path += " " + x.Path
		}
		out = append(out, lecture.Issue{Path: path, Message: msg, Advisory: true})
	}
	return out
}

func result(id string, ok bool, value float64, limit, detail string) RuleResult {
	status := Pass
	if !ok {
		status = Fail
	}
	return RuleResult{ID: id, Status: status, Value: value, Limit: limit, Detail: detail}
}

func na(id, detail string) RuleResult { return RuleResult{ID: id, Status: NA, Detail: detail} }

// G07: learning.EstimateMinutes in 15..25 (the product promises ~20 minutes).
func ruleMinutes(c lecture.Content) RuleResult {
	m := learning.EstimateMinutes(c)
	return result("G07", m >= MinMinutes && m <= MaxMinutes, float64(m), fmt.Sprintf("%d~%d분", MinMinutes, MaxMinutes),
		fmt.Sprintf("예상 읽기 %d분", m))
}

// interaction blocks break a stretch of reading.
var interaction = map[string]bool{
	lecture.BlockTrace: true, lecture.BlockPause: true, lecture.BlockExercise: true, lecture.BlockLab: true,
	lecture.BlockDiff: true, lecture.BlockChart: true, lecture.BlockRecall: true,
}

// proseChars is the reader prose of a block (code fences and code/log blocks excluded).
func proseChars(b lecture.Block) int {
	switch b.Type {
	case lecture.BlockMarkdown, lecture.BlockMisconception, lecture.BlockKeypoint, lecture.BlockAnswer,
		lecture.BlockSummary, lecture.BlockTerm:
		return visible(lecture.ProseOnly(b.Text))
	}
	return 0
}

func visible(s string) int {
	n := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n
}

// G08: the longest stretch of prose without an interaction block, across sections.
func ruleQuietProse(c lecture.Content, limit int) RuleResult {
	longest, run, where, start := 0, 0, "", ""
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			if interaction[b.Type] {
				run, start = 0, ""
				continue
			}
			if n := proseChars(b); n > 0 {
				if run == 0 {
					start = b.Anchor
				}
				run += n
				if run > longest {
					longest, where = run, start
				}
			}
		}
	}
	res := result("G08", longest <= limit, float64(longest), fmt.Sprintf("≤ %d자", limit),
		fmt.Sprintf("가장 긴 구간 %d자(%s부터) — 사이에 손계산·예측·연습·회상을 두세요", longest, where))
	res.Path = where
	return res
}

// G09: definition keypoints in principles: at most MaxDefinitions, and no more
// than MaxDefinitionRun in a row without an interaction block between them.
func ruleDefinitions(c lecture.Content) RuleResult {
	for _, s := range c.Sections {
		if s.Kind != lecture.SectionPrinciples {
			continue
		}
		total, run, longest := 0, 0, 0
		for _, b := range s.Blocks {
			switch {
			case b.Type == lecture.BlockKeypoint && b.Kind == lecture.KeypointDefinition:
				total++
				run++
				longest = max(longest, run)
			case interaction[b.Type]:
				run = 0
			}
		}
		ok := total <= MaxDefinitions && longest <= MaxDefinitionRun
		return result("G09", ok, float64(total), fmt.Sprintf("정의 ≤ %d개, 연속 ≤ %d개", MaxDefinitions, MaxDefinitionRun),
			fmt.Sprintf("정의 %d개, 상호작용 없이 연속 최대 %d개", total, longest))
	}
	return na("G09", "principles 없음")
}

// G10: prose chars from the principles start to its first trace/pause/exercise/recall.
func ruleFirstInteraction(c lecture.Content, limit int) RuleResult {
	for _, s := range c.Sections {
		if s.Kind != lecture.SectionPrinciples {
			continue
		}
		chars := 0
		for _, b := range s.Blocks {
			switch b.Type {
			case lecture.BlockTrace, lecture.BlockPause, lecture.BlockExercise, lecture.BlockRecall:
				return result("G10", chars <= limit, float64(chars), fmt.Sprintf("≤ %d자", limit),
					fmt.Sprintf("principles 시작 후 첫 손계산·연습까지 %d자", chars))
			}
			chars += proseChars(b)
		}
		return result("G10", false, float64(chars), fmt.Sprintf("≤ %d자", limit), "principles 에 손계산·연습이 없음")
	}
	return na("G10", "principles 없음")
}

var (
	presumptionRe = regexp.MustCompile(`당연히|알다시피|물론|익히 아는|잘 알려진|기본이므로|(?i)\b(?:of course|obviously|as you know|needless to say)\b`)
	overclaimRe   = regexp.MustCompile(`항상|절대로?|완벽하게|(?i)\b(?:always|never|perfectly)\b`)
	inlineCodeRe  = regexp.MustCompile("`[^`\n]*`")
)

// readerProse yields (anchor, prose) of every markdown-bodied block and pause
// question, code fences and inline code removed.
func readerProse(c lecture.Content, fn func(section string, b lecture.Block, prose string)) {
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			switch b.Type {
			case lecture.BlockMarkdown, lecture.BlockMisconception, lecture.BlockKeypoint, lecture.BlockExercise,
				lecture.BlockAnswer, lecture.BlockSummary, lecture.BlockPause, lecture.BlockRecall:
				fn(s.Kind, b, inlineCodeRe.ReplaceAllString(lecture.ProseOnly(b.Text), ""))
			}
		}
	}
}

// G11: presumption phrases (system prompt rule 12) and unconditional words (rule 8).
func rulePhrases(c lecture.Content) RuleResult {
	var found []string
	n := 0
	readerProse(c, func(_ string, b lecture.Block, prose string) {
		for _, re := range []*regexp.Regexp{presumptionRe, overclaimRe} {
			for _, m := range re.FindAllString(prose, -1) {
				n++
				if len(found) < 6 {
					found = append(found, fmt.Sprintf("%q(%s)", m, b.Anchor))
				}
			}
		}
	})
	return result("G11", n == 0, float64(n), "0개", "전제·과장 표현 "+fmt.Sprint(n)+"개: "+strings.Join(found, ", ")+
		" — 전제는 빼고, 단정은 조건을 붙이세요")
}

var termLinkRe = regexp.MustCompile(`\]\(term:([^)\s]+)\)`)

// G12: the same term: link after its first occurrence (the prompt links a term once).
func ruleTermLinks(c lecture.Content) RuleResult {
	seen := map[string]int{}
	var order []string
	readerProse(c, func(_ string, b lecture.Block, _ string) {
		for _, m := range termLinkRe.FindAllStringSubmatch(lecture.ProseOnly(b.Text), -1) {
			if seen[m[1]] == 0 {
				order = append(order, m[1])
			}
			seen[m[1]]++
		}
	})
	extra, dups := 0, []string{}
	for _, id := range order {
		if seen[id] > 1 {
			extra += seen[id] - 1
			dups = append(dups, fmt.Sprintf("%s×%d", id, seen[id]))
		}
	}
	return result("G12", extra == 0, float64(extra), "0개", fmt.Sprintf("처음 이후 다시 건 링크 %d개: %s", extra, strings.Join(dups, ", ")))
}

var (
	sentenceEndRe = regexp.MustCompile(`[.!?](?:["'”’)\]]*)(?:\s|$)`)
	abbrevRe      = regexp.MustCompile(`(?i)\b(?:e\.g|i\.e|vs|etc|cf)\.`)
	listLineRe    = regexp.MustCompile(`^\s*(?:[-*+>|#]|\d+[.)]\s)`)
)

// sentences approximates the sentence count of a paragraph (terminal . ! ?).
func sentences(p string) int {
	p = abbrevRe.ReplaceAllString(p, "")
	return len(sentenceEndRe.FindAllStringIndex(p, -1))
}

// G13: paragraphs (blank-line separated, lists/tables/headings excluded) with more than 5 sentences.
func ruleParagraphs(c lecture.Content) RuleResult {
	n, worst, where := 0, 0, ""
	readerProse(c, func(_ string, b lecture.Block, prose string) {
		if b.Type != lecture.BlockMarkdown && b.Type != lecture.BlockMisconception {
			return
		}
		for _, para := range strings.Split(prose, "\n\n") {
			var lines []string
			for _, line := range strings.Split(para, "\n") {
				if strings.TrimSpace(line) != "" && !listLineRe.MatchString(line) {
					lines = append(lines, line)
				}
			}
			k := sentences(strings.Join(lines, " "))
			if k > MaxParagraphSentences {
				n++
			}
			if k > worst {
				worst, where = k, b.Anchor
			}
		}
	})
	res := result("G13", n == 0, float64(n), fmt.Sprintf("문단당 ≤ %d문장", MaxParagraphSentences),
		fmt.Sprintf("%d문장 넘는 문단 %d개(가장 긴 문단 %d문장, %s) — 문단을 나누세요", MaxParagraphSentences, n, worst, where))
	if n > 0 {
		res.Path = where
	}
	return res
}

var sceneNumberRe = regexp.MustCompile(`\d[\d,]*(?:\.\d+)?`)

// sceneNumbers are the distinct number tokens of a text: thousands separators
// removed, kept when 3+ digits or decimal (2-digit and smaller numbers recur
// by chance), all-zero runs dropped.
func sceneNumbers(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range sceneNumberRe.FindAllString(text, -1) {
		tok := strings.TrimRight(strings.ReplaceAll(m, ",", ""), ".")
		digits := strings.ReplaceAll(tok, ".", "")
		if strings.Trim(digits, "0") == "" || (len(digits) < 3 && !strings.Contains(tok, ".")) || seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	return out
}

func sectionText(s lecture.Section) string {
	var b strings.Builder
	for _, blk := range s.Blocks {
		b.WriteString(blk.Text)
		b.WriteString("\n")
		if len(blk.Data) > 0 {
			b.Write(blk.Data)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// G14: at least two of the scene's numbers come back in judgment_next (the scene is closed).
func ruleSceneNumbers(c lecture.Content) RuleResult {
	var scene, judgment string
	for _, s := range c.Sections {
		switch s.Kind {
		case lecture.SectionSituation:
			scene = sectionText(s)
		case lecture.SectionJudgment:
			judgment = sectionText(s)
		}
	}
	nums := sceneNumbers(scene)
	if len(nums) < MinSceneNumbers {
		return na("G14", "장면에 회수할 숫자가 적음")
	}
	var back []string
	for _, n := range nums {
		if strings.Contains(strings.ReplaceAll(judgment, ",", ""), n) {
			back = append(back, n)
		}
	}
	return result("G14", len(back) >= MinSceneNumbers, float64(len(back)), fmt.Sprintf("≥ %d개", MinSceneNumbers),
		fmt.Sprintf("장면 숫자 %d개 중 judgment_next 에 다시 나온 것 %d개(%s)", len(nums), len(back), strings.Join(back, ", ")))
}

var digitOrCodeRe = regexp.MustCompile("[0-9]|`")

// G15: at most two misconceptions, each with ① ② ③ and a counterexample holding a number or code.
func ruleMisconceptions(c lecture.Content) RuleResult {
	n, bad := 0, []string{}
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			if b.Type != lecture.BlockMisconception {
				continue
			}
			n++
			i2, i3 := strings.Index(b.Text, "②"), strings.Index(b.Text, "③")
			switch {
			case !strings.Contains(b.Text, "①") || i2 < 0 || i3 < i2:
				bad = append(bad, b.Anchor+" ①②③ 없음")
			case !digitOrCodeRe.MatchString(b.Text[i2:i3]):
				bad = append(bad, b.Anchor+" 반례에 숫자·코드 없음")
			}
		}
	}
	if n == 0 {
		return na("G15", "오개념 없음")
	}
	return result("G15", n <= MaxMisconceptions && len(bad) == 0, float64(n), fmt.Sprintf("≤ %d개, ①②③ + 계산 가능한 반례", MaxMisconceptions),
		fmt.Sprintf("오개념 %d개 %s", n, strings.Join(bad, ", ")))
}

func exercises(c lecture.Content) []lecture.SectionBlock {
	var out []lecture.SectionBlock
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			if b.Type == lecture.BlockExercise {
				out = append(out, lecture.SectionBlock{Section: s.Kind, Block: b})
			}
		}
	}
	return out
}

// G16: with exercise kinds: the first is guided, the last error|free|modify, why ≤ 2.
func ruleExerciseKinds(c lecture.Content) RuleResult {
	list := exercises(c)
	kinds, why := []string{}, 0
	for _, e := range list {
		if e.Block.Kind != "" {
			kinds = append(kinds, e.Block.Kind)
		}
		if e.Block.Kind == lecture.ExerciseWhy {
			why++
		}
	}
	if len(kinds) == 0 {
		return na("G16", "연습 종류 표시 없음(이전 형식)")
	}
	first, last := list[0].Block.Kind, list[len(list)-1].Block.Kind
	var problems []string
	if first != lecture.ExerciseGuided {
		problems = append(problems, "첫 연습이 guided 가 아님("+first+")")
	}
	if last != lecture.ExerciseError && last != lecture.ExerciseFree && last != lecture.ExerciseModify {
		problems = append(problems, "마지막 연습이 error|free|modify 가 아님("+last+")")
	}
	if why > MaxWhyExercises {
		problems = append(problems, fmt.Sprintf("why %d개", why))
	}
	return result("G16", len(problems) == 0, float64(len(problems)), "첫 guided, 마지막 error|free|modify, why ≤ 2",
		"종류 "+strings.Join(kinds, "→")+" "+strings.Join(problems, ", "))
}

// G17: walkthrough has a pause and, after it, an exercise — a modify exercise when kinds are used.
func ruleWalkthrough(c lecture.Content) RuleResult {
	kinds := false
	for _, e := range exercises(c) {
		kinds = kinds || e.Block.Kind != ""
	}
	for _, s := range c.Sections {
		if s.Kind != lecture.SectionWalkthrough {
			continue
		}
		paused, after, modify := false, false, false
		for _, b := range s.Blocks {
			switch {
			case b.Type == lecture.BlockPause:
				paused = true
			case b.Type == lecture.BlockExercise && paused:
				after = true
				modify = modify || b.Kind == lecture.ExerciseModify
			}
		}
		ok := paused && after && (!kinds || modify)
		detail := fmt.Sprintf("예측 멈춤 %v, 그 뒤 연습 %v", yes(paused), yes(after))
		if kinds {
			detail += ", 수정(modify) 연습 " + yes(modify)
		}
		return result("G17", ok, boolValue(ok), "예측 → 출력 → (수정) 연습", detail)
	}
	return na("G17", "walkthrough 없음")
}

func yes(b bool) string {
	if b {
		return "있음"
	}
	return "없음"
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// G20: code blocks ≤ 30 lines; the scene's first log 6..12 lines (prompt: 8~10).
func ruleLengths(c lecture.Content) RuleResult {
	var problems []string
	sceneLog := -1
	for _, s := range c.Sections {
		for _, b := range s.Blocks {
			lines := len(strings.Split(strings.TrimRight(b.Text, "\n"), "\n"))
			if b.Type == lecture.BlockCode && lines > MaxCodeLines {
				problems = append(problems, fmt.Sprintf("%s 코드 %d줄", b.Anchor, lines))
			}
			if b.Type == lecture.BlockLog && s.Kind == lecture.SectionSituation && sceneLog < 0 {
				sceneLog = lines
			}
		}
	}
	if sceneLog >= 0 && (sceneLog < MinSceneLogLines || sceneLog > MaxSceneLogLines) {
		problems = append(problems, fmt.Sprintf("장면 로그 %d줄", sceneLog))
	}
	return result("G20", len(problems) == 0, float64(len(problems)), fmt.Sprintf("코드 ≤ %d줄, 장면 로그 %d~%d줄", MaxCodeLines, MinSceneLogLines, MaxSceneLogLines),
		strings.Join(problems, ", "))
}

// G23: principles and walkthrough each end with 1~2 recall prompts (only
// sources may follow them), each with a key; an objective, when given, must be the lecture's.
func ruleRecalls(c lecture.Content) RuleResult {
	objectives := map[string]bool{}
	for _, id := range c.ObjectiveIDs {
		objectives[id] = true
	}
	var problems []string
	total := 0
	for _, s := range c.Sections {
		if s.Kind != lecture.SectionPrinciples && s.Kind != lecture.SectionWalkthrough {
			continue
		}
		n, tail := 0, true
		for i := len(s.Blocks) - 1; i >= 0; i-- {
			b := s.Blocks[i]
			switch {
			case b.Type == lecture.BlockSources:
			case b.Type == lecture.BlockRecall:
				n++
				if !tail {
					problems = append(problems, b.Anchor+" 섹션 끝이 아님")
				}
				if b.Objective != "" && !objectives[b.Objective] {
					problems = append(problems, b.Anchor+" 목표 "+b.Objective+" 없음")
				}
			default:
				tail = false
			}
		}
		total += n
		if n < MinRecallsPerSection || n > MaxRecallsPerSection {
			problems = append(problems, fmt.Sprintf("%s 회상 %d개", s.Kind, n))
		}
	}
	return result("G23", len(problems) == 0, float64(total), fmt.Sprintf("principles·walkthrough 끝에 %d~%d개", MinRecallsPerSection, MaxRecallsPerSection),
		strings.Join(problems, ", "))
}
