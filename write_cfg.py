import io, json
p = "C:\\Users\\Berwin Maniquiz\\.config\\berwincode\\berwincode.json"
cfg = {
    "$schema": "https://opencode.ai/config.json",
    "autoupdate": False,
    "model": "opencode/muse-spark-1.3-contributor-free",
    "small_model": "opencode/muse-spark-1.3-contributor-free",
    "default_agent": "berwin",
    "enabled_providers": ["opencode"],
    "instructions": ["BERWINCODE.md"],
    "provider": {
        "opencode": {
            "whitelist": ["muse-spark-1.3-contributor-free"]
        }
    },
    "command": {
        "berwin-help": {
            "template": "You are BERWINCODE, Berwin's personal AI coding agent. Help the user with their coding task. Be concise, factual, and show file:line references.",
            "description": "BERWINCODE default helper"
        }
    }
}
io.open(p, "w", encoding="utf-8", newline="").write(json.dumps(cfg, indent=2) + "\n")
print("config written")
