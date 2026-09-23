//go:build unix

package rules

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	loadJobControl = "set -m"
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
		{"rule 1: the job-control fragment", loadJobControl},
		{"rule 1: the group kill", loadGroupKill},
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
	} {
		if has(rs.Match(prompt), "LOAD") {
			t.Errorf("ordinary prompt %q recalled LOAD", prompt)
		}
	}
}

// TestLoadDomainIdiomOwnsTheWholeGroup is the experiment behind the rule. A
// real child spawns two real grandchildren (a uniquely named sleep, so the
// test burns no CPU and its name probe can match nothing else). Recording the
// child's pid in a file and killing that pid — the incident's handle — leaves
// both grandchildren running; the prescribed idiom, one job-controlled
// background job killed as a process group, leaves nothing, and "nothing" is
// established by the two process queries the rule names, never by a list.
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
	burner := filepath.Join(dir, name)
	// A symlink, not a copy: a copied platform binary is refused by the
	// macOS kernel, while an exec through a symlink takes the link's name as
	// the process name that pgrep -x matches.
	if err := os.Symlink(sleep, burner); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = exec.Command("pkill", "-KILL", "-x", name).Run() })

	// waitForBoth is the script's own bounded wait until both grandchildren
	// exist, so a kill never races the launch.
	waitForBoth := `i=0; while [ "$(pgrep -x "$N" | wc -l)" -lt 2 ]; do i=$((i+1)); [ "$i" -gt 200 ] && exit 3; sleep 0.05; done`
	load := `( "$B" 300 & "$B" 300 & wait ) </dev/null >/dev/null 2>&1 &`

	t.Run("a pid file is not a handle", func(t *testing.T) {
		script := strings.Join([]string{
			load,
			`echo $! > "$D/pids"`,
			waitForBoth,
			`kill $(cat "$D/pids")`,
		}, "\n")
		runLoadScript(t, bash, script, burner, name, dir)
		// The kill landed on the recorded pid; give the signal time to land
		// anywhere it was going to, then show the grandchildren outlived it.
		time.Sleep(500 * time.Millisecond)
		if n := len(pgrepPIDs(t, "-x", name)); n != 2 {
			t.Fatalf("killing the recorded pid should orphan both grandchildren (the incident), found %d running", n)
		}
		_ = exec.Command("pkill", "-KILL", "-x", name).Run()
		waitGone(t, "-x", name)
	})

	t.Run("the owned group dies together", func(t *testing.T) {
		script := strings.Join([]string{
			loadJobControl,
			load,
			`pg=$!`,
			waitForBoth,
			loadGroupKill,
			`echo "$pg"`,
		}, "\n")
		out := runLoadScript(t, bash, script, burner, name, dir)
		pg, err := strconv.Atoi(strings.TrimSpace(out))
		if err != nil {
			t.Fatalf("the script did not report its process group: %q", out)
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
