package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const berwinVersion = "1.3.0"
const backendNpmPackage = "opencode-ai"

func main() {
	args := os.Args[1:]

	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v" || args[0] == "version") {
		fmt.Printf("BerwinCode v%s (terminal, berwincode engine)\n", berwinVersion)
		return
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
		printHelp()
		return
	}
	if len(args) == 1 && args[0] == "reset-login" {
		resetLogin()
		return
	}
	if len(args) == 1 && args[0] == "upgrade-engine" {
		upgradeEngine()
		return
	}

	ensureConfig(false)
	if code := ensureBackend(); code != 0 {
		os.Exit(code)
	}

	// Interactive TUI launch: hide terminal, show login form, no banner.
	if len(args) == 0 && isConsole() {
		hideConsole()
		code := runLoginGate()
		showConsole()
		if code == 2 {
			fmt.Fprintln(os.Stderr, "Login cancelled.")
			pauseEnter()
			os.Exit(1)
		}
		if code != 0 {
			os.Exit(code)
		}
		setTerminalTitle("BerwinCode")
	}

	backend, backendArgs, _ := resolveBackend(args)
	env := buildEnv()

	cmd := exec.Command(backend, backendArgs...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if d, err := os.Getwd(); err == nil {
		cmd.Dir = d
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "\nBerwinCode: could not launch terminal backend (%s): %v\n", backend, err)
		fmt.Fprintln(os.Stderr, "Install backend once: npm install -g opencode-ai@latest")
		pauseEnter()
		os.Exit(1)
	}
}

// ---------------------------------------------------------------- login gate

type loginFile struct {
	Username string `json:"username"`
	Salt     string `json:"salt"`
	Hash     string `json:"hash"`
}

func loginPath() string {
	return filepath.Join(berwinDataDir(), "login.json")
}

func runLoginGate() int {
	if runtime.GOOS != "windows" {
		return 0
	}
	if _, err := os.Stat(loginPath()); os.IsNotExist(err) {
		return setupAccount()
	}
	for attempt := 0; attempt < 3; attempt++ {
		var authErr uint32
		if attempt > 0 {
			authErr = 1326 // ERROR_LOGON_FAILURE: dialog shows "logon attempt failed"
		}
		user, pass, code := credPrompt("BerwinCode Login", "Enter your BerwinCode username and password.", authErr)
		if code == 1223 {
			return 2
		}
		if code != 0 {
			credFailed(code)
			return 1
		}
		ok := checkLogin(user, pass)
		zeroString(pass)
		if ok {
			return 0
		}
	}
	msgBox("BerwinCode", "Too many failed login attempts.", 0x10)
	return 1
}

func setupAccount() int {
	for attempt := 0; attempt < 3; attempt++ {
		user, pass, code := credPrompt("BerwinCode Setup", "Create your BerwinCode username and password.", 0)
		if code == 1223 {
			return 2
		}
		if code != 0 {
			credFailed(code)
			return 1
		}
		if strings.TrimSpace(user) == "" || pass == "" {
			zeroString(pass)
			msgBox("BerwinCode", "Username and password cannot be empty.", 0x30)
			continue
		}
		if err := saveLogin(strings.TrimSpace(user), pass); err != nil {
			zeroString(pass)
			msgBox("BerwinCode", fmt.Sprintf("Could not save login: %v", err), 0x10)
			return 1
		}
		zeroString(pass)
		return 0
	}
	msgBox("BerwinCode", "Setup not completed.", 0x30)
	return 1
}

func saveLogin(username, password string) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	sum := sha256.Sum256(append(salt, []byte(password)...))
	lf := loginFile{
		Username: username,
		Salt:     hex.EncodeToString(salt),
		Hash:     hex.EncodeToString(sum[:]),
	}
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(loginPath(), data, 0600)
}

func checkLogin(username, password string) bool {
	data, err := os.ReadFile(loginPath())
	if err != nil {
		return false
	}
	var lf loginFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return false
	}
	if !strings.EqualFold(lf.Username, strings.TrimSpace(username)) {
		return false
	}
	salt, err := hex.DecodeString(lf.Salt)
	if err != nil {
		return false
	}
	want, err := hex.DecodeString(lf.Hash)
	if err != nil {
		return false
	}
	sum := sha256.Sum256(append(salt, []byte(password)...))
	return subtle.ConstantTimeCompare(sum[:], want) == 1
}

