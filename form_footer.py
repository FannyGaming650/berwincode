import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\loginform.go"
s = io.open(p, encoding="utf-8").read()

def rep(old, new):
    global s
    assert s.count(old) == 1, old[:70]
    s = s.replace(old, new)

rep("\tstatusHwnd uintptr\n", "\tstatusHwnd uintptr\n\tfootHwnd   uintptr\n")

rep(r'''	statusHwnd = makeCtl("STATIC", "", base, 16, 134, 332, 44, formHwnd, 0, hInst)''',
r'''	statusHwnd = makeCtl("STATIC", "", base, 16, 134, 332, 44, formHwnd, 0, hInst)
	note := updateNote
	if note == "" {
		note = "starting"
	}
	footHwnd = makeCtl("STATIC", "BerwinCode v" + berwinVersion + " - " + note, base, 16, 186, 332, 16, formHwnd, 0, hInst)''')

rep("\tplace(statusHwnd)\n", "\tplace(statusHwnd)\n\tplace(footHwnd)\n")
io.open(p, "w", encoding="utf-8", newline="").write(s)

p2 = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\update.go"
s2 = io.open(p2, encoding="utf-8").read()
i = s2.find("func maybeOfferUpdate() {")
assert i > 0
j = s2.find("\n}\n", i)
assert j > i
s2 = s2[:i] + s2[j + len("\n}\n"):]
io.open(p2, "w", encoding="utf-8", newline="").write(s2)
print("footer added, dead code removed")
