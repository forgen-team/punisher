package phone

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// Label is the launchd service name.
const Label = "com.forgen.punisher"

func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func (m *Manager) plistPath() string {
	return filepath.Join(m.Home, "Library", "LaunchAgents", Label+".plist")
}

func (m *Manager) logPath() string {
	return filepath.Join(m.Home, "Library", "Logs", "Punisher", "punisher.log")
}

func (m *Manager) launchDomain() string { return "gui/" + strconv.Itoa(m.UID) }

// plist renders the LaunchAgent. It restarts the app only after a crash
// (non-zero exit), so an orderly quit stays quit.
func (m *Manager) plist() []byte {
	args := []string{m.Executable, "--port", strconv.Itoa(m.Port), "--no-browser"}
	if m.CustomDataDir {
		args = append(args, "--data-dir", m.DataDir)
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
`)
	key := func(k string) { fmt.Fprintf(&b, "  <key>%s</key>\n", k) }
	str := func(k, v string) {
		key(k)
		fmt.Fprintf(&b, "  <string>%s</string>\n", xmlEscape(v))
	}
	str("Label", Label)
	key("ProgramArguments")
	b.WriteString("  <array>\n")
	for _, a := range args {
		fmt.Fprintf(&b, "    <string>%s</string>\n", xmlEscape(a))
	}
	b.WriteString("  </array>\n")
	key("RunAtLoad")
	b.WriteString("  <true/>\n")
	key("KeepAlive")
	b.WriteString("  <dict>\n    <key>SuccessfulExit</key>\n    <false/>\n  </dict>\n")
	key("ThrottleInterval")
	b.WriteString("  <integer>30</integer>\n")
	str("StandardOutPath", m.logPath())
	str("StandardErrorPath", m.logPath())
	if len(m.Env) > 0 {
		key("EnvironmentVariables")
		b.WriteString("  <dict>\n")
		for _, kv := range m.Env {
			name, value, _ := strings.Cut(kv, "=")
			fmt.Fprintf(&b, "    <key>%s</key>\n    <string>%s</string>\n", xmlEscape(name), xmlEscape(value))
		}
		b.WriteString("  </dict>\n")
	}
	b.WriteString("</dict>\n</plist>\n")
	return []byte(b.String())
}
