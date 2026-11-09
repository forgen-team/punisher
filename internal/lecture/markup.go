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
//	@@sources
//	- [fork(2)](https://man7.org/linux/man-pages/man2/fork.2.html)
//	@@end

// ErrIncomplete means the markup ended before "@@end" (e.g. max_tokens).
var ErrIncomplete = errors.New("lecture markup is incomplete (missing @@end)")

var (
	directiveRe = regexp.MustCompile(`^@@([a-z_]+)(?:\s+(.*))?$`)
	sourceRe    = regexp.MustCompile(`^\s*[-*]\s*\[([^\]]+)\]\((\S+?)\)\s*$`)
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
	if b.Type == BlockSources {
		b.Sources = parseSources(body)
	} else if b.Type == BlockMarkdown || b.Type == BlockMisconception || b.Type == BlockTerm {
		b.Text = strings.TrimSpace(body)
	} else {
		b.Text = body
	}
	b.Anchor = fmt.Sprintf("%s-%d", p.section.Kind, len(p.section.Blocks)+1)
	p.section.Blocks = append(p.section.Blocks, b)
	p.block, p.body = nil, nil
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

// defaultSectionMinutes is the fixed 30-minute plan (3/12/10/5) used when the model omits a section's minutes.
var defaultSectionMinutes = map[string]int{
	"situation_question": 3,
	"principles":         12,
	"walkthrough":        10,
	"judgment_next":      5,
}

// parseAttrs parses `kind key=value flag ... title=rest of line`. The first
// bare token is stored under "_". title= and term= consume the rest of the line.
func parseAttrs(args string) map[string]string {
	attrs := map[string]string{}
	rest := strings.TrimSpace(args)
	for rest != "" {
		for _, greedy := range []string{"title=", "term="} {
			if strings.HasPrefix(rest, greedy) {
				attrs[strings.TrimSuffix(greedy, "=")] = strings.Trim(strings.TrimSpace(rest[len(greedy):]), `"`)
				return attrs
			}
		}
		token, tail, _ := strings.Cut(rest, " ")
		rest = strings.TrimSpace(tail)
		if key, value, ok := strings.Cut(token, "="); ok {
			attrs[key] = strings.Trim(value, `"`)
		} else {
			if _, seen := attrs["_"]; !seen {
				attrs["_"] = token
			}
			attrs[token] = ""
		}
	}
	return attrs
}

func parseSources(body string) []Source {
	out := []Source{}
	for _, line := range strings.Split(body, "\n") {
		if m := sourceRe.FindStringSubmatch(line); m != nil {
			out = append(out, Source{Title: strings.TrimSpace(m[1]), URL: m[2]})
		}
	}
	return out
}

func splitList(value string) []string {
	out := []string{}
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
