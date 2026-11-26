package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gittest"
)

// heldCall is one pre-tool-use payload of a sub-agent in a checkout.
type heldCall struct {
	session, agent, kind, cwd string
	transcript                string // the session's own transcript path
	tool                      string
	input                     map[string]any
}

func (c heldCall) payload(t *testing.T) string {
	t.Helper()
	p := map[string]any{
		"session_id": c.session, "cwd": c.cwd, "hook_event_name": "PreToolUse",
		"tool_name": c.tool, "tool_input": c.input,
	}
	if c.agent != "" {
		p["agent_id"], p["agent_type"] = c.agent, c.kind
	}
	if c.transcript != "" {
		p["transcript_path"] = c.transcript
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// holdFixture is a managed checkout and a session whose transcript stands under
// a temporary directory, the host's layout beside it.
type holdFixture struct {
	repo       *gittest.Repo
	transcript string
}

func newHoldFixture(t *testing.T) holdFixture {
	t.Helper()
	repo, _ := subagentRepo(t)
	return holdFixture{repo: repo, transcript: filepath.Join(t.TempDir(), "sess.jsonl")}
}

// says sets what the sub-agent's transcript holds.
func (f holdFixture) says(t *testing.T, agent, body string) {
	t.Helper()
	writeUnder(t, strings.TrimSuffix(f.transcript, ".jsonl"), "subagents/agent-"+agent+".jsonl", body)
}

func (f holdFixture) write(agent string) heldCall {
	return heldCall{session: "s1", agent: agent, kind: "general-purpose", cwd: f.repo.Root(), transcript: f.transcript,
		tool: "Write", input: map[string]any{"file_path": filepath.Join(f.repo.Root(), "x.txt"), "content": "y"}}
}

func (f holdFixture) shell(agent, command string) heldCall {
	c := f.write(agent)
	c.tool, c.input = "Bash", map[string]any{"command": command}
	return c
}

// held reports whether the guard holds the call, with its reason.
func held(t *testing.T, c heldCall) (bool, string) {
	t.Helper()
	so, se, code := runGuard(c.payload(t), "guard", "hook")
	if !isDeny(so) {
		return false, se
	}
	reason := mustDeny(t, so, se, code)
	return strings.Contains(reason, "unread conventions"), reason
}

func mustHold(t *testing.T, what string, c heldCall) string {
	t.Helper()
	ok, reason := held(t, c)
	if !ok {
		t.Errorf("%s: not held (%s)", what, reason)
	}
	return reason
}

func mustPass(t *testing.T, what string, c heldCall) {
	t.Helper()
	if so, _, _ := runGuard(c.payload(t), "guard", "hook"); isDeny(so) {
		t.Errorf("%s: refused: %s", what, so)
	}
}

func TestGuardHoldsASubagentsWriteUntilItHasReadTheConventions(t *testing.T) {
	f := newHoldFixture(t)
	startMessage(t, startHookPayload(t, "s1", f.repo.Root(), "a1", "t"))
	for i, tool := range []string{"Write", "Edit", "MultiEdit", "NotebookEdit"} {
		agent := fmt.Sprintf("a%d", i+1)
		f.says(t, agent, shellCallLine(t, "ls"))
		c := f.write(agent)
		c.tool = tool
		reason := mustHold(t, tool, c)
		for _, want := range []string{"AGENTS.md", f.repo.Root(), "cat AGENTS.md", "reading tool", "a search for it", "a call of its own", "same shell line", "same turn", "send this write again"} {
			if !strings.Contains(reason, want) {
				t.Errorf("%s: the refusal does not say %q: %s", tool, want, reason)
			}
		}
		if _, ok := fact(t, "s1", agent, rules.SubagentHeld); !ok {
			t.Errorf("%s: no held fact", tool)
		}
		mustHold(t, tool+" again", c)
		f.says(t, agent, shellCallLine(t, "ls")+readCallLine(t, "/w/AGENTS.md"))
		mustPass(t, tool+" after the read", c)
		if rec, ok := fact(t, "s1", agent, rules.SubagentPassed); !ok || rec.Reason != "read" {
			t.Errorf("%s: passed = %+v, %v", tool, rec, ok)
		}
		// A third time from the fact alone, with no transcript to read.
		if err := os.RemoveAll(strings.TrimSuffix(f.transcript, ".jsonl")); err != nil {
			t.Fatal(err)
		}
		mustPass(t, tool+" from the fact", c)
		if _, ok := fact(t, "s1", agent, rules.SubagentUnjudged); ok {
			t.Errorf("%s: a passed sub-agent was judged again", tool)
		}
	}
	// Another sub-agent of the session is held on its own account.
	f.says(t, "b1", shellCallLine(t, "ls"))
	mustHold(t, "another sub-agent", f.write("b1"))
}

func TestGuardTakesAShellReadAsAReading(t *testing.T) {
	f := newHoldFixture(t)
	for i, command := range []string{"cat AGENTS.md", "sed -n 1,40p ./AGENTS.md", "grep -n check AGENTS.md | head"} {
		agent := fmt.Sprintf("a%d", i)
		f.says(t, agent, shellCallLine(t, command))
		mustPass(t, command, f.write(agent))
		if rec, ok := fact(t, "s1", agent, rules.SubagentPassed); !ok || rec.Reason != "read" {
			t.Errorf("%s: passed = %+v, %v", command, rec, ok)
		}
	}
	f.says(t, "search", toolCallLine(t, "Grep", map[string]any{"pattern": "x", "path": "AGENTS.md"}))
	mustPass(t, "a search", f.write("search"))
	f.says(t, "ls", shellCallLine(t, "ls"))
	mustHold(t, "no reading", f.write("ls"))
}

func TestGuardHoldsTheGitCommandsThatRecordOrSendWork(t *testing.T) {
	f := newHoldFixture(t)
	f.says(t, "a1", shellCallLine(t, "ls"))
	for _, command := range []string{
		"git commit -m x",
		"git commit -m \"$(cat <<'EOF'\nfeat: x\nEOF\n)\"",
		"git merge x", "git rebase main", "git cherry-pick abc", "git revert abc", "git am x.patch", "git pull --rebase",
		"git push origin HEAD:lane",
		"git add -A && git commit -m x",
		"if ! git diff --cached --quiet; then git commit -m wip; fi",
		"{ git commit -m x; }",
		`for b in a b; do git push origin "$b"; done`,
		"function f { git commit -m x; }; f",
		"FOO+=1 git commit -m x",
		`g() { git "$@"; }; g commit -m x`,
	} {
		if reason := mustHold(t, command, f.shell("a1", command)); !strings.Contains(reason, "AGENTS.md") {
			t.Errorf("%s: %s", command, reason)
		}
	}
	f.says(t, "a1", readCallLine(t, "AGENTS.md"))
	mustPass(t, "a commit after the read", f.shell("a1", "git commit -m x"))
	// The registry judges the command once the hold lets it by.
	writeUnder(t, f.repo.Root(), ".abcd/guard.json", releaseBlockerCfg)
	so, _, _ := runGuard(f.shell("a1", "git commit -m x && make release").payload(t), "guard", "hook")
	if !isDeny(so) || strings.Contains(so, "unread conventions") {
		t.Errorf("the registry no longer judges a sub-agent's command: %s", so)
	}
}

func TestGuardDoesNotTakeAWriteThatNamesTheFileAsAReading(t *testing.T) {
	f := newHoldFixture(t)
	f.says(t, "a1", toolCallLine(t, "Write", map[string]any{"file_path": filepath.Join(f.repo.Root(), "AGENTS.md"), "content": "x"}))
	mustHold(t, "a refused write of the file itself", f.write("a1"))
	f.says(t, "a2", shellCallLine(t, "cat AGENTS.md && git commit -m x"))
	mustHold(t, "a read in the line that commits", f.shell("a2", "cat AGENTS.md && git commit -m x"))
	mustHold(t, "the same line again", f.shell("a2", "cat AGENTS.md && git commit -m x"))
}

func TestGuardHoldsASubagentTheStartHookNeverTold(t *testing.T) {
	f := newHoldFixture(t)
	f.says(t, "a1", shellCallLine(t, "ls"))
	reason := mustHold(t, "a write", f.write("a1"))
	if !strings.Contains(reason, f.repo.Root()) {
		t.Errorf("the refusal does not name the root: %s", reason)
	}
	told, ok := fact(t, "s1", "a1", rules.SubagentTold)
	if !ok || told.By != rules.SubagentToldByGuard || told.File != "AGENTS.md" || told.Root != f.repo.Root() {
		t.Errorf("told = %+v, %v", told, ok)
	}
	// CLAUDE.md is the file where there is no AGENTS.md.
	if err := os.Rename(filepath.Join(f.repo.Root(), "AGENTS.md"), filepath.Join(f.repo.Root(), "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	f.says(t, "a2", shellCallLine(t, "ls"))
	if reason := mustHold(t, "a write", f.write("a2")); !strings.Contains(reason, "CLAUDE.md") {
		t.Errorf("the refusal does not name CLAUDE.md: %s", reason)
	}
}

func TestGuardNeverHoldsTheSessionItself(t *testing.T) {
	f := newHoldFixture(t)
	for _, c := range []heldCall{f.write(""), f.shell("", "git commit -m x")} {
		mustPass(t, c.tool, c)
	}
	if files := factFiles(t); len(files) != 0 {
		t.Errorf("the session's own call left %v", files)
	}
	// A value of another JSON type names no sub-agent, and the payload is
	// still read.
	so, se, code := runGuard(fmt.Sprintf(`{"session_id":"s1","cwd":%q,"tool_name":"Bash","agent_id":7,"agent_type":[],"transcript_path":{},"tool_input":{"command":"git status"}}`, f.repo.Root()), "guard", "hook")
	if code != 0 || so != "" || se != "" {
		t.Errorf("a payload with loose fields: exit %d stdout %q stderr %q", code, so, se)
	}
}

func TestGuardNeverHoldsAColdReader(t *testing.T) {
	f := newHoldFixture(t)
	f.says(t, "a1", shellCallLine(t, "ls"))
	for _, kind := range []string{"abcd:cold-reading-detection", "cold-reading-widening"} {
		for _, c := range []heldCall{f.write("a1"), f.shell("a1", "git commit -m x")} {
			c.kind = kind
			mustPass(t, kind, c)
		}
	}
	if files := factFiles(t); len(files) != 0 {
		t.Errorf("a cold reader left %v", files)
	}
}

func TestGuardHoldsNothingWhereNothingApplies(t *testing.T) {
	f := newHoldFixture(t)
	f.says(t, "a1", shellCallLine(t, "ls"))
	nothing := func(what string, calls ...heldCall) {
		t.Helper()
		for _, c := range calls {
			mustPass(t, what, c)
		}
		if files := factFiles(t); len(files) != 0 {
			t.Errorf("%s left %v", what, files)
		}
	}
	nothing("a command that records nothing", f.shell("a1", "git status"), f.shell("a1", "git stash push"), f.shell("a1", "echo git commit"))

	plain := realPath(t, gittest.NewRepo(t).Root())
	writeUnder(t, plain, "AGENTS.md", "x\n")
	c := f.write("a1")
	c.cwd, c.input = plain, map[string]any{"file_path": filepath.Join(plain, "x.txt")}
	s := f.shell("a1", "git commit -m x")
	s.cwd = plain
	nothing("an unmanaged repository", c, s)

	writeUnder(t, f.repo.Root(), ".abcd/rules.json", `{"schema_version":1,"disabled":true,"domains":{}}`)
	nothing("rules switched off", f.write("a1"), f.shell("a1", "git commit -m x"))
	if err := os.Remove(filepath.Join(f.repo.Root(), ".abcd", "rules.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(f.repo.Root(), "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	nothing("no conventions file", f.write("a1"), f.shell("a1", "git commit -m x"))
	// Told by the start hook of no file, the same.
	startMessage(t, startHookPayload(t, "s1", f.repo.Root(), "a2", "t"))
	mustPass(t, "told of no file", f.write("a2"))
}

func TestGuardDoesNotHoldARunsBriefedAgent(t *testing.T) {
	f := newHoldFixture(t)
	brief := filepath.Join(f.repo.Root(), ".abcd/.work.local/run/run-1/lane-1/brief.md")
	f.says(t, "a1", readCallLine(t, brief))
	mustPass(t, "a briefed agent", f.shell("a1", "git commit -m x"))
	if rec, ok := fact(t, "s1", "a1", rules.SubagentPassed); !ok || rec.Reason != "briefed" {
		t.Errorf("passed = %+v, %v", rec, ok)
	}
	f.says(t, "a2", readCallLine(t, filepath.Join(filepath.Dir(brief), "validate/round-1/fix/brief.md")))
	mustHold(t, "a fix round's brief", f.write("a2"))
	f.says(t, "a3", shellCallLine(t, "ls"))
	t.Setenv(runner.StageWritesEnv, f.repo.Root())
	if ok, reason := held(t, f.write("a3")); ok {
		t.Errorf("a drive's stage is held: %s", reason)
	}
}

func TestGuardDoesNotHoldWhatItCannotJudge(t *testing.T) {
	f := newHoldFixture(t)
	// unjudged asserts a call is let by, that the fact says why, and that a
	// second call answers from it with no git asked and nothing read.
	unjudged := func(what, why string, c heldCall, undo func()) {
		t.Helper()
		mustPass(t, what, c)
		rec, ok := fact(t, c.session, c.agent, rules.SubagentUnjudged)
		if !ok || rec.Reason != why {
			t.Errorf("%s: unjudged = %+v, %v; want %s", what, rec, ok, why)
		}
		if undo != nil {
			undo()
		}
		f.says(t, c.agent, shellCallLine(t, "ls"))
		c.transcript = f.transcript
		if c.tool == "Bash" {
			mustPass(t, what+", a second call", c)
		} else {
			log := gitCallLog(t)
			mustPass(t, what+", a second call", c)
			if calls := strings.TrimSpace(ghCalls(t, log)); calls != "" {
				t.Errorf("%s: the second call ran git:\n%s", what, calls)
			}
		}
		for _, f := range []rules.SubagentFact{rules.SubagentHeld, rules.SubagentPassed} {
			if _, ok := fact(t, c.session, c.agent, f); ok {
				t.Errorf("%s: the second call left a %s fact", what, f)
			}
		}
	}
	none := f.write("p1")
	none.transcript = ""
	unjudged("no transcript path", "no-transcript-path", none, nil)
	rel := f.write("p2")
	rel.transcript = "sess.jsonl"
	unjudged("a relative transcript path", "no-transcript-path", rel, nil)
	odd := f.write("p3")
	odd.transcript = strings.TrimSuffix(f.transcript, ".jsonl")
	unjudged("a transcript of another form", "no-transcript-path", odd, nil)
	f.says(t, "t3", "x")
	big := filepath.Join(strings.TrimSuffix(f.transcript, ".jsonl"), "subagents", "agent-t3.jsonl")
	if err := os.Truncate(big, maxTranscriptBytes+1); err != nil {
		t.Fatal(err)
	}
	unjudged("a transcript over the cap", "transcript-unreadable", f.shell("t3", "git commit -m x"), nil)
	if err := os.Symlink(big, filepath.Join(filepath.Dir(big), "agent-t4.jsonl")); err != nil {
		t.Fatal(err)
	}
	unjudged("a transcript that is a link", "transcript-unreadable", f.write("t4"), func() {
		if err := os.Remove(filepath.Join(filepath.Dir(big), "agent-t4.jsonl")); err != nil {
			t.Fatal(err)
		}
	})

	// A told fact naming another file.
	tellSubagent(t, "s1", "n1", "AGENTS.md", rules.SubagentToldByStart)
	state := abcdhome.Path(os.Getenv("HOME"), "rules-state")
	entries, err := os.ReadDir(state)
	if err != nil {
		t.Fatal(err)
	}
	var told string
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".told") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(state, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), `"root":"/w"`) {
			told = filepath.Join(state, e.Name())
		}
	}
	if err := os.WriteFile(told, []byte(`{"file":"README.md","by":"start"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	unjudged("a told fact naming another file", "facts-unreadable", f.write("n1"), nil)

	// A rules file that will not load.
	rulesFile := filepath.Join(f.repo.Root(), ".abcd", "rules.json")
	writeUnder(t, f.repo.Root(), ".abcd/rules.json", "{not json")
	unjudged("a rules file that will not load", "conventions-unknown", f.write("r1"), func() { _ = os.Remove(rulesFile) })

	// Ids that can key nothing leave nothing.
	before := len(factFiles(t))
	for _, c := range []heldCall{f.write("a/b"), f.write("a b")} {
		mustPass(t, "an agent id that is no plain scalar", c)
	}
	s := f.write("k1")
	s.session = ""
	mustPass(t, "no session id", s)
	if after := len(factFiles(t)); after != before {
		t.Errorf("%d fact(s) kept under ids that can key nothing", after-before)
	}

	// A store that cannot be read.
	block := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(block, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	f.says(t, "z1", shellCallLine(t, "ls"))
	t.Setenv("ABCD_RULES_STATE_DIR", filepath.Join(block, "state"))
	mustPass(t, "a fact store that cannot be read", f.write("z1"))
}

// TestGuardJudgesAgainOnceTheTranscriptIsThere: a transcript that is absent or
// empty holds nothing and settles nothing, so the next write is judged on what
// stands then, and the stop still says that one was not judged.
func TestGuardJudgesAgainOnceTheTranscriptIsThere(t *testing.T) {
	f := newHoldFixture(t)
	for _, c := range []struct{ agent, what string }{{"t1", "an absent transcript"}, {"t2", "an empty transcript"}} {
		if c.agent == "t2" {
			f.says(t, c.agent, "")
		}
		for _, call := range []heldCall{f.write(c.agent), f.shell(c.agent, "git commit -m x")} {
			mustPass(t, c.what, call)
		}
		if rec, ok := fact(t, "s1", c.agent, rules.SubagentUnjudged); !ok || rec.Reason != "transcript-not-there" {
			t.Errorf("%s: unjudged = %+v, %v", c.what, rec, ok)
		}
		f.says(t, c.agent, shellCallLine(t, "ls"))
		mustHold(t, c.what+", then one with no reading", f.write(c.agent))
		mustHold(t, c.what+", then one with no reading", f.shell(c.agent, "git commit -m x"))
		if _, ok := fact(t, "s1", c.agent, rules.SubagentHeld); !ok {
			t.Errorf("%s: no held fact", c.what)
		}
		line, keep := subagentStopLine(hookInput{SessionID: "s1", AgentID: c.agent}, []byte(shellCallLine(t, "ls")), "")
		if !keep || line.Hold != history.SubagentHoldBoth {
			t.Errorf("%s: the stop's line is %+v, %v", c.what, line, keep)
		}
		if text := subagentLineText(line); !strings.HasSuffix(text, "; its writes were held, and at least one could not be judged and was not held") {
			t.Errorf("%s: the stop's line reads %q", c.what, text)
		}
		f.says(t, c.agent, shellCallLine(t, "ls")+readCallLine(t, "AGENTS.md"))
		mustPass(t, c.what+", then a reading", f.write(c.agent))
		if rec, ok := fact(t, "s1", c.agent, rules.SubagentPassed); !ok || rec.Reason != "read" {
			t.Errorf("%s: passed = %+v, %v", c.what, rec, ok)
		}
	}
}

// TestGuardAFinalCauseReplacesATranscriptNotThere: once a sub-agent could not
// be judged for a cause that settles it, it is not judged again, whatever the
// first call found.
func TestGuardAFinalCauseReplacesATranscriptNotThere(t *testing.T) {
	f := newHoldFixture(t)
	mustPass(t, "an absent transcript", f.write("a1"))
	f.says(t, "a1", "x")
	path := filepath.Join(strings.TrimSuffix(f.transcript, ".jsonl"), "subagents", "agent-a1.jsonl")
	if err := os.Truncate(path, maxTranscriptBytes+1); err != nil {
		t.Fatal(err)
	}
	mustPass(t, "a transcript over the cap", f.write("a1"))
	if rec, ok := fact(t, "s1", "a1", rules.SubagentUnjudged); !ok || rec.Reason != "transcript-unreadable" {
		t.Errorf("unjudged = %+v, %v", rec, ok)
	}
	f.says(t, "a1", shellCallLine(t, "ls"))
	mustPass(t, "a later call", f.write("a1"))
	// A transcript that is not there settles nothing a final cause settled.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	mustPass(t, "an absent transcript, after", f.write("a1"))
	if rec, ok := fact(t, "s1", "a1", rules.SubagentUnjudged); !ok || rec.Reason != "transcript-unreadable" {
		t.Errorf("unjudged, after = %+v, %v", rec, ok)
	}
}

// TestGuardTellsASubagentThatCannotReadWhoIsNotHeld: a sub-agent with no tool
// but one that writes stays held, and the refusal says what it does then.
func TestGuardTellsASubagentThatCannotReadWhoIsNotHeld(t *testing.T) {
	f := newHoldFixture(t)
	write := toolCallLine(t, "Write", map[string]any{"file_path": filepath.Join(f.repo.Root(), "x.txt"), "content": "AGENTS.md"})
	for i := range 3 {
		f.says(t, "a1", strings.Repeat(write, i+1))
		reason := mustHold(t, "a sub-agent that only writes", f.write("a1"))
		for _, want := range []string{"With no tool but ones that write", "end and say so to the session that started you", "never held"} {
			if !strings.Contains(reason, want) {
				t.Errorf("the refusal does not say %q: %s", want, reason)
			}
		}
	}
}

// TestGuardDoesNotHoldWhenGitDoesNotAnswer: git that stalls is unjudged, once.
func TestGuardDoesNotHoldWhenGitDoesNotAnswer(t *testing.T) {
	repo, pipe := pipeIncludedRepo(t)
	writeUnder(t, repo, "AGENTS.md", "x\n")
	setBound(t, &guardGitBudget, 500*time.Millisecond)
	c := heldCall{session: "s1", agent: "a1", kind: "t", cwd: repo, transcript: filepath.Join(t.TempDir(), "s.jsonl"),
		tool: "Write", input: map[string]any{"file_path": filepath.Join(repo, "x.txt")}}
	if a := runGuardHookWithin(t, c.payload(t), pipe, 10*time.Second); isDeny(a.stdout) {
		t.Errorf("a stalled git held the call: %s", a.stdout)
	}
	if rec, ok := fact(t, "s1", "a1", rules.SubagentUnjudged); !ok || rec.Reason != "git-did-not-answer" {
		t.Errorf("unjudged = %+v, %v", rec, ok)
	}
	if a := runGuardHookWithin(t, c.payload(t), pipe, 400*time.Millisecond); isDeny(a.stdout) {
		t.Errorf("the second call was held: %s", a.stdout)
	}
}
