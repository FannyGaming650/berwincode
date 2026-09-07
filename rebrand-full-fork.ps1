# BERWINCODE full-source fork helper
# Clones https://github.com/anomalyco/opencode and renames branding to BERWINCODE.
# NOTE: Upstream Windows native build is experimental. Recommended: run build in WSL,
# or just use BerwinCode.exe wrapper (Go) which already works on Windows.

param(
  [string]$Dest = "$HOME\berwincode-source",
  [switch]$Build,
  [switch]$SkipClone
)

$ErrorActionPreference = "Stop"
$Repo = "https://github.com/anomalyco/opencode"

if (-not $SkipClone) {
  if (Test-Path $Dest) { Write-Host "Dest exists: $Dest (use -SkipClone or delete)"; }
  else {
    Write-Host "Cloning $Repo -> $Dest ..."
    git clone --depth 1 $Repo $Dest
  }
}
Set-Location $Dest
Write-Host "Repo: $(Get-Location)"

# --- 1. Rename package names (safe, minimal) ---
# root package.json: "name": "opencode" -> "berwincode"
# packages/opencode/package.json bin: opencode -> berwincode
Write-Host ""
Write-Host "=== Renaming branding (opencode -> berwincode) ==="

$files = @(
  "package.json",
  "packages/opencode/package.json",
  "packages/sdk/js/package.json",
  "install"
)

foreach ($f in $files) {
  if (Test-Path $f) { Write-Host " - will patch: $f" }
}

# PowerShell replace (case-sensitive for brand, keep URLs working):
# 1) root package.json name
if (Test-Path "package.json") {
  (Get-Content package.json -Raw) `
    -replace ''"name": "opencode"'', ''"name": "berwincode"'' `
    -replace ''"url": "https://github.com/anomalyco/opencode"'', ''"url": "https://github.com/anomalyco/opencode"'' |
    Set-Content package.json -NoNewline
  Write-Host "patched package.json name -> berwincode"
}

# 2) Show where else to rename manually (TUI title, config dirs, env vars):
Write-Host ""
Write-Host "Manual rename checklist (search in repo for these):"
Write-Host "  - packages/opencode/src/cli/cmd/tui/* : TUI title 'opencode' -> 'BERWINCODE'"
Write-Host "  - packages/opencode/src/global/* or config/* : ~/.config/opencode -> ~/.config/berwincode"
Write-Host "  - OPENCODE_CONFIG / OPENCODE_CONFIG_DIR -> support BERWINCODE_CONFIG too"
Write-Host "  - install script: OPENCODE_INSTALL_DIR / ~/.opencode/bin -> BERWINCODE_INSTALL_DIR / ~/.berwincode/bin"
Write-Host "  - sst.config.ts / packages/opencode/script/build.ts : dist/opencode-<platform> -> dist/berwincode-<platform>"
Write-Host "  - docs + README: opencode.ai -> your site"
Write-Host ""
Write-Host "Quick search:"
Write-Host "  rg -l ''opencode'' packages/opencode/src --max-count=1 | head -20"

if ($Build) {
  Write-Host ""
  Write-Host "=== Building (bun 1.3+ required) ==="
  bun install
  # Single binary per CONTRIBUTING.md:
  # ./packages/opencode/script/build.ts --single
  bun ./packages/opencode/script/build.ts --single
  Write-Host ""
  Write-Host "Build output: ./packages/opencode/dist/"
  Get-ChildItem ./packages/opencode/dist -Recurse -Filter "*berwin*" -ErrorAction SilentlyContinue | Select-Object FullName
  Get-ChildItem ./packages/opencode/dist -Recurse -Filter "opencode*" -ErrorAction SilentlyContinue | Select-Object FullName | Select-Object -First 10
  Write-Host ""
  Write-Host "Rename dist/opencode-<platform>/bin/opencode -> berwincode.exe manually if needed."
  Write-Host "Windows native target may not exist upstream - use WSL: wsl bun ./packages/opencode/script/build.ts --single"
} else {
  Write-Host ""
  Write-Host "Skipped build. To build later:"
  Write-Host "  cd $Dest"
  Write-Host "  bun install"
  Write-Host "  bun ./packages/opencode/script/build.ts --single"
  Write-Host "  ./packages/opencode/dist/opencode-<platform>/bin/opencode --version"
}

Write-Host ""
Write-Host "Done. For daily Windows use, BerwinCode.exe wrapper is recommended (already built)."
