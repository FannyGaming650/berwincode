import io
p = "C:\\Users\\Berwin Maniquiz\\BerwinCode\\README.md"
s = io.open(p, encoding="utf-8").read()
old = """1. Double-click `BerwinCode.exe`.
2. If Node.js / engine are missing, they download automatically with progress.
3. A Windows form asks you to create your BerwinCode username + password.
4. Log in and the BerwinCode terminal opens."""
assert s.count(old) == 1
s = s.replace(old, """1. Double-click `BerwinCode.exe`.
2. If Node.js / engine are missing, they download automatically with progress.
3. A BerwinCode Login form appears. Press Send Code, read the 6-digit
   code in your Discord channel, type it in, press Login.
   Codes expire after 5 minutes; Resend has a 60s cooldown.
   Every launch needs a fresh code -- nothing is saved.
4. The BerwinCode terminal opens.

One-time setup per PC: `BerwinCode.exe set-webhook <discord-webhook-url>`
so it knows where to send codes. The webhook lives only in
`%USERPROFILE%\\.berwincode\\discord.json` on that PC, never in this repo.""")
old2 = "BerwinCode.exe reset-login     Remove the BerwinCode app login"
assert s.count(old2) == 1
s = s.replace(old2, "BerwinCode.exe set-webhook <url>  Save the Discord webhook for login codes")
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("readme login updated")
