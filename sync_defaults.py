import io
home = "C:\\Users\\Berwin Maniquiz"
live = {
    "defaultBerwinJSON": home + "\\.config\\berwincode\\berwincode.json",
    "defaultBerwinMD": home + "\\.config\\berwincode\\BERWINCODE.md",
    "defaultTuiJSON": home + "\\.config\\berwincode\\tui.json",
    "defaultAgentMD": home + "\\.config\\berwincode\\agents\\berwin-builder.md",
    "defaultBerwinAgentMD": home + "\\.config\\berwincode\\agents\\berwin.md",
}
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s = io.open(p, encoding="utf-8").read()
for func, path in live.items():
    content = io.open(path, encoding="utf-8").read()
    assert "`" not in content, func
    if not content.endswith("\n"):
        content += "\n"
    head = "func " + func + "() string {\n\treturn `"
    tail = "`\n}\n"
    i = s.find(head)
    assert i >= 0, func
    j = s.find(tail, i + len(head))
    assert j > i, func
    new = head + content + tail
    s = s[:i] + new + s[j + len(tail):]
    print("synced:", func, len(content), "bytes")
old = 'const berwinVersion = "1.4.0"'
assert s.count(old) == 1
s = s.replace(old, 'const berwinVersion = "1.4.1"')
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("WRITTEN + bumped 1.4.1")