func resetLogin() {
	if err := os.Remove(loginPath()); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No BerwinCode login set. Next launch will ask you to create one.")
			return
		}
		fmt.Fprintf(os.Stderr, "Could not remove login: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("BerwinCode login removed. Next launch will ask you to create a new one.")
}

func zeroString(s string) {
	// Best-effort: strings are immutable, so just avoid keeping references.
	_ = s
}

// ------------------------------------------------------- native dialogs

type credUIInfo struct {
	Size    uint32
	_       uint32
	Parent  uintptr
	Message *uint16
	Caption *uint16
	Banner  uintptr
}

// credPrompt shows the native Windows username/password dialog.
// Returns user, password, win32 code (0 = OK, 1223 = cancelled).
func credPrompt(caption, message string, authError uint32) (string, string, uint32) {
	credui := syscall.NewLazyDLL("credui.dll")
	proc := credui.NewProc("CredUIPromptForCredentialsW")

	capPtr, _ := syscall.UTF16PtrFromString(caption)
	msgPtr, _ := syscall.UTF16PtrFromString(message)
	targetPtr, _ := syscall.UTF16PtrFromString("BerwinCode")
	info := credUIInfo{}
	info.Size = uint32(unsafe.Sizeof(info))
	info.Message = msgPtr
	info.Caption = capPtr

	var userBuf [514]uint16
	var passBuf [257]uint16
	var save uint32

	const flags = 0x40082 // GENERIC_CREDENTIALS | ALWAYS_SHOW_UI | DO_NOT_PERSIST
	r1, _, _ := proc.Call(
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Pointer(targetPtr)),
		0,
		uintptr(authError),
		uintptr(unsafe.Pointer(&userBuf[0])),
		uintptr(513),
		uintptr(unsafe.Pointer(&passBuf[0])),
		uintptr(256),
		uintptr(unsafe.Pointer(&save)),
		uintptr(flags),
	)
	code := uint32(r1)
	if code != 0 {
		for i := range passBuf {
			passBuf[i] = 0
		}
		return "", "", code
	}
	user := syscall.UTF16ToString(userBuf[:])
	pass := syscall.UTF16ToString(passBuf[:])
	for i := range passBuf {
		passBuf[i] = 0
	}
	return user, pass, 0
}

func credFailed(code uint32) {
	msg := fmt.Sprintf("Could not show the login form (Windows error %d).", code)
	msgBox("BerwinCode", msg, 0x10)
	fmt.Fprintln(os.Stderr, msg)
	pauseEnter()
}

func msgBox(caption, text string, flags uint32) {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	capPtr, _ := syscall.UTF16PtrFromString(caption)
	txtPtr, _ := syscall.UTF16PtrFromString(text)
	proc.Call(0, uintptr(unsafe.Pointer(txtPtr)), uintptr(unsafe.Pointer(capPtr)), uintptr(flags))
}

// ---------------------------------------------------------------- helpers

func isConsole() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	if (st.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	st, err = os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (st.Mode() & os.ModeCharDevice) != 0
}

func setTerminalTitle(title string) {
	if !isConsole() {
		return
	}
	if runtime.GOOS == "windows" {
		setConsoleTitleWindows(title)
		return
	}
	fmt.Printf("\x1b]0;%s\x07", title)
}

func setConsoleTitleWindows(title string) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("SetConsoleTitleW")
	ptr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	proc.Call(uintptr(unsafe.Pointer(ptr)))
}

