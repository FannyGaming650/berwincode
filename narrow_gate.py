import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s = io.open(p, encoding="utf-8").read()
old = '''	// Strict gate: every engine launch needs a fresh Discord verification.
	// The TUI additionally hides the terminal while the form shows.
	if runtime.GOOS == "windows" {'''
assert s.count(old) == 1
s = s.replace(old, '''	// Strict gate: TUI and run need a fresh Discord verification.
	// Management commands (auth, models, agent list...) stay ungated.
	// The TUI additionally hides the terminal while the form shows.
	needsGate := len(args) == 0 || (len(args) > 0 && args[0] == "run")
	if runtime.GOOS == "windows" && needsGate {''')
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("narrowed")
