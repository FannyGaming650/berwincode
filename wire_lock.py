import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s = io.open(p, encoding="utf-8").read()

def rep(old, new):
    global s
    assert s.count(old) == 1, old[:70]
    s = s.replace(old, new)

rep('''	if len(args) == 1 && args[0] == "upgrade-engine" {
		upgradeEngine()
		return
	}''', '''	if len(args) == 1 && args[0] == "upgrade-engine" {
		upgradeEngine()
		return
	}
	if len(args) == 1 && args[0] == "lock-opencode" {
		os.Exit(cmdLock())
	}
	if len(args) == 1 && args[0] == "unlock-opencode" {
		os.Exit(cmdUnlock())
	}''')

rep('''	fmt.Println(`  BerwinCode.exe upgrade-engine  Rebrand a new stock engine after npm upgrades`)''',
'''	fmt.Println(`  BerwinCode.exe upgrade-engine  Rebrand a new stock engine after npm upgrades`)
	fmt.Println(`  BerwinCode.exe lock-opencode    Block the opencode command in shells`)
	fmt.Println(`  BerwinCode.exe unlock-opencode  Restore the opencode command`)''')

io.open(p, "w", encoding="utf-8", newline="").write(s)
print("wired")
