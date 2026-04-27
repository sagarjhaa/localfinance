// LocalFinance launcher — single-binary orchestrator for the macOS .app bundle.
//
// Responsibilities:
//   1. Resolve runtime layout (works inside .app bundle or in dev mode).
//   2. Verify Ollama is reachable and the chosen model is pulled.
//   3. Start an embedded Postgres (or system Postgres via brew fallback).
//   4. Spawn the 4 Go services + Iris (Node) as child processes.
//   5. Wait for Iris /api/health, then open the browser.
//   6. Forward signals; tear children down on shutdown or any-child-crash.
//
// Stdlib only — no module dependencies.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	defaultChatModel = "llama3.1:8b"
	defaultOllamaURL = "http://127.0.0.1:11434"

	pgPort        = 15432
	thesaurusPort = 8001
	sophiaPort    = 8002
	logosPort     = 8003
	hermesPort    = 3000
	irisPort      = 3001

	dbName     = "localfinance"
	dbUser     = "localfinance"
	dbPassword = "localfinance"

	jwtSecret = "local-installer-jwt-secret-change-me"
)

// Resources is the resolved layout of bundled assets and the chosen data dir.
type Resources struct {
	BundleRoot string // e.g. /Applications/LocalFinance.app  (empty if dev mode)
	BinDir     string // dir containing thesaurus/sophia/logos/hermes/node/iris-server.js
	IrisStatic string // dir to serve as Iris static (services/iris/client/build)
	PgBinDir   string // dir containing initdb/pg_ctl/postgres (may be empty -> use PATH)
	NodeBin    string // node executable (may be empty -> use PATH)
	IrisServer string // path to bundled iris-server.js
	DataDir    string // ~/Library/Application Support/LocalFinance (or override)
	LogDir     string
	PgDataDir  string
	UploadDir  string
}

// findResources walks up from the launcher executable to locate bundled assets.
// Layout when bundled:
//
//	LocalFinance.app/Contents/MacOS/launcher  <-- exePath
//	LocalFinance.app/Contents/Resources/bin/{thesaurus,sophia,logos,hermes,node,iris-server.js}
//	LocalFinance.app/Contents/Resources/bin/postgres/bin/{initdb,pg_ctl,postgres}
//	LocalFinance.app/Contents/Resources/iris-static/
//
// In dev mode (running `go run ./installer/launcher`), it falls back to repo dist/.
func findResources(exePath string, dataDir string) (*Resources, error) {
	abs, err := filepath.Abs(exePath)
	if err != nil {
		return nil, err
	}
	r := &Resources{DataDir: dataDir}

	macOSDir := filepath.Dir(abs)                 // .../Contents/MacOS
	contentsDir := filepath.Dir(macOSDir)         // .../Contents
	bundleRoot := filepath.Dir(contentsDir)       // .../LocalFinance.app
	resourcesDir := filepath.Join(contentsDir, "Resources")

	if filepath.Base(macOSDir) == "MacOS" && filepath.Base(contentsDir) == "Contents" {
		// Looks like a real .app bundle.
		r.BundleRoot = bundleRoot
		r.BinDir = filepath.Join(resourcesDir, "bin")
		r.IrisStatic = filepath.Join(resourcesDir, "iris-static")
		r.IrisServer = filepath.Join(r.BinDir, "iris-server.js")
		nodeCandidate := filepath.Join(r.BinDir, "node")
		if fileExists(nodeCandidate) {
			r.NodeBin = nodeCandidate
		}
		pgCandidate := filepath.Join(r.BinDir, "postgres", "bin")
		if dirExists(pgCandidate) {
			r.PgBinDir = pgCandidate
		}
	} else {
		// Dev mode: assume we're in repo root, use dist/.
		repoRoot, _ := os.Getwd()
		r.BinDir = filepath.Join(repoRoot, "dist")
		r.IrisStatic = filepath.Join(repoRoot, "services", "iris", "client", "build")
		r.IrisServer = filepath.Join(r.BinDir, "iris-server.js")
	}

	// Defaults that fall back to PATH-resolved binaries if the bundle didn't ship them.
	if r.NodeBin == "" {
		if p, err := exec.LookPath("node"); err == nil {
			r.NodeBin = p
		}
	}

	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		r.DataDir = filepath.Join(home, "Library", "Application Support", "LocalFinance")
	}
	r.LogDir = filepath.Join(r.DataDir, "logs")
	r.PgDataDir = filepath.Join(r.DataDir, "pgdata")
	r.UploadDir = filepath.Join(r.DataDir, "uploads")

	return r, nil
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// freePort picks an unused TCP port on 127.0.0.1.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForHealthy polls url until 200 OK or timeout.
func waitForHealthy(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("timed out")
	}
	return fmt.Errorf("waitForHealthy(%s): %w", url, lastErr)
}

