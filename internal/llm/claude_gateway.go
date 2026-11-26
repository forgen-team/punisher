package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ProviderClaudeGateway is the provider name for the host claude-gateway.
const ProviderClaudeGateway = "claude-gateway"

// Gateway model aliases understood by claude-gateway (`claude --model`).
const (
	GatewayModelSonnet = "sonnet"
	GatewayModelHaiku  = "haiku"
	GatewayModelOpus   = "opus"
)

const defaultGatewayTimeout = 180 * time.Second

// ClaudeGatewayClient calls apps/claude-gateway, which runs the host's
// logged-in Claude Code CLI. It implements the same Create/Stream shape as
// AnthropicMessagesClient; the apiKey argument is ignored (no BYOK key is
// involved — the gateway authenticates with X-Internal-Token).
type ClaudeGatewayClient struct {
	BaseURL       string // e.g. http://host.docker.internal:4790
	InternalToken string // WITEMS_INTERNAL_TOKEN
	HTTPClient    *http.Client
	// DefaultTimeout applies when a request has no Timeout; default 180s.
	DefaultTimeout time.Duration
	// BackgroundQueueWait is extra client-side deadline for background calls,
	// whose gateway timeout starts only after a slot is granted. It should
	// match CLAUDE_GATEWAY_BACKGROUND_QUEUE_TIMEOUT_SEC; default 300s.
	BackgroundQueueWait time.Duration
	// AppName is sent as X-Witems-App so the gateway picks this app's Claude
	// account profile. Empty means "read WITEMS_APP_NAME at request time";
	// both empty sends no header (the gateway uses its default profile).
	AppName string
}

// GatewayAppHeader names the calling app to the gateway (punisher in Witems).
const GatewayAppHeader = "X-Witems-App"

const defaultBackgroundQueueWait = 300 * time.Second

// NewClaudeGatewayClientFromEnv reads CLAUDE_GATEWAY_URL and
// WITEMS_INTERNAL_TOKEN. It returns nil when the URL is unset.
func NewClaudeGatewayClientFromEnv() *ClaudeGatewayClient {
	url := strings.TrimSpace(os.Getenv("CLAUDE_GATEWAY_URL"))
	if url == "" {
		return nil
	}
	return &ClaudeGatewayClient{
		BaseURL:       url,
		InternalToken: strings.TrimSpace(os.Getenv("WITEMS_INTERNAL_TOKEN")),
		AppName:       strings.TrimSpace(os.Getenv("WITEMS_APP_NAME")),
	}
}

// appName is the explicit AppName, else WITEMS_APP_NAME.
func (c *ClaudeGatewayClient) appName() string {
	if c != nil && strings.TrimSpace(c.AppName) != "" {
		return strings.TrimSpace(c.AppName)
	}
	return strings.TrimSpace(os.Getenv("WITEMS_APP_NAME"))
}

// setCommonHeaders adds the token and, when known, the app name.
func (c *ClaudeGatewayClient) setCommonHeaders(req *http.Request) {
	req.Header.Set("X-Internal-Token", c.InternalToken)
	if app := c.appName(); app != "" {
		req.Header.Set(GatewayAppHeader, app)
	}
}

// GatewayRequest is the gateway's /v1/complete and /v1/stream body.
type GatewayRequest struct {
	System     string `json:"system,omitempty"`
	Prompt     string `json:"prompt"`
	Model      string `json:"model,omitempty"`
	MaxTurns   int    `json:"maxTurns"`
	TimeoutSec int    `json:"timeoutSec,omitempty"`
	JSONMode   bool   `json:"jsonMode,omitempty"`
	// Priority selects the gateway lane: "interactive" (default) or
	// "background". Empty means "take it from ctx" (see WithPriority).
	Priority Priority `json:"priority,omitempty"`
}

type gatewayUsage struct {
	InputTokens              int `json:"inputTokens"`
	OutputTokens             int `json:"outputTokens"`
	CacheReadInputTokens     int `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens"`
}

type gatewayResult struct {
	Text       string        `json:"text"`
	Model      string        `json:"model"`
	Usage      *gatewayUsage `json:"usage"`
	DurationMs int64         `json:"durationMs"`
}

func (r gatewayResult) toMessages() MessagesResult {
	out := MessagesResult{Text: r.Text, Model: r.Model, StopReason: "end_turn", LatencyMs: r.DurationMs}
	if r.Usage != nil {
		out.Usage = Usage{
			InputTokens:  r.Usage.InputTokens + r.Usage.CacheReadInputTokens + r.Usage.CacheCreationInputTokens,
			OutputTokens: r.Usage.OutputTokens,
		}
	}
	return out
}

