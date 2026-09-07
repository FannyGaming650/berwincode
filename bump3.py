import io
for p, pairs in [
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\BerwinCode.iss", [('"1.4.0"', '"1.4.1"')]),
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\README.md", [("1.4.0", "1.4.1"), ("-Version 1.3.0", "-Version 1.4.1")]),
]:
    s = io.open(p, encoding="utf-8").read()
    for old, new in pairs:
        assert s.count(old) >= 1, (p, old)
        s = s.replace(old, new)
    io.open(p, "w", encoding="utf-8", newline="").write(s)
print("bumped 1.4.1")
