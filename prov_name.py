import io, json
p = "C:\\Users\\Berwin Maniquiz\\.config\\berwincode\\berwincode.json"
cfg = json.loads(io.open(p, encoding="utf-8").read())
cfg["provider"]["opencode"]["name"] = "Berwin"
io.open(p, "w", encoding="utf-8", newline="").write(json.dumps(cfg, indent=2) + "\n")
print("provider renamed in live config")