// waitForTCP polls until a TCP connect succeeds or timeout.
func waitForTCP(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 1*time.Second)
		if err == nil {
			c.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("waitForTCP(%s): timed out", addr)
}

// ollamaTags fetches the list of installed Ollama models. Returns names like "llama3.1:8b".
func ollamaTags(baseURL string) ([]string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(parsed.Models))
	for _, m := range parsed.Models {
		out = append(out, m.Name)
	}
	return out, nil
}

// hasModel checks whether the requested model name appears in the tag list.
// Ollama tags include the ":tag" suffix. We require an exact match.
func hasModel(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

// Args parsed from the CLI.
type Args struct {
	ChatModel   string
	SkipBrowser bool
	DataDir     string
	OllamaURL   string
}

func parseArgs(argv []string) (Args, error) {
	fs := flag.NewFlagSet("launcher", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	chat := fs.String("chat-model", defaultChatModel, "Ollama model name to use for Sophia")
	skip := fs.Bool("skip-browser", false, "Don't open the browser when ready (useful for tests)")
	data := fs.String("data-dir", "", "Override data directory (default: ~/Library/Application Support/LocalFinance)")
	ollama := fs.String("ollama-url", defaultOllamaURL, "Base URL for the local Ollama server")
	if err := fs.Parse(argv); err != nil {
		return Args{}, err
	}
	return Args{ChatModel: *chat, SkipBrowser: *skip, DataDir: *data, OllamaURL: *ollama}, nil
}

// child wraps an os/exec.Cmd plus its log file for tidy shutdown.
type child struct {
	name string
	cmd  *exec.Cmd
	log  *os.File
}

func (c *child) terminate() {
	if c.cmd == nil || c.cmd.Process == nil {
		return
	}
	_ = c.cmd.Process.Signal(syscall.SIGTERM)
}

func (c *child) kill() {
	if c.cmd == nil || c.cmd.Process == nil {
		return
	}
	_ = c.cmd.Process.Kill()
}

// Launcher is the runtime supervisor.
type Launcher struct {
	res      *Resources
	args     Args
	children []*child
	mu       sync.Mutex
	exitCh   chan childExit
	pgEnv    []string // DYLD_FALLBACK_LIBRARY_PATH etc. for Postgres binaries
}

type childExit struct {
	name string
	err  error
}

func newLauncher(res *Resources, args Args) *Launcher {
	return &Launcher{res: res, args: args, exitCh: make(chan childExit, 16)}
}

// dyldFallbackPath returns a colon-joined list of Homebrew opt/*/lib dirs to
// satisfy dynamic library deps for the bundled Postgres binaries (icu4c,
// openssl@1.1, krb5, etc.). Empty if no Homebrew prefix is found.
func dyldFallbackPath() string {
	var out []string
	add := func(lib string) {
		if dirExists(lib) {
			out = append(out, lib)
		}
	}
	// 1. Homebrew opt/<pkg>/lib (newest installed version of each formula).
	for _, prefix := range []string{"/opt/homebrew/opt", "/usr/local/opt"} {
		entries, err := os.ReadDir(prefix)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				add(filepath.Join(prefix, e.Name(), "lib"))
			}
		}
	}
	// 2. Homebrew Cellar/<pkg>/<version>/lib — covers the case where a formula
	//    has multiple versions installed and the binary needs an older one
	//    (e.g. postgresql@14 needs icu4c v67 but icu4c@78 is the linked one).
	for _, cellar := range []string{"/opt/homebrew/Cellar", "/usr/local/Cellar"} {
		pkgs, err := os.ReadDir(cellar)
		if err != nil {
			continue
		}
		for _, pkg := range pkgs {
			if !pkg.IsDir() {
				continue
			}
			versions, err := os.ReadDir(filepath.Join(cellar, pkg.Name()))
			if err != nil {
				continue
			}
			for _, v := range versions {
				if v.IsDir() {
					add(filepath.Join(cellar, pkg.Name(), v.Name(), "lib"))
				}
			}
		}
	}
	return strings.Join(out, ":")
}

