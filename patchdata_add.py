DST = "C:\\Users\\Berwin Maniquiz\\.berwincode\\bin\\berwincode.exe"
b = open(DST, "rb").read()

def pad(core, L):
    assert len(core) <= L, (core, len(core), L)
    return core + " " * (L - len(core))

o1 = "OpenCode includes free models so you can start immediately."
n1 = pad("BerwinCode includes free models - start now", len(o1))
o2 = "Connect from 75+ providers to use other models, including Claude, GPT, Gemini etc"
n2 = pad("Your BerwinCode model is ready - press Tab to see agents", len(o2))
o3 = 'S1("Getting started")'
n3 = 'S1("BerwinCode tips")'
assert b.count(o1.encode()) == 2
assert b.count(o2.encode()) == 1
assert b.count(o3.encode()) == 1
print("counts ok: 2, 1, 1")

def goq(s):
    return '"' + s.replace("\\", "\\\\").replace('"', '\\"') + '"'
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\patchdata.go"
s = open(p, encoding="ascii").read()
assert s.rstrip().endswith("}")
i = s.rfind("}")
add = "".join("\t{" + goq(o) + ", " + goq(n) + "},\n" for o, n in [(o1, n1), (o2, n2), (o3, n3)])
s = s[:i] + add + s[i:]
open(p, "w", encoding="ascii", newline="").write(s)
print("patchdata.go: 3 notice pairs queued")
