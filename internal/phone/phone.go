package phone

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Port is the fixed local port phone access publishes.
const Port = 4710

const httpsPort = "443"

// Manager performs `phone on` and `phone off`. Every outside program and
// location is a field so tests run against fakes and never touch the real
// Tailscale or launchd setup.
type Manager struct {
	DataDir       string
	CustomDataDir bool     // pass --data-dir to the LaunchAgent
	Executable    string   // absolute path of the punisher binary
	Home          string   // user's home directory
	UID           int      // for the launchd gui/<uid> domain
	Port          int      // local port
	Env           []string // KEY=VALUE pairs for the LaunchAgent
	Tailscale     string   // explicit CLI path ("" = search)
	AppPaths      []string // CLI locations tried after PATH
	Launchctl     string
	Out           io.Writer
	In            io.Reader

	// Probes (real ones by default; see New).
	IsPunisher func(port int) bool
	PortFree   func(port int) bool
	Wait       time.Duration // how long to wait for the app to stop or start
	Poll       time.Duration
}

// New returns a Manager for the current user.
func New(dataDir string, customDataDir bool, executable, home string, env []string) *Manager {
	return &Manager{
		DataDir: dataDir, CustomDataDir: customDataDir, Executable: executable, Home: home,
		UID: os.Getuid(), Port: Port, Env: env,
		AppPaths: []string{
			"/Applications/Tailscale.app/Contents/MacOS/Tailscale",
			"/usr/local/bin/tailscale",
			"/opt/homebrew/bin/tailscale",
		},
		Launchctl: "/bin/launchctl", Out: os.Stdout, In: os.Stdin,
		IsPunisher: localPunisher, PortFree: localPortFree,
		Wait: 15 * time.Second, Poll: 200 * time.Millisecond,
	}
}

// localPunisher reports whether Punisher answers on 127.0.0.1:port.
func localPunisher(port int) bool {
	return isPunisherAt("http://127.0.0.1:" + strconv.Itoa(port))
}

func isPunisherAt(baseURL string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(baseURL + "/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var body struct {
		Service string `json:"service"`
	}
	return json.NewDecoder(resp.Body).Decode(&body) == nil && body.Service == "punisher"
}

func localPortFree(port int) bool {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

const msgNoTailscale = `Tailscale을 찾을 수 없어요.
https://tailscale.com/download 에서 Mac용 Tailscale을 설치하고 로그인한 뒤 다시 실행해 주세요.`

const msgAskOwner = "직접 설정한 적이 없다면 이 문구를 Punisher를 준 사람에게 보여 주세요."

func (m *Manager) findTailscale() (string, error) {
	if m.Tailscale != "" {
		return m.Tailscale, nil
	}
	if p, err := exec.LookPath("tailscale"); err == nil {
		return p, nil
	}
	for _, p := range m.AppPaths {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return p, nil
		}
	}
	return "", errors.New(msgNoTailscale)
}

func runOutput(ctx context.Context, bin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
	}
	return out, err
}

type tailscaleStatus struct {
	BackendState string
	Self         struct {
		DNSName string
		UserID  int64
		Tags    []string
	}
	User map[string]struct {
		LoginName string
	}
}

// deviceAccess asks Tailscale for this Mac's full name and its owner.
func (m *Manager) deviceAccess(ctx context.Context, ts string) (Access, error) {
	const tryAgain = "메뉴 막대의 Tailscale 아이콘에서 로그인하고 연결한 뒤 다시 실행해 주세요."
	raw, err := runOutput(ctx, ts, "status", "--json")
	if err != nil && len(raw) == 0 {
		return Access{}, fmt.Errorf("Tailscale 상태를 읽지 못했어요. %s (%v)", tryAgain, err)
	}
	var st tailscaleStatus
	if err := json.Unmarshal(raw, &st); err != nil {
		return Access{}, fmt.Errorf("Tailscale 상태를 읽지 못했어요. %s (%v)", tryAgain, err)
	}
	if st.BackendState != "Running" {
		return Access{}, errors.New("Tailscale이 켜져 있지 않거나 로그인되어 있지 않아요. " + tryAgain)
	}
	host := strings.ToLower(strings.TrimSuffix(st.Self.DNSName, "."))
	if !ValidHost(host) {
		return Access{}, errors.New("이 Mac의 Tailscale 주소를 찾지 못했어요. https://login.tailscale.com/admin/dns 에서 MagicDNS를 켠 뒤 다시 실행해 주세요.")
	}
	login := st.User[strconv.FormatInt(st.Self.UserID, 10)].LoginName
	if len(st.Self.Tags) > 0 || !ValidLogin(login) {
		return Access{}, errors.New("이 Mac이 개인 계정이 아닌 태그로 Tailscale에 등록되어 있어요. 내 계정으로 다시 로그인한 뒤 실행해 주세요. " + msgAskOwner)
	}
	return Access{Host: host, Login: login}, nil
}

