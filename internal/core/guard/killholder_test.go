package guard

import "testing"

// TestStoppingAHolderIsBlockedLikeStoppingByName — iss-2609262259360005,
// ruling AS (product thinker, 2026-10-11): stopping whatever holds a port or
// a file blocks as stopping by name does. `fuser -k`, and a `kill` handed
// what `lsof` or `fuser` prints, signal every process that holds it, a peer
// session's server included, not the one the caller started.
func TestStoppingAHolderIsBlockedLikeStoppingByName(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`fuser -k 8080/tcp`, VerdictBlock, "fuser-kill"},
		{`fuser -km /home`, VerdictBlock, "fuser-kill"},
		{`fuser -mk /home`, VerdictBlock, "fuser-kill"},
		{`fuser -k -n tcp 8080`, VerdictBlock, "fuser-kill"},
		{`fuser -n tcp -k 8080`, VerdictBlock, "fuser-kill"},
		{`fuser --kill 8080/tcp`, VerdictBlock, "fuser-kill"},
		{`fuser -k -TERM 8080/tcp`, VerdictBlock, "fuser-kill"},
		{`fuser -ki 8080/tcp`, VerdictBlock, "fuser-kill"},
		{`sudo fuser -k 8080/tcp`, VerdictBlock, "fuser-kill"},
		{`kill $(lsof -t -i :8080)`, VerdictBlock, "kill-by-holder"},
		{`kill -9 $(lsof -ti:8080)`, VerdictBlock, "kill-by-holder"},
		{`kill $(lsof -t /var/run/app.sock)`, VerdictBlock, "kill-by-holder"},
		{`lsof -ti tcp:8080 | xargs kill`, VerdictBlock, "kill-by-holder"},
		{`lsof -t -i :8080 | xargs kill -9`, VerdictBlock, "kill-by-holder"},
		{`lsof -nP -iTCP:8080 -sTCP:LISTEN -t | xargs kill`, VerdictBlock, "kill-by-holder"},
		{`lsof -i :8080 | awk 'NR>1 {print $2}' | xargs kill`, VerdictBlock, "kill-by-holder"},
		{`kill $(fuser 8080/tcp 2>/dev/null)`, VerdictBlock, "kill-by-holder"},
		{`fuser 8080/tcp 2>/dev/null | xargs kill`, VerdictBlock, "kill-by-holder"},
		{`kill $(sh -c 'lsof -t -i :8080')`, VerdictBlock, "kill-by-holder"},
		{`lsof -t -i :8080 | xargs sh -c 'kill "$@"' _`, VerdictBlock, "kill-by-holder"},
		// Looking stays allowed: who holds it, and a kill by a pid you name.
		{`lsof -i :8080`, VerdictAllow, ""},
		{`lsof -t -i :8080`, VerdictAllow, ""},
		{`lsof -ti:8080 | wc -l`, VerdictAllow, ""},
		{`fuser 8080/tcp`, VerdictAllow, ""},
		{`fuser -v 8080/tcp`, VerdictAllow, ""},
		{`fuser -m /home`, VerdictAllow, ""},
		{`fuser -l`, VerdictAllow, ""},
		{`kill 4242`, VerdictAllow, ""},
		{`kill -- -4242`, VerdictAllow, ""},
		{`lsof -i :8080 && kill 4242`, VerdictAllow, ""},
		{`abcd capture "fuser -k 8080/tcp stopped a peer's server"`, VerdictAllow, ""},
	})
}
