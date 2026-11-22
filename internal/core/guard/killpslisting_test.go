package guard

import "testing"

// TestAPsListingReachingKillIsASearchUnlessPinnedToAPid: a ps listing that
// reaches kill, through any filter or none, is a search as pgrep is, since
// the pids it hands on are the ones a selection by name, user, terminal or
// the whole table printed. A ps that reports only the pids the caller names
// (`-p`, `--pid`, `-q`) is not, and a grep or a matching awk after it still
// is. `--ppid` lists every child of the pid it names, and BSD `T` every
// process on the terminal, so neither pins the listing.
func TestAPsListingReachingKillIsASearchUnlessPinnedToAPid(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`ps aux | sed -n '/sleep 40000/s/^[^ ]* *\([0-9]*\).*/\1/p' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`kill $(ps aux | sed -n '/make/s/^[^ ]* *\([0-9]*\).*/\1/p')`, VerdictBlock, "kill-by-search"},
		{`ps -eo pid,comm | sed -n 's/^ *\([0-9]*\) make$/\1/p' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps aux | perl -lane 'print $F[1] if /make/' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps -C make -o pid= | xargs kill`, VerdictBlock, "kill-by-search"},
		{`kill $(ps -eo pid,comm | sed -n '/make/s/ .*//p')`, VerdictBlock, "kill-by-search"},
		{`ps -u bob -o pid= | xargs kill -9`, VerdictBlock, "kill-by-search"},
		{`ps -e -o pid= | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps -o pid= | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps -o pid= -p 4242 -C make | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps ax -o pid,comm | cut -c1-6 | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps -eo pid,comm > p; kill $(sed -n '/make/s/ .*//p' p)`, VerdictBlock, "kill-by-search"},
		{`ps -o pid= -p 4242 | grep -v grep | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps aux | grep make | awk '{print $2}' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps aux | awk '/make/ {print $2}' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps -o pid= --ppid "$pid" | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps --ppid 1 -o pid= | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps T 4242 | sed -n 's/ sleep$//p' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`ps -o pid= T -p 4242 | xargs kill`, VerdictBlock, "kill-by-search"},

		{`ps -o pid= -p 4242 | xargs kill`, VerdictAllow, ""},
		{`ps -o pid= -p4242 | sed 's/ //g' | xargs kill`, VerdictAllow, ""},
		{`ps -o pid= --pid 4242,4243 | xargs kill`, VerdictAllow, ""},
		{`ps -o pid= -q 4242 | xargs kill`, VerdictAllow, ""},
		{`ps -o pid= -p 4242 | awk '{print $1}' | xargs kill`, VerdictAllow, ""},
		{`kill -- -$(ps -o pgid= -p $$)`, VerdictAllow, ""},
		{`ps aux | sed -n '/make/p'`, VerdictAllow, ""},
		{`ps -C make -o pid=`, VerdictAllow, ""},
		{`ps aux > procs.txt; kill $(cat pidfile)`, VerdictAllow, ""},
	})
}
