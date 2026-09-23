//go:build unix

package rules

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The LOAD domain carries the product thinker's ruling on iss-2609210828122412
// (DECISIONS.md, 2026-09-23): orphaned background burners from a load
// experiment in a managed repository starved a development machine into a
// kernel panic. Three of the ruling's four conditions hold as rules now, and
// the bundled domain is how they reach an agent in every repository abcd
// manages, not only this one.

// The idiom the domain's first two rules prescribe, as the literal fragments
// the rule text carries. TestLoadDomainIdiomOwnsTheWholeGroup runs exactly
// these fragments, and TestLoadDomainCarriesTheThreeRules pins that the rule
// text still names them, so the advice and its proof cannot drift apart.
const (
	// loadWrapper is the launch itself: the whole load inside ONE subshell
	// job. A loop of separate `&` jobs under `set -m` gives each job its own
	// group, so the group kill reaches only the last of them (the incident's
	// exact failure); the wrapper is what makes the group the whole load.
	loadWrapper    = "set -m; ( burner & burner & wait ) & pg=$!"
	loadLeaderPGID = `ps -o pgid= -p "$pg"`
	loadLeaderComm = `ps -o comm= -p "$pg"`
	loadGroupKill  = `kill -- -"$pg"`
	loadGroupProbe = `pgrep -g "$pg"`
	loadNameProbe  = "pgrep -x"
)

func TestLoadDomainIsBundledAndActive(t *testing.T) {
	d, ok := Defaults().Domains["LOAD"]
	if !ok {
		t.Fatal("the LOAD domain is not bundled: the load-experiment rules reach no managed repository")
	}
	if d.State == StateDormant {
		t.Fatal("the LOAD domain ships dormant: its rules would inject only on *LOAD")
	}
}

func TestLoadDomainCarriesTheThreeRules(t *testing.T) {
	d := Defaults().Domains["LOAD"]
	all := strings.Join(d.Rules, "\n")
	for _, want := range []struct{ what, text string }{
		{"rule 1: one owned process group", "process group"},
		{"rule 1: never nohup with a pid file as the only handle", "nohup"},
		{"rule 1: the one-job wrapper", loadWrapper},
		{"rule 1: the handle lives in a file the session owns", "a file the session owns"},
		{"rule 1: the handle is re-checked before use", "re-checked before use"},
		{"rule 1: the leader still leads its group", loadLeaderPGID},
		{"rule 1: the leader is the expected command", loadLeaderComm},
		{"rule 1: the group kill", loadGroupKill},
		{"rule 1: never a pattern kill", "kill by pattern"},
		{"rule 1: never pkill -f", "`pkill -f`"},
		{"rule 1: never killall", "`killall`"},
		{"rule 2: proof by what is running", "actually running"},
		{"rule 2: the group probe", loadGroupProbe},
		{"rule 2: the name probe", loadNameProbe},
		{"rule 3: explicit consent", "explicit consent"},
		{"rule 3: a cap below the core count", "below the core count"},
	} {
		if !strings.Contains(all, want.text) {
			t.Errorf("%s: the LOAD rules do not carry %q", want.what, want.text)
		}
	}
}

func TestLoadDomainRecallsLoadExperimentPrompts(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"run the selftest under load",
		"spawn eight burners to peg the cores",
		"nohup yes > /dev/null 2>&1 &",
		"kill the orphaned processes from the experiment",
		"stress the machine while the suite runs",
		"start a busy loop on every core",
		"check the load average first",
		"set up a load test for the queue",
		"peg the CPU while the tests run",
		"pegging the cpu for ten minutes",
		"spin up yes on every core",
		"spin up four workers on all cores",
		"saturate all cores and rerun the suite",
		"saturate the CPU before the gate starts",
		"max out the CPU during the run",
		"stress-test the machine with a CPU hog",
		"run a CPU stress test before the release",
	} {
		if !has(rs.Match(prompt), "LOAD") {
			t.Errorf("load-experiment prompt %q did not recall LOAD, got %v", prompt, names(rs.Match(prompt)))
		}
	}
}

