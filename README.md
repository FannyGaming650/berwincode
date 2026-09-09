# BerwinCode

BerwinCode is Berwin's personal AI coding agent for the Windows terminal.
Double-click the exe, log in, and code.

It is an independent project, not affiliated with or endorsed by the
OpenCode team. The terminal engine is built from the open-source OpenCode
project (https://github.com/anomalyco/opencode, MIT) with BerwinCode
branding patches applied automatically at install time.

## Download

Get `BerwinCode-Setup-1.6.9.exe` from the Releases page (recommended, no admin
rights), or the portable `BerwinCode-v1.6.9-windows-x64.zip`.
Just ~5MB -- everything else downloads itself on first run (~205MB,
one-time: portable Node LTS + engine).

The installer always removes any stock `opencode` command on that PC
(no checkbox, cannot be skipped in the wizard) so nobody can bypass the
BerwinCode login by typing `opencode` in a terminal. Locked files are
scheduled for deletion on reboot.

The block is permanent and has two layers, applied with no admin rights:
1. Shell layer: PowerShell profile functions plus a CMD AutoRun doskey
   script print "This PC uses BerwinCode. Please run BerwinCode.exe
   instead of opencode." instead of running anything.
2. Name layer: Explorer DisallowRun (HKCU) refuses to launch
   `opencode.exe` and its common installer names, even for copies that
   do not exist yet or live in other folders (takes effect for new
   sessions; log off/on if it seems ignored). If that key is
   admin-protected on a PC, the shell layer still applies and the
   installer reports which layer failed.
Uninstalling BerwinCode does NOT lift the block. To remove it on purpose:
`BerwinCode.exe unlock-opencode --force`.

Owner override (your PC only, keeps your stock install):
`BerwinCode-Setup-1.6.9.exe /KEEPOPENCODE=1`
or set env `BERWINCODE_KEEPOPENCODE=1` before running it.

## First run

1. Double-click `BerwinCode.exe`.
2. If Node.js / engine are missing, they download automatically with progress.
3. A BerwinCode Login form appears. Press Send Code, read the 6-digit
   code in your Discord channel, type it in, press Login.
   Codes expire after 5 minutes; Resend has a 60s cooldown.
   Every launch needs a fresh code -- nothing is saved.
4. The BerwinCode terminal opens.

A Discord webhook is built into the exe, so a fresh PC works with no
setup. Optional override per PC: `BerwinCode.exe set-webhook
<discord-webhook-url>` (saved to `%USERPROFILE%\.berwincode\discord.json`,
or set env `BERWINCODE_WEBHOOK`). Note: the built-in webhook URL is public
(it lives in this repo and in the exe), so keep it on a dedicated
login-codes channel and rotate it if abused.

Pick your AI model inside with `/models`, or press `ctrl+p` and run
`Hide tips` to hide the rotating tips.

## Daily use

```
BerwinCode.exe                 Open BerwinCode (login form, then terminal)
BerwinCode.exe run "prompt"    Run one prompt without the TUI
BerwinCode.exe auth login      Connect an AI provider (first time)
BerwinCode.exe agent list      List agents
BerwinCode.exe set-webhook <url>  Override the built-in Discord webhook on this PC
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
powershell -ExecutionPolicy Bypass -File .\release.ps1 -Version 1.6.9
```

Then upload the produced zip to a GitHub Release. See PUBLISHING below.

## License

MIT -- see LICENSE. Engine: MIT, belongs to its owners (see LICENSE).
