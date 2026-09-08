import io
for p, pairs in [
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\BerwinCode.iss", [('"1.5.1"', '"1.5.2"')]),
    ("C:\\Users\\Berwin Maniquiz\\BerwinCode\\README.md", [("1.5.1", "1.5.2")]),
]:
    s = io.open(p, encoding="utf-8").read()
    for old, new in pairs:
        assert s.count(old) >= 1, (p, old)
        s = s.replace(old, new)
    io.open(p, "w", encoding="utf-8", newline="").write(s)
print("bumped")
