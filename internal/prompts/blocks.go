package prompts

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
)

// Block ids of the user-editable instruction blocks. Dynamic context
// (concept, objectives, learned concepts, feedback, ...) is still assembled in
// code around these blocks.
const (
	BlockTeacherSystem  = "teacher.system"
	BlockOutlineTask    = "outline.task"
	BlockLectureFormat  = "lecture.format"
	BlockReviewTask     = "review.task"
	BlockReviewerSystem = "reviewer.system"
	BlockGlossSystem    = "gloss.system"
	BlockQASystem       = "qa.system"
	BlockExplainSystem  = "explain.system"
	BlockResolveSystem  = "resolve.system"
	BlockCheckItems     = "check.items"
	BlockCheckTutor     = "check.tutor"
	BlockCheckPretest   = "check.pretest"
)

// Var is a {{name}} placeholder a block may use.
type Var struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Block is one editable instruction block.
type Block struct {
	ID string `json:"id"`
	// Stage groups blocks in the UI, in pipeline order.
	Stage      string `json:"stage"`
	StageTitle string `json:"stageTitle"`
	Title      string `json:"title"`
	// Purpose: one line "이 단계가 하는 일".
	Purpose string `json:"purpose"`
	// AppliesTo: where a user override takes effect (shown in the UI).
	AppliesTo string `json:"appliesTo"`
	// Warning: shown before editing (e.g. format contract).
	Warning  string `json:"warning,omitempty"`
	Advanced bool   `json:"advanced"`
	Vars     []Var  `json:"vars"`
	Default  string `json:"default"`
}

// Stage ids in pipeline order.
const (
	StageOutline = "outline"
	StageLecture = "lecture"
	StageReview  = "review"
	StageGloss   = "gloss"
	StageQA      = "qa"
	StageExplain = "explain"
	StageResolve = "resolve"
	StageCheck   = "check"
	StagePretest = "pretest"
)

var (
	varConcept  = Var{Name: "concept_name", Description: "개념 이름(예: 페이지 캐시)"}
	varArea     = Var{Name: "concept_area", Description: "개념 영역 id(예: os)"}
	varDepth    = Var{Name: "depth", Description: "깊이 1(기본)~3(심화)"}
	varLanguage = Var{Name: "example_language", Description: "학습 설정의 예시 코드 언어(예: TypeScript)"}
	lectureVars = []Var{varConcept, varArea, varDepth, varLanguage}
)

const (
	appliesLecture   = "내 즉석 강의(더 깊게·직접 시작)와 '새 프롬프트로 다시 만들기'"
	appliesRegenOnly = "'새 프롬프트로 다시 만들기'(개요→본문→감수)에서만 쓰여요. 일반 즉석 강의는 개요 없이 본문을 바로 써서 이 블록을 쓰지 않아요"
	appliesReview    = "내 즉석 강의의 검수(통과해야 발행)와 '새 프롬프트로 다시 만들기'"
)

