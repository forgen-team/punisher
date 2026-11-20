package codexcli

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/forgen-team/punisher/internal/clicore"
)

// Result is a finished `codex exec --json` run.
type Result struct {
	Text         string
	InputTokens  int
	OutputTokens int
}

// Parser consumes `codex exec --json` JSONL events:
//
//	{"type":"thread.started","thread_id":"…"}
//	{"type":"turn.started"}
//	{"type":"item.started|item.updated|item.completed","item":{"id":"item_0","type":"agent_message","text":"…"}}
//	{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"…",…}}
//	{"type":"turn.completed","usage":{"input_tokens":…,"cached_input_tokens":…,"output_tokens":…}}
//	{"type":"turn.failed","error":{"message":"…"}}   {"type":"error","message":"…"}
//
// The answer is only the last completed agent_message: earlier ones are
// narration before a tool call ("I'll check the files…") and must not end
// up in a lecture or a JSON reply. Because an agent_message is known to be
// the last one only when the turn completes, OnDelta receives the answer
// once, at turn.completed (codex exec emits agent messages whole on
// item.completed anyway, so nothing streams earlier in practice).
type Parser struct {
	OnDelta func(string) error

	final    string // text of the last completed agent_message
	hasFinal bool
	done     bool
	failMsg  string // turn.failed: always a failure
	errMsg   string // error events: a failure only without turn.completed (they may be retries)
	usage    *usage
	// ToolItems are the non-message item types seen (command_execution,
	// web_search, error…), in order, once per item id.
	ToolItems []string
	seenItems map[string]bool
}

type item struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Text    string `json:"text"`
	Command string `json:"command"`
	Message string `json:"message"`
}

type event struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Item    *item  `json:"item"`
	Usage   *usage `json:"usage"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type usage struct {
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens"`
	OutputTokens      int `json:"output_tokens"`
}

// Feed handles one stdout line. Non-JSON lines are ignored. It returns
// clicore.ErrStop after the final event (turn.completed / turn.failed).
func (p *Parser) Feed(line []byte) error {
	line = bytes.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' {
		return nil
	}
	var e event
	if json.Unmarshal(line, &e) != nil {
		return nil
	}
	switch e.Type {
	case "item.started", "item.updated", "item.completed":
		if e.Item == nil {
			return nil
		}
		switch e.Item.Type {
		case "agent_message":
			if e.Type == "item.completed" {
				p.final, p.hasFinal = e.Item.Text, true
			}
		case "reasoning":
		default:
			p.toolItem(*e.Item)
		}
	case "turn.completed":
		p.done = true
		p.usage = e.Usage
		if p.OnDelta != nil && strings.TrimSpace(p.final) != "" {
			if err := p.OnDelta(p.final); err != nil {
				return err
			}
		}
		return clicore.ErrStop
	case "turn.failed":
		if e.Error != nil {
			p.failMsg = e.Error.Message
		}
		if p.failMsg == "" {
			p.failMsg = "codex turn failed"
		}
		return clicore.ErrStop
	case "error":
		if e.Message != "" {
			p.errMsg = e.Message
		}
	}
	return nil
}

// toolItem logs a non-message item once per id. The read-only sandbox
// still lets codex run (read-only) shell commands; Punisher does not fail
// the call but makes it visible.
func (p *Parser) toolItem(it item) {
	if p.seenItems == nil {
		p.seenItems = map[string]bool{}
	}
	if p.seenItems[it.ID] {
		return
	}
	p.seenItems[it.ID] = true
	p.ToolItems = append(p.ToolItems, it.Type)
	detail := it.Command
	if detail == "" {
		detail = it.Message
	}
	log.Printf("[codex] warning: run produced a non-message %q item (tool use or CLI error): %s", it.Type, clicore.Truncate(detail, 160))
}

// Finish returns the result. A run without turn.completed is an error (a
// truncated stream must never look successful); it is a fixed codex_error,
// never classified from stdout (which may be generated text) — Finish in
// clicore classifies stderr when the process failed.
func (p *Parser) Finish() (Result, error) {
	if p.failMsg != "" {
		return Result{}, codes.Classify(p.failMsg)
	}
	if !p.done {
		if p.errMsg != "" {
			return Result{}, codes.Classify(p.errMsg)
		}
		return Result{}, clicore.NewError(http.StatusBadGateway, CodeCLIError, "codex output ended without turn.completed")
	}
	if !p.hasFinal || strings.TrimSpace(p.final) == "" {
		return Result{}, clicore.NewError(http.StatusBadGateway, CodeCLIError, "codex returned no agent message")
	}
	out := Result{Text: p.final}
	if p.usage != nil {
		out.InputTokens, out.OutputTokens = p.usage.InputTokens, p.usage.OutputTokens
	}
	return out, nil
}