type gatewayError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GatewayHealth is the gateway's GET /health payload.
type GatewayHealth struct {
	Status        string `json:"status"`
	Reason        string `json:"reason,omitempty"`
	ClaudeVersion string `json:"claudeVersion,omitempty"`
	Credentials   string `json:"credentials,omitempty"`
	Active        int    `json:"active"`
	Queued        int    `json:"queued"`
	CallsToday    int    `json:"callsToday"`
	DailyLimit    int    `json:"dailyLimit"`
}

// Complete performs one non-streaming generation.
func (c *ClaudeGatewayClient) Complete(ctx context.Context, req GatewayRequest) (MessagesResult, error) {
	start := time.Now()
	req.Priority = resolvePriority(ctx, req.Priority)
	ctx, cancel := c.callContext(ctx, req.TimeoutSec, req.Priority)
	defer cancel()
	resp, err := c.do(ctx, "/v1/complete", req)
	if err != nil {
		return MessagesResult{}, err
	}
	defer resp.Body.Close()
	var parsed gatewayResult
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&parsed); err != nil {
		return MessagesResult{}, gatewayTransportError(ctx, err)
	}
	out := parsed.toMessages()
	out.LatencyMs = time.Since(start).Milliseconds()
	return out, nil
}

// StreamComplete streams text deltas. A stream without a final "done" event
// is an error.
func (c *ClaudeGatewayClient) StreamComplete(ctx context.Context, req GatewayRequest, onDelta func(string) error) (MessagesResult, error) {
	start := time.Now()
	req.Priority = resolvePriority(ctx, req.Priority)
	ctx, cancel := c.callContext(ctx, req.TimeoutSec, req.Priority)
	defer cancel()
	resp, err := c.do(ctx, "/v1/stream", req)
	if err != nil {
		return MessagesResult{}, err
	}
	defer resp.Body.Close()

	var (
		text      strings.Builder
		eventName string
		final     *MessagesResult
	)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		switch {
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			switch eventName {
			case "delta":
				var d struct {
					Text string `json:"text"`
				}
				if json.Unmarshal([]byte(data), &d) == nil && d.Text != "" {
					text.WriteString(d.Text)
					if onDelta != nil {
						if err := onDelta(d.Text); err != nil {
							return MessagesResult{Text: text.String()}, err
						}
					}
				}
			case "done":
				var r gatewayResult
				if err := json.Unmarshal([]byte(data), &r); err != nil {
					return MessagesResult{Text: text.String()}, &LLMError{Message: "claude-gateway: invalid done event", Err: ErrServerError}
				}
				res := r.toMessages()
				if res.Text == "" {
					res.Text = text.String()
				}
				final = &res
			case "error":
				var ge gatewayError
				_ = json.Unmarshal([]byte(data), &ge)
				return MessagesResult{Text: text.String()}, gatewayCodeError(http.StatusBadGateway, ge)
			}
		case line == "":
			eventName = ""
		}
	}
	if err := scanner.Err(); err != nil {
		return MessagesResult{Text: text.String()}, gatewayTransportError(ctx, err)
	}
	if final == nil {
		return MessagesResult{Text: text.String()}, &LLMError{Message: "claude-gateway: stream ended without done event", Err: ErrServerError}
	}
	final.LatencyMs = time.Since(start).Milliseconds()
	return *final, nil
}

// Create adapts MessagesRequest to the gateway (apiKey is ignored).
func (c *ClaudeGatewayClient) Create(ctx context.Context, _ string, req MessagesRequest) (MessagesResult, error) {
	return c.Complete(ctx, gatewayRequestFrom(req))
}

// Stream adapts MessagesRequest to the gateway SSE endpoint (apiKey is ignored).
func (c *ClaudeGatewayClient) Stream(ctx context.Context, _ string, req MessagesRequest, onDelta func(string) error) (MessagesResult, error) {
	return c.StreamComplete(ctx, gatewayRequestFrom(req), onDelta)
}