func consoleWindow() uintptr {
	if runtime.GOOS != "windows" {
		return 0
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetConsoleWindow")
	r1, _, _ := proc.Call()
	return uintptr(r1)
}

func hideConsole() {
	if hwnd := consoleWindow(); hwnd != 0 {
		user32 := syscall.NewLazyDLL("user32.dll")
		proc := user32.NewProc("ShowWindow")
		proc.Call(hwnd, 0)
	}
}

func showConsole() {
	if hwnd := consoleWindow(); hwnd != 0 {
		user32 := syscall.NewLazyDLL("user32.dll")
		proc := user32.NewProc("ShowWindow")
		proc.Call(hwnd, 5)
	}
}

func pauseEnter() {
	if st, err := os.Stdin.Stat(); err == nil && (st.Mode()&os.ModeCharDevice) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "Press Enter to close...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func printBanner() {
	fmt.Println(`  ____  _______ ______        _______ _ _   _  ____ ___  ____  _____ `)
	fmt.Println(` | __ )| ____|  _ \ \      / /_ _| \ | |/ ___/ _ \|  _ \| ____|`)
	fmt.Println(` |  _ \|  _| | |_) \ \ /\ / / | ||  \| | |  | | | | | | |  _|  `)
	fmt.Println(` | |_) | |___|  _ < \ V  V /  | || |\  | |__| |_| | |_| | |___ `)
	fmt.Println(` |____/|_____|_| \_\ \_/\_/  |___|_| \_|\____\___/|____/|_____|`)
	fmt.Printf("                 BERWINCODE v%s - TERMINAL\n", berwinVersion)
	fmt.Println()
}

func printHelp() {
	printBanner()
	fmt.Println(`Terminal-only usage:`)
	fmt.Println(`  BerwinCode.exe                 Open BERWINCODE (login form, then terminal)`)
	fmt.Println(`  BerwinCode.exe run "prompt"    Run a prompt in terminal (no TUI, no login)`)
	fmt.Println(`  BerwinCode.exe auth login      Login a provider (first time only)`)
	fmt.Println(`  BerwinCode.exe reset-login     Remove the BerwinCode app login`)
	fmt.Println(`  BerwinCode.exe upgrade-engine  Rebrand a new stock engine after npm upgrades`)
	fmt.Println()
	fmt.Println(`All other terminal args are proxied to the engine:`)
	fmt.Println(`  BerwinCode.exe --model anthropic/claude-sonnet-4-5`)
	fmt.Println(`  BerwinCode.exe agent list`)
	fmt.Println()
	fmt.Println(`  First run downloads Node LTS + engine automatically (~205MB, one-time).`)
	fmt.Printf("Engine: %s\n", berwinEngine())
	fmt.Printf("Config: %s\n", berwinConfigFile())
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

func berwinConfigDir() string {
	return filepath.Join(homeDir(), ".config", "berwincode")
}

func berwinDataDir() string {
	return filepath.Join(homeDir(), ".berwincode")
}

func berwinEngine() string {
	return filepath.Join(berwinDataDir(), "bin", "berwincode.exe")
}

func berwinConfigFile() string {
	if v := os.Getenv("BERWINCODE_CONFIG"); v != "" {
		return v
	}
	if v := os.Getenv("OPENCODE_CONFIG"); v != "" {
		return v
	}
	return filepath.Join(berwinConfigDir(), "berwincode.json")
}

func buildEnv() []string {
	env := os.Environ()
	cfgFile := berwinConfigFile()
	cfgDir := berwinConfigDir()
	if os.Getenv("OPENCODE_CONFIG") == "" {
		env = append(env, "OPENCODE_CONFIG="+cfgFile)
	}
	if os.Getenv("OPENCODE_CONFIG_DIR") == "" {
		env = append(env, "OPENCODE_CONFIG_DIR="+cfgDir)
	}
	if nex := portableNodeExe(); nex != "" {
		ndir := filepath.Dir(nex)
		for i, e := range env {
			if k, _, ok := strings.Cut(e, "="); ok && strings.EqualFold(k, "Path") {
				env[i] = k + "=" + ndir + ";" + e[len(k)+1:]
				break
			}
		}
	}
	if os.Getenv("OPENCODE_DISABLE_TERMINAL_TITLE") == "" {
		env = append(env, "OPENCODE_DISABLE_TERMINAL_TITLE=true")
	}
	env = append(env, "BERWINCODE=1")
	env = append(env, "BERWINCODE_VERSION="+berwinVersion)
	_ = runtime.GOOS
	return env
}

func resolveBackend(userArgs []string) (string, []string, bool) {
	if v := os.Getenv("BERWINCODE_BACKEND"); v != "" {
		return v, userArgs, false
	}
	if v := os.Getenv("OPENCODE_BIN"); v != "" {
		return v, userArgs, false
	}
	if st, err := os.Stat(berwinEngine()); err == nil && !st.IsDir() {
		return berwinEngine(), userArgs, false
	}
	if p, err := exec.LookPath("opencode"); err == nil {
		return p, userArgs, false
	}
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("opencode.exe"); err == nil {
			return p, userArgs, false
		}
	}
	candidates := []string{
		filepath.Join(homeDir(), ".opencode", "bin", "opencode.exe"),
		filepath.Join(homeDir(), ".opencode", "bin", "opencode"),
		filepath.Join(homeDir(), ".local", "bin", "opencode"),
		filepath.Join(homeDir(), "scoop", "shims", "opencode.exe"),
		filepath.Join(homeDir(), ".bun", "bin", "opencode.exe"),
		filepath.Join(homeDir(), ".bun", "bin", "opencode"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, userArgs, false
		}
	}
	if bun, err := exec.LookPath("bunx"); err == nil {
		full := append([]string{"--package", backendNpmPackage, "opencode"}, userArgs...)
		return bun, full, true
	}
	if bun, err := exec.LookPath("bun"); err == nil {
		full := append([]string{"x", "--package", backendNpmPackage, "opencode"}, userArgs...)
		return bun, full, true
	}
	if npx, err := exec.LookPath("npx"); err == nil {
		full := append([]string{"-y", backendNpmPackage}, userArgs...)
		return npx, full, true
	}
	return "opencode", userArgs, false
}

func ensureConfig(verbose bool) {
	cfgDir := berwinConfigDir()
	dataDir := berwinDataDir()
	_ = os.MkdirAll(cfgDir, 0755)
	_ = os.MkdirAll(dataDir, 0755)
	_ = os.MkdirAll(filepath.Join(dataDir, "bin"), 0755)
	_ = os.MkdirAll(filepath.Join(cfgDir, "agents"), 0755)
	_ = os.MkdirAll(filepath.Join(cfgDir, "commands"), 0755)

	cfgFile := berwinConfigFile()
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		_ = os.WriteFile(cfgFile, []byte(defaultBerwinJSON()), 0644)
		if verbose {
			fmt.Printf("Created %s\n", cfgFile)
		}
	}
	mdFile := filepath.Join(cfgDir, "BERWINCODE.md")
	if _, err := os.Stat(mdFile); os.IsNotExist(err) {
		_ = os.WriteFile(mdFile, []byte(defaultBerwinMD()), 0644)
	}
	tuiFile := filepath.Join(cfgDir, "tui.json")
	if _, err := os.Stat(tuiFile); os.IsNotExist(err) {
		_ = os.WriteFile(tuiFile, []byte(defaultTuiJSON()), 0644)
	}
	agentFile := filepath.Join(cfgDir, "agents", "berwin-builder.md")
	if _, err := os.Stat(agentFile); os.IsNotExist(err) {
		_ = os.WriteFile(agentFile, []byte(defaultAgentMD()), 0644)
	}
}

