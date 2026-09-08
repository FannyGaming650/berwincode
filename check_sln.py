import io, os, re
proj = "C:\\Users\\Berwin Maniquiz\\Desktop\\LibrarySystem\\LibrarySystem.vbproj"
sln = "C:\\Users\\Berwin Maniquiz\\Desktop\\LibrarySystem\\LibrarySystem.sln"
vbp = io.open(proj, encoding="utf-8", errors="replace").read()
files = re.findall(r"<Compile Include=\"([^\"]+)\"", vbp) + re.findall(r"<EmbeddedResource Include=\"([^\"]+)\"", vbp) + re.findall(r"<None Include=\"([^\"]+)\"", vbp)
root = os.path.dirname(proj)
missing = [f for f in files if not os.path.exists(os.path.join(root, f.replace("/", os.sep)))]
print("vbproj lists:", len(files), "files; missing:", len(missing))
for f in missing[:10]:
    print("  MISSING:", f)
ss = io.open(sln, encoding="utf-8", errors="replace").read()
print("sln references vbproj:", "LibrarySystem.vbproj" in ss)
print("sln configs:", "Debug|Any CPU" in ss)
