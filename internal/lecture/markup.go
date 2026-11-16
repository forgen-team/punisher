package lecture

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// The LLM writes lectures in a line-based markup instead of JSON: long Korean
// prose and code are error-prone to escape inside JSON, and the markup can be
// streamed and rendered progressively. Directive lines start with "@@word"
// ("@@ -1,2" diff hunks do not match). Format:
//
//	@@lecture
//	title: 프로세스는 무엇을 가지고 있나
//	objectives: os.process#1, os.process#2
//	new_scope: 이번에 새로 다루는 범위 한 줄 (여러 줄 가능)
//	prerequisites: os.kernel-and-syscall
//	@@section situation_question minutes=3 title=상황과 오늘의 질문
//	@@markdown
//	본문...
//	@@code lang=bash
//	ps -ef
//	@@term concept=os.kernel-and-syscall term=시스템 콜
//	짧은 설명
//	@@term concept=os.ebpf proposed term=eBPF
//	@@keypoint kind=definition
//	정식 정의 한 문장 + 예 + 비예(마크다운)
//	@@trace
//	{"caption":"…","columns":["단계","값"],"rows":[["1","…"],["2","…"]]}
//	@@pause
//	다음 출력에서 무엇이 보일까요?
//	@@exercise / @@answer / @@summary  (마크다운)
//	@@chart, @@diff lang=sql            (본문 JSON 하나 — structured.go)
//	@@sources
//	- [fork(2)](https://man7.org/linux/man-pages/man2/fork.2.html)
//	@@end

// ErrIncomplete means the markup ended before "@@end" (e.g. max_tokens).
var ErrIncomplete = errors.New("lecture markup is incomplete (missing @@end)")

var (
	directiveRe = regexp.MustCompile(`^@@([a-z_]+)(?:\s+(.*))?$`)
	// trailing text after the link (e.g. "— 이 문서가 근거하는 내용") is allowed and ignored.
	sourceRe = regexp.MustCompile(`^\s*[-*]\s*\[([^\]]+)\]\((\S+?)\)(?:\s.*)?$`)
)

type parser struct {
	content Content
	section *Section
	block   *Block
	body    []string
	inMeta  bool
	ended   bool
}

// ParseMarkup parses LLM lecture markup. conceptID and depth come from the
// generation plan (never trusted from model output).
func ParseMarkup(text, conceptID string, depth int) (Content, error) {
	p := &parser{content: Content{
		SchemaVersion: SchemaVersion, ConceptID: conceptID, Depth: depth,
		ObjectiveIDs: []string{}, NewScope: []string{}, PrerequisitesReferenced: []string{}, Sections: []Section{},
	}}
	text = strings.TrimPrefix(strings.TrimSpace(text), "\ufeff")
	for i, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if p.ended {
			break
		}
		if err := p.line(line); err != nil {
			return p.content, fmt.Errorf("line %d: %w", i+1, err)
		}
	}
	if !p.ended {
		return p.content, ErrIncomplete
	}
	p.content.EstimatedMinutes = p.content.TotalMinutes()
	return p.content, nil
}

func (p *parser) line(line string) error {
	m := directiveRe.FindStringSubmatch(strings.TrimRight(line, " \t"))
	if m == nil {
		if p.inMeta {
			return p.metaLine(line)
		}
		if p.block != nil {
			p.body = append(p.body, line)
			return nil
		}
		if strings.TrimSpace(line) == "" || p.section == nil {
			return nil // prose outside blocks is ignored
		}
		// Loose text right after a section header becomes a markdown block.
		p.openBlock(BlockMarkdown, nil)
		p.body = append(p.body, line)
		return nil
	}
	name, args := m[1], m[2]
	switch name {
	case "lecture":
		p.inMeta = true
		return nil
	case "section":
		return p.openSection(args)
	case "end":
		p.flushBlock()
		p.flushSection()
		p.ended = true
		return nil
	}
	if !IsBlockType(name) {
		return fmt.Errorf("unknown directive @@%s", name)
	}
	if p.section == nil {
		return fmt.Errorf("@@%s outside of a section", name)
	}
	p.openBlock(name, parseAttrs(args))
	return nil
}

