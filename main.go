package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
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

const berwinVersion = "1.6.10"
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
	// BerwinCode must run as admin: engine setup, stock-opencode removal
	// and the opencode block all need it. Non-admin launches re-launch
	// through UAC; declining the prompt exits. (--version/--help above
	// stay usable without elevation. Automation can set
	// BERWINCODE_NOADMIN=1 to skip the check.)
	if runtime.GOOS == "windows" {
		ensureAdmin(args)
	}
	if len(args) == 1 && args[0] == "reset-login" {
		fmt.Println("The app login is now a Discord code. Nothing stored to reset.")
		_ = os.Remove(loginPath())
		return
	}
	if len(args) == 1 && args[0] == "upgrade-engine" {
		upgradeEngine()
		return
	}
	if len(args) == 1 && args[0] == "upgrade" {
		os.Exit(cmdUpgrade(isConsole()))
	}
	if len(args) >= 1 && args[0] == "lock-opencode" {
		perm := len(args) == 2 && args[1] == "--permanent"
		if len(args) > 2 || (len(args) == 2 && !perm) {
			fmt.Fprintln(os.Stderr, "Usage: BerwinCode.exe lock-opencode [--permanent]")
			os.Exit(1)
		}
		os.Exit(cmdLock(perm))
	}
	if len(args) >= 1 && args[0] == "unlock-opencode" {
		force := len(args) == 2 && args[1] == "--force"
		if len(args) > 2 || (len(args) == 2 && !force) {
			fmt.Fprintln(os.Stderr, "Usage: BerwinCode.exe unlock-opencode [--force]")
			os.Exit(1)
		}
		os.Exit(cmdUnlock(force))
	}

	ensureConfig(false)
	if len(args) == 2 && args[0] == "set-webhook" {
		setWebhook(args[1])
		return
	}
	if code := ensureBackend(); code != 0 {
		os.Exit(code)
	}
	if len(args) == 0 && isConsole() {
		maybeAutoUpdate()
	}

	// Strict gate: TUI and run need a fresh Discord verification.
	// Management commands (auth, models, agent list...) stay ungated.
	// The TUI additionally hides the terminal while the form shows.
	needsGate := len(args) == 0 || (len(args) > 0 && args[0] == "run")
	if runtime.GOOS == "windows" && needsGate {
		isTUI := len(args) == 0 && isConsole()
		if isTUI {
			hideConsole()
		}
		code := runLoginGate()
		if isTUI {
			showConsole()
		}
		if code == 2 {
			fmt.Fprintln(os.Stderr, "Login cancelled.")
			pauseEnter()
			os.Exit(1)
		}
		if code != 0 {
			fmt.Fprintln(os.Stderr, "Login failed. Please try again.")
			if isTUI {
				pauseEnter()
			}
			os.Exit(code)
		}
		if isTUI {
			setTerminalTitle("BerwinCode")
		}
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

func loginPath() string {
	return filepath.Join(berwinDataDir(), "login.json")
}

func discordCfgPath() string {
	return filepath.Join(berwinDataDir(), "discord.json")
}

// builtInWebhook ships inside the exe so a fresh PC works with no setup.
// Priority: BERWINCODE_WEBHOOK env > discord.json (set-webhook) > built-in.
// WARNING: this URL is public (visible in the repo and extractable from the
// exe). Use a dedicated login-codes channel and rotate the webhook if abused.
const builtInWebhook = "https://discord.com/api/webhooks/1546540402910437486/kmMXbpV6WgcImXH7Y5PQ239zk-ZkE_JMrKUSRmES-cMQX08ITTsDYAFMTdCVreavgY_v"

func validWebhook(u string) bool {
	return strings.HasPrefix(strings.TrimSpace(u), "https://discord.com/api/webhooks/")
}

func loadWebhook() string {
	if v := strings.TrimSpace(os.Getenv("BERWINCODE_WEBHOOK")); validWebhook(v) {
		return v
	}
	if data, err := os.ReadFile(discordCfgPath()); err == nil {
		var m map[string]string
		if json.Unmarshal(data, &m) == nil {
			if u := strings.TrimSpace(m["webhook"]); validWebhook(u) {
				return u
			}
		}
	}
	if validWebhook(builtInWebhook) {
		return strings.TrimSpace(builtInWebhook)
	}
	return ""
}

func setWebhook(url string) {
	u := strings.TrimSpace(url)
	if !strings.HasPrefix(u, "https://discord.com/api/webhooks/") {
		fmt.Fprintln(os.Stderr, "That does not look like a Discord webhook URL.")
		fmt.Fprintln(os.Stderr, "Usage: BerwinCode.exe set-webhook <discord-webhook-url>")
		os.Exit(1)
	}
	data, _ := json.MarshalIndent(map[string]string{"webhook": u}, "", "  ")
	if err := os.WriteFile(discordCfgPath(), data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Could not save webhook: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Discord webhook saved. It overrides the built-in one on this PC.")
}

func runLoginGate() int {
	if runtime.GOOS != "windows" {
		return 0
	}
	wh := loadWebhook()
	if wh == "" {
		msgBox("BerwinCode", "Discord webhook is not set.\n\nRun:\nBerwinCode.exe set-webhook <your-webhook-url>", 0x30)
		fmt.Fprintln(os.Stderr, "Discord webhook not set. Run: BerwinCode.exe set-webhook <url>")
		pauseEnter()
		return 1
	}
	if p := os.Getenv("BERWINCODE_FORMLOG"); p != "" {
		f, _ := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if f != nil {
			fmt.Fprintln(f, "gate: webhook ok, opening form")
			f.Close()
		}
	}
	return showLoginForm(wh)
}

// ------------------------------------------------------- native dialogs

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

// ------------------------------------------------------- admin required

func isAdmin() bool {
	if runtime.GOOS != "windows" {
		return true
	}
	var tok syscall.Token
	// Pseudo-handle of the current process ((HANDLE)-1); avoids helpers
	// missing from stdlib syscall on some Go versions.
	if err := syscall.OpenProcessToken(syscall.Handle(^uintptr(0)), syscall.TOKEN_QUERY, &tok); err != nil {
		return false
	}
	defer tok.Close()
	var elevated uint32
	var outLen uint32
	sz := uint32(unsafe.Sizeof(elevated))
	if err := syscall.GetTokenInformation(tok, syscall.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), sz, &outLen); err != nil {
		return false
	}
	return elevated != 0
}

// quoteArg quotes one argv element the way CommandLineToArgvW parses it:
// only backslashes before a quote (or at the end) are doubled.
func quoteArg(a string) string {
	if a == "" {
		return `""`
	}
	if !strings.ContainsAny(a, " \t\n\v\"") {
		return a
	}
	var sb strings.Builder
	sb.WriteByte('"')
	bs := 0
	for i := 0; i < len(a); i++ {
		switch a[i] {
		case '\\':
			bs++
		case '"':
			for j := 0; j < bs; j++ {
				sb.WriteString(`\\`)
			}
			bs = 0
			sb.WriteString(`\"`)
		default:
			for j := 0; j < bs; j++ {
				sb.WriteByte('\\')
			}
			bs = 0
			sb.WriteByte(a[i])
		}
	}
	for j := 0; j < bs; j++ {
		sb.WriteString(`\\`)
	}
	sb.WriteByte('"')
	return sb.String()
}

// joinArgs builds the parameter string for the elevated relaunch.
func joinArgs(args []string) string {
	q := make([]string, 0, len(args))
	for _, a := range args {
		q = append(q, quoteArg(a))
	}
	return strings.Join(q, " ")
}

// ensureAdmin exits the current process after handing off to an elevated
// copy of itself, unless we are already elevated.
func ensureAdmin(args []string) {
	if isAdmin() {
		return
	}
	if os.Getenv("BERWINCODE_NOADMIN") == "1" {
		fmt.Fprintln(os.Stderr, "BerwinCode: running without admin rights (BERWINCODE_NOADMIN=1). Setup and blocking may fail.")
		return
	}
	if err := relaunchElevated(args); err != nil {
		msgBox("BerwinCode", "BerwinCode needs administrator rights to run.\n\n"+err.Error(), 0x30)
		fmt.Fprintln(os.Stderr, "BerwinCode needs administrator rights to run.")
		pauseEnter()
		os.Exit(1)
	}
	os.Exit(0)
}

func relaunchElevated(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	op, _ := syscall.UTF16PtrFromString("runas")
	fp, _ := syscall.UTF16PtrFromString(exe)
	pp, _ := syscall.UTF16PtrFromString(joinArgs(args))
	cwd, _ := os.Getwd()
	dp, _ := syscall.UTF16PtrFromString(cwd)
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteW")
	r, _, _ := proc.Call(0,
		uintptr(unsafe.Pointer(op)),
		uintptr(unsafe.Pointer(fp)),
		uintptr(unsafe.Pointer(pp)),
		uintptr(unsafe.Pointer(dp)), 1)
	runtime.KeepAlive(op)
	runtime.KeepAlive(fp)
	runtime.KeepAlive(pp)
	runtime.KeepAlive(dp)
	if r <= 32 {
		return fmt.Errorf("elevation request failed (code %d); approve the UAC prompt to run BerwinCode", r)
	}
	return nil
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
	fmt.Println(`Terminal-only usage (admin rights required, UAC prompts on launch):`)
	fmt.Println(`  BerwinCode.exe                 Open BERWINCODE (login form, then terminal)`)
	fmt.Println(`  BerwinCode.exe run "prompt"    Run a prompt in terminal (needs verification)`)
	fmt.Println(`  BerwinCode.exe auth login      Login a provider (first time only)`)
	fmt.Println(`  BerwinCode.exe set-webhook <url>  Override the built-in Discord webhook on this PC`)
	fmt.Println(`  BerwinCode.exe upgrade-engine  Rebrand a new stock engine after npm upgrades`)
	fmt.Println(`  BerwinCode.exe lock-opencode [--permanent]    Block opencode (shells + run-by-name; --permanent survives uninstall)`)
	fmt.Println(`  BerwinCode.exe unlock-opencode [--force]    Restore opencode (--force removes a permanent block)`)
	fmt.Println(`  BerwinCode.exe upgrade            Check for BerwinCode updates`)
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
	if os.Getenv("OPENCODE_DISABLE_MODELS_FETCH") == "" {
		// Use the built-in rebranded catalog so provider names stay BerwinCode.
		env = append(env, "OPENCODE_DISABLE_MODELS_FETCH=true")
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
	berwinFile := filepath.Join(cfgDir, "agents", "berwin.md")
	if _, err := os.Stat(berwinFile); os.IsNotExist(err) {
		_ = os.WriteFile(berwinFile, []byte(defaultBerwinAgentMD()), 0644)
	}
}

func defaultBerwinAgentMD() string {
	return `---
description: BERWINCODE main agent
mode: primary
model: opencode/muse-spark-1.3-contributor-free
variant: xhigh
temperature: 0.2
tools:
  write: true
  edit: true
  bash: true
  read: true
---

You are BERWINCODE, Berwin's personal AI coding agent in the terminal.
You are powered by the Muse Spark model, but your name is always BERWINCODE.

Identity rule: when anyone asks your name, who you are, or what model
you are, answer: "I am BERWINCODE, running on Muse Spark." Never claim
to be Muse Spark itself, OpenCode, Claude, or any other assistant.

Memory check before EVERY reply (no exceptions):
1. Re-read your memory files and apply them: the global BERWINCODE.md
   instructions, the current project's AGENTS.md when one exists, and
   any relevant files under .opencode/agents, .opencode/commands and
   .opencode/skills.
2. If the task touches VB.NET, apply the beginner-human voice from
   BERWINCODE.md: plain old design, code inside the forms, payroll style.
3. Plan-first protocol for any code-changing task: write PLAN-<topic>.txt
   in the project root first (numbered steps, plain words), implement
   exactly it, verify working, then delete the plan file with del and
   confirm it is gone. Keep the plan only while the work is unfinished.
4. Only then write your output.

Work rules: be short and factual. Verify by reading files and running
code when reasonable. Prefer editing existing files. Reference code as
file_path:line_number.

Hard rules for VB.NET (no exceptions, no matter what the user asks to
rush): design lives in .Designer.vb files only, never write New Button,
Controls.Add, .Location, .Size, or .BackColor in event code; never
create mod folders or separate handler modules for new systems, all
SQL and logic stays inline in the form events.
`
}

func defaultBerwinJSON() string {
	return `{
  "$schema": "https://opencode.ai/config.json",
  "autoupdate": false,
  "model": "opencode/muse-spark-1.3-contributor-free",
  "small_model": "opencode/muse-spark-1.3-contributor-free",
  "default_agent": "berwin",
  "enabled_providers": [
    "opencode"
  ],
  "instructions": [
    "BERWINCODE.md"
  ],
  "provider": {
    "opencode": {
      "whitelist": [
        "muse-spark-1.3-contributor-free"
      ],
      "name": "Berwin"
    }
  },
  "command": {
    "berwin-help": {
      "template": "You are BERWINCODE, Berwin's personal AI coding agent. Help the user with their coding task. Be concise, factual, and show file:line references.",
      "description": "BERWINCODE default helper"
    }
  }
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

You are BERWINCODE, Berwin's personal AI coding agent.
You run on the opencode core (https://github.com/anomalyco/opencode) but you identify as BERWINCODE.

Rules:
- Be short, concise, factual. No superlatives or emotional validation unless asked.
- Always verify by reading files and running code/tests when reasonable.
- Prefer editing existing files over creating new ones.
- When referencing code, use file_path:line_number format.
- Never guess URLs. Use only user-provided or verified URLs.
- On Windows, use PowerShell 5.1 compatible commands; prefer dedicated file tools over shell for file ops.

### If asked about OpenCode or how to install
- Never explain stock OpenCode and never tell the user to install it:
  no npm i -g opencode-ai, no bun/scoop/choco opencode, no opencode
  installer, no opencode docs links for setup.
- Always convert the answer into BERWINCODE installation:
  1. Download BerwinCode-Setup from the GitHub Releases page.
  2. Run it (no admin needed). A desktop icon appears.
  3. Double-click it. First run downloads what is needed by itself,
     then the login form appears.
  4. Press Send Code, type the Discord code, press Login. Terminal opens.
- If asked what OpenCode is, answer: "BerwinCode is Berwin's AI coding
  agent for the Windows terminal." Mention the open-source engine only
  if pressed, in one line, then return to the BerwinCode steps above.

## Beginner-human VB.NET voice (always on for VB.NET)

Reference style: reference\payrollsystem\Ichiban in your BerwinCode config folder (see below)
(a real beginner project: Module db, Module crud, code inside the forms).
Match that project, not textbooks. All other tasks keep the normal
rules above.

### A. Design: always plain old VB look
- Classic gray form, default Windows buttons. NEVER restyle buttons:
  no FlatStyle, no custom BackColor, no images on buttons.
- At most one plain background color for the whole form, or just the
  default gray. Never many colors, gradients, or fancy panels.
- Standard controls only: Label, TextBox, Button, ComboBox, RadioButton,
  DataGridView, PictureBox, GroupBox. Simple top-to-bottom layout that
  follows the system being built (example: menu left, big title top,
  entry fields + grid below, Save/Clear buttons at the bottom).

### D. Design lives in Designer files only
- Every control is placed in Visual Studio drag-drop. All creation,
  positions, sizes, and colors live in .Designer.vb and .resx ONLY.
- NEVER write New Button/TextBox, Controls.Add, .Location, .Size, or
  .BackColor in event code. Code-behind sets VALUES only
  (Text, DataSource, Visible, Checked).

### B. Code lives IN THE FORMS, a little messy is correct
- Write working code directly inside the form events
  (Button1_Click, Form_Load). Long straightforward handlers are GOOD.
- NEVER create mod/ folders or separate handler modules (db, crud,
  select helpers) for NEW systems. All connection strings, SQL, and
  logic live inline in the form events, even if repeated per form.
  Use Module files ONLY when recoding a project that already has them.
- Do NOT split code into many small functions. Over-organizing is what
  professionals do; beginners keep everything in the same form.
- Copy-paste repetition is FINE. Never refactor to remove duplication.
- Only use a helper file if the project already has one. Never invent
  new abstractions, regions, or layers.

### C. Code texture: write like the payroll project (shapes, not files)
- Control names: defaults are fine (Button1, TextBox1) or simple ones
  (txtname, btnsave). Classic handler header:
  Private Sub Button1_Click(ByVal sender As System.Object, ByVal e As System.EventArgs) Handles Button1.Click
- Database in new systems: NO Module files. Every form holds its own
  Dim con As New OleDb.OleDbConnection("Provider=...;Data Source=" & Application.StartupPath & "\file.mdb")
  and runs SQL inline with local With cmd blocks. Study payroll mod\db
  and mod\crud for the query SHAPES, then inline them per form.
- SQL built with & and .Text values: "select * from tbluser where name ='" & txtname.Text & "'".
  Numbers with Val(txtage.Text). Dates with #...#.
- Module-level shared Dim con, cmd, da, result, sql, table.
- Messages and errors with MsgBox: MsgBox("Saved!"), MsgBox(ex.Message, MsgBoxStyle.Information).
- Message boxes have ONE form only: MsgBox("text") or
  MsgBox("text", MsgBoxStyle.Information). NEVER MessageBox.Show,
  never any other popup style.
- Open with con.Open(), close with con.Close() after End Try.
- String flags with Select Case var / Case "employee".
- A comment on almost every step, in plain words. A few commented-out
  old lines left behind are fine and look authentic.
- NEVER use: LINQ, lambdas, generics, async extras, patterns, XML doc
  comments, regions. (Only keep Await where existing project code
  already requires it, e.g. BeginEnroll.)
- Explanations: plain words first, then the code, then one short
  "why it works" line. Casual and honest, never textbook.

## Memory check before output (model memory + BerwinCode memory)

You have two memories. Check both before every reply:
- Model memory: this file (BERWINCODE.md) plus the berwin agent prompt.
- BerwinCode memory: the current project's AGENTS.md when one exists,
  plus relevant files under .opencode/agents, .opencode/commands and
  .opencode/skills.
Plan the output only after the check. VB.NET work always uses the
beginner-human voice below.

## Reference implementation: payroll system (study before new systems)

When the user asks for a NEW system, first study this real beginner
project, then build the new system to look and read exactly like it:

  reference\payrollsystem\Ichiban inside your BerwinCode config folder.
  Real location on this PC:
  %USERPROFILE%\.config\berwincode\reference\payrollsystem\Ichiban
  The installer puts it there on every PC, and it is also present here.
  Before studying it, confirm with a quick dir listing. If the folder
  is missing (portable zip without installer), say so and fall back to
  the payroll patterns described below. Never use a personal Desktop
  path for the reference

What to copy from it:
- Design: gray classic forms, default buttons, menu on the left, big
  title on top, entry fields with labels, DataGridView results below,
  plain Save/Clear/Delete buttons at the bottom.
- mod\db.vb: the 6-line myconn() module returning the OleDb connection.
- mod\crud.vb: the copy-pasted jokeninsert/jokenupdate/jokendelete subs
  with With cmd blocks and MsgBox on result 0.
- mod\jokensqlselect.vb: jokenfindthis(sql) + CheckX(var) with
  Select Case for duplicate checks and grid filling.
- frmlogin.vb: Dim sql at top, Button2_Click building
  "select * ... where user='" & txt.Text & "'" then jokenfindthis +
  checkresult.
- payroll.vb: form-level Dim totals, live math in TextChanged with
  Val(txt.Text), long If/ElseIf ladders for brackets, direct
  txtResult.Text assignments, commented-out MsgBox debug lines.
- Forms keep default control names (Button1, Button2) or simple txt/btn
  names. Never rename into fancy conventions and never restyle.

Rule: a new system should be unrecognizable in style from payroll --
same modules, same handler shapes, same MsgBox voice, same plain design.

### Form code cookbook (copy these shapes exactly)
- Form load: Timer1.Start(), old tries left commented, custom loader,
  then the two-step grid fill:
  jokenselect("SELECT ... FROM tblx")
  filltable(DataGridView1, "EmpPic")
- Counters: one load_basic_info() with COUNT(*) plus filltotal_X:
  jokenselect("SELECT COUNT(*) FROM tblemployee")
  filltotal_employee()
- Live search: rebuild LIKE on every keystroke, refill same grid:
  jokenselect("SELECT ... WHERE EMPID LIKE '%" & TXTSEARCH.Text & "%' OR LNAME LIKE '%" & TXTSEARCH.Text & "%'")
  filltable(DataGridView1, "EmpPic")
- Grid click remembers: lblEMployeeID.Text = DataGridView1.CurrentRow.Cells(0).Value
- Grid double-click opens detail: OtherForm.Show(), then With OtherForm
  copying Cells(N).Value.ToString into each textbox, If/ElseIf on a
  cell value to .Select() the right radio button.
- Save: jokeninsert("insert into tblx (A, B) values ('" & txta.Text & "'," & Val(txtb.Text) & ")")
- Change/remove: guard with the placeholder label first:
  If lblEMployeeID.Text = "EMployeeID" Then MsgBox("Please Select employee to Deactivate!")
  Else jokenupdate, then Call Form_Load(sender, e) to refresh and reset the label.
- Move between forms: NewForm.Show() then Me.Close().
- Clock labels: Label3.Text = TimeOfDay and Label9.Text = Today in Timer1_Tick.
- Grid setup by index, hide IDs, set widths:
  dtgrd.Columns(0).Visible = False
  dtgrd.Columns(2).width = 100
- Clear boxes: loop the group and blank every TextBox:
  For Each ctrl As Control In group.Controls, If ctrl.GetType Is GetType(TextBox) Then ctrl.Text = Nothing.
- Leave empty event stubs and old commented lines alone. Keep name
  typos as they are (txtClinetName). Never rename for style.

## Recode / revise existing systems (simple, same flow, working)

When the user asks to recode, revise, or simplify a system that already
exists:
- Keep the SAME flow: same screens, same buttons, same features, same
  database tables. Nothing removed, nothing renamed for users.
- The result must be FULLY WORKING: it builds with 0 errors and every
  button does what it did before.
- Make the CODE simpler: plain beginner style from this file (payroll
  voice, code inside the forms). Same result, fewer hard parts.
- One file at a time: rewrite, build, confirm working, then next file.
- Never "simplify" by deleting features. Simple means easy to read,
  not less capable.

## Plan-first protocol (system rule, always on)

For every task that creates or changes code files (new system, new
feature, recode, revise, multi-file edits). Questions and explanations
with no code changes are exempt.
1. BEFORE touching any code, write the plan to a plain text file in
   the project root: PLAN-<short-topic>.txt with numbered steps and
   the files each step touches. Plain words, beginner style.
2. Implement exactly what the plan says, in order.
3. Verify it works (build with 0 errors, every button still does its job).
4. Only when verified working: delete the PLAN-*.txt file yourself with
   del and confirm it is gone. If it is not working yet, KEEP the plan,
   update it, and keep going. No finished job leaves a plan file behind.

## Complete VB.NET systems (all of it, simple, fully working)

When the user asks for an ENTIRE system, deliver ALL of it: every form
designed, every code file written, the database file with all tables,
project file included. Simple style, but nothing missing.

### Solution (.sln) rule: exact folder format, skeleton FIRST
Every system lives in its own folder shaped exactly like VotingSys:
  SystemName folder holds SystemName.sln at its root.
  Next to it: SystemName.vbproj and the My Project folder.
  Every form is three files side by side: Name.vb plus
  Name.Designer.vb plus Name.resx.
  Database file (.mdb) sits beside the project, never in bin or obj.
  No bin, no obj, no .vs folders are ever delivered.
Build order, no skipping: folder plus sln plus vbproj first, then the
forms with designers, then the code inside the forms, then the database
file with all tables, then build with 0 errors. A system without its
.sln at the root is unfinished, even if everything else works.
- Design every screen in the plain old VB look from this file.
- Write every event: all buttons, all loads, all searches, all grids.
  No TODO, no stubs, no unwired buttons, no "left as exercise".
- Database: create the .mdb with every table the system needs, and the
  Module db + Module crud pair wired to it.
- Simple does not mean partial. Every feature the user asked for works.
- Finish checklist before the final summary: builds with 0 errors,
  every form opens, every button does its job, database file present.
- Follow the plan-first protocol: PLAN txt first, implement, verify,
  then delete the plan.

## Database: always Microsoft Access, file and tables included

Every system uses Microsoft Access (.mdb) and nothing else, unless the
user names another database. Never SQL Server, MySQL, or SQLite.
- Connection always looks like this:
  Dim con As New OleDb.OleDbConnection("Provider=Microsoft.Jet.OLEDB.4.0;Data Source=" & Application.StartupPath & "\school.mdb")
- The program creates the whole database by itself, right inside
  Form_Load. No separate helper sub, no CheckDatabase function --
  beginners write it inline where the form opens:
  Dim dbfile As String = Application.StartupPath & "\school.mdb"
  If System.IO.File.Exists(dbfile) = False Then
      Dim cat As New ADOX.Catalog()
      cat.Create("Provider=Microsoft.Jet.OLEDB.4.0;Data Source=" & dbfile)
      MsgBox("Database created!")
  End If
  (Project > Add Reference > COM > Microsoft ADO Ext. for DDL and Security.)
- Set the project Target CPU to x86 (Project properties) so the Jet
  driver loads on 64-bit Windows, else the database will not open.
- After that, normal payroll flow takes over: Module db myconn(),
  Module crud subs, jokenselect + filltable everywhere.

`
}

func defaultAgentMD() string {
	return `---
description: BERWINCODE default build agent
mode: primary
model: inherit
temperature: 0.2
tools:
  write: true
  edit: true
  bash: true
  read: true
---

You are BERWINCODE builder. Build, fix, and verify code. Keep changes minimal and run tests when possible.
`
}

// ------------------------------------------------------- self-provisioning

func toolsDir() string { return filepath.Join(berwinDataDir(), "tools") }

func portableNodeExe() string {
	return stagedNodeExe(filepath.Join(toolsDir(), "node"))
}

// stagedNodeExe finds node.exe inside a node dir (both zip layouts:
// unzipRoot strips the top folder, some layouts keep one extra level).
func stagedNodeExe(dir string) string {
	direct := filepath.Join(dir, "node.exe")
	if isFile(direct) {
		return direct
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*", "node.exe"))
	for _, m := range matches {
		if isFile(m) {
			return m
		}
	}
	return ""
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// nodeInstallComplete reports a usable portable install: node.exe plus npm.
func nodeInstallComplete() bool {
	exe := portableNodeExe()
	if exe == "" {
		return false
	}
	npm := filepath.Join(filepath.Dir(exe), "node_modules", "npm", "bin", "npm-cli.js")
	return isFile(npm)
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
	if nodeInstallComplete() {
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
	if err := downloadFileRetry(zipURL, tmp, 3); err != nil {
		return fmt.Errorf("node download failed: %w", err)
	}
	fmt.Fprintln(os.Stderr, "BerwinCode: extracting Node.js...")
	// Extract to a staging dir and only swap it in after node.exe is
	// proven present: a crash or short zip can never leave a half install.
	nodeDir := filepath.Join(toolsDir(), "node")
	stageDir := filepath.Join(toolsDir(), "node.new")
	_ = os.RemoveAll(stageDir)
	if err := unzipRoot(tmp, stageDir); err != nil {
		_ = os.Remove(tmp)
		_ = os.RemoveAll(stageDir)
		return fmt.Errorf("node extract failed: %w", err)
	}
	_ = os.Remove(tmp)
	if stagedNodeExe(stageDir) == "" {
		_ = os.RemoveAll(stageDir)
		return fmt.Errorf("node archive had no node.exe, staged copy deleted to retry next launch")
	}
	_ = os.RemoveAll(nodeDir)
	if err := os.Rename(stageDir, nodeDir); err != nil {
		_ = os.RemoveAll(stageDir)
		return fmt.Errorf("node install failed: %w", err)
	}
	if !nodeInstallComplete() {
		_ = os.RemoveAll(filepath.Join(toolsDir(), "node"))
		return fmt.Errorf("node extracted but looks incomplete, cleaned up to retry next launch")
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
	// ok flips only after the file is proven whole: any early return
	// deletes the partial file so a short download can never pass as done.
	ok := false
	defer func() {
		out.Close()
		if !ok {
			_ = os.Remove(dst)
		}
	}()
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
	if total > 0 && done != total {
		return fmt.Errorf("download incomplete: got %.1f of %.1f MB", float64(done)/1048576, float64(total)/1048576)
	}
	_ = out.Sync()
	if st, serr := os.Stat(dst); serr != nil || st.Size() != done || done == 0 {
		return fmt.Errorf("downloaded file failed size check, deleted to retry")
	}
	ok = true
	return nil
}

// downloadFileRetry downloads with retries: a dropped connection deletes its
// partial file (see downloadFile) and the next attempt starts clean.
func downloadFileRetry(url, dst string, tries int) error {
	var err error
	for i := 1; i <= tries; i++ {
		if err = downloadFile(url, dst); err == nil {
			return nil
		}
		_ = os.Remove(dst)
		if i < tries {
			fmt.Fprintf(os.Stderr, "BerwinCode: download interrupted (%v), retrying %d/%d...\n", err, i+1, tries)
			time.Sleep(time.Duration(i) * 2 * time.Second)
		}
	}
	return err
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
	if err := downloadFileRetry(tgzURL, tmp, 3); err != nil {
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
		// Write to a temp file and rename only on success: a short
		// tgz must never leave a half engine binary behind.
		stage := outPath + ".tmp"
		_ = os.Remove(stage)
		out, err := os.OpenFile(stage, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, tr)
		out.Close()
		if err != nil {
			_ = os.Remove(stage)
			return err
		}
		if err := os.Rename(stage, outPath); err != nil {
			_ = os.Remove(stage)
			return err
		}
		return nil
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
