package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/forgen-team/punisher/internal/claudecli"
	"github.com/forgen-team/punisher/internal/services"
)

// ClaudeStatusSource reports the local claude CLI state.
type ClaudeStatusSource interface {
	Status(ctx context.Context) claudecli.Status
}

// StatusHandler serves GET /v1/status for the UI's Claude banner.
type StatusHandler struct {
	Claude     ClaudeStatusSource
	AppVersion string
	DataDir    string
}

type statusResponse struct {
	Claude  claudecli.Status `json:"claude"`
	Message string           `json:"message,omitempty"`
	// Steps are Korean fix-it steps shown when Claude is not usable.
	Steps      []string `json:"steps,omitempty"`
	AppVersion string   `json:"appVersion"`
	DataDir    string   `json:"dataDir"`
}

func (h *StatusHandler) Status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	var st claudecli.Status
	if h.Claude != nil {
		st = h.Claude.Status(ctx)
	}
	resp := statusResponse{Claude: st, AppVersion: h.AppVersion, DataDir: h.DataDir}
	switch st.Code {
	case claudecli.CodeCLIMissing:
		resp.Message = services.MsgClaudeMissing
		resp.Steps = []string{
			"Claude Code를 설치합니다: 터미널에서 curl -fsSL https://claude.ai/install.sh | bash",
			"터미널에서 claude 를 실행하고 안내에 따라 로그인합니다.",
			"이 화면을 새로고침합니다(Punisher를 다시 켤 필요는 없습니다).",
		}
	case claudecli.CodeNotLoggedIn:
		resp.Message = services.MsgClaudeLoggedOut
		resp.Steps = []string{
			"터미널에서 claude 를 실행합니다.",
			"안내에 따라 로그인(/login)을 마칩니다.",
			"이 화면을 새로고침합니다.",
		}
	case claudecli.CodeRateLimited:
		resp.Message = "Claude 사용 한도에 도달했습니다. 한도가 풀린 뒤 다시 시도해 주세요. 이미 만든 강의는 계속 읽을 수 있습니다."
	}
	writeJSON(w, http.StatusOK, resp)
}
