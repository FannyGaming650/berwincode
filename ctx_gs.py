b = open("C:\\Users\\Berwin Maniquiz\\.berwincode\\bin\\berwincode.exe", "rb").read()
for j in [112969966]:
    seg = b[max(0,j-200):j+120]
    safe = "".join(chr(x) if 32 <= x < 127 else "." for x in seg)
    print(safe)
