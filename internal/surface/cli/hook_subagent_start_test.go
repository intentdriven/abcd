package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/gittest"
)

// subagentCheckWord stands in a fixture's conventions file and must reach
// nothing the hook sends.
const subagentCheckWord = "quillfennel"

// subagentRepo is a checkout abcd manages, holding a conventions file, with the
// account's home in a folder of the test's own. It returns the repository and
// its root commit.
func subagentRepo(t *testing.T) (repo *gittest.Repo, sha string) {
	t.Helper()
	repo = gittest.OpenRepo(t, realPath(t, gittest.NewRepo(t).Root()))
	writeUnder(t, repo.Root(), "AGENTS.md", "# AGENTS.md\n\nConventions-file check word: "+subagentCheckWord+".\n")
	writeUnder(t, repo.Root(), ".abcd/work/CONTEXT.md", "x\n")
	repo.Git("add", "-A")
	repo.Git("commit", "-q", "-m", "seed")
	sha = repo.Git("rev-list", "--max-parents=0", "HEAD")
	t.Setenv("HOME", realPath(t, t.TempDir()))
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("ABCD_RULES_STATE_DIR", "")
	return repo, sha
}

func writeUnder(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// startHookPayload is the payload the host writes at a sub-agent's start.
func startHookPayload(t *testing.T, session, cwd, agent, agentType string) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"hook_event_name": "SubagentStart", "session_id": session, "cwd": cwd,
		"agent_id": agent, "agent_type": agentType,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// runSubagentStart runs the hook's command on its own, with no tree around it.
func runSubagentStart(t *testing.T, stdin io.Reader) (stdout, stderr string, code int) {
	t.Helper()
	cmd := newSubagentStartCommand()
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	var so, se bytes.Buffer
	cmd.SetOut(&so)
	cmd.SetErr(&se)
	cmd.SetIn(stdin)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		code = 1
		var coded interface{ ExitCode() int }
		if errors.As(err, &coded) {
			code = coded.ExitCode()
		}
	}
	return so.String(), se.String(), code
}

// startMessage runs the hook on a payload and returns what the sub-agent is
// sent, failing on anything but the host's one object at exit 0.
func startMessage(t *testing.T, payload string) string {
	t.Helper()
	so, se, code := runSubagentStart(t, strings.NewReader(payload))
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	var doc map[string]map[string]string
	if err := json.Unmarshal([]byte(so), &doc); err != nil {
		t.Fatalf("stdout is not one JSON object: %v\n%s", err, so)
	}
	out := doc["hookSpecificOutput"]
	if len(doc) != 1 || len(out) != 2 || out["hookEventName"] != "SubagentStart" || out["additionalContext"] == "" {
		t.Fatalf("stdout = %s; want hookSpecificOutput carrying the event and additionalContext alone", so)
	}
	return out["additionalContext"]
}

