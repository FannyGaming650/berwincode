import io, difflib
home = "C:\\Users\\Berwin Maniquiz"
for f in ["berwincode.json", "BERWINCODE.md", "agents\\berwin-builder.md"]:
    print("=" * 20, f)
    gofn = {"berwincode.json": "defaultBerwinJSON", "BERWINCODE.md": "defaultBerwinMD",
            "agents\\berwin-builder.md": "defaultAgentMD"}[f]
    src = io.open("C:\\Users\\Berwin Maniquiz\\BerwinCode\\main.go", encoding="utf-8").read()
    i = src.find("func " + gofn + "() string {")
    seg = src[i:i + 200]
    print("func region head:", repr(seg[:120]))
    break
