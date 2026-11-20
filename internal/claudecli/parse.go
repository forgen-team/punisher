package claudecli

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"github.com/forgen-team/punisher/internal/clicore"
)

// Error codes returned to clients.
const (
	CodeUnauthorized = clicore.CodeUnauthorized
	CodeInvalid      = clicore.CodeInvalid
	CodeQueueFull    = clicore.CodeQueueFull
	CodeDailyCap     = clicore.CodeDailyCap
	CodeNotLoggedIn  = "claude_not_logged_in"
	CodeRateLimited  = "claude_rate_limited"
	CodeTimeout      = "claude_timeout"
	CodeCLIError     = "claude_error"
	CodeCLIMissing   = "claude_unavailable"
	CodeClientGone   = clicore.CodeClientGone
)

// APIError is an error with an HTTP status and a stable code.
type APIError = clicore.APIError

func apiErr(status int, code, msg string) *APIError { return clicore.NewError(status, code, msg) }

// Usage mirrors the CLI's token accounting.
type Usage struct {
	InputTokens              int `json:"inputTokens"`
	OutputTokens             int `json:"outputTokens"`
	CacheReadInputTokens     int `json:"cacheReadInputTokens,omitempty"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens,omitempty"`
}

// Result is a finished CLI call.
type Result struct {
	Text       string  `json:"text"`
	Model      string  `json:"model"`
	Usage      *Usage  `json:"usage,omitempty"`
	CostUSD    float64 `json:"costUsd,omitempty"`
	DurationMs int64   `json:"durationMs"`
}

// cliResult is the `type:"result"` message of --output-format json/stream-json.
type cliResult struct {
	Type       string          `json:"type"`
	Subtype    string          `json:"subtype"`
	IsError    bool            `json:"is_error"`
	Result     string          `json:"result"`
	TotalCost  float64         `json:"total_cost_usd"`
	Usage      *cliUsage       `json:"usage"`
	ModelUsage map[string]any  `json:"modelUsage"`
	Errors     json.RawMessage `json:"errors"`
}

type cliUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

func (r cliResult) toResult(fallbackModel string) (Result, error) {
	if r.IsError || (r.Subtype != "" && r.Subtype != "success") {
		msg := r.Result
		if msg == "" {
			msg = r.Subtype + " " + string(r.Errors)
		}
		return Result{}, classifyFailure(msg)
	}
	out := Result{Text: r.Result, Model: fallbackModel, CostUSD: r.TotalCost}
	if len(r.ModelUsage) > 0 {
		names := make([]string, 0, len(r.ModelUsage))
		for name := range r.ModelUsage {
			names = append(names, name)
		}
		sort.Strings(names)
		out.Model = names[0]
	}
	if r.Usage != nil {
		out.Usage = &Usage{InputTokens: r.Usage.InputTokens, OutputTokens: r.Usage.OutputTokens,
			CacheReadInputTokens: r.Usage.CacheReadInputTokens, CacheCreationInputTokens: r.Usage.CacheCreationInputTokens}
	}
	return out, nil
}

// ParseJSONOutput parses `claude -p --output-format json` stdout. Older and
// --verbose builds print an array of messages; the last result entry wins.
func ParseJSONOutput(stdout []byte, fallbackModel string) (Result, error) {
	trimmed := bytes.TrimSpace(stdout)
	if len(trimmed) == 0 {
		return Result{}, classifyFailure("empty output from claude")
	}
	if trimmed[0] == '[' {
		var msgs []json.RawMessage
		if err := json.Unmarshal(trimmed, &msgs); err != nil {
			return Result{}, classifyFailure(string(trimmed))
		}
		for i := len(msgs) - 1; i >= 0; i-- {
			var r cliResult
			if json.Unmarshal(msgs[i], &r) == nil && r.Type == "result" {
				return r.toResult(fallbackModel)
			}
		}
		return Result{}, classifyFailure("no result message in claude output")
	}
	var r cliResult
	if err := json.Unmarshal(trimmed, &r); err != nil || r.Type != "result" {
		return Result{}, classifyFailure(string(trimmed))
	}
	return r.toResult(fallbackModel)
}

// StreamParser consumes `--output-format stream-json` lines.
type StreamParser struct {
	FallbackModel string
	OnDelta       func(text string) error

	sawPartial bool
	done       bool
	result     Result
	err        error
}

type streamLine struct {
	Type  string `json:"type"`
	Event *struct {
		Type  string `json:"type"`
		Delta *struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	} `json:"event"`
	Message *struct {
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
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
	switch l.Type {
	case "stream_event":
		if l.Event != nil && l.Event.Type == "content_block_delta" && l.Event.Delta != nil &&
			l.Event.Delta.Type == "text_delta" && l.Event.Delta.Text != "" {
			p.sawPartial = true
			return p.OnDelta(l.Event.Delta.Text)
		}
	case "assistant":
		// Without --include-partial-messages only whole messages arrive.
		if l.Message == nil {
			return nil
		}
		if l.Message.Model != "" {
			p.FallbackModel = l.Message.Model
		}
		if p.sawPartial {
			return nil
		}
		for _, c := range l.Message.Content {
			if c.Type == "text" && c.Text != "" {
				if err := p.OnDelta(c.Text); err != nil {
					return err
				}
			}
		}
	case "result":
		var r cliResult
		if err := json.Unmarshal(line, &r); err != nil {
			return nil
		}
		p.done = true
		p.result, p.err = r.toResult(p.FallbackModel)
		return clicore.ErrStop // the result is the last message: stop reading, end the process group
	}
	return nil
}

// Finish returns the final result. A stream without a result message is an
// error (a truncated stream must never look successful).
func (p *StreamParser) Finish(rawTail string) (Result, error) {
	if !p.done {
		if rawTail == "" {
			rawTail = "stream ended without a result message"
		}
		return Result{}, classifyFailure(rawTail)
	}
	return p.result, p.err
}

// classifyFailure maps CLI failure text to an APIError.
func classifyFailure(text string) *APIError { return codes.Classify(text) }

// StripCodeFence removes a surrounding ```json ... ``` block.
func StripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if nl := strings.Index(s, "\n"); nl >= 0 {
		s = s[nl+1:]
	} else {
		return s
	}
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
}