// factFiles lists the sub-agents' facts under the test's home.
func factFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(abcdhome.Path(os.Getenv("HOME"), "rules-state"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// fact reads one fact, failing on a fact that cannot be read.
func fact(t *testing.T, session, agent string, f rules.SubagentFact) (rules.SubagentRecord, bool) {
	t.Helper()
	rec, ok, err := rules.ReadSubagentFact(session, agent, f)
	if err != nil {
		t.Fatalf("%s fact: %v", f, err)
	}
	return rec, ok
}

func TestSubagentStartNamesTheConventionsFileAndTheCommands(t *testing.T) {
	repo, _ := subagentRepo(t)
	exe := "/opt/a b/abcd"
	swapExecutable(t, func() (string, error) { return exe, nil })
	msg := startMessage(t, startHookPayload(t, "s1", repo.Root(), "a1", "general-purpose"))
	for _, want := range []string{
		"read AGENTS.md at the root of the worktree you work in first",
		"This hook looked in " + repo.Root(),
		"if you were given another worktree, read the copy at its root",
		`'/opt/a b/abcd' rules --prompt "<your task in one sentence>"`,
		"'/opt/a b/abcd' rules <DOMAIN>",
		"COMMITTING",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("the message does not say %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "general-purpose") {
		t.Errorf("the message carries the payload's agent type:\n%s", msg)
	}
	told, ok := fact(t, "s1", "a1", rules.SubagentTold)
	if !ok || told.File != "AGENTS.md" || told.Root != repo.Root() || told.By != rules.SubagentToldByStart {
		t.Errorf("told = %+v, %v", told, ok)
	}
	// An executable, or a root, whose path holds a byte a command cannot carry
	// is left out, never escaped.
	exe = "/opt/$(x)/abcd"
	if msg := startMessage(t, startHookPayload(t, "s1", repo.Root(), "a2", "t")); strings.Contains(msg, "$(x)") || !strings.Contains(msg, "  abcd rules <DOMAIN>") {
		t.Errorf("an unspeakable executable path is not left out:\n%s", msg)
	}
	odd := gittest.NewRepo(t).Root()
	named := filepath.Join(filepath.Dir(odd), "odd;name")
	if err := os.Rename(odd, named); err != nil {
		t.Fatal(err)
	}
	writeUnder(t, named, "AGENTS.md", "x\n")
	writeUnder(t, named, ".abcd/x", "x\n")
	if msg := startMessage(t, startHookPayload(t, "s1", named, "a3", "t")); strings.Contains(msg, "odd;name") || strings.Contains(msg, "looked in") || !strings.Contains(msg, "read AGENTS.md") {
		t.Errorf("an unspeakable root is not left out:\n%s", msg)
	}
	// A payload whose ids can key nothing is told all the same, and no fact is
	// kept for it.
	before := len(factFiles(t))
	for _, p := range []string{
		startHookPayload(t, "", repo.Root(), "a4", "t"),
		startHookPayload(t, "s1", repo.Root(), "", "t"),
		startHookPayload(t, "s1", repo.Root(), "a/4", "t"),
	} {
		if msg := startMessage(t, p); !strings.Contains(msg, "read AGENTS.md") {
			t.Errorf("a payload with no usable id is told nothing: %s", p)
		}
	}
	if after := len(factFiles(t)); after != before {
		t.Errorf("%d fact(s) were kept under ids that can key nothing", after-before)
	}
}

// swapExecutable replaces the running binary's path for one test.
func swapExecutable(t *testing.T, f func() (string, error)) {
	t.Helper()
	old := osExecutable
	osExecutable = f
	t.Cleanup(func() { osExecutable = old })
}

func TestSubagentStartFallsBackToClaudeMdThenSaysNothingOfAFile(t *testing.T) {
	repo, _ := subagentRepo(t)
	root := repo.Root()
	move := func(from, to string) {
		t.Helper()
		if err := os.Rename(filepath.Join(root, from), filepath.Join(root, to)); err != nil {
			t.Fatal(err)
		}
	}
	move("AGENTS.md", "CLAUDE.md")
	msg := startMessage(t, startHookPayload(t, "s1", root, "a1", "t"))
	if !strings.Contains(msg, "read CLAUDE.md at the root") || strings.Contains(msg, "AGENTS.md") {
		t.Errorf("with CLAUDE.md alone:\n%s", msg)
	}
	// A conventions file that is a link is still found, and AGENTS.md leads.
	if err := os.Symlink("CLAUDE.md", filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if msg := startMessage(t, startHookPayload(t, "s1", root, "a2", "t")); !strings.Contains(msg, "read AGENTS.md at the root") {
		t.Errorf("with a linked AGENTS.md:\n%s", msg)
	}
	for _, f := range []string{"AGENTS.md", "CLAUDE.md"} {
		if err := os.Remove(filepath.Join(root, f)); err != nil {
			t.Fatal(err)
		}
	}
	// A folder of that name is no conventions file.
	if err := os.Mkdir(filepath.Join(root, "AGENTS.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	msg = startMessage(t, startHookPayload(t, "s1", root, "a3", "t"))
	for _, not := range []string{"AGENTS.md", "CLAUDE.md", "onventions", "guard"} {
		if strings.Contains(msg, not) {
			t.Errorf("with no conventions file the message says %q:\n%s", not, msg)
		}
	}
	for _, want := range []string{"rules --prompt", "rules <DOMAIN>", "Domains in force"} {
		if !strings.Contains(msg, want) {
			t.Errorf("with no conventions file the message does not say %q:\n%s", want, msg)
		}
	}
	if told, ok := fact(t, "s1", "a3", rules.SubagentTold); !ok || told.File != "" {
		t.Errorf("told = %+v, %v; want one naming no file", told, ok)
	}
}

func TestSubagentStartNamesTheWorktreeItLookedIn(t *testing.T) {
	repo, _ := subagentRepo(t)
	lane := filepath.Join(realPath(t, t.TempDir()), "lane")
	repo.Git("worktree", "add", "-q", "-b", "lane", lane)
	sub := filepath.Join(lane, "internal", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	msg := startMessage(t, startHookPayload(t, "s1", sub, "a1", "t"))
	if !strings.Contains(msg, "This hook looked in "+lane+";") {
		t.Errorf("the message does not name the linked worktree's root %s:\n%s", lane, msg)
	}
	if strings.Contains(msg, repo.Root()) {
		t.Errorf("the message names the main worktree:\n%s", msg)
	}
	// With no directory in the payload, the hook's own is asked.
	t.Chdir(sub)
	if msg := startMessage(t, `{"session_id":"s1","agent_id":"a2"}`); !strings.Contains(msg, lane) {
		t.Errorf("with no cwd in the payload:\n%s", msg)
	}
}

func TestSubagentStartNamesTheCheckoutsOwnScriptInAbcdsSourceTree(t *testing.T) {
	repo, _ := subagentRepo(t)
	swapExecutable(t, func() (string, error) { return "/opt/release/abcd", nil })
	writeUnder(t, repo.Root(), "cmd/abcd/main.go", "package main\n")
	if msg := startMessage(t, startHookPayload(t, "s1", repo.Root(), "a1", "t")); !strings.Contains(msg, "  /opt/release/abcd rules") {
		t.Errorf("a source tree with no script names something other than the executable:\n%s", msg)
	}
	writeUnder(t, repo.Root(), "scripts/abcd.sh", "#!/bin/sh\n")
	msg := startMessage(t, startHookPayload(t, "s1", repo.Root(), "a2", "t"))
	if !strings.Contains(msg, "  scripts/abcd.sh rules --prompt") || strings.Contains(msg, "/opt/release") {
		t.Errorf("abcd's own source tree is not sent to its own script:\n%s", msg)
	}
}

func TestSubagentStartNeverCarriesTheCheckWord(t *testing.T) {
	repo, _ := subagentRepo(t)
	for i, mode := range []os.FileMode{0o644, 0} {
		if err := os.Chmod(filepath.Join(repo.Root(), "AGENTS.md"), mode); err != nil {
			t.Fatal(err)
		}
		so, se, code := runSubagentStart(t, strings.NewReader(startHookPayload(t, "s1", repo.Root(), fmt.Sprintf("a%d", i), "t")))
		if code != 0 || !strings.Contains(so, "AGENTS.md") {
			t.Errorf("mode %v: exit %d stdout %q stderr %q", mode, code, so, se)
		}
		if strings.Contains(so+se, subagentCheckWord) {
			t.Errorf("mode %v: the hook carries the file's check word: %s%s", mode, so, se)
		}
	}
}

func TestSubagentStartSaysNothingToAColdReader(t *testing.T) {
	repo, _ := subagentRepo(t)
	for _, kind := range []string{"abcd:cold-reading-detection", "cold-reading-detection", "cold-reading-widening", "abcd:cold-reading-entailment", "abcd:cold-reading-comparative"} {
		so, se, code := runSubagentStart(t, strings.NewReader(startHookPayload(t, "s1", repo.Root(), "a1", kind)))
		if so != "" || se != "" || code != 0 {
			t.Errorf("%s: exit %d stdout %q stderr %q; want silence", kind, code, so, se)
		}
	}
	if files := factFiles(t); len(files) != 0 {
		t.Errorf("a cold reader left %v", files)
	}
	// Another agent of a like name is told.
	if msg := startMessage(t, startHookPayload(t, "s1", repo.Root(), "a2", "x:cold-reading-detection")); msg == "" {
		t.Error("another plugin's agent is told nothing")
	}
}

func TestSubagentStartFitsTheHostLimitWithEveryDomainNamed(t *testing.T) {
	repo, _ := subagentRepo(t)
	writeUnder(t, repo.Root(), ".abcd/rules.json", `{"schema_version":1,"domains":{"OWNDOMAIN":{"recall":["x"],"rules":["a rule no message carries <&> `+strings.Repeat("y", 3000)+`"]},"SCRATCH":{"state":"dormant"}}}`)
	so, se, code := runSubagentStart(t, strings.NewReader(startHookPayload(t, "s1", repo.Root(), "a1", "t")))
	if code != 0 || se != "" {
		t.Fatalf("exit %d stderr %q", code, se)
	}
	if len(so) > rules.InjectionBudgetBytes {
		t.Errorf("the document is %d bytes, over %d", len(so), rules.InjectionBudgetBytes)
	}
	rs, err := rulesForHook(repo.Root(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	active := rs.Active()
	var names []string
	for _, d := range active {
		names = append(names, d.Name)
		for _, r := range d.Rules {
			if strings.Contains(so, r) {
				t.Errorf("the message carries a rule's text of %s", d.Name)
			}
		}
	}
	if want := fmt.Sprintf("Domains in force (%d): %s", len(names), strings.Join(names, ", ")); !strings.Contains(so, want) {
		t.Errorf("the message does not name every domain in force (%s):\n%s", want, so)
	}
	if strings.Contains(so, "no message carries") || strings.Contains(so, "SCRATCH") || !strings.Contains(so, "OWNDOMAIN") {
		t.Errorf("the message names the wrong domains or a rule's text:\n%s", so)
	}
	if strings.Contains(so, `\u003c`) {
		t.Errorf("the document escapes HTML: %s", so)
	}
}

func TestSubagentStartCountsTheDomainsItCannotFitAndCutsNone(t *testing.T) {
	repo, _ := subagentRepo(t)
	swapExecutable(t, func() (string, error) { return "/opt/abcd", nil })
	var b strings.Builder
	for i := range 300 {
		fmt.Fprintf(&b, `%q:{"recall":["x"],"rules":["r"]},`, fmt.Sprintf("DOMAIN_%d_%s", i, strings.Repeat("Z", 40)))
	}
	writeUnder(t, repo.Root(), ".abcd/rules.json", `{"schema_version":1,"domains":{`+strings.TrimSuffix(b.String(), ",")+`}}`)
	so, se, code := runSubagentStart(t, strings.NewReader(startHookPayload(t, "s1", repo.Root(), "a1", "t")))
	if code != 0 {
		t.Fatalf("exit %d stderr %q", code, se)
	}
	if len(so) > rules.InjectionBudgetBytes || len(so) == 0 {
		t.Errorf("the document is %d bytes; want some, within %d", len(so), rules.InjectionBudgetBytes)
	}
	rs, err := rulesForHook(repo.Root(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if want := fmt.Sprintf("Domains in force: %d", len(rs.Active())); !strings.Contains(so, want) || !strings.Contains(so, "/opt/abcd rules lists them") {
		t.Errorf("the message does not give the count (%s) and the command:\n%s", want, so)
	}
	for _, not := range []string{"DOMAIN_", "COMMITTING"} {
		if strings.Contains(so, not) {
			t.Errorf("a partial list is sent (%s):\n%s", not, so)
		}
	}
	if !strings.Contains(so, "read AGENTS.md") || strings.Count(se, "\n") != 1 || !strings.Contains(se, "do not fit") {
		t.Errorf("stdout %q, stderr %q", so, se)
	}
}

func TestSubagentStartSaysTheGuardHoldsWrites(t *testing.T) {
	repo, _ := subagentRepo(t)
	msg := startMessage(t, startHookPayload(t, "s1", repo.Root(), "a1", "t"))
	if want := "the guard refuses your file writes, and the git commands that record or send your work, until a call of your own has read AGENTS.md"; !strings.Contains(msg, want) {
		t.Errorf("the message does not say the hold:\n%s", msg)
	}
}

func TestSubagentStartAddsNothingOutsideAManagedRepository(t *testing.T) {
	t.Setenv("HOME", realPath(t, t.TempDir()))
	t.Setenv("ABCD_RULES_STATE_DIR", "")
	home := os.Getenv("HOME")
	plain := realPath(t, gittest.NewRepo(t).Root())
	writeUnder(t, plain, "AGENTS.md", "x\n")
	t.Chdir(plain)
	for _, cwd := range []string{plain, filepath.Join(plain, "gone"), realPath(t, t.TempDir()), ""} {
		so, se, code := runSubagentStart(t, strings.NewReader(startHookPayload(t, "s1", cwd, "a1", "t")))
		if so != "" || se != "" || code != 0 {
			t.Errorf("%q: exit %d stdout %q stderr %q; want silence", cwd, code, so, se)
		}
	}
	if files := homeFiles(t, home); len(files) != 0 {
		t.Errorf("the hook left %v under the home", files)
	}
}

func TestSubagentStartAddsNothingWhenRulesAreSwitchedOff(t *testing.T) {
	repo, _ := subagentRepo(t)
	silent := func(which string) {
		t.Helper()
		so, se, code := runSubagentStart(t, strings.NewReader(startHookPayload(t, "s1", repo.Root(), "a1", "t")))
		if so != "" || se != "" || code != 0 {
			t.Errorf("%s: exit %d stdout %q stderr %q; want silence", which, code, so, se)
		}
		if files := factFiles(t); len(files) != 0 {
			t.Errorf("%s: the hook left %v", which, files)
		}
	}
	off := `{"schema_version":1,"disabled":true,"domains":{}}`
	writeUnder(t, repo.Root(), ".abcd/rules.json", off)
	silent("the repository's switch")
	if err := os.Remove(filepath.Join(repo.Root(), ".abcd", "rules.json")); err != nil {
		t.Fatal(err)
	}
	home := abcdhome.Path(os.Getenv("HOME"))
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "rules.json"), []byte(off), 0o600); err != nil {
		t.Fatal(err)
	}
	silent("the machine's switch")
}

// failedStart asserts the hook's failure: nothing on stdout, one line on
// stderr, exit 1. It returns the line.
func failedStart(t *testing.T, what string, stdin io.Reader) string {
	t.Helper()
	so, se, code := runSubagentStart(t, stdin)
	if code != 1 || so != "" || strings.Count(se, "\n") != 1 || !strings.HasSuffix(se, "\n") {
		t.Errorf("%s: exit %d stdout %q stderr %q; want exit 1, nothing on stdout and one line on stderr", what, code, so, se)
	}
	return se
}

// startFailures is how many start failures the lane's marker counts.
func startFailures(t *testing.T, repo, sha string) int {
	t.Helper()
	note, _, err := history.ReadSubagentLines(repo, sha)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, l := range note.Lines {
		if l.Kind == history.SubagentStartFailed {
			n++
		}
	}
	return n
}

func TestSubagentStartFailureExitsOneWithOneLineAndIsCounted(t *testing.T) {
	repo, sha := subagentRepo(t)
	if err := history.NoteSessionRepo(repo.Root(), sha, "s1"); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"{not json", ""} {
		if line := failedStart(t, "an unreadable payload", strings.NewReader(body)); !strings.Contains(line, "payload") {
			t.Errorf("the line does not name the cause: %q", line)
		}
	}
	if n := startFailures(t, repo.Root(), sha); n != 0 {
		t.Errorf("a payload that names no session was counted %d time(s)", n)
	}

	writeUnder(t, repo.Root(), ".abcd/rules.json", "{\nnot json")
	if line := failedStart(t, "a rules file that will not load", strings.NewReader(startHookPayload(t, "s1", repo.Root(), "a1", "t\nx"))); !strings.Contains(line, "rules") || strings.Contains(line, "not counted") {
		t.Errorf("the line: %q", line)
	}
	if n := startFailures(t, repo.Root(), sha); n != 1 {
		t.Errorf("the failure was counted %d time(s), want 1", n)
	}
	if line := failedStart(t, "a session no store noted", strings.NewReader(startHookPayload(t, "s-unknown", repo.Root(), "a1", "t"))); !strings.Contains(line, "not counted") {
		t.Errorf("the line does not say the failure was not counted: %q", line)
	}
	if err := os.Remove(filepath.Join(repo.Root(), ".abcd", "rules.json")); err != nil {
		t.Fatal(err)
	}

	block := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(block, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ABCD_RULES_STATE_DIR", filepath.Join(block, "state"))
	failedStart(t, "a told fact that cannot be written", strings.NewReader(startHookPayload(t, "s1", repo.Root(), "a2", "t")))
	if n := startFailures(t, repo.Root(), sha); n != 2 {
		t.Errorf("the failure was counted %d time(s), want 2", n)
	}
	t.Setenv("ABCD_RULES_STATE_DIR", "")

	// A failure runs no git: the store is the session's note.
	log := gitCallLog(t)
	if err := subagentStartFailedForTest(t, "s1"); err == nil {
		t.Error("the failure path returned no error")
	}
	if calls := strings.TrimSpace(ghCalls(t, log)); calls != "" {
		t.Errorf("the failure path ran git:\n%s", calls)
	}
}

// subagentStartFailedForTest runs the failure path alone.
func subagentStartFailedForTest(t *testing.T, session string) error {
	t.Helper()
	cmd := newSubagentStartCommand()
	cmd.SetErr(io.Discard)
	return subagentStartFailed(cmd, hookInput{SessionID: session, AgentID: "a9"}, errors.New("x"))
}

func TestSubagentStartStopsAtItsOwnBound(t *testing.T) {
	repo, sha := subagentRepo(t)
	if err := history.NoteSessionRepo(repo.Root(), sha, "s1"); err != nil {
		t.Fatal(err)
	}
	setBound(t, &subagentStartBound, 300*time.Millisecond)
	never := func() io.Reader {
		r, w := io.Pipe()
		t.Cleanup(func() { _ = w.Close() })
		return r
	}
	t.Chdir(repo.Root())
	start := time.Now()
	failedStart(t, "a stdin that never ends", never())
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("the hook answered after %s", took)
	}
	// Silent where the checkout does not look managed.
	t.Chdir(realPath(t, t.TempDir()))
	if so, se, code := runSubagentStart(t, never()); so != "" || se != "" || code != 0 {
		t.Errorf("outside a managed checkout: exit %d stdout %q stderr %q", code, so, se)
	}
}

// TestSubagentStartStopsWhenGitDoesNotAnswer: git waiting on a configuration
// it cannot finish reading spends the bound, and the failure is the hook's.
func TestSubagentStartStopsWhenGitDoesNotAnswer(t *testing.T) {
	repo, _ := pipeIncludedRepo(t)
	writeUnder(t, repo, "AGENTS.md", "x\n")
	setBound(t, &subagentStartBound, 500*time.Millisecond)
	start := time.Now()
	failedStart(t, "git not answering", strings.NewReader(startHookPayload(t, "s1", repo, "a1", "t")))
	if took := time.Since(start); took > 10*time.Second {
		t.Errorf("the hook answered after %s", took)
	}
	if files := factFiles(t); len(files) != 0 {
		t.Errorf("a sub-agent that was not told left %v", files)
	}
}

func TestSubagentStartRecoversAPanicAtExitOne(t *testing.T) {
	repo, sha := subagentRepo(t)
	if err := history.NoteSessionRepo(repo.Root(), sha, "s1"); err != nil {
		t.Fatal(err)
	}
	swapExecutable(t, func() (string, error) { panic("a fault\nin two lines") })
	if line := failedStart(t, "a panic", strings.NewReader(startHookPayload(t, "s1", repo.Root(), "a1", "t"))); !strings.Contains(line, "internal fault") {
		t.Errorf("the line does not name the fault: %q", line)
	}
	if n := startFailures(t, repo.Root(), sha); n != 1 {
		t.Errorf("the panic was counted %d time(s), want 1", n)
	}
}
