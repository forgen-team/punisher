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
	"syscall"
	"time"

	seedfiles "github.com/wooo-jin/punisher/curriculum"
	"github.com/wooo-jin/punisher/internal/claudecli"
	"github.com/wooo-jin/punisher/internal/config"
	"github.com/wooo-jin/punisher/internal/curriculum"
	httpapi "github.com/wooo-jin/punisher/internal/http"
	"github.com/wooo-jin/punisher/internal/llm"
	"github.com/wooo-jin/punisher/internal/middleware"
	"github.com/wooo-jin/punisher/internal/scheduler"
	"github.com/wooo-jin/punisher/internal/services"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/webui"
	"github.com/wooo-jin/punisher/migrations"
)

// version is set at build time (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	cfg := config.Load()
	showVersion := flag.Bool("version", false, "버전을 출력하고 종료")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "포트 (0 = 4710, 사용 중이면 빈 포트)")
	flag.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "데이터 폴더 (punisher.db 위치)")
	noBrowser := flag.Bool("no-browser", false, "시작할 때 브라우저를 열지 않음")
	flag.Parse()
	if *showVersion {
		fmt.Printf("punisher %s\npart of the forgen family — https://github.com/wooo-jin/forgen\n© 2026 jang-ujin · MIT License\n", version)
		return
	}
	if *noBrowser {
		cfg.OpenBrowser = false
	}
	if err := run(cfg); err != nil {
		log.Fatalf("punisher: %v", err)
	}
}

func run(cfg config.Config) error {
	ln, existing, err := listen(cfg.Port)
	if err != nil {
		return err
	}
	if existing != "" {
		fmt.Printf("Punisher가 이미 실행 중입니다: %s\n", existing)
		if cfg.OpenBrowser {
			openBrowser(existing)
		}
		return nil
	}
	url := "http://" + ln.Addr().String()

	ctx := context.Background()
	dbPath := filepath.Join(cfg.DataDir, "punisher.db")
	st, err := store.Open(dbPath)
	if err != nil {
		_ = ln.Close()
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx, migrations.Files); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	if err := syncCurriculum(ctx, cfg, st); err != nil {
		return fmt.Errorf("curriculum seed: %w", err)
	}

	claude := claudecli.New(claudecli.Config{
		Bin:          cfg.ClaudeBin,
		TempBase:     filepath.Join(cfg.DataDir, "tmp"),
		Concurrency:  cfg.Concurrency,
		DefaultModel: cfg.LectureModel,
		DailyLimit:   cfg.DailyCallLimit,
	})
	svc := &services.Service{
		Store: st,
		LLM:   claude,
		Keys:  services.GatewayKeys{},
		Hub:   services.NewHub(),
		Cfg: services.Config{
			LectureModel:      cfg.LectureModel,
			FastModel:         cfg.FastModel,
			OwnerUserID:       middleware.LocalUserID,
			DailyLectureLimit: cfg.DailyLectureLimit,
			DailyTokenLimit:   cfg.DailyTokenLimit,
			LibraryAhead:      cfg.LibraryAhead,
			StudyPregenerate:  cfg.StudyPregenerate,
			LectureMaxTokens:  cfg.LectureMaxTokens,
			// 단독판은 사용자가 한 명이라 라이브러리(추천 미리 만들기) 강의도 그 사람의 프롬프트를 따른다.
			LibraryUsesUserPrompts: true,
		},
	}

	// No WriteTimeout: instant lectures stream over SSE for minutes.
	srv := &http.Server{
		Handler: httpapi.NewRouter(httpapi.RouterDependencies{
			Store: st, Services: svc, Claude: claude, UI: webui.Handler(), AppVersion: version, DataDir: cfg.DataDir,
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
	fmt.Printf("\nPunisher 실행 중: %s\n데이터: %s\n종료하려면 이 창에서 Ctrl+C 를 누르세요.\n\n", url, dbPath)
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
	fmt.Println("종료합니다...")
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
		return nil, "", fmt.Errorf("포트 %d 를 사용할 수 없습니다: %w", port, err)
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
		fmt.Println("[주의] " + services.MsgClaudeMissing)
	case claudecli.CodeNotLoggedIn:
		fmt.Println("[주의] " + services.MsgClaudeLoggedOut)
	default:
		fmt.Printf("Claude Code 연결됨 (%s)\n", st.Version)
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
		log.Printf("브라우저를 열지 못했습니다(%v). 직접 %s 를 여세요.", err, url)
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
