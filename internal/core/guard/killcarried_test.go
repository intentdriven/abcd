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