// TestLoadDomainStaysQuietOnOrdinaryLoadWords pins the recall set's
// narrowness: "load" alone is everyday vocabulary here (the rule loader,
// loading a config), so the domain recalls on the experiment's own words and
// never on the bare verb.
func TestLoadDomainStaysQuietOnOrdinaryLoadWords(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"the rule loader injects matched domains",
		"load the config file before the hook runs",
		"download the release archive",
		"the page loads slowly",
		// "orphan" is ordinary vocabulary here: orphaned stages, drafts,
		// fences and transcripts, none of them a process.
		"the orphan sweep rolls back the stage",
		"a failed stamp leaves an orphan draft",
		"an orphaned BEGIN fence breaks the marker block",
		// "stress-test" alone is ordinary review vocabulary: a design or
		// an argument is stress-tested, not a machine.
		"stress-test the plan before we build it",
		"stress test this argument",
		"the stress on the second syllable",
	} {
		if has(rs.Match(prompt), "LOAD") {
			t.Errorf("ordinary prompt %q recalled LOAD", prompt)
		}
	}
}

// TestLoadDomainIdiomOwnsTheWholeGroup is the experiment behind the rule. A
// real child spawns two real grandchildren (a uniquely named sleep, so the
// test burns no CPU and its name probe can match nothing else). Recording the
// child's pid and killing that pid alone — the incident's handle — leaves
// both grandchildren running. The prescribed idiom runs the rule's own launch
// text verbatim, keeps the group's handle in a file, and stops the load from
// a SECOND shell, as an agent does, since its shell state does not survive
// between tool calls: that shell re-checks the handle before the group kill,
// and afterwards "nothing" is established by the two process queries the rule
// names, never by a list.
func TestLoadDomainIdiomOwnsTheWholeGroup(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not on PATH")
	}
	if _, err := exec.LookPath("pgrep"); err != nil {
		t.Skip("pgrep not on PATH")
	}
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep not on PATH")
	}

	dir := t.TempDir()
	name := burnerName(t)
	burnerPath := filepath.Join(dir, name)
	// A symlink, not a copy: a copied platform binary is refused by the
	// macOS kernel, while an exec through a symlink takes the link's name as
	// the process name that pgrep -x matches.
	if err := os.Symlink(sleep, burnerPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { killBurners(t, name) })

	// burner is the name the rule's launch text calls: each one execs the
	// uniquely named sleep, detached from the script's output so the script
	// returns while the load runs on.
	burner := `burner() { exec "$B" 300 </dev/null >/dev/null 2>&1; }`
	// detach points a launching script's own descriptors away from the
	// test's pipes before the launch line runs, so the subshell the load
	// lives in holds none of them and the call returns while the load runs
	// on. A launching script reports by exit status and by the handle file.
	detach := `exec </dev/null >/dev/null 2>&1`
	// waitForBoth is the script's own bounded wait until both grandchildren
	// exist, so a kill never races the launch.
	waitForBoth := `i=0; while [ "$(pgrep -x "$N" | wc -l)" -lt 2 ]; do i=$((i+1)); [ "$i" -gt 200 ] && exit 3; sleep 0.05; done`
	// isPID refuses (exit 4) an operand that is empty, not a number, 0 or 1,
	// so no kill below can ever be aimed at 0, -1 or an empty operand.
	isPID := func(v string) string {
		return `case "$` + v + `" in ''|*[!0-9]*|0|1) exit 4;; esac`
	}
	// ownChild adds that the pid is this script's own direct child.
	ownChild := func(v string) string {
		return isPID(v) + `; [ "$(ps -o ppid= -p "$` + v + `" | tr -d ' ')" = "$$" ] || exit 4`
	}

	t.Run("the child's pid is not a handle", func(t *testing.T) {
		// The incident's handle: the pid of the process that was started,
		// killed by pid alone. It is the script's own child ($!), checked
		// before the kill, never a pid read back from a file.
		script := strings.Join([]string{
			detach,
			burner,
			`( burner & burner & wait ) & p=$!`,
			ownChild("p"),
			waitForBoth,
			`kill "$p"`,
		}, "\n")
		runLoadScript(t, bash, script, burnerPath, name, dir)
		// The kill landed on the child's pid; give the signal time to land
		// anywhere it was going to, then show the grandchildren outlived it.
		time.Sleep(500 * time.Millisecond)
		if n := len(pgrepPIDs(t, "-x", name)); n != 2 {
			t.Fatalf("killing the child's pid should orphan both grandchildren (the incident), found %d running", n)
		}
		killBurners(t, name)
		waitGone(t, "-x", name)
	})

	t.Run("the owned group dies together", func(t *testing.T) {
		handle := filepath.Join(dir, "load.pg")
		// The first tool call: the rule's launch text verbatim, and the
		// handle written to a file the session owns before the call ends.
		start := strings.Join([]string{
			detach,
			burner,
			loadWrapper,
			ownChild("pg"),
			waitForBoth,
			`printf '%s\n' "$pg" > "$D/load.pg"`,
		}, "\n")
		runLoadScript(t, bash, start, burnerPath, name, dir)

		// The second tool call: a fresh shell holding nothing but the file.
		// It re-checks the handle before the kill: the pid still leads its
		// own group, the leader is still the shell that launched the load,
		// and the group holds exactly this test's two burners. Any failed
		// check stops the script before the kill, so a stale or reused pid
		// is never signalled.
		stop := strings.Join([]string{
			`pg=$(cat "$D/load.pg")`,
			isPID("pg"),
			`[ "$(` + loadLeaderPGID + ` | tr -d ' ')" = "$pg" ] || exit 5`,
			`[ "$(basename "$(` + loadLeaderComm + `)")" = bash ] || exit 6`,
			`[ "$(pgrep -g "$pg" -x "$N" | wc -l)" -eq 2 ] || exit 7`,
			loadGroupKill,
			`echo "$pg"`,
		}, "\n")
		out := runLoadScript(t, bash, stop, burnerPath, name, dir)
		pg, err := strconv.Atoi(strings.TrimSpace(out))
		if err != nil || pg <= 1 || pg == syscall.Getpgrp() {
			t.Fatalf("the script did not report a process group of its own: %q", out)
		}
		if b, err := os.ReadFile(handle); err != nil || strings.TrimSpace(string(b)) != strconv.Itoa(pg) {
			t.Fatalf("the handle file does not hold the group the stop call killed: %q, %v", b, err)
		}
		// Proof by what is running: both queries the rule names come back
		// empty. Delivery of the signal is asynchronous, so the proof polls
		// under a deadline rather than trusting the kill's exit status.
		waitGone(t, "-g", strconv.Itoa(pg))
		waitGone(t, "-x", name)
	})
}

