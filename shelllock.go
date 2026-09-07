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

func psProfileDirs() []string {
	docs := filepath.Join(homeDir(), "Documents")
	dirs := []string{filepath.Join(docs, "WindowsPowerShell")}
	if st, err := os.Stat(filepath.Join(docs, "PowerShell")); err == nil && st.IsDir() {
		dirs = append(dirs, filepath.Join(docs, "PowerShell"))
	}
	return dirs
}

func lockBlock() string {
	msg := "The 'opencode' command is disabled on this PC. Use BerwinCode.exe instead."
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

func lockCmd() (bool, error) {
	batch := cmdBatchPath()
	content := "@echo off\r\n" +
		"doskey opencode=echo The 'opencode' command is disabled on this PC. Use BerwinCode.exe instead.\r\n" +
		"doskey opencode.exe=echo The 'opencode' command is disabled on this PC. Use BerwinCode.exe instead.\r\n"
	_ = os.MkdirAll(berwinDataDir(), 0755)
	if err := os.WriteFile(batch, []byte(content), 0644); err != nil {
		return false, err
	}
	cur, present := regQueryAutoRun()
	if hasBerwinMark(cur) {
		return false, nil
	}
	if present {
		_ = os.WriteFile(autoRunBakPath(), []byte(cur), 0644)
		return true, regSetAutoRun(cur + " & \"" + batch + "\"")
	}
	_ = os.Remove(autoRunBakPath())
	return true, regSetAutoRun("\"" + batch + "\"")
}

func unlockCmd() (bool, error) {
	cur, present := regQueryAutoRun()
	if !present || !hasBerwinMark(cur) {
		return false, nil
	}
	batch := cmdBatchPath()
	if bkb, err := os.ReadFile(autoRunBakPath()); err == nil {
		_ = os.Remove(autoRunBakPath())
		_ = os.Remove(batch)
		if strings.TrimSpace(string(bkb)) == "" {
			return true, regDelAutoRun()
		}
		return true, regSetAutoRun(string(bkb))
	}
	_ = os.Remove(batch)
	q := "\"" + batch + "\""
	nv := strings.Replace(cur, " & "+q, "", 1)
	if nv == cur {
		nv = strings.Replace(cur, q, "", 1)
	}
	nv = strings.TrimSpace(nv)
	if nv == "" {
		return true, regDelAutoRun()
	}
	return true, regSetAutoRun(nv)
}

func cmdLock() int {
	c1, e1 := lockPowerShell()
	c2, e2 := lockCmd()
	if e1 != nil {
		fmt.Fprintf(os.Stderr, "BerwinCode: PowerShell lock failed: %v\n", e1)
	}
	if e2 != nil {
		fmt.Fprintf(os.Stderr, "BerwinCode: CMD lock failed: %v\n", e2)
	}
	if e1 != nil || e2 != nil {
		return 1
	}
	if !c1 && !c2 {
		fmt.Println("BerwinCode: opencode command already blocked.")
		return 0
	}
	fmt.Println("BerwinCode: opencode command blocked in PowerShell and CMD.")
	fmt.Println("Run BerwinCode.exe unlock-opencode to restore.")
	return 0
}

func cmdUnlock() int {
	c1, e1 := unlockPowerShell()
	c2, e2 := unlockCmd()
	if e1 != nil {
		fmt.Fprintf(os.Stderr, "BerwinCode: PowerShell unlock failed: %v\n", e1)
	}
	if e2 != nil {
		fmt.Fprintf(os.Stderr, "BerwinCode: CMD unlock failed: %v\n", e2)
	}
	if e1 != nil || e2 != nil {
		return 1
	}
	if !c1 && !c2 {
		fmt.Println("BerwinCode: no opencode block found. Nothing to restore.")
		return 0
	}
	fmt.Println("BerwinCode: opencode command restored.")
	return 0
}
