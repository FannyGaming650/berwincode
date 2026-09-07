# BerwinCode

BerwinCode is Berwin's personal AI coding agent for the Windows terminal.
Double-click the exe, log in, and code.

It is an independent project, not affiliated with or endorsed by the
OpenCode team. The terminal engine is built from the open-source OpenCode
project (https://github.com/anomalyco/opencode, MIT) with BerwinCode
branding patches applied automatically at install time.

## Download

Get `BerwinCode-Setup-1.3.0.exe` from the Releases page (recommended, no admin
rights), or the portable `BerwinCode-v1.3.0-windows-x64.zip`.
Just ~5MB -- everything else downloads itself on first run (~205MB,
one-time: portable Node LTS + engine).

The installer always removes any stock `opencode` command on that PC
(no checkbox, cannot be skipped in the wizard) so nobody can bypass the
BerwinCode login by typing `opencode` in a terminal. Locked files are
scheduled for deletion on reboot.

Owner override (your PC only, keeps your stock install):
`BerwinCode-Setup-1.3.0.exe /KEEPOPENCODE=1`
or set env `BERWINCODE_KEEPOPENCODE=1` before running it.

## First run

1. Double-click `BerwinCode.exe`.
2. If Node.js / engine are missing, they download automatically with progress.
3. A Windows form asks you to create your BerwinCode username + password.
4. Log in and the BerwinCode terminal opens.

Pick your AI model inside with `/models`, or press `ctrl+p` and run
`Hide tips` to hide the rotating tips.

## Daily use

```
BerwinCode.exe                 Open BerwinCode (login form, then terminal)
BerwinCode.exe run "prompt"    Run one prompt without the TUI
BerwinCode.exe auth login      Connect an AI provider (first time)
BerwinCode.exe agent list      List agents
BerwinCode.exe reset-login     Remove the BerwinCode app login
BerwinCode.exe upgrade-engine  Re-apply branding after `npm i -g opencode-ai`
BerwinCode.exe --help          Help
BerwinCode.exe --version       Version
```

Config lives in `%USERPROFILE%\.config\berwincode\` (`berwincode.json`,
`BERWINCODE.md` system prompt, `tui.json`). Engine lives in
`%USERPROFILE%\.berwincode\bin\berwincode.exe`.

Forgot your app login? Run `BerwinCode.exe reset-login`.

## The app login

The username/password gate keeps casual users out of your terminal. It is a
local gate, not encryption: anyone who can run the engine binary directly
can bypass it.

## Build from source

Requires Go 1.23+ only. No other downloads for the launcher itself.

```
cd BerwinCode
go build -trimpath -o BerwinCode.exe .
```

## Publish a new release (maintainers)

```
powershell -ExecutionPolicy Bypass -File .\release.ps1 -Version 1.3.0
```

Then upload the produced zip to a GitHub Release. See PUBLISHING below.

## License

MIT -- see LICENSE. Engine: MIT, belongs to its owners (see LICENSE).
