# BerwinCode release helper. Usage:
#   powershell -ExecutionPolicy Bypass -File .\release.ps1 -Version 1.3.0
param([Parameter(Mandatory = $true)][string]$Version)
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

$main = ".\main.go"
$orig = Get-Content -LiteralPath $main -Raw
if ($orig -notmatch 'const berwinVersion = "([^"]+)"') { throw "version const not found" }
$old = $Matches[1]
(Get-Content -LiteralPath $main -Raw).Replace(
  'const berwinVersion = "' + $old + '"',
  'const berwinVersion = "' + $Version + '"'
) | Set-Content -LiteralPath $main -NoNewline

try {
  go vet ./...
  if ($LASTEXITCODE -ne 0) { throw "go vet failed" }
  go build -trimpath -o "BerwinCode.exe" .
  if ($LASTEXITCODE -ne 0) { throw "go build failed" }
  .\BerwinCode.exe --version
  $zip = "BerwinCode-v$Version-windows-x64.zip"
  if (Test-Path $zip) { Remove-Item $zip -Force }
  Compress-Archive -LiteralPath @(".\BerwinCode.exe", ".\README.md", ".\LICENSE") -DestinationPath $zip
  $hash = (Get-FileHash $zip -Algorithm SHA256).Hash
  $iscc = "$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe"
  if (Test-Path $iscc) {
    & $iscc "$PSScriptRoot\BerwinCode.iss"
    Write-Host "Installer built."
  } else {
    Write-Host "Inno Setup not found, skipped installer (winget install JRSoftware.InnoSetup)."
  }
  Write-Host ""
  Write-Host "Release ready: $zip"
  Write-Host "SHA256: $hash"
  Write-Host ""
  Write-Host "Next: create a GitHub Release (web UI, no tools needed) and upload the zip."
} finally {
  (Get-Content -LiteralPath $main -Raw).Replace(
    'const berwinVersion = "' + $Version + '"',
    'const berwinVersion = "' + $old + '"'
  ) | Set-Content -LiteralPath $main -NoNewline
  go build -trimpath -o "BerwinCode.exe" .
}
