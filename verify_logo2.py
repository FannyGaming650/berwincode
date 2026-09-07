b = open("C:\\Users\\Berwin Maniquiz\\.berwincode\\bin\\berwincode.exe", "rb").read()
def fit(rendered, L):
    W = len(rendered)
    m = (L - W) // 5
    return ("".join("\\u%04x" % ord(c) for c in rendered[:m]) + rendered[m:]).encode("ascii")
new_l1 = fit("             BERWIN", 99)
new_r1 = fit("CODE" + " " * 15, 99)
old_l1 = fit("      BERWIN       ", 99)
print("new single-line BERWIN x2:", b.count(new_l1))
print("new single-line CODE x2:", b.count(new_r1))
print("old double BERWIN gone:", b.count(old_l1) == 0)
print("zen gone:", b.count(b"OpenCode Zen") == 0)
