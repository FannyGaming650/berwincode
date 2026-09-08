import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\update.go"
s = io.open(p, encoding="utf-8").read()
old = "func maybeOfferUpdate() {"
assert s.count(old) == 1
new = r'''func maybeAutoUpdate() {
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

func maybeOfferUpdate() {'''
s = s.replace(old, new)
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("inserted")
