package agycli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/forgen-team/punisher/internal/clicore"
)

// Result is a finished headless run.
type Result struct {
	Text         string
	Model        string
	InputTokens  int
	OutputTokens int
}

// envelope is the `--output-format json` object and the stream's result payload:
//
//	{"conversation_id":"…","status":"SUCCESS","response":"…","error":"…",
//	 "usage":{"input_tokens":…,"output_tokens":…,"thinking_tokens":…,"cache_read_tokens":…,"total_tokens":…}}
//
// input_tokens includes cache reads and output_tokens includes thinking
// (total_tokens = input + output in the documented examples).
type envelope struct {
	Status   string `json:"status"`
	Response string `json:"response"`
	Error    string `json:"error"`
	Usage    *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (e envelope) result(model string) (Result, error) {
	if !strings.EqualFold(e.Status, "SUCCESS") {
		msg := strings.TrimSpace(e.Error)
		if msg == "" {
			msg = "agy run ended with status " + e.Status
		}
		return Result{}, codes.Classify(msg)
	}
	out := Result{Text: e.Response, Model: model}
	if e.Usage != nil {
		out.InputTokens, out.OutputTokens = e.Usage.InputTokens, e.Usage.OutputTokens
	}
	return out, nil
}

// ParseJSON parses `agy -p --output-format json` stdout: the whole output
// as one envelope (it may be pretty-printed over several lines), else its
// last line (anything logged before the envelope is ignored). Output that
// is no envelope is a fixed agy_error — never classified for auth/rate
// markers, since it may be generated text (clicore.Finish reads stderr).
func ParseJSON(stdout []byte) (Result, error) {
	trimmed := bytes.TrimSpace(stdout)
	if len(trimmed) == 0 {
		return Result{}, clicore.NewError(http.StatusBadGateway, CodeCLIError, "empty output from agy")
	}
	candidates := [][]byte{trimmed}
	if i := bytes.LastIndexByte(trimmed, '\n'); i >= 0 {
		candidates = append(candidates, bytes.TrimSpace(trimmed[i+1:]))
	}
	for _, c := range candidates {
		var e envelope
		if json.Unmarshal(c, &e) == nil && e.Status != "" {
			return e.result("")
		}
	}
	return Result{}, clicore.NewError(http.StatusBadGateway, CodeCLIError, "agy output is not a JSON result envelope")
}

// StreamParser consumes `--output-format stream-json` NDJSON:
//
//	{"event":"init","init":{"model":"…",…}}
//	{"event":"step_update","step_update":{"step_type":"agent_response","state":"ACTIVE|DONE","text_delta":"…"}}
//	{"event":"result","result":{<envelope>}}
//
// Every agent_response text_delta (ACTIVE fragments and the final DONE) is
// incremental text.
type StreamParser struct {
	OnDelta func(string) error

	model  string
	done   bool
	result Result
	err    error
}

type streamLine struct {
	Event string `json:"event"`
	Init  *struct {
		Model string `json:"model"`
	} `json:"init"`
	Step *struct {
		StepType  string `json:"step_type"`
		TextDelta string `json:"text_delta"`
	} `json:"step_update"`
	Result *envelope `json:"result"`
}

// Feed handles one stdout line. Non-JSON lines are ignored.
func (p *StreamParser) Feed(line []byte) error {
	line = bytes.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' {
		return nil
	}
	var l streamLine
	if json.Unmarshal(line, &l) != nil {
		return nil
	}
	switch l.Event {
	case "init":
		if l.Init != nil {
			p.model = l.Init.Model
		}
	case "step_update":
		if l.Step != nil && l.Step.StepType == "agent_response" && l.Step.TextDelta != "" {
			return p.OnDelta(l.Step.TextDelta)
		}
	case "result":
		if l.Result == nil {
			return nil
		}
		p.done = true
		p.result, p.err = l.Result.result(p.model)
		return clicore.ErrStop // the result is the last event: stop reading, end the process group
	}
	return nil
}

// Finish returns the final result. A stream without a result event is a
// fixed agy_error (a truncated stream must never look successful, and its
// last line may be generated text, so it is not classified).
func (p *StreamParser) Finish() (Result, error) {
	if !p.done {
		return Result{}, clicore.NewError(http.StatusBadGateway, CodeCLIError, "agy stream ended without a result event")
	}
	return p.result, p.err
}