type webConfig struct {
	Handlers map[string]struct {
		Proxy string
	}
}

type serveConfig struct {
	TCP map[string]struct {
		TCPForward string
	}
	Web         map[string]webConfig
	AllowFunnel map[string]bool
	Foreground  map[string]serveConfig
}

func (m *Manager) readServe(ctx context.Context, ts string) (serveConfig, error) {
	var sc serveConfig
	raw, err := runOutput(ctx, ts, "serve", "status", "--json")
	if err != nil {
		return sc, fmt.Errorf("Tailscale 연결 설정을 읽지 못했어요. 메뉴 막대에서 Tailscale이 켜져 있는지 확인하고 다시 실행해 주세요. (%w)", err)
	}
	if strings.TrimSpace(string(raw)) == "" {
		return sc, nil
	}
	if err := json.Unmarshal(raw, &sc); err != nil {
		return sc, fmt.Errorf("Tailscale 연결 설정을 읽지 못했어요. 메뉴 막대에서 Tailscale이 켜져 있는지 확인하고 다시 실행해 주세요. (%w)", err)
	}
	return sc, nil
}

func (m *Manager) isOurProxy(target string) bool {
	p := strconv.Itoa(m.Port)
	switch strings.TrimRight(target, "/") {
	case "http://127.0.0.1:" + p, "http://localhost:" + p:
		return true
	}
	return false
}

// isOurs reports whether a web entry is exactly what `phone on` publishes.
func (m *Manager) isOurs(web webConfig) bool {
	return len(web.Handlers) == 1 && m.isOurProxy(web.Handlers["/"].Proxy)
}

// checkServable refuses anything `phone on` must not take over or expose,
// including what a foreground `tailscale serve` session holds.
func (m *Manager) checkServable(sc serveConfig, ts, host string) error {
	key := host + ":" + httpsPort
	configs := []serveConfig{sc}
	for _, fg := range sc.Foreground {
		configs = append(configs, fg)
	}
	occupied := errors.New("이 Mac의 Tailscale 주소를 이미 다른 프로그램이 쓰고 있어요. 그 설정을 지우지 않으려고 멈췄어요. " + msgAskOwner)
	for _, c := range configs {
		if c.AllowFunnel[key] {
			return fmt.Errorf("이 Mac에 인터넷 공개 기능(Tailscale Funnel)이 켜져 있어요. Punisher는 로그인 화면이 없어서 공개되면 위험해요. 터미널에 아래를 입력해 끈 뒤 다시 실행해 주세요.\n  %q funnel --https=443 off", ts)
		}
		if c.TCP[httpsPort].TCPForward != "" {
			return occupied
		}
		if web, ok := c.Web[key]; ok && !m.isOurs(web) {
			return occupied
		}
	}
	return nil
}

func (m *Manager) launchctl(ctx context.Context, args ...string) ([]byte, error) {
	return runOutput(ctx, m.Launchctl, args...)
}

func (m *Manager) agentLoaded(ctx context.Context) bool {
	_, err := m.launchctl(ctx, "print", m.launchDomain()+"/"+Label)
	return err == nil
}

func (m *Manager) bootout(ctx context.Context) error {
	if !m.agentLoaded(ctx) {
		return nil
	}
	if _, err := m.launchctl(ctx, "bootout", m.launchDomain()+"/"+Label); err != nil {
		return fmt.Errorf("launchctl bootout: %w", err)
	}
	m.waitUntil(ctx, func() bool { return !m.agentLoaded(ctx) })
	return nil
}

