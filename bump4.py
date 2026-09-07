import io
for p, pairs in [
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go", [("1.4.1", "1.4.2")]),
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\BerwinCode.iss", [('"1.4.1"', '"1.4.2"')]),
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\README.md", [("1.4.1", "1.4.2")]),
]:
    s = io.open(p, encoding="utf-8").read()
    for old, new in pairs:
        n = 0
        if old == "1.4.1":
            assert s.count(old) >= 1, (p, old)
            s = s.replace(old, new)
        else:
            assert s.count(old) >= 1, (p, old)
            s = s.replace(old, new)
    io.open(p, "w", encoding="utf-8", newline="").write(s)
print("bumped 1.4.2")
