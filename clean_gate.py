import io, re
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go"
s = io.open(p, encoding="utf-8").read()

start = s.find("type credUIInfo struct")
assert start > 0, "credui block not found"
endmark = "// ---------------------------------------------------------------- helpers"
end = s.find(endmark, start)
assert end > start
# keep msgBox: it starts at "func msgBox"
mb = s.find("func msgBox", start)
assert start < mb < end
s = s[:start] + s[mb:end] + s[end:]

for imp in ['\t"crypto/rand"\n', '\t"crypto/sha256"\n', '\t"crypto/subtle"\n', '\t"encoding/hex"\n']:
    assert s.count(imp) == 1, imp
    s = s.replace(imp, "")

io.open(p, "w", encoding="utf-8", newline="").write(s)
print("dead code + imports cleaned")
