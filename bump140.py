import io
for p, pairs in [
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\BerwinCode.iss",
     [('#define MyAppVersion "1.3.0"', '#define MyAppVersion "1.4.0"')]),
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\README.md",
     [("BerwinCode-Setup-1.3.0.exe", "BerwinCode-Setup-1.4.0.exe"),
      ("BerwinCode-v1.3.0-windows-x64.zip", "BerwinCode-v1.4.0-windows-x64.zip")]),
]:
    s = io.open(p, encoding="utf-8").read()
    for old, new in pairs:
        assert s.count(old) >= 1, (p, old)
        s = s.replace(old, new)
    io.open(p, "w", encoding="utf-8", newline="").write(s)
print("bumped")
