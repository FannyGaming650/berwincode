import zipfile
z = zipfile.ZipFile("C:\\Users\\Berwin Maniquiz\\BerwinCode\\fake-node.zip", "w")
top = "node-v24.20.0-win-x64/"
for name in ["node.exe", "npm.cmd", "node_modules/npm/bin/npm-cli.js", "README.md"]:
    z.writestr(top + name, b"fake-" + name.encode()[:10])
z.close()
print("fake zip written")
