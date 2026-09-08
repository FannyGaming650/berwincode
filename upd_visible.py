import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\update.go"
s = io.open(p, encoding="utf-8").read()

def rep(old, new):
    global s
    assert s.count(old) == 1, old[:70]
    s = s.replace(old, new)

rep(r'''func checkForUpdate() (tag, setupURL, pageURL string, available bool) {
	if updateDisabled() {
		return "", "", "", false
	}
	tag, setupURL, pageURL, err := latestRelease()
	if err != nil || tag == "" {
		return "", "", "", false
	}
	if !newerAvailable(tag) {
		return "", "", "", false
	}
	return tag, setupURL, pageURL, true
}''', r'''func checkForUpdate() (tag, setupURL, pageURL string, available bool, checkErr error) {
	if updateDisabled() {
		return "", "", "", false, nil
	}
	tag, setupURL, pageURL, err := latestRelease()
	if err != nil || tag == "" {
		return "", "", "", false, err
	}
	if !newerAvailable(tag) {
		return "", "", "", false, nil
	}
	return tag, setupURL, pageURL, true, nil
}''')

rep(r'''func cmdUpgrade(interactive bool) int {
	tag, setupURL, pageURL, ok := checkForUpdate()
	if !ok {
		fmt.Printf("BerwinCode v%s is the latest version.\n", berwinVersion)
		return 0
	}''', r'''func cmdUpgrade(interactive bool) int {
	tag, setupURL, pageURL, ok, err := checkForUpdate()
	if !ok {
		if err != nil {
			fmt.Printf("BerwinCode: update check failed (%v). You have v%s.\n", err, berwinVersion)
			return 1
		}
		fmt.Printf("BerwinCode v%s is the latest version.\n", berwinVersion)
		return 0
	}''')

rep(r'''func maybeAutoUpdate() {
	tag, setupURL, pageURL, ok := checkForUpdate()
	if !ok {
		return
	}''', r'''// updateNote is shown in the login form footer so the check is visible.
var updateNote = ""

func maybeAutoUpdate() {
	fmt.Fprintf(os.Stderr, "BerwinCode v%s: checking for updates...\n", berwinVersion)
	tag, setupURL, pageURL, ok, err := checkForUpdate()
	if !ok {
		if err != nil {
			fmt.Fprintln(os.Stderr, "BerwinCode: update server unreachable, continuing offline.")
			updateNote = "update check offline"
		} else {
			fmt.Fprintln(os.Stderr, "BerwinCode: up to date.")
			updateNote = "up to date"
		}
		return
	}
	updateNote = "updating to " + tag + "..."''')

io.open(p, "w", encoding="utf-8", newline="").write(s)
print("update.go visible")
