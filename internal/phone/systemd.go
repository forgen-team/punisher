package phone

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Linux: the app runs as a systemd user service (punisher.service) instead of
// a LaunchAgent. It starts at login; to keep it running while logged out the
// user enables lingering themselves (printed hint — the app never runs sudo).
const unitName = "punisher.service"

const msgLinger = "로그아웃해도 폰에서 열리게 하려면 한 번만 실행하세요: loginctl enable-linger $USER"

const msgOperator = "tailscale serve 권한이 없다면 한 번만 실행하세요: sudo tailscale set --operator=$USER"

func (m *Manager) unitPath() string {
	return filepath.Join(m.Home, ".config", "systemd", "user", unitName)
}

func (m *Manager) linuxLogPath() string {
	return filepath.Join(m.Home, ".local", "state", "punisher", "punisher.log")
}

func (m *Manager) systemctl(ctx context.Context, args ...string) ([]byte, error) {
	return runOutput(ctx, m.Systemctl, append([]string{"--user"}, args...)...)
}

func (m *Manager) unitActive(ctx context.Context) bool {
	_, err := m.systemctl(ctx, "is-active", "--quiet", unitName)
	return err == nil
}

// unit renders the service: restart only after a crash, like the LaunchAgent.
func (m *Manager) unit() []byte {
	args := []string{m.Executable, "--port", strconv.Itoa(m.Port), "--no-browser"}
	if m.CustomDataDir {
		args = append(args, "--data-dir", m.DataDir)
	}
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = systemdQuote(a)
	}
	var b strings.Builder
	b.WriteString("[Unit]\nDescription=Punisher (phone access)\n\n[Service]\n")
	fmt.Fprintf(&b, "ExecStart=%s\n", strings.Join(quoted, " "))
	for _, kv := range m.Env {
		fmt.Fprintf(&b, "Environment=%s\n", systemdQuote(kv))
	}
	b.WriteString("Restart=on-failure\nRestartSec=30\n")
	fmt.Fprintf(&b, "StandardOutput=append:%s\nStandardError=append:%s\n", m.linuxLogPath(), m.linuxLogPath())
	b.WriteString("\n[Install]\nWantedBy=default.target\n")
	return []byte(b.String())
}

// systemdQuote double-quotes a word for ExecStart/Environment (backslash, quote, %, $ escaped).
func systemdQuote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `%`, `%%`, `$`, `$$`)
	return `"` + r.Replace(s) + `"`
}

func (m *Manager) installUnit(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(m.unitPath()), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.linuxLogPath()), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(m.unitPath(), m.unit(), 0o600); err != nil {
		return err
	}
	if _, err := m.systemctl(ctx, "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	if _, err := m.systemctl(ctx, "enable", "--now", unitName); err != nil {
		return fmt.Errorf("systemctl enable: %w", err)
	}
	fmt.Fprintln(m.Out, msgLinger)
	return nil
}

func (m *Manager) stopUnit(ctx context.Context) error {
	if !m.unitActive(ctx) {
		_, _ = m.systemctl(ctx, "disable", unitName)
		return nil
	}
	if _, err := m.systemctl(ctx, "disable", "--now", unitName); err != nil {
		return fmt.Errorf("systemctl disable: %w", err)
	}
	m.waitUntil(ctx, func() bool { return !m.unitActive(ctx) })
	return nil
}