// Blocks is the registry, in UI order. Library (shared) lectures always use Default.
var Blocks = []Block{
	{ID: BlockOutlineTask, Stage: StageOutline, StageTitle: "개요 짜기", Title: "개요 설계 지시",
		Purpose:   "20분 강의의 섹션 4개와 핵심 정의·선행 계산·논지 점검·다룰 오해·출처를 JSON 개요로 먼저 설계합니다.",
		AppliesTo: appliesRegenOnly, Vars: lectureVars, Default: OutlineTask},
	{ID: BlockTeacherSystem, Stage: StageLecture, StageTitle: "본문 쓰기", Title: "강사 원칙(시스템)",
		Purpose:   "강사의 태도와 원칙 — 실무 상황에서 출발, 단계별 메커니즘, 공식 문서 근거, 이미 배운 것은 링크만.",
		AppliesTo: appliesLecture, Vars: lectureVars, Default: SystemTeacher},
	{ID: BlockLectureFormat, Stage: StageLecture, StageTitle: "본문 쓰기", Title: "본문 출력 형식",
		Purpose:   "강의 본문을 화면이 읽을 수 있는 @@ 마크업(섹션 4개·정의·손계산·예측·연습·요약 블록·출처)으로 쓰게 합니다.",
		AppliesTo: appliesLecture, Vars: lectureVars, Default: MarkupSpec,
		Warning: "형식 규칙입니다. @@ 지시어·섹션 이름·순서를 바꾸면 강의 저장이 실패하거나 화면이 깨질 수 있어요. 문장 표현만 고치는 것을 권합니다."},
	{ID: BlockReviewerSystem, Stage: StageReview, StageTitle: "감수", Title: "감수자 태도(시스템)",
		Purpose:   "강사와 다른 시니어 감수자 역할 — 칭찬 없이 문제만 찾습니다.",
		AppliesTo: appliesReview, Default: SystemReviewer},
	{ID: BlockReviewTask, Stage: StageReview, StageTitle: "감수", Title: "감수 기준",
		Purpose:   "초안의 사실 오류·구조·교수법·누락·과장·출처·용어 문제와 코드 출력 대조를 심각도와 함께 JSON으로 지적합니다.",
		AppliesTo: appliesReview, Vars: lectureVars, Default: ReviewTask,
		Warning: "JSON 출력 형식(verdict·issues)은 코드가 읽습니다. 형식 부분은 그대로 두세요."},
	{ID: BlockGlossSystem, Stage: StageGloss, StageTitle: "용어 설명", Title: "용어 팝오버",
		Purpose:   "본문 용어를 눌렀을 때 보이는 2~3문장 설명을 씁니다.",
		AppliesTo: "내가 보는 용어 설명(바꾸면 내 전용 캐시로 새로 만들어요)", Vars: []Var{varConcept}, Default: SystemGloss},
	{ID: BlockQASystem, Stage: StageQA, StageTitle: "질문 답변", Title: "조교 답변(시스템)",
		Purpose:   "강의 중 질문과 '설명 부족' 재설명에 답하고, 관련 개념을 연결합니다.",
		AppliesTo: "내 질문·재설명", Vars: []Var{varConcept, varLanguage}, Default: SystemQA,
		Warning: "답변은 JSON(answer·linked_concept_ids·proposed_concepts)으로 받습니다. '출력은 JSON 하나뿐' 문장은 남겨 두세요."},
	{ID: BlockExplainSystem, Stage: StageExplain, StageTitle: "구절 설명", Title: "선택 구절 설명(시스템)",
		Purpose:   "드래그로 고른 구절을 섹션 맥락에서 풀어 설명합니다.",
		AppliesTo: "내 '이 부분 설명'", Vars: []Var{varConcept, varLanguage}, Default: SystemExplain},
	{ID: BlockResolveSystem, Stage: StageResolve, StageTitle: "질문 분류(고급)", Title: "개념 분류기(시스템)",
		Purpose:   "자유 질문·학습 요청을 지식 지도의 개념 id로 연결합니다.",
		AppliesTo: "내 자유 질문·학습 요청의 개념 연결", Advanced: true, Default: SystemResolve,
		Warning: "분류 결과를 JSON으로 받습니다. 바꾸면 학습 요청이 엉뚱한 개념에 붙을 수 있어요."},
	{ID: BlockCheckItems, Stage: StageCheck, StageTitle: "확인", Title: "확인 문항 만들기(시스템)",
		Purpose:   "강의를 읽은 뒤 '확인하기'에서 풀 문항(목표마다 확인·복습 1개씩과 통합 1개)과 채점 기준을 만듭니다.",
		AppliesTo: "내 '확인하기' 문항(바꾸면 내 전용 문항으로 새로 만들어요)", Vars: []Var{varConcept, varLanguage}, Default: SystemCheckItems,
		Warning: "문항은 JSON(items·rubric)으로 받습니다. 키 이름과 '출력은 JSON 하나뿐' 문장은 남겨 두세요."},
	{ID: BlockCheckTutor, Stage: StageCheck, StageTitle: "확인", Title: "확인 튜터(시스템)",
		Purpose:   "'확인하기'에서 답 하나를 채점 기준으로 판정하고, 좁힌 질문·설명·반론·다음 문항 중 정해진 행동을 합니다.",
		AppliesTo: "내 '확인하기' 채점과 되묻기", Vars: []Var{varConcept}, Default: SystemCheckTutor,
		Warning: "판정은 JSON(verdict·confidence·tutor_move 등)으로 받습니다. 키 이름과 '출력은 JSON 하나뿐' 문장은 남겨 두세요."},
	{ID: BlockCheckPretest, Stage: StagePretest, StageTitle: "사전 진단", Title: "사전 진단 문항 만들기(시스템)",
		Purpose:   "강의 시작 때의 예측 질문, '이미 앎' 확인 문항, 새 챕터 배치 진단 문항을 개념·목표만으로 만듭니다.",
		AppliesTo: "내 예측 질문·'이미 앎' 확인·배치 진단 문항", Vars: []Var{varConcept, varLanguage}, Default: SystemCheckPretest,
		Warning: "문항은 JSON(items·material.choices·rubric.correct)으로 받습니다. 키 이름과 '출력은 JSON 하나뿐' 문장은 남겨 두세요."},
}

