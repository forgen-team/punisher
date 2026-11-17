// Command punisher runs the standalone Punisher learning app: a local web
// server on 127.0.0.1 with the UI embedded, SQLite storage and lecture
// generation through the locally installed Claude Code CLI.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/backup"
	"github.com/forgen-team/punisher/internal/claudecli"
	"github.com/forgen-team/punisher/internal/config"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/glossary"
	httpapi "github.com/forgen-team/punisher/internal/http"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/middleware"
	"github.com/forgen-team/punisher/internal/phone"
	"github.com/forgen-team/punisher/internal/scheduler"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/forgen-team/punisher/internal/webui"
	"github.com/forgen-team/punisher/migrations"
)

// version is set at build time (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "phone" {
		os.Exit(runPhone(os.Args[2:]))
	}
	if len(os.Args) > 1 && os.Args[1] == "samples" {
		os.Exit(runSamples(os.Args[2:]))
	}
	if len(os.Args) > 1 && (os.Args[1] == "export" || os.Args[1] == "import") {
		os.Exit(runBackup(os.Args[1], os.Args[2:]))
	}
	cfg := config.Load()
	showVersion := flag.Bool("version", false, tr("버전을 출력하고 종료", "print the version and exit"))
	flag.IntVar(&cfg.Port, "port", cfg.Port, tr("포트 (0 = 4710, 사용 중이면 빈 포트)", "port (0 = 4710, or a free port if taken)"))
	flag.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, tr("데이터 폴더 (punisher.db 위치)", "data folder (where punisher.db lives)"))
	noBrowser := flag.Bool("no-browser", false, tr("시작할 때 브라우저를 열지 않음", "do not open the browser on start"))
	flag.Usage = func() {
		flag.CommandLine.SetOutput(os.Stderr)
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\n"+phoneUsage+"\n\n"+backupUsage)
	}
	flag.Parse()
	if flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, phoneUsage)
		os.Exit(2)
	}
	if *showVersion {
		fmt.Printf("punisher %s\npart of the forgen family — https://github.com/forgen-team/forgen\n© 2026 jang-ujin · MIT License\n", version)
		return
	}
	if *noBrowser {
		cfg.OpenBrowser = false
	}
	if err := run(cfg); err != nil {
		log.Fatalf("punisher: %v", err)
	}
}

const phoneUsage = `폰 연결 (순서는 README.ko.md의 8번을 보세요):
  punisher phone on    내 폰에서 열 수 있게 켜기. Tailscale이 필요하고, 로그인할 때 자동으로 켜져요
  punisher phone off   끄기. 자동 시작도 함께 꺼져요`

// runPhone handles `punisher phone on|off` and returns the exit code.
func runPhone(args []string) int {
	if len(args) != 1 || (args[0] != "on" && args[0] != "off") {
		fmt.Fprintln(os.Stderr, phoneUsage)
		return 2
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		fmt.Fprintln(os.Stderr, tr("폰 연결은 macOS와 Linux에서만 쓸 수 있어요.", "Phone access is available on macOS and Linux only."))
		return 1
	}
	m, err := newPhoneManager()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// The first Ctrl+C cleans up; a second one is free to end the process.
	context.AfterFunc(ctx, stop)
	if args[0] == "on" {
		err = m.On(ctx)
	} else {
		err = m.Off(ctx)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func newPhoneManager() (*phone.Manager, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf(tr("실행 파일 위치를 찾지 못했어요: %w", "could not find the executable: %w"), err)
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return nil, fmt.Errorf(tr("실행 파일 위치를 찾지 못했어요: %w", "could not find the executable: %w"), err)
	}
	if strings.Contains(exe, "/go-build") {
		return nil, errors.New(tr("go run 으로는 켤 수 없어요. 설치한 punisher 로 실행해 주세요.", "this cannot be enabled from go run; run the installed punisher instead"))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf(tr("홈 폴더를 찾지 못했어요: %w", "could not find the home folder: %w"), err)
	}
	cfg := config.Load()
	// launchd starts with a minimal environment; keep what finding claude needs
	// and this user's Punisher settings.
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "PATH=") || strings.HasPrefix(kv, "LANG=") || strings.HasPrefix(kv, "LC_ALL=") ||
			strings.HasPrefix(kv, "LC_MESSAGES=") || strings.HasPrefix(kv, "PUNISHER_") {
			env = append(env, kv)
		}
	}
	return phone.New(cfg.DataDir, cfg.DataDir != config.DefaultDataDir(), exe, home, env), nil
}

