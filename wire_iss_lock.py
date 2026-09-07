import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\BerwinCode.iss"
s = io.open(p, encoding="utf-8").read()

def rep(old, new):
    global s
    assert s.count(old) == 1, old[:70]
    s = s.replace(old, new)

rep('''[Run]
Filename: "{app}\\{#MyAppExe}"; Description: "Launch {#MyAppName}"; Flags: nowait postinstall skipifsilent''',
'''[Run]
Filename: "{app}\\{#MyAppExe}"; Description: "Launch {#MyAppName}"; Flags: nowait postinstall skipifsilent
Filename: "{app}\\{#MyAppExe}"; Parameters: "lock-opencode"; Flags: runhidden; Check: not RemovalSkipped; StatusMsg: "Blocking stock opencode command..."

[UninstallRun]
Filename: "{app}\\{#MyAppExe}"; Parameters: "unlock-opencode"; Flags: runhidden skipifdoesntexist''')

io.open(p, "w", encoding="utf-8", newline="").write(s)
print("iss wired")
