import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\shelllock.go"
s = io.open(p, encoding="utf-8").read()
old = "Use BerwinCode instead."
assert s.count(old) == 3, s.count(old)
s = s.replace(old, "Use BerwinCode.exe instead.")
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("message updated")
