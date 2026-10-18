package guard

import "testing"

// TestApproveRefusesCodeReachingAnInterpreterOutsideItsFlag: code handed to an
// interpreter other than through -c or -e is still code the guard does not
// read: a here-document, a here-string, a pipe or a redirected file on the
// standard input of an interpreter that names no script file; an awk program
// that runs a command; a sed script that runs one; tar's options that run a
// program; and a vim or ex command holding a `!`.
func TestApproveRefusesCodeReachingAnInterpreterOutsideItsFlag(t *testing.T) {
	r := plainRig(t)
	code := refusedAs(inlineCodeEntryID)
	r.run(t, []approveCase{
		{"python3 <<'EOF'\nimport os\nEOF", code},
		{"python3 - <<'EOF'\nimport os\nEOF", code},
		{`python3 <<< 'import os'`, code},
		{`python3 < sub/x.py`, code},
		{`cat sub/x.py | python3`, code},
		{`cat sub/x.js | node -`, code},
		{`node --require ./hook.js < sub/x.js`, code},
		{`perl < sub/x.pl`, code},
		{`ruby -w < sub/x.rb`, code},
		{`cat x | sudo python3 /dev/stdin`, code},
		{`ex -s README <<'EOF'` + "\n%s/a/b/\nwq\nEOF", code},
		{`vim -es README < cmds.vim`, code},
		{`awk 'BEGIN { system("rm -rf x") }'`, code},
		{`awk '{ print | "sh" }' f`, code},
		{`gawk '{ "date" | getline d }' f`, code},
		{`awk -F: -v x=1 '{system ($1)}' /etc/passwd`, code},
		{`awk "$PROG" f`, code},
		{`sed 's/x/date/e' f`, code},
		{`sed -n '1e date' f`, code},
		{`sed -e '/a/ { e date' -e '}' f`, code},
		{`sed --expression='$e ls' f`, code},
		{`tar -cf x.tar --checkpoint=1 --checkpoint-action=exec=sh\ x.sh dir`, code},
		{`tar xf x.tar --to-command='sh -c x'`, code},
		{`tar -cf x.tar --use-compress-program=./z.sh dir`, code},
		{`tar -cf x.tar --rsh-command=./r.sh host:x dir`, code},
		{`vim -c '!rm -rf x' README`, code},
		{`vi +'r !date' README`, code},
		{`nvim --cmd 'silent !touch x' README`, code},
		{`ex -c 'w !sh' README`, code},
		// Data on the standard input of a script that is a file, or of a
		// command that runs nothing.
		{`python3 sub/x.py < input.txt`, nil},
		{`cat data.csv | python3 -m csvkit`, nil},
		{`node sub/x.js <<< 'data'`, nil},
		{`awk '{ print $1 }' f > sub/out.txt`, nil},
		{`awk -F, '/x|y/ { n++ } END { print n }' f`, nil},
		{`awk -f prog.awk f`, nil},
		{`sed -n '/^e/p' f`, nil},
		{`sed -e 's/e/E/g' -e '/^$/d' f`, nil},
		{`sed 's/a/b/w sub/changes.txt' f`, nil},
		{`sed -i '' 's/a/e/' sub/x.txt`, nil},
		{`tar -czf x.tar.gz --checkpoint=100 dir`, nil},
		{`vim -c 'wq' README`, nil},
		{`ex -c '%s/a/b/' -c 'wq' README`, nil},
	})
}

// TestApproveReadsWritesASedScriptNames: a sed script's w command writes the
// file it names, which is judged as every other write.
func TestApproveReadsWritesASedScriptNames(t *testing.T) {
	r := plainRig(t)
	r.run(t, []approveCase{
		{`sed -n 'w ~/.zshrc' f`, protectedOrRegistry},
		{`sed 's/a/b/w .mcp.json' f`, protectedOrRegistry},
		{`sed -n '/x/W .abcd/guard.json' f`, protectedOrRegistry},
	})
}