func (m *Manager) waitUntil(ctx context.Context, cond func() bool) bool {
	deadline := time.Now().Add(m.Wait)
	for {
		if cond() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(m.Poll):
		}
	}
}

func (m *Manager) installAgent(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(m.plistPath()), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.logPath()), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(m.plistPath(), m.plist(), 0o600); err != nil {
		return err
	}
	if _, err := m.launchctl(ctx, "bootstrap", m.launchDomain(), m.plistPath()); err != nil {
		return fmt.Errorf("launchctl bootstrap: %w", err)
	}
	return nil
}

func (m *Manager) removeAgent(ctx context.Context) error {
	err := m.bootout(ctx)
	if rerr := os.Remove(m.plistPath()); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
		err = errors.Join(err, rerr)
	}
	return err
}

// serveOn publishes the port. Output is passed through because the first run
// may print a link for turning on HTTPS and wait for the user to approve it.
func (m *Manager) serveOn(ctx context.Context, ts string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, ts, "serve", "--bg", "--https="+httpsPort, strconv.Itoa(m.Port))
	cmd.Stdout, cmd.Stderr, cmd.Stdin = m.Out, m.Out, m.In
	return cmd.Run()
}

// On turns phone access on. Refusals happen before anything changes; if a
// later step fails or the user cancels, phone access ends up fully off.
func (m *Manager) On(ctx context.Context) error {
	ts, err := m.findTailscale()
	if err != nil {
		return err
	}
	access, err := m.deviceAccess(ctx, ts)
	if err != nil {
		return err
	}
	sc, err := m.readServe(ctx, ts)
	if err != nil {
		return err
	}
	if err := m.checkServable(sc, ts, access.Host); err != nil {
		return err
	}
	if err := m.switchOn(ctx, ts, access); err != nil {
		return m.rollback(ctx, ts, err)
	}
	return m.printOn(access.Host)
}

func (m *Manager) switchOn(ctx context.Context, ts string, access Access) error {
	fmt.Fprintln(m.Out, "Punisher를 백그라운드로 시작하는 중이에요. 30초쯤 걸릴 수 있어요.")
	if err := m.bootout(ctx); err != nil {
		return err
	}
	if !m.waitUntil(ctx, func() bool { return m.PortFree(m.Port) }) {
		if m.IsPunisher(m.Port) {
			return errors.New("Punisher가 이미 다른 터미널 창에서 실행 중이에요. 그 창에서 Ctrl+C로 끈 뒤 다시 실행해 주세요.")
		}
		return fmt.Errorf("다른 프로그램이 %d 포트를 쓰고 있어요. 폰 연결에는 이 포트가 꼭 필요해요. Mac을 재시동한 뒤 바로 다시 실행해 주세요.", m.Port)
	}
	if err := saveAccess(m.DataDir, access); err != nil {
		return err
	}
	if err := m.installAgent(ctx); err != nil {
		return err
	}
	if !m.waitUntil(ctx, func() bool { return m.IsPunisher(m.Port) }) {
		return fmt.Errorf("Punisher를 백그라운드로 시작하지 못해서 폰 연결을 껐어요. 다시 실행해 보고, 그래도 안 되면 터미널에 아래를 입력해 나온 내용을 Punisher를 준 사람에게 보여 주세요.\n  open %q", m.logPath())
	}
	fmt.Fprint(m.Out, `
Tailscale에 연결하는 중이에요.
처음이라면 아래에 https://login.tailscale.com/... 링크가 나와요.
Command(⌘) 키를 누른 채 링크를 클릭하고, 열린 화면에서 켜기(Enable)를 누르세요. 승인하면 자동으로 이어져요.

`)
	if err := m.serveOn(ctx, ts); err != nil {
		return fmt.Errorf("Tailscale 연결을 켜지 못해서 폰 연결을 껐어요. 링크 승인을 마치지 못했다면 punisher phone on 을 다시 실행해 주세요. (%w)", err)
	}
	after, err := m.readServe(ctx, ts)
	if err != nil {
		return err
	}
	key := access.Host + ":" + httpsPort
	if after.AllowFunnel[key] || !m.isOurs(after.Web[key]) {
		return errors.New("Tailscale 설정이 예상과 달라서 폰 연결을 껐어요. " + msgAskOwner)
	}
	return nil
}

