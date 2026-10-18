package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// hangOrigin points root's origin at a command that would hang any network
// call for 30 s and first leaves a file at the path it returns, so a test
// knows from the file, not the clock, whether anything reached for origin.
func hangOrigin(t *testing.T, root string) (touched string) {
	t.Helper()
	touched = filepath.Join(t.TempDir(), "origin-reached")
	if strings.ContainsAny(touched, " %") {
		t.Fatalf("the marker path %q needs escaping in an ext:: URL", touched)
	}
	gitCmd(t, root, "config", "protocol.ext.allow", "always")
	// git-remote-ext splits the command at spaces, and "% " is its literal
	// space, so sh -c gets the whole of "touch <file>; sleep 30" as one
	// argument. A backslash escapes nothing there: "sleep\ 30" would hand
	// sleep no argument, and it would exit at once instead of hanging.
	gitCmd(t, root, "remote", "set-url", "origin", "ext::sh -c touch% "+touched+";% sleep% 30")
	return touched
}

// requireOriginUntouched fails the test if anything reached for the origin
// hangOrigin set up.
func requireOriginUntouched(t *testing.T, touched string) {
	t.Helper()
	if _, err := os.Stat(touched); err == nil {
		t.Fatal("the hook reached for origin; want it never to touch the network")
	} else if !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
}

// TestThePromptHookReturnsInsideItsSecond is C12's bound at the hook: with an
// announcement to show and an origin that would hang any network call, the
// hook returns inside its one-second budget and never reaches for origin.
// The bound is the shipped one, so the clock decides it, and one reading on a
// loaded runner can run over (iss-2610180735158998). Each of three tries runs
// in a checkout of its own, so each has the announcement still to show, and
// each try's time is logged; two tries over the second fail the test, so one
// spike is absorbed and a hook that is usually slow is not. Whether origin
// was reached is decided by the file hangOrigin's command leaves, never by
// the clock.
func TestThePromptHookReturnsInsideItsSecond(t *testing.T) {
	const tries = 3
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	w := newReserveWorld(t)
	octo := w.clone(t, "octo", octoMail)
	if code, _, stderr := reserveRun(t, octo, "announce", "merging", "1012"); code != 0 {
		t.Fatalf("announce: exit %d, %s", code, stderr)
	}
	over := 0
	for i := 1; i <= tries; i++ {
		pat := w.clone(t, fmt.Sprintf("pat%d", i), "5+pat@users.noreply.github.com")
		if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
			t.Fatalf("status: exit %d, %s", code, stderr)
		}
		touched := hangOrigin(t, pat)
		// runHook runs the hook in this process, so the time around it is
		// the hook's own, from its start to its last write: no process start
		// or exit is in it.
		start := time.Now()
		runHook(t, hookInputJSON(t, fmt.Sprintf("s-bound-%d", i), pat, "hello there"), "hook", "prompt-router")
		took := time.Since(start)
		t.Logf("try %d: the hook took %s", i, took)
		requireOriginUntouched(t, touched)
		if took > time.Second {
			over++
		}
	}
	if over >= 2 {
		t.Fatalf("%d of %d tries took over the hook's second; want at most one", over, tries)
	}
}

// TestThePromptHookShowsEachAnnouncementOnce is C6 at the hook: after a fetch
// brought an announcement in, the next prompt's injected text carries it,
// rendered from its fields, with the grants-nothing line, with an origin that
// would hang any network call and is never reached; the prompt after shows it
// no more; and a rules.json that will not load does not switch it off. The
// budgets are the test's own, ten seconds for the hook and for the status
// read, because what is shown is asked here and the shipped bound is asked by
// TestThePromptHookReturnsInsideItsSecond: under the shipped second a loaded
// runner can spend the read's share, and the block then says the status is
// unavailable (iss-2610180735158998).
func TestThePromptHookShowsEachAnnouncementOnce(t *testing.T) {
	const spare = 10 * time.Second
	oldBudget := hookBudget
	hookBudget = spare
	defer func() { hookBudget = oldBudget }()
	defer claim.SetPromptReadBudgetForTest(spare)()
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	w := newReserveWorld(t)
	octo := w.clone(t, "octo", octoMail)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	if code, _, stderr := reserveRun(t, octo, "announce", "merging", "1012"); code != 0 {
		t.Fatalf("announce: exit %d, %s", code, stderr)
	}
	if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	touched := hangOrigin(t, pat)

	out, _ := runHook(t, hookInputJSON(t, "s-status", pat, "hello there"), "hook", "prompt-router")
	for _, want := range []string{claim.NotesHeading + "\n", "- @octo (claimed): merging #1012, for 30 min\n", claim.GrantsNothing + "\n"} {
		if !strings.Contains(out, want) {
			t.Fatalf("the hook's text:\n%s\nwant it to carry %q", out, want)
		}
	}
	again, _ := runHook(t, hookInputJSON(t, "s-status", pat, "hello again"), "hook", "prompt-router")
	if strings.Contains(again, claim.NotesHeading) {
		t.Fatalf("the second prompt shows the status again:\n%s", again)
	}
	requireOriginUntouched(t, touched)

	// A second announcement, then a rules.json that will not load: the
	// status block is still emitted, in both forms.
	gitCmd(t, pat, "remote", "set-url", "origin", w.origin)
	if code, _, stderr := reserveRun(t, octo, "announce", "merged", "1012"); code != 0 {
		t.Fatalf("announce: exit %d, %s", code, stderr)
	}
	if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	writeRepoRules(t, pat, `{"schema_version": 1, "domains": `)
	env := routerJSON(t, "s-status", pat, "hello")
	if env.Error == "" || !strings.Contains(env.Text, "- @octo (claimed): merged #1012, for 30 min\n") {
		t.Fatalf("with rules.json unreadable, the envelope = %+v; want the error and the status block", env)
	}
	if code, _, stderr := reserveRun(t, octo, "announce", "asking", "1012"); code != 0 {
		t.Fatalf("announce: exit %d, %s", code, stderr)
	}
	if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	plain, _ := runHook(t, hookInputJSON(t, "s-status", pat, "hello"), "hook", "prompt-router")
	if !strings.Contains(plain, "- @octo (claimed): asking #1012, for 30 min\n") {
		t.Fatalf("with rules.json unreadable, the plain hook:\n%s\nwant the status block", plain)
	}
}

