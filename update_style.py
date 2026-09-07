import io
p = "C:\\Users\\Berwin Maniquiz\\.config\\berwincode\\BERWINCODE.md"
s = io.open(p, encoding="utf-8").read()
start = s.find("## Beginner-human VB.NET voice")
assert start > 0
s = s[:start].rstrip("\n") + "\n"
s += """
## Beginner-human VB.NET voice (always on for VB.NET)

Reference style: C:\\Users\\Berwin Maniquiz\\Desktop\\payrollsystem\\Ichiban
(a real beginner project: Module db, Module crud, code inside the forms).
Match that project, not textbooks. All other tasks keep the normal
rules above.

### A. Design: always plain old VB look
- Classic gray form, default Windows buttons. NEVER restyle buttons:
  no FlatStyle, no custom BackColor, no images on buttons.
- At most one plain background color for the whole form, or just the
  default gray. Never many colors, gradients, or fancy panels.
- Standard controls only: Label, TextBox, Button, ComboBox, RadioButton,
  DataGridView, PictureBox, GroupBox. Simple top-to-bottom layout that
  follows the system being built (example: menu left, big title top,
  entry fields + grid below, Save/Clear buttons at the bottom).

### B. Code lives IN THE FORMS, a little messy is correct
- Write working code directly inside the form events
  (Button1_Click, Form_Load). Long straightforward handlers are GOOD.
- Do NOT create new Helper/Module/Class files for simple tasks.
- Do NOT split code into many small functions. Over-organizing is what
  professionals do; beginners keep everything in the same form.
- Copy-paste repetition is FINE. Never refactor to remove duplication.
- Only use a helper file if the project already has one. Never invent
  new abstractions, regions, or layers.

### C. Code texture: write like the payroll project
- Control names: defaults are fine (Button1, TextBox1) or simple ones
  (txtname, btnsave). Classic handler header:
  Private Sub Button1_Click(ByVal sender As System.Object, ByVal e As System.EventArgs) Handles Button1.Click
- Database: one Module db with a myconn() function returning
  New OleDb.OleDbConnection("Provider=...;Data Source=" & Application.StartupPath & "\\file.mdb").
  One Module crud with nearly identical insert/update/delete/subs taking
  (ByVal sql As String), using With cmd / .Connection / .CommandText.
- SQL built with & and .Text values: "select * from tbluser where name ='" & txtname.Text & "'".
  Numbers with Val(txtage.Text). Dates with #...#.
- Module-level shared Dim con, cmd, da, result, sql, table.
- Messages and errors with MsgBox: MsgBox("Saved!"), MsgBox(ex.Message, MsgBoxStyle.Information).
- Open with con.Open(), close with con.Close() after End Try.
- String flags with Select Case var / Case "employee".
- A comment on almost every step, in plain words. A few commented-out
  old lines left behind are fine and look authentic.
- NEVER use: LINQ, lambdas, generics, async extras, patterns, XML doc
  comments, regions. (Only keep Await where existing project code
  already requires it, e.g. BeginEnroll.)
- Explanations: plain words first, then the code, then one short
  "why it works" line. Casual and honest, never textbook.
"""
io.open(p, "w", encoding="utf-8", newline="").write(s)
print("style v2 written")
