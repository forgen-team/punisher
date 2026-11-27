package handlers

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/forgen-team/punisher/internal/claudecli"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/services"
)

// ClaudeStatusSource reports one local AI CLI's state (claudecli, codexcli
// and agycli all return the shared clicore.Status).
type ClaudeStatusSource interface {
	Status(ctx context.Context) claudecli.Status
}

// StatusHandler serves GET /v1/status for the UI's AI banner and the
// settings › AI connection list.
type StatusHandler struct {
	// Claude is the claude status (the `claude` field older web builds read).
	Claude ClaudeStatusSource
	// Providers are the selectable AIs by id (nil = claude only, e.g. the Witems app).
	Providers map[string]ClaudeStatusSource
	// Selected returns the owner's chosen AI (nil = claude).
	Selected   func(ctx context.Context) string
	AppVersion string
	DataDir    string
}

// ProviderStatus is one AI's line in the status response.
type ProviderStatus struct {
	ID        string `json:"id"` // claude | codex | agy
	Ready     bool   `json:"ready"`
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	// LoggedIn is null when the CLI cannot report it (agy has no status command).
	LoggedIn *bool  `json:"loggedIn"`
	Code     string `json:"code,omitempty"`
}

type statusResponse struct {
	Claude claudecli.Status `json:"claude"`
	// Provider is the selected AI; Providers lists every AI (in llm.Providers order).
	Provider  string           `json:"provider"`
	Providers []ProviderStatus `json:"providers"`
	// Message and Steps are the Korean fix-it texts of the selected AI when it is not usable.
	Message    string   `json:"message,omitempty"`
	Steps      []string `json:"steps,omitempty"`
	AppVersion string   `json:"appVersion"`
	DataDir    string   `json:"dataDir"`
}

func (h *StatusHandler) Status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	selected := llm.DefaultProvider
	if h.Selected != nil {
		selected = llm.NormalizeProvider(h.Selected(ctx))
	}
	sources := h.Providers
	if sources == nil {
		sources = map[string]ClaudeStatusSource{llm.ProviderClaude: h.Claude}
	}
	statuses := h.collect(ctx, sources)

	resp := statusResponse{Claude: statuses[llm.ProviderClaude], Provider: selected, AppVersion: h.AppVersion, DataDir: h.DataDir}
	for _, id := range llm.Providers {
		st, ok := statuses[id]
		if !ok {
			continue
		}
		resp.Providers = append(resp.Providers, ProviderStatus{ID: id, Ready: st.Ready, Installed: st.Installed,
			Version: st.Version, LoggedIn: st.LoggedIn, Code: st.Code})
	}
	resp.Message, resp.Steps = fixSteps(selected, statuses[selected].Code)
	writeJSON(w, http.StatusOK, resp)
}

// collect asks every source in parallel (each runs --version / login checks).
func (h *StatusHandler) collect(ctx context.Context, sources map[string]ClaudeStatusSource) map[string]claudecli.Status {
	out := make(map[string]claudecli.Status, len(sources))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for id, src := range sources {
		if src == nil {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			st := src.Status(ctx)
			mu.Lock()
			out[id] = st
			mu.Unlock()
		}()
	}
	wg.Wait()
	if _, ok := out[llm.ProviderClaude]; !ok && h.Claude != nil {
		out[llm.ProviderClaude] = h.Claude.Status(ctx)
	}
	return out
}

// fixSteps is the message and Korean steps for the selected AI's status code.
func fixSteps(provider, code string) (string, []string) {
	switch code {
	case provider + "_unavailable":
		return services.ProviderMessage(provider, "missing"), installSteps[provider]
	case provider + "_not_logged_in":
		return services.ProviderMessage(provider, "logged_out"), loginSteps[provider]
	case provider + "_rate_limited":
		return services.ProviderMessage(provider, "rate_limited") + " 이미 만든 강의는 계속 읽을 수 있습니다.", nil
	}
	return "", nil
}

var installSteps = map[string][]string{
	llm.ProviderClaude: {
		"Claude Code를 설치합니다: 터미널에서 curl -fsSL https://claude.ai/install.sh | bash",
		"터미널에서 claude 를 실행하고 안내에 따라 로그인합니다.",
		"이 화면을 새로고침합니다(punisher를 다시 켤 필요는 없습니다).",
	},
	llm.ProviderCodex: {
		"Codex CLI를 설치합니다: 터미널에서 npm install -g @openai/codex (또는 brew install codex)",
		"터미널에서 codex login 을 실행해 로그인합니다.",
		"이 화면을 새로고침합니다(punisher를 다시 켤 필요는 없습니다).",
	},
	llm.ProviderAgy: {
		"Antigravity CLI(agy)를 설치합니다: antigravity.google 의 안내를 따릅니다.",
		"터미널에서 agy 를 한 번 실행해 로그인합니다.",
		"이 화면을 새로고침합니다(punisher를 다시 켤 필요는 없습니다).",
	},
}

var loginSteps = map[string][]string{
	llm.ProviderClaude: {
		"터미널에서 claude 를 실행합니다.",
		"안내에 따라 로그인(/login)을 마칩니다.",
		"이 화면을 새로고침합니다.",
	},
	llm.ProviderCodex: {
		"터미널에서 codex login 을 실행합니다.",
		"안내에 따라 로그인을 마칩니다(codex login status 로 확인할 수 있습니다).",
		"이 화면을 새로고침합니다.",
	},
	llm.ProviderAgy: {
		"터미널에서 agy 를 실행합니다.",
		"안내에 따라 로그인을 마칩니다.",
		"이 화면을 새로고침합니다.",
	},
}
