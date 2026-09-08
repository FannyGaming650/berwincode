import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s = io.open(p, encoding="utf-8").read()
content = io.open("C:\\Users\\Berwin Maniquiz\\.config\\berwincode\\BERWINCODE.md", encoding="utf-8").read()
assert "`" not in content
if not content.endswith("\n"):
    content += "\n"
head = "func defaultBerwinMD() string {\n\treturn `"
tail = "`\n}\n"
i = s.find(head)
assert i >= 0
j = s.find(tail, i + len(head))
assert j > i
s = s[:i] + head + content + tail + s[j + len(tail):]
old = 'const berwinVersion = "1.5.4"'
assert s.count(old) == 1
s = s.replace(old, 'const berwinVersion = "1.5.5"')
io.open(p, "w", encoding="utf-8", newline="").write(s)
for p2, pairs in [
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\BerwinCode.iss", [('"1.5.4"', '"1.5.5"')]),
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\README.md", [("1.5.4", "1.5.5")]),
]:
    t = io.open(p2, encoding="utf-8").read()
    for a, b in pairs:
        assert t.count(a) >= 1, (p2, a)
        t = t.replace(a, b)
    io.open(p2, "w", encoding="utf-8", newline="").write(t)
print("synced + bumped 1.5.5")