func (p *parser) metaLine(line string) error {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return nil
	}
	value = strings.TrimSpace(value)
	switch strings.TrimSpace(strings.ToLower(key)) {
	case "title":
		p.content.Title = value
	case "objectives":
		p.content.ObjectiveIDs = append(p.content.ObjectiveIDs, splitList(value)...)
	case "new_scope":
		if value != "" {
			p.content.NewScope = append(p.content.NewScope, value)
		}
	case "prerequisites":
		p.content.PrerequisitesReferenced = append(p.content.PrerequisitesReferenced, splitList(value)...)
	}
	return nil
}

func (p *parser) openSection(args string) error {
	p.inMeta = false
	p.flushBlock()
	p.flushSection()
	attrs := parseAttrs(args)
	kind := attrs["_"]
	title := attrs["title"]
	raw := attrs["minutes"]
	if raw == "" {
		raw = attrs["min"]
	}
	// LLM이 title= 뒤에 분량을 붙이는 경우(title=... minutes=10)가 있다 — title= 이 줄 끝까지 먹으므로 떼어낸다.
	if m := trailingMinutesRe.FindStringSubmatch(title); m != nil {
		title = strings.TrimSpace(title[:len(title)-len(m[0])])
		if raw == "" {
			raw = m[1]
		}
	}
	minutes, err := strconv.Atoi(raw)
	if err != nil || minutes <= 0 {
		// 섹션 구조는 고정이므로 분량 표기가 빠지면 종류별 기본 분량을 쓴다.
		minutes = defaultSectionMinutes[kind]
	}
	p.section = &Section{Anchor: kind, Kind: kind, Title: title, Minutes: minutes, Blocks: []Block{}}
	return nil
}

func (p *parser) openBlock(kind string, attrs map[string]string) {
	p.flushBlock()
	b := Block{Type: kind}
	if attrs != nil {
		b.Lang = attrs["lang"]
		// "@@code bash" (lang= omitted) — accept the first bare token as the language.
		if b.Lang == "" && (kind == BlockCode || kind == BlockLog || kind == BlockDiff) {
			b.Lang = attrs["_"]
		}
		if kind == BlockKeypoint {
			b.Kind = attrs["kind"]
		}
		if kind == BlockLab {
			b.Kind = attrs["os"]
		}
		b.ConceptID = attrs["concept"]
		b.Term = attrs["term"]
		_, b.Proposed = attrs["proposed"]
	}
	p.block = &b
	p.body = nil
}

func (p *parser) flushBlock() {
	if p.block == nil || p.section == nil {
		p.block = nil
		return
	}
	b := *p.block
	body := strings.Trim(strings.Join(p.body, "\n"), "\n")
	switch {
	case b.Type == BlockSources:
		b.Sources = parseSources(body)
	case StructuredBlock(b.Type):
		// 본문은 JSON 하나. 해석되지 않으면 원문을 Text 에 남겨 검증이 재작성 지적으로 돌려준다.
		if data, ok := compactJSONObject(body); ok {
			b.Data = data
		} else {
			b.Text = strings.TrimSpace(body)
		}
	case proseBlocks[b.Type]:
		b.Text = strings.TrimSpace(body)
	default:
		b.Text = body
	}
	b.Anchor = fmt.Sprintf("%s-%d", p.section.Kind, len(p.section.Blocks)+1)
	p.section.Blocks = append(p.section.Blocks, b)
	p.block, p.body = nil, nil
}

// proseBlocks keep their text trimmed (markdown or a single line).
var proseBlocks = map[string]bool{
	BlockMarkdown: true, BlockMisconception: true, BlockTerm: true, BlockKeypoint: true,
	BlockPause: true, BlockExercise: true, BlockAnswer: true, BlockSummary: true, BlockLab: true,
}

func (p *parser) flushSection() {
	if p.section == nil {
		return
	}
	p.content.Sections = append(p.content.Sections, *p.section)
	p.section = nil
}

// trailingMinutesRe matches a `minutes=N` / `min=N` suffix the model sometimes appends after title=.
var trailingMinutesRe = regexp.MustCompile(`\s+(?:minutes|mins?)=(\d+)\s*$`)

// defaultSectionMinutes is the fixed 20-minute plan (TargetMinutes 2/9/6/3) used when the model omits a section's minutes.
var defaultSectionMinutes = TargetMinutes