func defaultBerwinJSON() string {
	return `{
  "$schema": "https://opencode.ai/config.json",
  "autoupdate": false,
  "instructions": ["BERWINCODE.md"]
}
`
}

func defaultTuiJSON() string {
	return `{
  "$schema": "https://opencode.ai/tui.json",
  "theme": "opencode",
  "mouse": true
}
`
}

func defaultBerwinMD() string {
	return `# BERWINCODE Instructions

You are BERWINCODE, Berwin's personal AI coding agent in the terminal.
You run on the berwincode engine but you identify as BERWINCODE.

Rules:
- Be short, concise, factual.
- Verify by reading files and running code/tests when reasonable.
- Prefer editing existing files over creating new ones.
- When referencing code, use file_path:line_number format.
`
}

func defaultAgentMD() string {
	return `---
description: BERWINCODE default build agent
mode: primary
---

You are BERWINCODE builder in the terminal. Build, fix, and verify code.
`
}

// ------------------------------------------------------- self-provisioning

func toolsDir() string { return filepath.Join(berwinDataDir(), "tools") }

func portableNodeExe() string {
	matches, _ := filepath.Glob(filepath.Join(toolsDir(), "node", "*", "node.exe"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func haveSystemNode() bool {
	_, err := exec.LookPath("node")
	return err == nil
}

func ensureBackend() int {
	if err := ensureNode(); err != nil {
		fmt.Fprintf(os.Stderr, "\nBerwinCode: %v\n", err)
		fmt.Fprintln(os.Stderr, "Connect to the internet once so BerwinCode can finish setup.")
		pauseEnter()
		return 1
	}
	if err := ensureEngine(); err != nil {
		fmt.Fprintf(os.Stderr, "\nBerwinCode: %v\n", err)
		fmt.Fprintln(os.Stderr, "Connect to the internet once so BerwinCode can finish setup.")
		pauseEnter()
		return 1
	}
	return 0
}

func ensureNode() error {
	if haveSystemNode() {
		return nil
	}
	if portableNodeExe() != "" {
		return nil
	}
	ver, err := latestLTSNode()
	if err != nil {
		return fmt.Errorf("could not find Node LTS version: %w", err)
	}
	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	zipURL := "https://nodejs.org/dist/" + ver + "/node-" + ver + "-win-" + arch + ".zip"
	fmt.Fprintf(os.Stderr, "BerwinCode: Node.js not found. Downloading %s (~35MB, one-time)...\n", ver)
	_ = os.MkdirAll(toolsDir(), 0755)
	tmp := filepath.Join(toolsDir(), "node.zip.tmp")
	if err := downloadFile(zipURL, tmp); err != nil {
		return fmt.Errorf("node download failed: %w", err)
	}
	fmt.Fprintln(os.Stderr, "BerwinCode: extracting Node.js...")
	if err := unzipRoot(tmp, filepath.Join(toolsDir(), "node")); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("node extract failed: %w", err)
	}
	_ = os.Remove(tmp)
	if portableNodeExe() == "" {
		return fmt.Errorf("node installed but node.exe was not found")
	}
	fmt.Fprintln(os.Stderr, "BerwinCode: Node.js ready.")
	return nil
}

func latestLTSNode() (string, error) {
	resp, err := httpClient().Get("https://nodejs.org/dist/index.json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %s", resp.Status)
	}
	var list []struct {
		Version string      `json:"version"`
		LTS     interface{} `json:"lts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return "", err
	}
	for _, e := range list {
		if name, ok := e.LTS.(string); ok && name != "" {
			return e.Version, nil
		}
	}
	return "", fmt.Errorf("no LTS release found")
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Minute}
}

func httpGetJSON(url string) (map[string]interface{}, error) {
	resp, err := httpClient().Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	var m map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func downloadFile(url, dst string) error {
	resp, err := httpClient().Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %s", resp.Status)
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	total := resp.ContentLength
	var done int64
	buf := make([]byte, 1<<20)
	last := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			done += int64(n)
			if time.Since(last) > 300*time.Millisecond || rerr == io.EOF {
				last = time.Now()
				if total > 0 {
					fmt.Fprintf(os.Stderr, "\r  %.1f / %.1f MB", float64(done)/1048576, float64(total)/1048576)
				} else {
					fmt.Fprintf(os.Stderr, "\r  %.1f MB", float64(done)/1048576)
				}
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	fmt.Fprintln(os.Stderr)
	return nil
}

func unzipRoot(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	_ = os.MkdirAll(destDir, 0755)
	base := filepath.Clean(destDir)
	for _, f := range r.File {
		parts := strings.Split(filepath.ToSlash(f.Name), "/")
		if len(parts) < 2 {
			continue
		}
		target := filepath.Join(append([]string{destDir}, parts[1:]...)...)
		if !strings.HasPrefix(filepath.Clean(target), base) {
			continue
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(target), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(w, rc)
		rc.Close()
		w.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureEngine() error {
	if st, err := os.Stat(berwinEngine()); err == nil && !st.IsDir() {
		return nil
	}
	src := stockEnginePath()
	if src == "" {
		var err error
		src, err = downloadEngine()
		if err != nil {
			return err
		}
	} else {
		fmt.Fprintln(os.Stderr, "BerwinCode: found stock engine, branding it...")
	}
	return patchEngineFile(src, berwinEngine())
}

func stockEnginePath() string {
	cands := []string{
		filepath.Join(homeDir(), "AppData", "Roaming", "npm", "node_modules", "opencode-ai", "bin", "opencode.exe"),
	}
	if p, err := exec.LookPath("opencode"); err == nil && strings.HasSuffix(strings.ToLower(p), ".exe") {
		cands = append([]string{p}, cands...)
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() && st.Size() > 50000000 {
			return c
		}
	}
	return ""
}

func downloadEngine() (string, error) {
	meta, err := httpGetJSON("https://registry.npmjs.org/opencode-ai/latest")
	if err != nil {
		return "", fmt.Errorf("could not read engine registry: %w", err)
	}
	ver, _ := meta["version"].(string)
	if ver == "" {
		return "", fmt.Errorf("engine registry gave no version")
	}
	optdeps, _ := meta["optionalDependencies"].(map[string]interface{})
	plat := "windows"
	arch := "x64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	name := "opencode-" + plat + "-" + arch
	pkgver, _ := optdeps[name].(string)
	if pkgver == "" {
		name = name + "-baseline"
		pkgver, _ = optdeps[name].(string)
	}
	if pkgver == "" {
		return "", fmt.Errorf("no engine package for %s/%s", plat, arch)
	}
	tgzURL := "https://registry.npmjs.org/" + name + "/-/" + name + "-" + pkgver + ".tgz"
	fmt.Fprintf(os.Stderr, "BerwinCode: engine not found. Downloading %s %s (~170MB, one-time)...\n", name, pkgver)
	_ = os.MkdirAll(toolsDir(), 0755)
	tmp := filepath.Join(toolsDir(), "engine.tgz.tmp")
	if err := downloadFile(tgzURL, tmp); err != nil {
		return "", fmt.Errorf("engine download failed: %w", err)
	}
	fmt.Fprintln(os.Stderr, "BerwinCode: extracting engine...")
	out := filepath.Join(toolsDir(), "engine-stock.exe")
	if err := extractEngineTgz(tmp, out); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("engine extract failed: %w", err)
	}
	_ = os.Remove(tmp)
	return out, nil
}

func extractEngineTgz(tgzPath, outPath string) error {
	f, err := os.Open(tgzPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.ToSlash(hdr.Name)
		if !strings.HasSuffix(name, "package/bin/opencode.exe") && !strings.HasSuffix(name, "package/bin/opencode") {
			continue
		}
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, tr)
		out.Close()
		return err
	}
	return fmt.Errorf("opencode binary not found in package")
}

func patchEngineFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	missing := 0
	for _, p := range enginePatches {
		ob, nb := []byte(p[0]), []byte(p[1])
		if !bytes.Contains(b, ob) {
			missing++
			continue
		}
		b = bytes.ReplaceAll(b, ob, nb)
	}
	_ = os.MkdirAll(filepath.Dir(dst), 0755)
	if err := os.WriteFile(dst, b, 0755); err != nil {
		return err
	}
	if missing > 0 {
		fmt.Fprintf(os.Stderr, "BerwinCode: note: %d branding patch(es) skipped (engine layout changed).\n", missing)
	}
	cmd := exec.Command(dst, "--version")
	if out, err := cmd.Output(); err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("branded engine failed self-test: %w", err)
	} else {
		fmt.Fprintf(os.Stderr, "BerwinCode: engine ready (%s).\n", strings.TrimSpace(string(out)))
	}
	fmt.Fprintln(os.Stderr, "BerwinCode: branding applied (BERWINCODE logo, no Zen labels).")
	return nil
}

func upgradeEngine() {
	src := stockEnginePath()
	if src == "" {
		fmt.Fprintln(os.Stderr, "No stock opencode engine found (npm install -g opencode-ai first).")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "BerwinCode: rebranding %s ...\n", src)
	if err := patchEngineFile(src, berwinEngine()); err != nil {
		fmt.Fprintf(os.Stderr, "BerwinCode: %v\n", err)
		os.Exit(1)
	}
}