// rollback leaves phone access fully off. It keeps working after ctx is
// cancelled, because Ctrl+C is exactly when cleaning up matters.
func (m *Manager) rollback(ctx context.Context, ts string, cause error) error {
	if ctx.Err() != nil {
		cause = errors.New("취소했어요. 폰 연결은 꺼진 상태예요. 다시 켜려면 punisher phone on 을 실행해 주세요.")
	}
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer cancel()
	errs := []error{cause, m.unpublish(cleanup, ts), m.removeAgent(cleanup), removeAccess(m.DataDir)}
	return errors.Join(errs...)
}

func (m *Manager) printOn(host string) error {
	url := "https://" + host + "/"
	fmt.Fprintf(m.Out, "\n폰 연결을 켰어요.\n\n폰에서 Tailscale을 켜고 이 주소를 열거나 QR을 찍어 주세요.\n\n  %s\n\n", url)
	if err := writeQR(m.Out, url); err != nil {
		return err
	}
	fmt.Fprint(m.Out, `
이 주소는 내 Tailscale 계정의 기기에서만 열려요.

[중요] 이 Mac이 잠들면(맥북 뚜껑을 닫아도) 폰에서 열리지 않아요.
       밖에서 보려면 전원을 꽂고 뚜껑을 열어 둔 채로 나가세요. 화면은 꺼져도 괜찮아요.
       (안내문 4번의 배터리 설정도 켜 두어야 해요.)

이 터미널 창은 닫아도 돼요. 이제 Mac에 로그인하면 Punisher가 자동으로 켜져요.
끄려면: punisher phone off
`)
	return nil
}

// Off undoes On. It always tries every step and removes only what On created:
// another service published on this Mac is left alone.
func (m *Manager) Off(ctx context.Context) error {
	// Stopping halfway would leave the agent loaded with its plist gone.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer cancel()
	var tsErr error
	ts, err := m.findTailscale()
	if err == nil {
		tsErr = m.unpublish(ctx, ts)
	} else {
		fmt.Fprintln(m.Out, "Tailscale을 찾지 못해 Tailscale 쪽 설정은 건너뛰었어요.")
	}
	if err := errors.Join(m.removeAgent(ctx), removeAccess(m.DataDir)); err != nil {
		return errors.Join(err, tsErr)
	}
	if tsErr != nil {
		return fmt.Errorf("폰 연결은 껐어요. 이제 폰에서 열리지 않아요.\nTailscale 쪽 설정은 지우지 못했어요. Tailscale에 로그인한 뒤 punisher phone off 를 한 번 더 실행해 주세요. (%w)", tsErr)
	}
	fmt.Fprintln(m.Out, "폰 연결을 껐어요. 이제 이 Mac에서만 열리고, 로그인할 때 자동으로 켜지지도 않아요.")
	return nil
}

// unpublish removes the Serve entry On created, wherever the name now points.
func (m *Manager) unpublish(ctx context.Context, ts string) error {
	sc, err := m.readServe(ctx, ts)
	if err != nil {
		return err
	}
	ours, shared := false, false
	for key, web := range sc.Web {
		if !strings.HasSuffix(key, ":"+httpsPort) {
			continue
		}
		ours = ours || m.isOurs(web)
		shared = shared || (m.isOurProxy(web.Handlers["/"].Proxy) && !m.isOurs(web))
	}
	if shared {
		fmt.Fprintln(m.Out, "Tailscale 주소를 다른 프로그램과 함께 쓰고 있어서 Tailscale 설정은 그대로 두었어요.")
	}
	if !ours {
		return nil
	}
	if _, err := runOutput(ctx, ts, "serve", "--https="+httpsPort, "off"); err != nil {
		return fmt.Errorf("Tailscale Serve를 끄지 못했어요: %w", err)
	}
	return nil
}