// spawn starts a child process with merged env, capturing stdout/stderr to a log file.
func (l *Launcher) spawn(name, binary string, args []string, env []string) (*child, error) {
	logPath := filepath.Join(l.res.LogDir, name+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log %s: %w", logPath, err)
	}
	cmd := exec.Command(binary, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(), env...)
	// Put the child in its own process group so we can signal the whole group.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("start %s: %w", name, err)
	}
	c := &child{name: name, cmd: cmd, log: logFile}
	l.mu.Lock()
	l.children = append(l.children, c)
	l.mu.Unlock()

	go func() {
		err := cmd.Wait()
		logFile.Close()
		l.exitCh <- childExit{name: name, err: err}
	}()
	fmt.Printf("  → started %s (pid %d) — log: %s\n", name, cmd.Process.Pid, logPath)
	return c, nil
}

// shutdownAll signals every child SIGTERM, waits up to 10s, then SIGKILLs survivors.
func (l *Launcher) shutdownAll() {
	l.mu.Lock()
	children := append([]*child(nil), l.children...)
	l.mu.Unlock()

	fmt.Println("Stopping all services...")
	for _, c := range children {
		c.terminate()
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		alive := false
		for _, c := range children {
			if c.cmd.ProcessState == nil {
				alive = true
				break
			}
		}
		if !alive {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	for _, c := range children {
		if c.cmd.ProcessState == nil {
			fmt.Printf("  ! force-killing %s\n", c.name)
			c.kill()
		}
	}
}

// startPostgres initdbs (if needed) and starts Postgres on pgPort. Idempotent.
func (l *Launcher) startPostgres() error {
	pgInitdb := l.pgBinary("initdb")
	pgCtl := l.pgBinary("pg_ctl")
	pgPostgres := l.pgBinary("postgres")
	if pgInitdb == "" || pgCtl == "" || pgPostgres == "" {
		return errors.New(`postgres binaries not found.

LocalFinance needs Postgres installed once on your Mac:
    brew install postgresql@15

(See installer/POSTGRES.md for why Postgres is not bundled in Phase 1.)`)
	}

	if err := os.MkdirAll(l.res.PgDataDir, 0700); err != nil {
		return err
	}
	// DYLD_FALLBACK_LIBRARY_PATH lets the bundled (Homebrew-sourced) Postgres
	// binaries locate icu4c/openssl/krb5 dylibs via Homebrew prefixes at
	// runtime. Also include the bundle's own Postgres lib dir.
	dyld := dyldFallbackPath()
	if l.res.PgBinDir != "" {
		bundleLib := filepath.Join(filepath.Dir(l.res.PgBinDir), "lib")
		if dirExists(bundleLib) {
			if dyld == "" {
				dyld = bundleLib
			} else {
				dyld = bundleLib + ":" + dyld
			}
		}
	}
	pgEnv := []string{}
	if dyld != "" {
		pgEnv = append(pgEnv, "DYLD_FALLBACK_LIBRARY_PATH="+dyld)
	}
	l.pgEnv = pgEnv

	pgVerFile := filepath.Join(l.res.PgDataDir, "PG_VERSION")
	if !fileExists(pgVerFile) {
		fmt.Println("Initializing Postgres data directory...")
		// Use trust auth on local socket — single-user installer, no network exposure.
		cmd := exec.Command(pgInitdb,
			"-D", l.res.PgDataDir,
			"-U", dbUser,
			"--auth-local=trust",
			"--auth-host=trust",
			"-E", "UTF8",
		)
		cmd.Env = append(os.Environ(), pgEnv...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("initdb: %w", err)
		}
	}

	// Spawn `postgres` directly (not via pg_ctl) so it's a normal child of the launcher.
	args := []string{
		"-D", l.res.PgDataDir,
		"-p", strconv.Itoa(pgPort),
		"-k", l.res.PgDataDir, // unix socket dir under data dir
		"-c", "listen_addresses=127.0.0.1",
		"-c", "unix_socket_directories=" + l.res.PgDataDir,
	}
	if _, err := l.spawn("postgres", pgPostgres, args, pgEnv); err != nil {
		return err
	}

	addr := fmt.Sprintf("127.0.0.1:%d", pgPort)
	fmt.Printf("Waiting for Postgres on %s...\n", addr)
	if err := waitForTCP(addr, 30*time.Second); err != nil {
		return err
	}
	// Allow a beat for Postgres to finish accepting connections.
	time.Sleep(500 * time.Millisecond)
	if err := l.ensureDatabase(); err != nil {
		return fmt.Errorf("ensure db: %w", err)
	}
	return nil
}

// ensureDatabase creates the localfinance database if it doesn't exist using psql.
func (l *Launcher) ensureDatabase() error {
	psql := l.pgBinary("psql")
	if psql == "" {
		return errors.New("psql not found")
	}
	check := exec.Command(psql,
		"-h", "127.0.0.1", "-p", strconv.Itoa(pgPort),
		"-U", dbUser, "-d", "postgres",
		"-tAc", fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", dbName),
	)
	check.Env = append(os.Environ(), l.pgEnv...)
	out, err := check.Output()
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(out)) == "1" {
		return nil
	}
	create := exec.Command(psql,
		"-h", "127.0.0.1", "-p", strconv.Itoa(pgPort),
		"-U", dbUser, "-d", "postgres",
		"-c", fmt.Sprintf("CREATE DATABASE %s OWNER %s", dbName, dbUser),
	)
	create.Env = append(os.Environ(), l.pgEnv...)
	create.Stdout = os.Stdout
	create.Stderr = os.Stderr
	return create.Run()
}

// pgBinary resolves a Postgres CLI binary, preferring bundled ones, then PATH.
func (l *Launcher) pgBinary(name string) string {
	if l.res.PgBinDir != "" {
		p := filepath.Join(l.res.PgBinDir, name)
		if fileExists(p) {
			return p
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

// dbEnv returns the env vars every Go service expects for Postgres.
func (l *Launcher) dbEnv() []string {
	return []string{
		"DB_HOST=127.0.0.1",
		fmt.Sprintf("DB_PORT=%d", pgPort),
		"DB_USER=" + dbUser,
		"DB_PASSWORD=" + dbPassword,
		"DB_NAME=" + dbName,
		"DB_SSLMODE=disable",
		"JWT_SECRET=" + jwtSecret,
		"ENV=production",
	}
}

// startServices spawns thesaurus, sophia, logos, hermes, iris in order.
func (l *Launcher) startServices() error {
	thesaurusURL := fmt.Sprintf("http://127.0.0.1:%d", thesaurusPort)
	sophiaURL := fmt.Sprintf("http://127.0.0.1:%d", sophiaPort)
	logosURL := fmt.Sprintf("http://127.0.0.1:%d", logosPort)

	// 1. Thesaurus — every other service depends on it.
	if _, err := l.spawn("thesaurus", filepath.Join(l.res.BinDir, "thesaurus"), nil,
		append(l.dbEnv(), fmt.Sprintf("PORT=%d", thesaurusPort)),
	); err != nil {
		return err
	}
	if err := waitForHealthy(thesaurusURL+"/health", 30*time.Second); err != nil {
		return err
	}

	// 2. Sophia — needs Thesaurus + Ollama.
	sophiaEnv := append(l.dbEnv(),
		fmt.Sprintf("PORT=%d", sophiaPort),
		"OLLAMA_HOST="+l.args.OllamaURL,
		"MODEL_NAME="+l.args.ChatModel,
		"OLLAMA_KEEP_ALIVE=24h",
		"THESAURUS_URL="+thesaurusURL,
	)
	if _, err := l.spawn("sophia", filepath.Join(l.res.BinDir, "sophia"), nil, sophiaEnv); err != nil {
		return err
	}
	if err := waitForHealthy(sophiaURL+"/health", 60*time.Second); err != nil {
		return err
	}

	// 3. Logos — stateless processor.
	logosEnv := append(l.dbEnv(),
		fmt.Sprintf("PORT=%d", logosPort),
		"THESAURUS_URL="+thesaurusURL,
		"SOPHIA_URL="+sophiaURL,
		"OLLAMA_HOST="+l.args.OllamaURL,
		"TEMP_DIR="+l.res.UploadDir,
	)
	if _, err := l.spawn("logos", filepath.Join(l.res.BinDir, "logos"), nil, logosEnv); err != nil {
		return err
	}
	if err := waitForHealthy(logosURL+"/health", 30*time.Second); err != nil {
		return err
	}

	// 4. Hermes — gateway (currently passive but bundled for parity).
	hermesEnv := append(l.dbEnv(),
		fmt.Sprintf("PORT=%d", hermesPort),
		"THESAURUS_URL="+thesaurusURL,
		"SOPHIA_URL="+sophiaURL,
		"LOGOS_URL="+logosURL,
	)
	if _, err := l.spawn("hermes", filepath.Join(l.res.BinDir, "hermes"), nil, hermesEnv); err != nil {
		return err
	}
	// Hermes is best-effort; don't block on its health.

	// 5. Iris — Node + Express. Last so we can open the browser when it's healthy.
	if l.res.NodeBin == "" {
		return errors.New("no node binary found (bundle is missing Resources/bin/node and PATH has no node)")
	}
	if !fileExists(l.res.IrisServer) {
		return fmt.Errorf("iris-server.js not found at %s", l.res.IrisServer)
	}
	irisEnv := []string{
		fmt.Sprintf("PORT=%d", irisPort),
		"THESAURUS_URL=" + thesaurusURL,
		"SOPHIA_URL=" + sophiaURL,
		"LOGOS_URL=" + logosURL,
		"IRIS_STATIC_DIR=" + l.res.IrisStatic,
		"NODE_ENV=production",
	}
	if _, err := l.spawn("iris", l.res.NodeBin, []string{l.res.IrisServer}, irisEnv); err != nil {
		return err
	}
	if err := waitForHealthy(fmt.Sprintf("http://127.0.0.1:%d/api/health", irisPort), 60*time.Second); err != nil {
		return err
	}
	return nil
}

func openBrowser(url string) error {
	if runtime.GOOS != "darwin" {
		return errors.New("openBrowser only supports darwin")
	}
	return exec.Command("open", url).Start()
}

func main() {
	args, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "argument error:", err)
		os.Exit(2)
	}

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot resolve executable path:", err)
		os.Exit(1)
	}
	res, err := findResources(exe, args.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot resolve resources:", err)
		os.Exit(1)
	}
	for _, d := range []string{res.DataDir, res.LogDir, res.UploadDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Fprintln(os.Stderr, "mkdir:", err)
			os.Exit(1)
		}
	}

	fmt.Println("LocalFinance launcher starting")
	fmt.Println("  bundle:    ", res.BundleRoot)
	fmt.Println("  bin dir:   ", res.BinDir)
	fmt.Println("  data dir:  ", res.DataDir)
	fmt.Println("  chat model:", args.ChatModel)

	// 1. Verify Ollama is up.
	tags, err := ollamaTags(args.OllamaURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "ERROR: Cannot reach Ollama at", args.OllamaURL)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "LocalFinance needs a local Ollama server.")
		fmt.Fprintln(os.Stderr, "Install and start it with:")
		fmt.Fprintln(os.Stderr, "    brew install ollama")
		fmt.Fprintln(os.Stderr, "    ollama serve")
		fmt.Fprintln(os.Stderr, "")
		os.Exit(1)
	}
	if !hasModel(tags, args.ChatModel) {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "ERROR: Ollama is running but model %q is not pulled.\n", args.ChatModel)
		fmt.Fprintln(os.Stderr, "Pull it with:")
		fmt.Fprintf(os.Stderr, "    ollama pull %s\n\n", args.ChatModel)
		fmt.Fprintln(os.Stderr, "Available models:", strings.Join(tags, ", "))
		os.Exit(1)
	}
	fmt.Println("  ollama ok, model present:", args.ChatModel)

	l := newLauncher(res, args)

	// SIGINT/SIGTERM forwarding.
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 2. Postgres.
	if err := l.startPostgres(); err != nil {
		fmt.Fprintln(os.Stderr, "postgres start failed:", err)
		l.shutdownAll()
		os.Exit(1)
	}

	// 3. Services.
	if err := l.startServices(); err != nil {
		fmt.Fprintln(os.Stderr, "service start failed:", err)
		l.shutdownAll()
		os.Exit(1)
	}

	url := fmt.Sprintf("http://localhost:%d", irisPort)
	fmt.Println("LocalFinance is ready at", url)
	if !args.SkipBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Fprintln(os.Stderr, "warning: failed to open browser:", err)
		}
	}

	// Block until either a signal or a child exit.
	select {
	case sig := <-sigCh:
		fmt.Println("Received signal:", sig)
	case ex := <-l.exitCh:
		fmt.Fprintf(os.Stderr, "Child %q exited unexpectedly: %v\n", ex.name, ex.err)
		// Surface the tail of the offending log.
		tailLog(filepath.Join(res.LogDir, ex.name+".log"), 30)
	}

	l.shutdownAll()
	fmt.Println("Bye.")
}

// tailLog prints the last n lines of a log file to stderr.
func tailLog(path string, n int) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	fmt.Fprintln(os.Stderr, "--- last", len(lines), "log lines ---")
	for _, ln := range lines {
		fmt.Fprintln(os.Stderr, ln)
	}
	fmt.Fprintln(os.Stderr, "--- end ---")
}

// silence unused-import warnings in builds where context isn't directly referenced.
var _ = context.Background
