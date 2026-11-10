// Package phone turns on access from the user's own phone: `punisher phone on`
// keeps the app running with launchd and publishes it to the user's tailnet
// with Tailscale Serve. The app itself stays bound to 127.0.0.1 and the HTTP
// layer accepts exactly the saved host and owner (see httpapi.localOnly).
// Funnel, which would publish it to the internet, is never used.
package phone

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const settingsFile = "phone.json"

// Access is who may reach the app besides this Mac: requests for Host from the
// Tailscale user Login. The zero value means phone access is off.
type Access struct {
	Host  string `json:"allowedHost"`
	Login string `json:"ownerLogin"`
}

// hostRE is a full MagicDNS name: device.tailnet.ts.net (lower-case, no port).
var hostRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+\.ts\.net$`)

// ValidHost reports whether h is a complete Tailscale MagicDNS name.
func ValidHost(h string) bool {
	return hostRE.MatchString(h)
}

// taggedLogin is the login Tailscale reports for devices owned by a tag.
const taggedLogin = "tagged-devices"

// ValidLogin reports whether l can identify one Tailscale user.
func ValidLogin(l string) bool {
	if l == "" || l == taggedLogin || len(l) > 320 {
		return false
	}
	return strings.IndexFunc(l, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) < 0
}

func (a Access) valid() bool { return ValidHost(a.Host) && ValidLogin(a.Login) }

// LoadAccess returns what `phone on` saved (zero when phone access is off). A
// malformed file is an error so the caller stays loopback-only.
func LoadAccess(dataDir string) (Access, error) {
	raw, err := os.ReadFile(filepath.Join(dataDir, settingsFile))
	if errors.Is(err, os.ErrNotExist) {
		return Access{}, nil
	}
	if err != nil {
		return Access{}, fmt.Errorf("read phone settings: %w", err)
	}
	var a Access
	if err := json.Unmarshal(raw, &a); err != nil {
		return Access{}, fmt.Errorf("parse phone settings: %w", err)
	}
	if !a.valid() {
		return Access{}, fmt.Errorf("phone settings: invalid host or owner (%q)", a.Host)
	}
	return a, nil
}

func saveAccess(dataDir string, a Access) error {
	if !a.valid() {
		return fmt.Errorf("invalid host or owner (%q)", a.Host)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, settingsFile), raw, 0o600)
}

func removeAccess(dataDir string) error {
	err := os.Remove(filepath.Join(dataDir, settingsFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
