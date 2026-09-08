import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s = io.open(p, encoding="utf-8").read()

def rep(old, new):
    global s
    assert s.count(old) == 1, old[:70]
    s = s.replace(old, new)

rep('''	if len(args) == 0 && isConsole() {
		maybeOfferUpdate()
	}''', '''	if len(args) == 0 && isConsole() {
		maybeAutoUpdate()
	}''')

rep('const berwinVersion = "1.5.0"', 'const berwinVersion = "1.5.1"')
io.open(p, "w", encoding="utf-8", newline="").write(s)

p2 = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\update.go"
s2 = io.open(p2, encoding="utf-8").read()

def rep2(old, new):
    global s2
    assert s2.count(old) == 1, old[:70]
    s2 = s2.replace(old, new)

rep2('''func maybeOfferUpdate() {''', '''func maybeAutoUpdate() {
	tag, setupURL, pageURL, ok := checkForUpdate()
	if !ok {
		return
	}
	fmt.Fprintf(os.Stderr, "BerwinCode %s found - updating automatically...\n", tag)
	if setupURL == "" {
		fmt.Fprintf(os.Stderr, "No installer found. Get it here: %s\n", pageURL)
		return
	}
	dst := filepath.Join(os.TempDir(), "BerwinCode-"+strings.TrimPrefix(tag, "v")+"-setup.exe")
	if err := downloadFile(setupURL, dst); err != nil {
		fmt.Fprintf(os.Stderr, "Auto-update download failed: %v\nContinuing with v%s.\n", err, berwinVersion)
		return
	}
	_ = exec.Command(dst).Start()
	fmt.Fprintln(os.Stderr, "BerwinCode: installer started, closing so it can update. Reopen BerwinCode after.")
	os.Exit(0)
}

func maybeOfferUpdate() {''')

io.open(p2, "w", encoding="utf-8", newline="").write(s2)

p3 = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\BerwinCode.iss"
s3 = io.open(p3, encoding="utf-8").read()
old3 = 'Source: "BerwinCode.exe"; DestDir: "{app}"; Flags: ignoreversion'
assert s3.count(old3) == 1
s3 = s3.replace(old3, 'Source: "BerwinCode.exe"; DestDir: "{app}"; Flags: ignoreversion restartreplace')
io.open(p3, "w", encoding="utf-8", newline="").write(s3)
print("auto-update wired")
