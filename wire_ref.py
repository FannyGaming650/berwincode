import io

p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\BerwinCode.iss"
s = io.open(p, encoding="utf-8").read()
old = '''Source: "LICENSE"; DestDir: "{app}"; Flags: ignoreversion'''
assert s.count(old) == 1
s = s.replace(old, old + '''
Source: "reference\\*"; DestDir: "{userprofile}\\.config\\berwincode\\reference"; Flags: ignoreversion recursesubdirs''')
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("iss wired")

p2 = "C:\\Users\\Berwin Maniquiz\\.config\\berwincode\\BERWINCODE.md"
s2 = io.open(p2, encoding="utf-8").read()
old2 = """  C:\\Users\\Berwin Maniquiz\\Desktop\\payrollsystem\\Ichiban"""
assert s2.count(old2) == 1
s2 = s2.replace(old2, """  reference\\payrollsystem\\Ichiban inside your BerwinCode config folder
  (%USERPROFILE%\\.config\\berwincode\\reference\\payrollsystem\\Ichiban).
  The installer puts it there on every PC, so never use a personal
  Desktop path for the reference""")
io.open(p2, "w", encoding="utf-8", newline="").write(s2)
print("memory repointed")

p3 = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s3 = io.open(p3, encoding="utf-8").read()
content = io.open(p2, encoding="utf-8").read()
assert "`" not in content
if not content.endswith("\n"):
    content += "\n"
head = "func defaultBerwinMD() string {\n\treturn `"
tail = "`\n}\n"
i = s3.find(head)
assert i >= 0
j = s3.find(tail, i + len(head))
assert j > i
s3 = s3[:i] + head + content + tail + s3[j + len(tail):]
oldv = 'const berwinVersion = "1.5.3"'
assert s3.count(oldv) == 1
s3 = s3.replace(oldv, 'const berwinVersion = "1.5.4"')
io.open(p3, "w", encoding="utf-8", newline="").write(s3)
print("baked + bumped 1.5.4")
