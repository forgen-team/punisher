package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func gatewayServer(t *testing.T, h http.HandlerFunc) *ClaudeGatewayClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &ClaudeGatewayClient{BaseURL: srv.URL, InternalToken: "tok"}
}

func TestGatewayCreate(t *testing.T) {
	var got GatewayRequest
	c := gatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/complete" || r.Header.Get("X-Internal-Token") != "tok" {
			t.Errorf("path %s token %q", r.URL.Path, r.Header.Get("X-Internal-Token"))
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		fmt.Fprint(w, `{"text":"hello","model":"claude-sonnet-x","usage":{"inputTokens":3,"outputTokens":5,"cacheReadInputTokens":10},"durationMs":12}`)
	})
	res, err := c.Create(context.Background(), "ignored-key", MessagesRequest{Model: "sonnet", System: "S", User: "U", Timeout: 1500 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "hello" || res.Model != "claude-sonnet-x" || res.Usage.InputTokens != 13 || res.Usage.OutputTokens != 5 || res.Truncated() {
		t.Fatalf("res %+v", res)
	}
	if got.System != "S" || got.Prompt != "U" || got.Model != "sonnet" || got.MaxTurns != 1 || got.TimeoutSec != 2 {
		t.Fatalf("request %+v", got)
	}
}

func TestGatewayStream(t *testing.T) {
	c := gatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/stream" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: delta\ndata: {\"text\":\"Hel\"}\n\nevent: delta\ndata: {\"text\":\"lo\"}\n\n")
		fmt.Fprint(w, "event: done\ndata: {\"text\":\"Hello\",\"model\":\"m\",\"usage\":{\"inputTokens\":1,\"outputTokens\":2}}\n\n")
	})
	var deltas []string
	res, err := c.Stream(context.Background(), "", MessagesRequest{User: "x"}, func(d string) error { deltas = append(deltas, d); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(deltas, "|") != "Hel|lo" || res.Text != "Hello" || res.Usage.OutputTokens != 2 {
		t.Fatalf("deltas %v res %+v", deltas, res)
	}
}

func TestGatewayStreamWithoutDoneIsError(t *testing.T) {
	c := gatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "event: delta\ndata: {\"text\":\"par\"}\n\n")
	})
	res, err := c.Stream(context.Background(), "", MessagesRequest{User: "x"}, nil)
	if err == nil || res.Text != "par" {
		t.Fatalf("expected error with partial text, got %+v %v", res, err)
	}
}

func TestGatewayStreamErrorEvent(t *testing.T) {
	c := gatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "event: delta\ndata: {\"text\":\"a\"}\n\nevent: error\ndata: {\"code\":\"claude_rate_limited\",\"message\":\"limit\"}\n\n")
	})
	_, err := c.Stream(context.Background(), "", MessagesRequest{User: "x"}, nil)
	if !errors.Is(err, ErrRateLimited) || GatewayErrorCode(err) != "claude_rate_limited" {
		t.Fatalf("err %v", err)
	}
}

func TestGatewayErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		code   string
		want   error
	}{
		{401, "claude_not_logged_in", ErrGenerationUnavailable},
		{429, "claude_rate_limited", ErrRateLimited},
		{429, "daily_cap_reached", ErrRateLimited},
		{504, "claude_timeout", ErrTimeout},
		{502, "claude_error", ErrServerError},
	}
	for _, tc := range cases {
		c := gatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			fmt.Fprintf(w, `{"error":{"code":%q,"message":"m"}}`, tc.code)
		})
		_, err := c.Create(context.Background(), "", MessagesRequest{User: "x"})
		if !errors.Is(err, tc.want) || GatewayErrorCode(err) != tc.code {
			t.Errorf("%s: err %v", tc.code, err)
		}
	}
}

func TestGatewayUnreachableAndUnconfigured(t *testing.T) {
	c := &ClaudeGatewayClient{BaseURL: "http://127.0.0.1:1"}
	if _, err := c.Create(context.Background(), "", MessagesRequest{User: "x"}); !errors.Is(err, ErrGenerationUnavailable) {
		t.Fatalf("unreachable err %v", err)
	}
	var nilClient *ClaudeGatewayClient
	if _, err := nilClient.Complete(context.Background(), GatewayRequest{Prompt: "x"}); !errors.Is(err, ErrGenerationUnavailable) {
		t.Fatalf("nil client err %v", err)
	}
	if _, err := nilClient.Health(context.Background()); !errors.Is(err, ErrGenerationUnavailable) {
		t.Fatalf("nil health err %v", err)
	}
}

func TestGatewayHealth(t *testing.T) {
	c := gatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"degraded","reason":"claude_not_logged_in","callsToday":3,"dailyLimit":150}`)
	})
	h, err := c.Health(context.Background())
	if err != nil || h.Status != "degraded" || h.Reason != "claude_not_logged_in" || h.CallsToday != 3 {
		t.Fatalf("health %+v err %v", h, err)
	}
}

func TestGatewayPriority(t *testing.T) {
	type seen struct{ body, header Priority }
	ch := make(chan seen, 8)
	c := gatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
		var got GatewayRequest
		_ = json.NewDecoder(r.Body).Decode(&got)
		ch <- seen{got.Priority, Priority(r.Header.Get("X-Priority"))}
		if r.URL.Path == "/v1/stream" {
			fmt.Fprint(w, "event: done\ndata: {\"text\":\"x\"}\n\n")
			return
		}
		fmt.Fprint(w, `{"text":"x"}`)
	})
	bg := WithPriority(context.Background(), PriorityBackground)
	cases := []struct {
		name string
		call func() error
		want Priority
	}{
		{"default sync", func() error { _, err := c.Create(context.Background(), "", MessagesRequest{User: "u"}); return err }, PriorityInteractive},
		{"ctx sync", func() error { _, err := c.Create(bg, "", MessagesRequest{User: "u"}); return err }, PriorityBackground},
		{"ctx stream", func() error { _, err := c.Stream(bg, "", MessagesRequest{User: "u"}, nil); return err }, PriorityBackground},
		{"explicit stream", func() error {
			_, err := c.StreamComplete(context.Background(), GatewayRequest{Prompt: "u", Priority: PriorityBackground}, nil)
			return err
		}, PriorityBackground},
		{"explicit overrides ctx", func() error {
			_, err := c.Create(bg, "", MessagesRequest{User: "u", Priority: PriorityInteractive})
			return err
		}, PriorityInteractive},
		{"short lane is kept", func() error {
			_, err := c.Create(context.Background(), "", MessagesRequest{User: "u", Priority: ShortUnlessBackground(context.Background())})
			return err
		}, PriorityShort},
		{"short inside background work stays background", func() error {
			_, err := c.Create(bg, "", MessagesRequest{User: "u", Priority: ShortUnlessBackground(bg)})
			return err
		}, PriorityBackground},
	}
	for _, tc := range cases {
		if err := tc.call(); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if s := <-ch; s.body != tc.want || s.header != tc.want {
			t.Errorf("%s: body=%q header=%q want %q", tc.name, s.body, s.header, tc.want)
		}
	}
	if !IsBackground(bg) || IsBackground(context.Background()) {
		t.Fatal("IsBackground")
	}
}