// loadPhoneAccess reads the phone-access settings. A broken file means phone
// access stays off rather than failing the app.
func loadPhoneAccess(dataDir string) phone.Access {
	access, err := phone.LoadAccess(dataDir)
	if err != nil {
		log.Printf(tr("폰 연결 설정을 무시합니다: %v", "ignoring the phone access setting: %v"), err)
		return phone.Access{}
	}
	return access
}

func run(cfg config.Config) error {
	ln, existing, err := listen(cfg.Port)
	if err != nil {
		return err
	}
	if existing != "" {
		fmt.Printf(tr("Punisher가 이미 실행 중입니다: %s\n", "Punisher is already running: %s\n"), existing)
		if cfg.OpenBrowser {
			openBrowser(existing)
		}
		return nil
	}
	url := "http://" + ln.Addr().String()

	// 같은 데이터 폴더를 두 번 열지 않게(가져오기와 겹치지 않게) 잠근다.
	lock, err := backup.Lock(cfg.DataDir)
	if err != nil {
		_ = ln.Close()
		return err
	}
	defer backup.Unlock(lock)
	ctx := context.Background()
	dbPath := filepath.Join(cfg.DataDir, "punisher.db")
	st, err := store.Open(dbPath)
	if err != nil {
		_ = ln.Close()
		return err
	}
	defer st.Close()
	// 설정을 저장한 적 없는 사용자의 언어는 OS 로캘(PUNISHER_LOCALE 로 덮어쓰기)을 따른다.
	store.SetDefaultLocale(cfg.Locale)
	if err := st.Migrate(ctx, migrations.Files); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	if err := syncCurriculum(ctx, cfg, st); err != nil {
		return fmt.Errorf("curriculum seed: %w", err)
	}
	seedSamples(ctx, st)
	// 확인 문항 만들기·채점은 이 프로세스 안에서만 돌므로, 지난 실행이 남긴 진행 중 작업은 끝나지 않는다.
	if r, err := st.RecoverInterruptedChecks(ctx); err != nil {
		log.Printf("recover interrupted checks: %v", err)
	} else if r != (store.RecoveredChecks{}) {
		log.Printf("recovered interrupted checks: %d jobs, %d sessions, %d turns, %d exercise attempts", r.Jobs, r.Sessions, r.Turns, r.Attempts)
	}

	claude := claudecli.New(claudecli.Config{
		Bin:          cfg.ClaudeBin,
		TempBase:     filepath.Join(cfg.DataDir, "tmp"),
		Concurrency:  cfg.Concurrency,
		DefaultModel: cfg.LectureModel,
		DailyLimit:   cfg.DailyCallLimit,
	})
	terms, err := glossary.Load(seedfiles.Glossary)
	if err != nil {
		log.Printf("glossary: %v (term glosses are generated instead)", err)
	}
	svc := &services.Service{
		Glossary: terms,
		Store:    st,
		LLM:      claude,
		Keys:     services.GatewayKeys{},
		Hub:      services.NewHub(),
		Cfg: services.Config{
			LectureModel:      cfg.LectureModel,
			FastModel:         cfg.FastModel,
			OwnerUserID:       middleware.LocalUserID,
			DailyLectureLimit: cfg.DailyLectureLimit,
			DailyTokenLimit:   cfg.DailyTokenLimit,
			LibraryAhead:      cfg.LibraryAhead,
			NightAhead:        cfg.NightAhead,
			NightStartHour:    cfg.NightStartHour,
			NightEndHour:      cfg.NightEndHour,
			StudyPregenerate:  cfg.StudyPregenerate,
			LectureMaxTokens:  cfg.LectureMaxTokens,
			// 단독판은 사용자가 한 명이라 라이브러리(추천 미리 만들기) 강의도 그 사람의 프롬프트를 따른다.
			LibraryUsesUserPrompts: true,

			CheckModel:           cfg.CheckModel,
			CheckRegradeModel:    cfg.CheckRegradeModel,
			DailyCheckTokenLimit: cfg.DailyCheckTokenLimit,

			BackfillReviewEnabled: cfg.BackfillReviewEnabled,
		},
	}

	access := loadPhoneAccess(cfg.DataDir)
	// No WriteTimeout: instant lectures stream over SSE for minutes.
	srv := &http.Server{
		Handler: httpapi.NewRouter(httpapi.RouterDependencies{
			Store: st, Services: svc, Claude: claude, UI: webui.Handler(), AppVersion: version, DataDir: cfg.DataDir,
			AllowedHost: access.Host, AllowedLogin: access.Login,
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	schedCtx, stopScheduler := context.WithCancel(llm.WithBackground(context.Background()))
	defer stopScheduler()
	if cfg.SchedulerEnabled {
		sched := &scheduler.Scheduler{Svc: svc, Interval: cfg.SchedulerInterval, InitialDelay: 2 * time.Minute}
		go sched.Run(schedCtx)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	reportClaude(ctx, claude)
	fmt.Printf(tr("\nPunisher 실행 중: %s\n데이터: %s\n종료하려면 이 창에서 Ctrl+C 를 누르세요.\n\n", "\nPunisher is running: %s\nData: %s\nPress Ctrl+C in this window to quit.\n\n"), url, dbPath)
	if cfg.OpenBrowser {
		openBrowser(url)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-stop:
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server: %w", err)
		}
	}
	fmt.Println(tr("종료합니다...", "Shutting down..."))
	stopScheduler()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	return nil
}

// listen binds 127.0.0.1 on the requested port (default 4710). When the
// default port is taken by another running Punisher its URL is returned so
// the caller can just open it; when taken by something else a free port is
// used instead.
func listen(port int) (net.Listener, string, error) {
	explicit := port != 0
	if port == 0 {
		port = config.DefaultPort
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		return ln, "", nil
	}
	if url := "http://" + addr; isPunisher(url) {
		return nil, url, nil
	}
	if explicit {
		return nil, "", fmt.Errorf(tr("포트 %d 를 사용할 수 없습니다: %w", "port %d is not available: %w"), port, err)
	}
	ln, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", fmt.Errorf("listen: %w", err)
	}
	return ln, "", nil
}

func isPunisher(url string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url + "/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var body struct {
		Service string `json:"service"`
	}
	return json.NewDecoder(resp.Body).Decode(&body) == nil && body.Service == "punisher"
}

func reportClaude(ctx context.Context, c *claudecli.Client) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	st := c.Status(ctx)
	switch st.Code {
	case claudecli.CodeCLIMissing:
		fmt.Println(tr("[주의] "+services.MsgClaudeMissing, "[warning] Claude Code (the claude command) was not found. Install Claude Code, run `claude` once in a terminal to sign in, then try again."))
	case claudecli.CodeNotLoggedIn:
		fmt.Println(tr("[주의] "+services.MsgClaudeLoggedOut, "[warning] Claude Code is not signed in. Run `claude` in a terminal and sign in (/login), then try again."))
	default:
		fmt.Printf(tr("Claude Code 연결됨 (%s)\n", "Claude Code connected (%s)\n"), st.Version)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf(tr("브라우저를 열지 못했습니다(%v). 직접 %s 를 여세요.", "Could not open the browser (%v). Open %s yourself."), err, url)
		return
	}
	go func() { _ = cmd.Wait() }()
}

// syncCurriculum loads the embedded seed (or PUNISHER_CURRICULUM_DIR in dev),
// validates it and upserts it in one transaction. An invalid seed aborts startup.
func syncCurriculum(ctx context.Context, cfg config.Config, st store.Store) error {
	var source fs.FS = seedfiles.Files
	if cfg.CurriculumDir != "" {
		source = os.DirFS(cfg.CurriculumDir)
		log.Printf("loading curriculum from %s", cfg.CurriculumDir)
	}
	seed, err := curriculum.Load(source)
	if err != nil {
		return err
	}
	applied, err := st.SyncCurriculum(ctx, seed)
	if err != nil {
		return err
	}
	if applied {
		log.Printf("curriculum %s applied: %d concepts, %d edges, %d aliases",
			seed.Version[:12], len(seed.Concepts), len(seed.Edges), len(seed.Aliases))
	}
	return nil
}
