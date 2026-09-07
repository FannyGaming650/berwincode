import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s = io.open(p, encoding="utf-8").read()

def rep(old, new):
    global s
    assert s.count(old) == 1, (old[:70], s.count(old))
    s = s.replace(old, new)

rep('const berwinVersion = "1.3.0"', 'const berwinVersion = "1.4.0"')

rep('''	if len(args) == 1 && args[0] == "reset-login" {
		resetLogin()
		return
	}''', '''	if len(args) == 1 && args[0] == "reset-login" {
		fmt.Println("The app login is now a Discord code. Nothing stored to reset.")
		_ = os.Remove(loginPath())
		return
	}''')

rep('''	fmt.Println(`  BerwinCode.exe reset-login     Remove the BerwinCode app login`)''',
'''	fmt.Println(`  BerwinCode.exe set-webhook <url>  Save your Discord webhook for login codes`)''')

rep('''	ensureConfig(false)
	if code := ensureBackend(); code != 0 {''', '''	ensureConfig(false)
	if len(args) == 2 && args[0] == "set-webhook" {
		setWebhook(args[1])
		return
	}
	if code := ensureBackend(); code != 0 {''')

start = s.find("// ---------------------------------------------------------------- login gate")
end = s.find("// ------------------------------------------------------- native dialogs")
assert start > 0 and end > start
s = s[:start] + '''// ---------------------------------------------------------------- login gate

func loginPath() string {
	return filepath.Join(berwinDataDir(), "login.json")
}

func discordCfgPath() string {
	return filepath.Join(berwinDataDir(), "discord.json")
}

func loadWebhook() string {
	data, err := os.ReadFile(discordCfgPath())
	if err != nil {
		return ""
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	u := strings.TrimSpace(m["webhook"])
	if !strings.HasPrefix(u, "https://discord.com/api/webhooks/") {
		return ""
	}
	return u
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
		fmt.Fprintf(os.Stderr, "Could not save webhook: %v\\n", err)
		os.Exit(1)
	}
	fmt.Println("Discord webhook saved. Codes will be sent there from now on.")
}

func runLoginGate() int {
	if runtime.GOOS != "windows" {
		return 0
	}
	wh := loadWebhook()
	if wh == "" {
		msgBox("BerwinCode", "Discord webhook is not set.\\n\\nRun:\\nBerwinCode.exe set-webhook <your-webhook-url>", 0x30)
		fmt.Fprintln(os.Stderr, "Discord webhook not set. Run: BerwinCode.exe set-webhook <url>")
		pauseEnter()
		return 1
	}
	return showLoginForm(wh)
}

''' + s[end:]
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("gate swapped")
