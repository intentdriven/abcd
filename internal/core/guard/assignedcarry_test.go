package guard

import "testing"

// TestAVariableAnExpansionOrReadSetsIsFollowed: a name the line assigns by
// `${NAME:=v}` or `${NAME=v}`, and REPLY, which a bare `read` or a `select`
// sets, are the line's own variables, so a word holding one still reads
// every file the line wrote; and a word whose fixed tail is `/NAME` can
// name the file the line wrote as the relative NAME.
func TestAVariableAnExpansionOrReadSetsIsFollowed(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`pgrep make > p; : ${f:=p}; kill $(cat $f)`, VerdictBlock, "kill-by-search"},
		{`pgrep make > p; echo ${f=p} > /dev/null; kill $(cat "$f")`, VerdictBlock, "kill-by-search"},
		{`pgrep make > p; read < q; kill $(cat $REPLY)`, VerdictBlock, "kill-by-search"},
		{`pgrep make > p; select x in a; do kill $(cat $REPLY); done`, VerdictBlock, "kill-by-search"},
		{`pgrep make > p; kill $(cat $PWD/p)`, VerdictBlock, "kill-by-search"},
		{`pgrep make > sub/p; xargs kill < "$PWD"/sub/p`, VerdictBlock, "kill-by-search"},

		{`pgrep make > search.log; : ${f:=x}; echo $pid | xargs kill`, VerdictAllow, ""},
		{`pgrep make > p; kill $(cat $PWD/q)`, VerdictAllow, ""},
		{`pgrep -l make > search.log; echo $pid | xargs kill`, VerdictAllow, ""},
	})
}