var blockIndex = func() map[string]Block {
	m := make(map[string]Block, len(Blocks))
	for _, b := range Blocks {
		m[b.ID] = b
	}
	return m
}()

// LookupBlock returns the registered block.
func LookupBlock(id string) (Block, bool) {
	b, ok := blockIndex[id]
	return b, ok
}

// Set resolves block texts: a user's overrides on top of the defaults.
// The zero value is the default set (used for library lectures).
type Set struct {
	overrides map[string]string
}

// Defaults is the code default set.
func Defaults() Set { return Set{} }

// NewSet keeps only overrides of registered blocks.
func NewSet(overrides map[string]string) Set {
	kept := map[string]string{}
	for id, body := range overrides {
		if _, ok := blockIndex[id]; ok {
			kept[id] = body
		}
	}
	return Set{overrides: kept}
}

// Overridden reports whether the user changed block id.
func (s Set) Overridden(id string) bool {
	_, ok := s.overrides[id]
	return ok
}

// Raw is the effective block body before variable substitution.
func (s Set) Raw(id string) string {
	if body, ok := s.overrides[id]; ok {
		return body
	}
	return blockIndex[id].Default
}

// Text is the effective block body with the block's declared {{vars}}
// substituted. Undeclared or missing placeholders are left as written.
func (s Set) Text(id string, vars map[string]string) string {
	return Substitute(s.Raw(id), blockIndex[id].Vars, vars)
}

// Substitute replaces {{name}} for each declared var present in values.
func Substitute(body string, declared []Var, values map[string]string) string {
	if !strings.Contains(body, "{{") {
		return body
	}
	for _, v := range declared {
		if val, ok := values[v.Name]; ok {
			body = strings.ReplaceAll(body, "{{"+v.Name+"}}", val)
		}
	}
	return body
}

// Version is the prompt version to record for a generation that used the
// given blocks: the code Version when none of them is overridden, otherwise
// Version+"+u:"+first 8 hex of sha256 over the effective (id, body) pairs.
func (s Set) Version(ids ...string) string {
	used := append([]string(nil), ids...)
	sort.Strings(used)
	overridden := false
	h := sha256.New()
	for _, id := range used {
		overridden = overridden || s.Overridden(id)
		h.Write([]byte(id))
		h.Write([]byte{0})
		h.Write([]byte(s.Raw(id)))
		h.Write([]byte{0})
	}
	if !overridden {
		return Version
	}
	return Version + "+u:" + hex.EncodeToString(h.Sum(nil))[:8]
}

// Hash is a short hash of the effective bodies of ids (cache keys).
func (s Set) Hash(ids ...string) string {
	v := s.Version(ids...)
	if i := strings.Index(v, "+u:"); i >= 0 {
		return v[i+3:]
	}
	return ""
}

// Blocks used per generation kind (for Version).
var (
	InstantBlocks = []string{BlockTeacherSystem, BlockLectureFormat, BlockReviewerSystem, BlockReviewTask}
	FullBlocks    = []string{BlockOutlineTask, BlockTeacherSystem, BlockLectureFormat, BlockReviewerSystem, BlockReviewTask}
)

// vars exposes lecture context values to block placeholders.
func (c LectureContext) vars() map[string]string {
	return map[string]string{
		varConcept.Name: c.ConceptName, varArea.Name: c.Area,
		varDepth.Name: itoa(c.Depth), varLanguage.Name: c.ExampleLanguage,
	}
}

// LectureVars is c.vars() for callers outside the package (system blocks).
func LectureVars(c LectureContext) map[string]string { return c.vars() }

func itoa(n int) string { return strconv.Itoa(n) }

// With returns a copy of s with block id set to body.
func (s Set) With(id, body string) Set {
	next := make(map[string]string, len(s.overrides)+1)
	for k, v := range s.overrides {
		next[k] = v
	}
	if _, ok := blockIndex[id]; ok {
		next[id] = body
	}
	return Set{overrides: next}
}

// UnknownVars lists {{name}} placeholders in body that the block does not declare.
func UnknownVars(body string, declared []Var) []string {
	known := map[string]bool{}
	for _, v := range declared {
		known[v.Name] = true
	}
	out := []string{}
	seen := map[string]bool{}
	for rest := body; ; {
		i := strings.Index(rest, "{{")
		if i < 0 {
			break
		}
		j := strings.Index(rest[i+2:], "}}")
		if j < 0 {
			break
		}
		name := strings.TrimSpace(rest[i+2 : i+2+j])
		if name != "" && !known[name] && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
		rest = rest[i+2+j+2:]
	}
	return out
}
