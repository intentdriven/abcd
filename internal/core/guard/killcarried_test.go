package guard

import "testing"

// TestAPidListFromASearchBlocksHoweverItTravels — iss-2610112214257996, in the
// spirit of ruling AS (product thinker, 2026-10-11): a pid list that came from
// a search blocks, a recorded pid does not. A search written to a file the
// same line then reads into a kill is a kill by name, and so is a `ps` listing
// a grep or an awk program filters by a name. A `ps` that reports the pid the
// caller names, and a pid file the caller recorded, stay allowed.
func TestAPidListFromASearchBlocksHoweverItTravels(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		// A file the line wrote from a search.
		{`pgrep make > p; kill $(cat p)`, VerdictBlock, "kill-by-search"},
		{`pgrep make > p; xargs kill < p`, VerdictBlock, "kill-by-search"},
		{`pgrep -f make > /tmp/pids; kill -9 $(< /tmp/pids)`, VerdictBlock, "kill-by-search"},
		{`pgrep make >> p && cat p | xargs kill`, VerdictBlock, "kill-by-search"},
		{`pgrep make > p; xargs -a p kill`, VerdictBlock, "kill-by-search"},
		{`pgrep make > p; while read q; do kill $q; done < p`, VerdictBlock, "kill-by-search"},
		{`pgrep make > p; read q < p; kill $q`, VerdictBlock, "kill-by-search"},
		{`pgrep make | sort > p; kill $(cat p)`, VerdictBlock, "kill-by-search"},
		{`lsof -t -i :8080 > p; kill $(cat p)`, VerdictBlock, "kill-by-holder"},
		// A ps listing filtered by a name.
		{`ps aux | grep make | awk '{print $2}' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`kill $(ps aux | grep '[m]ake' | awk '{print $2}')`, VerdictBlock, "kill-by-search"},
		{`ps -ef | grep -v grep | grep node | awk '{print $2}' | xargs kill -9`, VerdictBlock, "kill-by-search"},
		{`ps aux | awk '/make/ {print $2}' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps -eo pid,comm | awk '$2 == "make" {print $1}' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps -eo pid,args | grep -F 'make preflight' | cut -d' ' -f1 > p; kill $(cat p)`, VerdictBlock, "kill-by-search"},

		// A recorded pid, and looking.
		{`ps -o pid= -p 4242 | xargs kill`, VerdictAllow, ""},
		{`ps -o pid= -p "$pid" | xargs kill`, VerdictAllow, ""},
		{`ps -o pid= -p 4242 | awk '{print $1}' | xargs kill`, VerdictAllow, ""},
		{`kill $(cat pidfile)`, VerdictAllow, ""},
		{`echo $! > run.pid; kill $(cat run.pid)`, VerdictAllow, ""},
		{`pgrep make > p; cat p`, VerdictAllow, ""},
		{`pgrep -l make > procs.txt; wc -l procs.txt`, VerdictAllow, ""},
		{`ps aux | grep make`, VerdictAllow, ""},
		{`ps aux | grep make > procs.txt; cat procs.txt`, VerdictAllow, ""},
		{`pgrep make > p; kill $(cat q)`, VerdictAllow, ""},
		{`cat p | xargs kill; pgrep make > p`, VerdictAllow, ""},
	})
}

// TestAVariableTheLineDidNotSetReadsNoFileItWrote — iss-2610120916220240. A
// word whose unknown parts are only variables the line did not set reads as
// it did before the file carry: a recorded pid, not the file a search was
// logged to, so the kill entries' own successor (`echo "$pid" | xargs
// kill`) allows after a search is merely logged. A file the line wrote under
// a name it does not fix can still be the one such a word names, and a
// variable the line set from the file is still followed.
func TestAVariableTheLineDidNotSetReadsNoFileItWrote(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`pgrep -l make > search.log; echo $pid | xargs kill`, VerdictAllow, ""},
		{`pgrep -l make > search.log; echo "$pid" | xargs kill`, VerdictAllow, ""},
		{`pgrep -l make > search.log; printf '%s\n' "${pid}" | xargs kill -TERM`, VerdictAllow, ""},
		{`pgrep make > search.log && xargs kill <<< "$pid"`, VerdictAllow, ""},

		// The search to a file to the kill still blocks.
		{`pgrep make > p; kill $(cat p)`, VerdictBlock, "kill-by-search"},
		{`pgrep make > "$f"; kill $(cat "$f")`, VerdictBlock, "kill-by-search"},
		{`pgrep make > $TMPDIR/p; xargs kill < $TMPDIR/p`, VerdictBlock, "kill-by-search"},
		{`pgrep make > p; f=p; kill $(cat $f)`, VerdictBlock, "kill-by-search"},
		{`pgrep make > p; cat $(echo p) | xargs kill`, VerdictBlock, "kill-by-search"},
	})
}
