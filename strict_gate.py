import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s = io.open(p, encoding="utf-8").read()

def rep(old, new):
    global s
    assert s.count(old) == 1, old[:70]
    s = s.replace(old, new)

rep('''	// Interactive TUI launch: hide terminal, show login form, no banner.
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
	}''', '''	// Strict gate: every engine launch needs a fresh Discord verification.
	// The TUI additionally hides the terminal while the form shows.
	if runtime.GOOS == "windows" {
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
			os.Exit(code)
		}
		if isTUI {
			setTerminalTitle("BerwinCode")
		}
	}''')

rep('Run a prompt in terminal (no TUI, no login)', 'Run a prompt in terminal (needs verification)')
rep('const berwinVersion = "1.4.2"', 'const berwinVersion = "1.4.3"')

io.open(p, "w", encoding="utf-8", newline="").write(s)
print("strict gate + 1.4.3")
