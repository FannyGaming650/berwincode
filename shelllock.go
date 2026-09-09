package main

// Shell-level opencode lock (defense in depth, after file removal).
// lock-opencode: shadows the `opencode` command in PowerShell profiles
// and CMD autorun with a "disabled" message. Fully reversible.
// unlock-opencode: restores everything.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const lockBegin = "# >>> BERWINCODE-LOCK >>>"
const lockEnd = "# <<< BERWINCODE-LOCK <<<"
const autoRunKey = `HKCU\Software\Microsoft\Command Processor`
const explorerKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Policies\Explorer`
const disallowSub = "DisallowRun"

// Permanent lock: survives BerwinCode uninstall. The CMD batch lives
// outside .berwincode and a marker makes plain unlock-opencode (the exact
// command the uninstaller runs) refuse. Removal then needs:
//
//	BerwinCode.exe unlock-opencode --force
func permanentMarkerPath() string { return filepath.Join(berwinDataDir(), "permanent.lock") }

func isPermanentLock() bool { return isFile(permanentMarkerPath()) }

func markPermanent() {
	_ = os.MkdirAll(berwinDataDir(), 0755)
	_ = os.WriteFile(permanentMarkerPath(), []byte("permanent"), 0644)
}

// permanentBatchPath lives in Roaming AppData so wiping .berwincode or
// uninstalling never deletes the CMD blocker itself.
func permanentBatchPath() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		base = filepath.Join(homeDir(), "AppData", "Roaming")
	}
	return filepath.Join(base, "BerwinCode", "cmdlock.bat")
}

// disallowNames are blocked by file NAME via Explorer DisallowRun, so the
// block works even for copies that do not exist yet or live elsewhere.
// HKCU (not HKLM): same per-user effect, no admin rights needed.
var disallowNames = []string{
	"opencode.exe",
	"opencode.cmd",
	"opencode.ps1",
	"opencode-setup.exe",
	"opencode_installer.exe",
	"opencode-installer.exe",
}

func psProfileDirs() []string {
	docs := filepath.Join(homeDir(), "Documents")
	dirs := []string{filepath.Join(docs, "WindowsPowerShell")}
	if st, err := os.Stat(filepath.Join(docs, "PowerShell")); err == nil && st.IsDir() {
		dirs = append(dirs, filepath.Join(docs, "PowerShell"))
	}
	return dirs
}

func lockBlock() string {
	msg := "This PC uses BerwinCode. Please run BerwinCode.exe instead of opencode."
	return lockBegin + " (managed by BerwinCode, do not edit)\r\n" +
		`function opencode { Write-Host "` + msg + `" -ForegroundColor Red }` + "\r\n" +
		`function opencode.exe { Write-Host "` + msg + `" -ForegroundColor Red }` + "\r\n" +
		`function opencode.cmd { Write-Host "` + msg + `" -ForegroundColor Red }` + "\r\n" +
		lockEnd + "\r\n"
}

func lockPowerShell() (bool, error) {
	changed := false
	for _, d := range psProfileDirs() {
		prof := filepath.Join(d, "Microsoft.PowerShell_profile.ps1")
		data, err := os.ReadFile(prof)
		if err != nil {
			if !os.IsNotExist(err) {
				return changed, err
			}
			if err := os.MkdirAll(d, 0755); err != nil {
				return changed, err
			}
			if err := os.WriteFile(prof, []byte(lockBlock()), 0644); err != nil {
				return changed, err
			}
			changed = true
			continue
		}
		if strings.Contains(string(data), lockBegin) {
			continue
		}
		sep := "\r\n"
		if !strings.HasSuffix(string(data), "\n") {
			sep = "\r\n\r\n"
		}
		if err := os.WriteFile(prof, []byte(string(data)+sep+lockBlock()), 0644); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
}

func unlockPowerShell() (bool, error) {
	changed := false
	for _, d := range psProfileDirs() {
		prof := filepath.Join(d, "Microsoft.PowerShell_profile.ps1")
		data, err := os.ReadFile(prof)
		if err != nil {
			continue
		}
		s := string(data)
		if !strings.Contains(s, lockBegin) {
			continue
		}
		block := lockBlock()
		done := false
		for _, sep := range []string{"\r\n\r\n", "\r\n", "\n\n", "\n", ""} {
			if sep == "" {
				if s == block {
					_ = os.Remove(prof)
					done = true
				}
				break
			}
			if strings.HasSuffix(s, sep+block) {
				out := strings.TrimSuffix(s, sep+block)
				if strings.TrimSpace(out) == "" {
					_ = os.Remove(prof)
				} else if err := os.WriteFile(prof, []byte(out), 0644); err != nil {
					return changed, err
				}
				done = true
				break
			}
		}
		if done {
			changed = true
			continue
		}
		var kept []string
		skip := false
		for _, ln := range strings.Split(s, "\n") {
			t := strings.TrimRight(ln, "\r")
			if strings.Contains(t, lockBegin) {
				skip = true
				continue
			}
			if strings.Contains(t, lockEnd) {
				skip = false
				continue
			}
			if skip {
				continue
			}
			kept = append(kept, ln)
		}
		out := strings.Join(kept, "\n")
		if strings.TrimSpace(out) == "" {
			_ = os.Remove(prof)
		} else if err := os.WriteFile(prof, []byte(out), 0644); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
}

// regRun runs reg.exe and returns an error carrying reg's own message
// (plain .Run() only yields a bare "exit status 1").
func regRun(args ...string) error {
	out, err := exec.Command("reg", args...).Output()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("reg %s: %s", strings.Join(args, " "), msg)
	}
	return nil
}

func regQueryAutoRun() (string, bool) {
	out, err := exec.Command("reg", "query", autoRunKey, "/v", "AutoRun").Output()
	if err != nil {
		return "", false
	}
	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimRight(ln, "\r")
		if !strings.Contains(ln, "AutoRun") {
			continue
		}
		for _, typ := range []string{"REG_SZ", "REG_EXPAND_SZ"} {
			if i := strings.Index(ln, typ); i >= 0 {
				return strings.TrimSpace(ln[i+len(typ):]), true
			}
		}
	}
	return "", false
}

func regSetAutoRun(v string) error {
	return exec.Command("reg", "add", autoRunKey, "/v", "AutoRun", "/t", "REG_SZ", "/d", v, "/f").Run()
}

func regDelAutoRun() error {
	return exec.Command("reg", "delete", autoRunKey, "/v", "AutoRun", "/f").Run()
}

func hasBerwinMark(s string) bool {
	return strings.Contains(strings.ToLower(s), "berwincode")
}

func cmdBatchPath() string {
	return filepath.Join(berwinDataDir(), "cmdlock.bat")
}

func autoRunBakPath() string {
	return filepath.Join(berwinDataDir(), "autorun.bak")
}

func lockCmd(permanent bool) (bool, error) {
	if isPermanentLock() {
		permanent = true // once permanent, every lock stays permanent
	}
	batch := cmdBatchPath()
	if permanent {
		batch = permanentBatchPath()
	}
	content := "@echo off\r\n" +
		"doskey opencode=echo This PC uses BerwinCode. Please run BerwinCode.exe instead of opencode.\r\n" +
		"doskey opencode.exe=echo This PC uses BerwinCode. Please run BerwinCode.exe instead of opencode.\r\n"
	_ = os.MkdirAll(filepath.Dir(batch), 0755)
	if err := os.WriteFile(batch, []byte(content), 0644); err != nil {
		return false, err
	}
	if permanent {
		markPermanent()
	}
	cur, present := regQueryAutoRun()
	want := "\"" + batch + "\""
	if strings.Contains(cur, want) {
		return false, nil
	}
	nv := stripBerwinBatches(cur)
	if strings.TrimSpace(nv) != "" {
		// The user had their own AutoRun content: keep it, chain ours after.
		if present && !hasBerwinMark(cur) {
			_ = os.WriteFile(autoRunBakPath(), []byte(cur), 0644)
		}
		return true, regSetAutoRun(nv + " & " + want)
	}
	_ = os.Remove(autoRunBakPath())
	return true, regSetAutoRun(want)
}

// stripBerwinBatches removes our blocker refs (both batch locations) from an
// AutoRun value while keeping anything else the user had.
func stripBerwinBatches(s string) string {
	for _, b := range []string{cmdBatchPath(), permanentBatchPath()} {
		q := "\"" + b + "\""
		s = strings.Replace(s, " & "+q, "", 1)
		if s == q {
			s = ""
		} else {
			s = strings.Replace(s, q, "", 1)
		}
	}
	return strings.TrimSpace(s)
}

func unlockCmd(force bool) (bool, error) {
	if isPermanentLock() && !force {
		return false, fmt.Errorf("permanent block is active (survives uninstall); remove with BerwinCode.exe unlock-opencode --force")
	}
	cur, present := regQueryAutoRun()
	if !present || !hasBerwinMark(cur) {
		_ = os.Remove(cmdBatchPath())
		_ = os.Remove(permanentBatchPath())
		if force {
			_ = os.Remove(permanentMarkerPath())
		}
		return false, nil
	}
	batch := cmdBatchPath()
	if bkb, err := os.ReadFile(autoRunBakPath()); err == nil {
		_ = os.Remove(autoRunBakPath())
		_ = os.Remove(batch)
		_ = os.Remove(permanentBatchPath())
		_ = os.Remove(permanentMarkerPath())
		if strings.TrimSpace(string(bkb)) == "" {
			return true, regDelAutoRun()
		}
		return true, regSetAutoRun(string(bkb))
	}
	_ = os.Remove(batch)
	_ = os.Remove(permanentBatchPath())
	_ = os.Remove(permanentMarkerPath())
	nv := stripBerwinBatches(cur)
	if nv == "" {
		return true, regDelAutoRun()
	}
	return true, regSetAutoRun(nv)
}

func cmdLock(permanent bool) int {
	c1, e1 := lockPowerShell()
	c2, e2 := lockCmd(permanent)
	c3, e3 := lockDisallowRun()
	for _, e := range []error{e1, e2, e3} {
		if e != nil {
			fmt.Fprintf(os.Stderr, "BerwinCode: lock failed: %v\n", e)
		}
	}
	if e1 != nil || e2 != nil || e3 != nil {
		return 1
	}
	if !c1 && !c2 && !c3 {
		if permanent || isPermanentLock() {
			fmt.Println("BerwinCode: permanent opencode block already active (survives uninstall).")
		} else {
			fmt.Println("BerwinCode: opencode command already blocked.")
		}
		return 0
	}
	if permanent || isPermanentLock() {
		fmt.Println("BerwinCode: opencode PERMANENTLY blocked (shells + run-by-name, survives uninstall).")
		fmt.Println("Run BerwinCode.exe unlock-opencode --force to remove.")
		return 0
	}
	fmt.Println("BerwinCode: opencode command blocked in PowerShell and CMD.")
	fmt.Println("Run BerwinCode.exe unlock-opencode to restore.")
	return 0
}

func cmdUnlock(force bool) int {
	if isPermanentLock() && !force {
		fmt.Println("BerwinCode: permanent opencode block is active (survives uninstall).")
		fmt.Println("Run BerwinCode.exe unlock-opencode --force to remove it.")
		return 1
	}
	c1, e1 := unlockPowerShell()
	c2, e2 := unlockCmd(force)
	c3, e3 := unlockDisallowRun()
	for _, e := range []error{e1, e2, e3} {
		if e != nil {
			fmt.Fprintf(os.Stderr, "BerwinCode: unlock failed: %v\n", e)
		}
	}
	if e1 != nil || e2 != nil || e3 != nil {
		return 1
	}
	if !c1 && !c2 && !c3 {
		fmt.Println("BerwinCode: no opencode block found. Nothing to restore.")
		return 0
	}
	fmt.Println("BerwinCode: opencode command restored.")
	return 0
}

// ------------------------------------------------- DisallowRun name block
//
// Explorer-level block by file NAME (HKCU, no admin): double-clicking or
// launching opencode.exe (or its common installer names) is refused even
// for copies that do not exist yet or live in other folders. Takes effect
// for new Explorer sessions (log off/on if it seems ignored).

func disallowStatePath() string { return filepath.Join(berwinDataDir(), "disallow.json") }

// parseRegValues parses `reg query <key>` output into name->data.
func parseRegValues(out string) map[string]string {
	m := map[string]string{}
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimRight(ln, "\r")
		f := strings.Fields(ln)
		if len(f) < 3 {
			continue
		}
		if f[1] != "REG_SZ" && f[1] != "REG_EXPAND_SZ" && f[1] != "REG_DWORD" {
			continue
		}
		m[f[0]] = strings.Join(f[2:], " ")
	}
	return m
}

func regQueryDisallow() (enabled bool, vals map[string]string) {
	vals = map[string]string{}
	out, err := exec.Command("reg", "query", explorerKey, "/v", "DisallowRun").Output()
	if err == nil {
		v := parseRegValues(string(out))["DisallowRun"]
		enabled = v == "0x1" || v == "1"
	}
	out, err = exec.Command("reg", "query", explorerKey+"\\"+disallowSub).Output()
	if err == nil {
		vals = parseRegValues(string(out))
	}
	return enabled, vals
}

func disallowWeEnabled() bool {
	data, err := os.ReadFile(disallowStatePath())
	if err != nil {
		return false
	}
	return strings.Contains(string(data), `"enabledByUs":true`)
}

func setDisallowState(enabledByUs bool) {
	_ = os.MkdirAll(berwinDataDir(), 0755)
	v := "false"
	if enabledByUs {
		v = "true"
	}
	_ = os.WriteFile(disallowStatePath(), []byte(`{"enabledByUs":`+v+`}`), 0644)
}

func lockDisallowRun() (bool, error) {
	changed := false
	enabled, vals := regQueryDisallow()
	have := map[string]bool{}
	for _, d := range vals {
		have[strings.ToLower(d)] = true
	}
	next := 1
	for len(vals) > 0 {
		if _, taken := vals[fmt.Sprint(next)]; !taken {
			break
		}
		next++
	}
	for _, n := range disallowNames {
		if have[strings.ToLower(n)] {
			continue
		}
		for {
			if _, taken := vals[fmt.Sprint(next)]; !taken {
				break
			}
			next++
		}
		if err := regRun("add", explorerKey+"\\"+disallowSub, "/v", fmt.Sprint(next), "/t", "REG_SZ", "/d", n, "/f"); err != nil {
			return changed, err
		}
		vals[fmt.Sprint(next)] = n
		next++
		changed = true
	}
	if !enabled {
		if err := regRun("add", explorerKey, "/v", "DisallowRun", "/t", "REG_DWORD", "/d", "1", "/f"); err != nil {
			return changed, err
		}
		setDisallowState(true)
		changed = true
	}
	return changed, nil
}

func unlockDisallowRun() (bool, error) {
	changed := false
	_, vals := regQueryDisallow()
	for name, data := range vals {
		want := false
		for _, n := range disallowNames {
			if strings.EqualFold(data, n) {
				want = true
				break
			}
		}
		if !want {
			continue
		}
		if err := regRun("delete", explorerKey+"\\"+disallowSub, "/v", name, "/f"); err != nil {
			return changed, err
		}
		delete(vals, name)
		changed = true
	}
	if len(vals) == 0 {
		_ = regRun("delete", explorerKey+"\\"+disallowSub, "/f")
		if disallowWeEnabled() {
			_ = regRun("add", explorerKey, "/v", "DisallowRun", "/t", "REG_DWORD", "/d", "0", "/f")
		}
		_ = os.Remove(disallowStatePath())
	}
	return changed, nil
}
