b = open("C:\\Users\\Berwin Maniquiz\\.berwincode\\bin\\berwincode.exe", "rb").read()
o = b"Getting started"
start = 0
while True:
    j = b.find(o, start)
    if j < 0:
        break
    seg = b[max(0,j-60):j+len(o)+10]
    safe = "".join(chr(x) if 32 <= x < 127 else "." for x in seg)
    print("at", j, ":", safe)
    start = j + 1
