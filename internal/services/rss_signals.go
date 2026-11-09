package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LearningSignals are the RSS-side inputs of next-chapter suggestions (source c).
type LearningSignals struct {
	Interests []string `json:"interests"`
	Trending  []struct {
		Topic string  `json:"topic"`
		Z     float64 `json:"z"`
	} `json:"trending"`
	ExplanationConcepts []struct {
		ConceptID string `json:"conceptId"`
		Count     int    `json:"count"`
	} `json:"explanationConcepts"`
}

// SignalSource provides LearningSignals for a user. Nil on the Service = off
// (standalone build has no RSS).
type SignalSource interface {
	LearningSignals(ctx context.Context, userID string) (LearningSignals, error)
}

// RSSSignals calls rss-server GET /v1/internal/users/{userId}/learning-signals
// with the service-to-service internal token.
type RSSSignals struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func (r RSSSignals) LearningSignals(ctx context.Context, userID string) (LearningSignals, error) {
	var out LearningSignals
	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	endpoint := strings.TrimRight(r.BaseURL, "/") + "/v1/internal/users/" + url.PathEscape(userID) + "/learning-signals"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("X-Internal-Token", r.Token)
	req.Header.Set("X-Internal-User-Id", userID)
	resp, err := client.Do(req)
	if err != nil {
		return out, fmt.Errorf("rss learning signals: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return out, fmt.Errorf("rss learning signals: read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("rss learning signals: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, fmt.Errorf("rss learning signals: decode: %w", err)
	}
	return out, nil
}
