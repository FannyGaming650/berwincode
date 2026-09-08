import sqlite3
db = sqlite3.connect("file:C:\\Users\\Berwin Maniquiz\\.local\\share\\opencode\\opencode.db?mode=ro", uri=True)
tables = [r[0] for r in db.execute("select name from sqlite_master where type = 'table'")]
print("tables:", tables)
for t in tables:
    try:
        n = db.execute("select count(*) from " + t).fetchone()[0]
        print(t, "rows:", n)
    except Exception as e:
        print(t, "err:", e)
