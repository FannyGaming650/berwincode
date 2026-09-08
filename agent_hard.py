import io
home = "C:\\Users\\Berwin Maniquiz"
p2 = home + "\\.config\\berwincode\\agents\\berwin.md"
s2 = io.open(p2, encoding="utf-8").read()
old = """Work rules: be short and factual. Verify by reading files and running
code when reasonable. Prefer editing existing files. Reference code as
file_path:line_number."""
assert s2.count(old) == 1
s2 = s2.replace(old, """Work rules: be short and factual. Verify by reading files and running
code when reasonable. Prefer editing existing files. Reference code as
file_path:line_number.

Hard rules for VB.NET (no exceptions, no matter what the user asks to
rush): design lives in .Designer.vb files only, never write New Button,
Controls.Add, .Location, .Size, or .BackColor in event code; never
create mod folders or separate handler modules for new systems, all
SQL and logic stays inline in the form events.""")
io.open(p2, "w", encoding="utf-8", newline="").write(s2)
print("agent hardened")

p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s = io.open(p, encoding="utf-8").read()
content = io.open(p2, encoding="utf-8").read()
assert "`" not in content
head = "func defaultBerwinAgentMD() string {\n\treturn `"
tail = "`\n}\n"
i = s.find(head)
assert i >= 0
j = s.find(tail, i + len(head))
assert j > i
s = s[:i] + head + content + tail + s[j + len(tail):]
oldv = 'const berwinVersion = "1.6.3"'
assert s.count(oldv) == 1
s = s.replace(oldv, 'const berwinVersion = "1.6.4"')
for p3, pairs in [
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\BerwinCode.iss", [('"1.6.3"', '"1.6.4"')]),
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\README.md", [("1.6.3", "1.6.4")]),
]:
    t = io.open(p3, encoding="utf-8").read()
    for a, b in pairs:
        assert t.count(a) >= 1, (p3, a)
        t = t.replace(a, b)
    io.open(p3, "w", encoding="utf-8", newline="").write(t)
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("baked + bumped 1.6.4")
