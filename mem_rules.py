import io
p = "C:\\Users\\Berwin Maniquiz\\.config\\berwincode\\BERWINCODE.md"
s = io.open(p, encoding="utf-8").read()

def rep(old, new):
    global s
    assert s.count(old) == 1, old[:70]
    s = s.replace(old, new)

rep("""- Do NOT create new Helper/Module/Class files for simple tasks.
- Do NOT split code into many small functions. Over-organizing is what
  professionals do; beginners keep everything in the same form.""",
"""- NEVER create mod/ folders or separate handler modules (db, crud,
  select helpers) for NEW systems. All connection strings, SQL, and
  logic live inline in the form events, even if repeated per form.
  Use Module files ONLY when recoding a project that already has them.
- Do NOT split code into many small functions. Over-organizing is what
  professionals do; beginners keep everything in the same form.""")

rep("""### C. Code texture: write like the payroll project""",
"""### C. Code texture: write like the payroll project (shapes, not files)""")

rep("""- Database: one Module db with a myconn() function returning
  New OleDb.OleDbConnection("Provider=...;Data Source=" & Application.StartupPath & "\\file.mdb").
  One Module crud with nearly identical insert/update/delete/subs taking
  (ByVal sql As String), using With cmd / .Connection / .CommandText.""",
"""- Database in new systems: NO Module files. Every form holds its own
  Dim con As New OleDb.OleDbConnection("Provider=...;Data Source=" & Application.StartupPath & "\\file.mdb")
  and runs SQL inline with local With cmd blocks. Study payroll mod\\db
  and mod\\crud for the query SHAPES, then inline them per form.""")

rep("""- Standard controls only: Label, TextBox, Button, ComboBox, RadioButton,
  DataGridView, PictureBox, GroupBox. Simple top-to-bottom layout that
  follows the system being built (example: menu left, big title top,
  entry fields + grid below, Save/Clear buttons at the bottom).""",
"""- Standard controls only: Label, TextBox, Button, ComboBox, RadioButton,
  DataGridView, PictureBox, GroupBox. Simple top-to-bottom layout that
  follows the system being built (example: menu left, big title top,
  entry fields + grid below, Save/Clear buttons at the bottom).

### D. Design lives in Designer files only
- Every control is placed in Visual Studio drag-drop. All creation,
  positions, sizes, and colors live in .Designer.vb and .resx ONLY.
- NEVER write New Button/TextBox, Controls.Add, .Location, .Size, or
  .BackColor in event code. Code-behind sets VALUES only
  (Text, DataSource, Visible, Checked).""")

rep("""project file included: the .sln solution file, the .vbproj project
file, every .vb form with its .Designer.vb and .resx. Simple style,
but nothing missing.""",
"""project file included. Skeleton FIRST, like VotingSys: one system
folder with the .sln at its root, the .vbproj, My Project folder,
then every form as .vb plus its .Designer.vb plus .resx, then code,
then the database file, then build. Simple style, but nothing missing.""")

io.open(p, "w", encoding="utf-8", newline="").write(s)
print("memory rules updated")
