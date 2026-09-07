import io
home = "C:\\Users\\Berwin Maniquiz"
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s = io.open(p, encoding="utf-8").read()

def sync_func(func, path):
    global s
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
    s = s[:i] + head + content + tail + s[j + len(tail):]
    print("synced:", func, len(content), "bytes")

sync_func("defaultBerwinJSON", home + "\\.config\\berwincode\\berwincode.json")
sync_func("defaultBerwinMD", home + "\\.config\\berwincode\\BERWINCODE.md")
sync_func("defaultTuiJSON", home + "\\.config\\berwincode\\tui.json")
sync_func("defaultAgentMD", home + "\\.config\\berwincode\\agents\\berwin-builder.md")

old_ensure = '''	agentFile := filepath.Join(cfgDir, "agents", "berwin-builder.md")
	if _, err := os.Stat(agentFile); os.IsNotExist(err) {
		_ = os.WriteFile(agentFile, []byte(defaultAgentMD()), 0644)
	}
}'''
assert s.count(old_ensure) == 1
s = s.replace(old_ensure, '''	agentFile := filepath.Join(cfgDir, "agents", "berwin-builder.md")
	if _, err := os.Stat(agentFile); os.IsNotExist(err) {
		_ = os.WriteFile(agentFile, []byte(defaultAgentMD()), 0644)
	}
	berwinFile := filepath.Join(cfgDir, "agents", "berwin.md")
	if _, err := os.Stat(berwinFile); os.IsNotExist(err) {
		_ = os.WriteFile(berwinFile, []byte(defaultBerwinAgentMD()), 0644)
	}
}''')

content = io.open(home + "\\.config\\berwincode\\agents\\berwin.md", encoding="utf-8").read()
assert "`" not in content
if not content.endswith("\n"):
    content += "\n"
anchor = "func defaultBerwinJSON() string {"
assert s.count(anchor) == 1
s = s.replace(anchor, "func defaultBerwinAgentMD() string {\n\treturn `" + content + "`\n}\n\n" + anchor)
print("inserted: defaultBerwinAgentMD", len(content), "bytes")

old = 'const berwinVersion = "1.4.0"'
assert s.count(old) == 1
s = s.replace(old, 'const berwinVersion = "1.4.1"')
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("WRITTEN + bumped 1.4.1")