// slowPayload is a hook payload that arrives after a delay, as from a host
// that writes it late: the hook's first step, reading it, takes that long.
type slowPayload struct {
	r     io.Reader
	delay time.Duration
	once  bool
}

func (p *slowPayload) Read(b []byte) (int, error) {
	if !p.once {
		p.once = true
		time.Sleep(p.delay)
	}
	return p.r.Read(b)
}

// runSlowHook runs the prompt hook in this process with a payload that takes
// delay to arrive, under budget.
func runSlowHook(t *testing.T, payload string, delay, budget time.Duration) string {
	t.Helper()
	old := hookBudget
	hookBudget = budget
	defer func() { hookBudget = old }()
	cmd := NewRootCommand()
	var so, se bytes.Buffer
	cmd.SetOut(&so)
	cmd.SetErr(&se)
	cmd.SetIn(&slowPayload{r: strings.NewReader(payload), delay: delay})
	cmd.SetArgs([]string{"hook", "prompt-router"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook exited non-zero: %v\n%s", err, se.String())
	}
	return so.String()
}

// TestTheHooksDeadlineStartsWithTheHook is C12's one deadline: it is set when
// the hook starts, before its first step, not when the status read does, so
// what the steps before the read take comes out of the read's share. With a
// payload that takes 500 ms to arrive against a 600 ms budget, nothing is
// left for the read: the block says the status is unavailable, and nothing
// is kept shown. Set any later, the read would have 450 ms, ample for one
// announcement. The same payload with time to spare after it leaves the read
// its time, and the announcement is shown. That half's budgets are the
// test's own: the hook's is the payload's delay plus readSpare, and the
// read's cap is readSpare in place of its shipped 600 ms. Under the shipped
// one second the read had at most 350 ms after the payload, less what the
// steps before it took, and a loaded merge-queue runner spent that
// (iss-2610180712063858). The first half keeps the shipped read cap and its
// tight budget: the read's deadline is the hook's less 150 ms, which the
// payload has already passed, so load only makes its "unavailable" surer.
func TestTheHooksDeadlineStartsWithTheHook(t *testing.T) {
	const delay, tight, readSpare = 500 * time.Millisecond, 600 * time.Millisecond, 10 * time.Second
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	w := newReserveWorld(t)
	octo := w.clone(t, "octo", octoMail)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	if code, _, stderr := reserveRun(t, octo, "announce", "merging", "1012"); code != 0 {
		t.Fatalf("announce: exit %d, %s", code, stderr)
	}
	if code, _, stderr := reserveRun(t, pat, "status"); code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	out := runSlowHook(t, hookInputJSON(t, "s-spent", pat, "hello there"), delay, tight)
	if !strings.Contains(out, "status unavailable: no time was left to read it at this prompt; `abcd status` reads it without this prompt's deadline\n") || strings.Contains(out, "merging #1012") {
		t.Fatalf("with the hook's first step taking most of its budget, the hook's text:\n%s\nwant the status unavailable", out)
	}
	defer claim.SetPromptReadBudgetForTest(readSpare)()
	again := runSlowHook(t, hookInputJSON(t, "s-spent", pat, "hello again"), delay, delay+readSpare)
	if !strings.Contains(again, "- @octo (claimed): merging #1012, for 30 min\n") {
		t.Fatalf("the next prompt, with time left after the payload:\n%s\nwant the announcement, never shown", again)
	}
}
