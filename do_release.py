import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\make_release.ps1"
s = io.open(p, encoding="utf-8").read()
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("touch ok")