// Health calls GET /health. An error means the gateway is unreachable or
// rejected the token.
func (c *ClaudeGatewayClient) Health(ctx context.Context) (GatewayHealth, error) {
	if c == nil || strings.TrimSpace(c.BaseURL) == "" {
		return GatewayHealth{}, &LLMError{Message: "claude-gateway: CLAUDE_GATEWAY_URL is not set", Err: ErrGenerationUnavailable, Code: "gateway_not_configured"}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/health", nil)
	if err != nil {
		return GatewayHealth{}, err
	}
	c.setCommonHeaders(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return GatewayHealth{}, gatewayTransportError(ctx, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return GatewayHealth{}, decodeGatewayError(resp)
	}
	var h GatewayHealth
	if err := json.NewDecoder(resp.Body).Decode(&h); err != nil {
		return GatewayHealth{}, &LLMError{Message: "claude-gateway: invalid health body", Err: ErrServerError}
	}
	return h, nil
}

func gatewayRequestFrom(req MessagesRequest) GatewayRequest {
	prompt := req.User
	if len(req.Messages) > 0 {
		var b strings.Builder
		for _, m := range req.Messages {
			fmt.Fprintf(&b, "[%s]\n%s\n\n", m.Role, m.Content)
		}
		if req.User != "" {
			fmt.Fprintf(&b, "[user]\n%s\n", req.User)
		}
		prompt = strings.TrimSpace(b.String())
	}
	out := GatewayRequest{System: req.System, Prompt: prompt, Model: req.Model, MaxTurns: 1, Priority: req.Priority}
	if req.Timeout > 0 {
		out.TimeoutSec = int((req.Timeout + time.Second - 1) / time.Second)
	}
	return out
}

// resolvePriority: explicit request priority wins, else the ctx priority.
func resolvePriority(ctx context.Context, p Priority) Priority {
	if p == "" {
		return PriorityFromContext(ctx)
	}
	return p.normalized()
}

func (c *ClaudeGatewayClient) callContext(ctx context.Context, timeoutSec int, priority Priority) (context.Context, context.CancelFunc) {
	timeout := defaultGatewayTimeout
	if c != nil && c.DefaultTimeout > 0 {
		timeout = c.DefaultTimeout
	}
	if timeout <= 0 {
		timeout = defaultGatewayTimeout
	}
	if timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}
	if priority == PriorityBackground {
		wait := defaultBackgroundQueueWait
		if c != nil && c.BackgroundQueueWait > 0 {
			wait = c.BackgroundQueueWait
		}
		timeout += wait
	}
	// Slack so the gateway reports its own timeout (504) before we give up.
	return context.WithTimeout(ctx, timeout+15*time.Second)
}

func (c *ClaudeGatewayClient) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return defaultGatewayHTTP
}

// defaultGatewayHTTP has no overall Timeout (streams last minutes); calls are
// bounded by callContext. No proxy: the gateway is on the host network.
var defaultGatewayHTTP = &http.Client{Transport: &http.Transport{IdleConnTimeout: 90 * time.Second}}

func (c *ClaudeGatewayClient) do(ctx context.Context, path string, body GatewayRequest) (*http.Response, error) {
	if c == nil || strings.TrimSpace(c.BaseURL) == "" {
		return nil, &LLMError{Message: "claude-gateway: CLAUDE_GATEWAY_URL is not set", Err: ErrGenerationUnavailable, Code: "gateway_not_configured"}
	}
	if strings.TrimSpace(body.Prompt) == "" {
		return nil, &LLMError{StatusCode: http.StatusBadRequest, Message: "claude-gateway: empty prompt", Err: ErrInvalidRequest}
	}
	body.MaxTurns = 1
	body.Priority = resolvePriority(ctx, body.Priority)
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setCommonHeaders(req)
	req.Header.Set("X-Priority", string(body.Priority))
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, gatewayTransportError(ctx, err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, decodeGatewayError(resp)
	}
	return resp, nil
}

// maxErrorBodyBytes caps how much of an error response is read.
const maxErrorBodyBytes = 8 << 10

func decodeGatewayError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	var body struct {
		Error gatewayError `json:"error"`
	}
	if json.Unmarshal(raw, &body) != nil || body.Error.Code == "" {
		body.Error = gatewayError{Code: fmt.Sprintf("http_%d", resp.StatusCode), Message: strings.TrimSpace(string(raw))}
	}
	return gatewayCodeError(resp.StatusCode, body.Error)
}

// gatewayCodeError maps gateway error codes onto the llm sentinel errors.
func gatewayCodeError(status int, ge gatewayError) error {
	sentinel := ErrServerError
	switch ge.Code {
	case "claude_not_logged_in", "claude_unavailable", "unauthorized":
		sentinel = ErrGenerationUnavailable
	case "claude_rate_limited", "queue_full", "daily_cap_reached":
		sentinel = ErrRateLimited
	case "claude_timeout":
		sentinel = ErrTimeout
	case "invalid_request":
		sentinel = ErrInvalidRequest
	default:
		switch {
		case status == http.StatusTooManyRequests:
			sentinel = ErrRateLimited
		case status == http.StatusUnauthorized || status == http.StatusServiceUnavailable:
			sentinel = ErrGenerationUnavailable
		case status == http.StatusGatewayTimeout:
			sentinel = ErrTimeout
		}
	}
	return &LLMError{StatusCode: status, Code: ge.Code, Message: "claude-gateway: " + ge.Code + ": " + ge.Message, Err: sentinel}
}

func gatewayTransportError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &LLMError{Message: "claude-gateway: timeout", Err: ErrTimeout, Code: "client_timeout"}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return &LLMError{Message: "claude-gateway: unreachable: " + err.Error(), Err: ErrGenerationUnavailable, Code: "gateway_unreachable"}
}

// GatewayErrorCode returns the gateway error code carried by err, if any.
func GatewayErrorCode(err error) string {
	var le *LLMError
	if errors.As(err, &le) {
		return le.Code
	}
	return ""
}