func burnerName(t *testing.T) string {
	t.Helper()
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	// Under the 15-character process-name limit Linux keeps, so pgrep -x
	// matches the whole name on every host.
	return "abcdburn" + hex.EncodeToString(b)[:6]
}

var burnerNameRe = regexp.MustCompile(`^abcdburn[0-9a-f]{6}$`)

// killBurners is the test's safety net: it signals only processes whose name
// is exactly this test's random burner name, and refuses any other operand.
func killBurners(t *testing.T, name string) {
	t.Helper()
	if !burnerNameRe.MatchString(name) {
		t.Fatalf("refusing to pkill %q: not this test's burner name", name)
	}
	_ = exec.Command("pkill", "-KILL", "-x", name).Run()
}

func runLoadScript(t *testing.T, bash, script, burner, name, dir string) string {
	t.Helper()
	cmd := exec.Command(bash, "-c", script)
	cmd.Env = append(os.Environ(), "B="+burner, "N="+name, "D="+dir)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	done := make(chan struct{})
	var out []byte
	var runErr error
	go func() {
		out, runErr = cmd.Output()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("the load script did not finish")
	}
	if runErr != nil {
		t.Fatalf("the load script failed: %v (stderr %q)", runErr, stderr.String())
	}
	return string(out)
}

// pgrepPIDs runs pgrep with one selector and returns the matching pids. pgrep
// exits 1 when nothing matches, which is the answer "none", not a failure.
func pgrepPIDs(t *testing.T, flag, value string) []string {
	t.Helper()
	out, err := exec.Command("pgrep", flag, value).Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return nil
	}
	if err != nil {
		t.Fatalf("pgrep %s %s: %v", flag, value, err)
	}
	return strings.Fields(string(out))
}

func waitGone(t *testing.T, flag, value string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		pids := pgrepPIDs(t, flag, value)
		if len(pids) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("pgrep %s %s still finds %v after the group kill", flag, value, pids)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
