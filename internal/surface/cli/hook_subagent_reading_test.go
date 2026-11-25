package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/core/rules"
)

// toolCallLine is one assistant line carrying one tool call.
func toolCallLine(t *testing.T, tool string, input any) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"type": "assistant", "message": map[string]any{
		"content": []any{map[string]any{"type": "tool_use", "name": tool, "input": input}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return string(b) + "\n"
}

func shellCallLine(t *testing.T, command string) string {
	t.Helper()
	return toolCallLine(t, "Bash", map[string]any{"command": command})
}

func readCallLine(t *testing.T, path string) string {
	t.Helper()
	return toolCallLine(t, "Read", map[string]any{"file_path": path})
}

// tellSubagent keeps a told fact as the start hook, or the guard, would.
func tellSubagent(t *testing.T, session, agent, file, by string) {
	t.Helper()
	if err := rules.WriteSubagentFact(session, agent, rules.SubagentTold, rules.SubagentRecord{File: file, Root: "/w", By: by}); err != nil {
		t.Fatal(err)
	}
}

// stopSubagent runs the stop hook for one sub-agent whose transcript holds body.
func stopSubagent(t *testing.T, repo, session, agent, body string) (stderr string) {
	t.Helper()
	tp := writeAgentTranscript(t, t.TempDir(), agent, body)
	_, se := runHook(t, subagentPayload(t, session, repo, agent, tp, "general-purpose"), "hook", "subagent-stop")
	return se
}

// keptLines are the lines the lane's marker holds, oldest first.
func keptLines(t *testing.T, repo, sha string) []history.SubagentLine {
	t.Helper()
	note, _, err := history.ReadSubagentLines(repo, sha)
	if err != nil {
		t.Fatal(err)
	}
	return note.Lines
}

func newestLine(t *testing.T, repo, sha string) history.SubagentLine {
	t.Helper()
	lines := keptLines(t, repo, sha)
	if len(lines) == 0 {
		return history.SubagentLine{}
	}
	return lines[len(lines)-1]
}

func TestSubagentStopKeepsALineForWhatWasSkipped(t *testing.T) {
	repo, sha := sessionEndRepo(t)
	read := readCallLine(t, "/any/AGENTS.md")
	rulesRun := shellCallLine(t, "scripts/abcd.sh rules COMMITTING")
	brief := readCallLine(t, "/w/.abcd/.work.local/run/run-1/lane-1/brief.md")
	ls := shellCallLine(t, "ls")
	for i, c := range []struct {
		name, file, by, body string
		want                 history.SubagentLineKind
	}{
		{"neither", "AGENTS.md", rules.SubagentToldByStart, ls, history.SubagentSkippedBoth},
		{"the rules alone", "AGENTS.md", rules.SubagentToldByStart, rulesRun, history.SubagentFileUnread},
		{"the file alone", "AGENTS.md", rules.SubagentToldByStart, read, history.SubagentRulesNotRun},
		{"a brief alone", "AGENTS.md", rules.SubagentToldByStart, brief, history.SubagentRulesNotRun},
		{"both", "AGENTS.md", rules.SubagentToldByStart, read + rulesRun, ""},
		{"a brief and the rules", "AGENTS.md", rules.SubagentToldByStart, brief + rulesRun, ""},
		{"no file to read", "", rules.SubagentToldByStart, ls, history.SubagentRulesNotRun},
		{"no file to read, rules run", "", rules.SubagentToldByStart, rulesRun, ""},
		{"told by the guard, neither", "AGENTS.md", rules.SubagentToldByGuard, ls, history.SubagentFileUnread},
		{"told by the guard, read", "AGENTS.md", rules.SubagentToldByGuard, read, ""},
		{"a write of the file", "AGENTS.md", rules.SubagentToldByStart, toolCallLine(t, "Write", map[string]any{"file_path": "/w/AGENTS.md"}) + rulesRun, history.SubagentFileUnread},
	} {
		agent := fmt.Sprintf("a%d", i)
		tellSubagent(t, "s1", agent, c.file, c.by)
		before := len(keptLines(t, repo, sha))
		stopSubagent(t, repo, "s1", agent, c.body)
		lines := keptLines(t, repo, sha)
		if c.want == "" {
			if len(lines) != before {
				t.Errorf("%s: a line was kept: %+v", c.name, lines[len(lines)-1])
			}
			continue
		}
		if len(lines) != before+1 {
			t.Errorf("%s: %d line(s) kept, want 1", c.name, len(lines)-before)
			continue
		}
		if l := lines[len(lines)-1]; l.Kind != c.want || l.AgentID != agent || l.AgentType != "general-purpose" {
			t.Errorf("%s: line %+v, want kind %s", c.name, l, c.want)
		}
		if _, ok := fact(t, "s1", agent, rules.SubagentJudged); !ok {
			t.Errorf("%s: no judged fact beside the line", c.name)
		}
	}
	// The transcript is staged as it was before.
	staged, err := history.ListStaged(repo, sha)
	if err != nil || len(staged) != 11 {
		t.Errorf("%d staged transcript(s) (%v), want 11", len(staged), err)
	}
}

func TestSubagentStopKeepsNoLineForASubagentNeverTold(t *testing.T) {
	repo, sha := sessionEndRepo(t)
	tellSubagent(t, "s1", "a1", "AGENTS.md", rules.SubagentToldByStart)
	stopSubagent(t, repo, "s2", "a1", shellCallLine(t, "ls"))
	stopSubagent(t, repo, "s1", "a2", shellCallLine(t, "ls"))
	if lines := keptLines(t, repo, sha); len(lines) != 0 {
		t.Errorf("a sub-agent never told left %+v", lines)
	}
}

func TestSubagentStopKeepsOneLineForASubagentThatStopsTwice(t *testing.T) {
	repo, sha := sessionEndRepo(t)
	tellSubagent(t, "s1", "a1", "AGENTS.md", rules.SubagentToldByStart)
	body := shellCallLine(t, "ls")
	for range 3 {
		stopSubagent(t, repo, "s1", "a1", body)
		body += shellCallLine(t, "pwd")
	}
	if lines := keptLines(t, repo, sha); len(lines) != 1 {
		t.Errorf("%d lines for one sub-agent: %+v", len(lines), lines)
	}
}

// TestSubagentStopSaysWhenASubagentCouldNotBeJudged: an absent, an unreadable
// and an oversized transcript each leave a line saying why, and so does a told
// fact that cannot be read.
func TestSubagentStopSaysWhenASubagentCouldNotBeJudged(t *testing.T) {
	repo, sha := sessionEndRepo(t)
	dir := t.TempDir()
	big := filepath.Join(dir, "agent-big.jsonl")
	if err := os.WriteFile(big, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(big, maxTranscriptBytes+1); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "agent-link.jsonl")
	if err := os.Symlink(big, link); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		agent, path string
		want        history.SubagentLineCause
	}{
		{"none", "", history.SubagentNoTranscript},
		{"gone", filepath.Join(dir, "gone.jsonl"), history.SubagentTranscriptUnread},
		{"link", link, history.SubagentTranscriptUnread},
		{"empty", writeAgentTranscript(t, dir, "empty", ""), history.SubagentTranscriptUnread},
		{"big", big, history.SubagentTranscriptTooLarge},
	} {
		tellSubagent(t, "s1", c.agent, "AGENTS.md", rules.SubagentToldByStart)
		runHook(t, subagentPayload(t, "s1", repo, c.agent, c.path, "t"), "hook", "subagent-stop")
		if l := newestLine(t, repo, sha); l.AgentID != c.agent || l.Kind != history.SubagentNotJudged || l.Cause != c.want {
			t.Errorf("%s: line %+v, want not judged for %s", c.agent, l, c.want)
		}
		// An untold sub-agent with the same transcript leaves nothing.
		before := len(keptLines(t, repo, sha))
		runHook(t, subagentPayload(t, "s1", repo, c.agent+"-untold", c.path, "t"), "hook", "subagent-stop")
		if n := len(keptLines(t, repo, sha)); n != before {
			t.Errorf("%s: an untold sub-agent left a line", c.agent)
		}
	}
	// A told fact that cannot be read.
	tellSubagent(t, "s1", "torn", "AGENTS.md", rules.SubagentToldByStart)
	state := abcdhome.Path(os.Getenv("HOME"), "rules-state")
	entries, err := os.ReadDir(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".told") {
			if err := os.WriteFile(filepath.Join(state, e.Name()), []byte("{torn"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	stopSubagent(t, repo, "s1", "torn", shellCallLine(t, "ls"))
	if l := newestLine(t, repo, sha); l.AgentID != "torn" || l.Kind != history.SubagentNotJudged || l.Cause != history.SubagentFactUnread {
		t.Errorf("a torn told fact: line %+v", l)
	}
}

// TestSubagentStopSaysWhenTheHoldCouldNotBeJudged: for a sub-agent with no
// reading the line says what the guard did about its writes.
func TestSubagentStopSaysWhenTheHoldCouldNotBeJudged(t *testing.T) {
	repo, sha := sessionEndRepo(t)
	note := func(agent string, f rules.SubagentFact) {
		t.Helper()
		if err := rules.WriteSubagentFact("s1", agent, f, rules.SubagentRecord{Reason: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	write := toolCallLine(t, "Edit", map[string]any{"file_path": "/w/x.go"})
	commit := shellCallLine(t, `git commit -m "$(cat <<'EOF'`+"\nx\nEOF\n"+`)"`)
	read := readCallLine(t, "AGENTS.md")
	for _, c := range []struct {
		agent, body string
		facts       []rules.SubagentFact
		kind        history.SubagentLineKind
		hold        history.SubagentLineHold
	}{
		{"never-wrote", shellCallLine(t, "ls"), nil, history.SubagentSkippedBoth, ""},
		{"held", write, []rules.SubagentFact{rules.SubagentHeld}, history.SubagentSkippedBoth, history.SubagentHoldHeld},
		{"unjudged", write, []rules.SubagentFact{rules.SubagentUnjudged}, history.SubagentSkippedBoth, history.SubagentHoldNotJudged},
		{"both", write, []rules.SubagentFact{rules.SubagentHeld, rules.SubagentUnjudged}, history.SubagentSkippedBoth, history.SubagentHoldBoth},
		{"a-write-no-fact", write, nil, history.SubagentSkippedBoth, history.SubagentHoldNotJudged},
		{"a-commit-no-fact", commit, nil, history.SubagentSkippedBoth, history.SubagentHoldNotJudged},
		{"held-then-read", write + read, []rules.SubagentFact{rules.SubagentHeld, rules.SubagentPassed}, history.SubagentRulesNotRun, ""},
	} {
		tellSubagent(t, "s1", c.agent, "AGENTS.md", rules.SubagentToldByStart)
		for _, f := range c.facts {
			note(c.agent, f)
		}
		stopSubagent(t, repo, "s1", c.agent, c.body)
		if l := newestLine(t, repo, sha); l.AgentID != c.agent || l.Kind != c.kind || l.Hold != c.hold {
			t.Errorf("%s: line %+v, want %s with hold %q", c.agent, l, c.kind, c.hold)
		}
	}
	// The guard could not learn whether the conventions apply, so nothing
	// told the sub-agent: the line is where that is said.
	note("untold", rules.SubagentUnjudged)
	stopSubagent(t, repo, "s1", "untold", write)
	if l := newestLine(t, repo, sha); l.AgentID != "untold" || l.Kind != history.SubagentNotJudged ||
		l.Cause != history.SubagentConventionsUnknown || l.Hold != history.SubagentHoldNotJudged {
		t.Errorf("unjudged and never told: line %+v", l)
	}
}

// TestJudgeKnowsTheRulesCommandTheStartMessagePrinted: both commands, as the
// start hook printed them, are the rules command at the stop.
func TestJudgeKnowsTheRulesCommandTheStartMessagePrinted(t *testing.T) {
	repo, _ := subagentRepo(t)
	printed := func(agent string) []string {
		t.Helper()
		var out []string
		for _, l := range strings.Split(startMessage(t, startHookPayload(t, "s1", repo.Root(), agent, "t")), "\n") {
			if cmd, ok := strings.CutPrefix(l, "  "); ok {
				cmd, _, _ = strings.Cut(cmd, "   ")
				out = append(out, cmd)
			}
		}
		if len(out) != 2 {
			t.Fatalf("the message prints %d commands, want 2: %q", len(out), out)
		}
		return out
	}
	swapExecutable(t, func() (string, error) { return "/opt/a b/abcd", nil })
	cmds := printed("a1")
	writeUnder(t, repo.Root(), "cmd/abcd/main.go", "package main\n")
	writeUnder(t, repo.Root(), "scripts/abcd.sh", "#!/bin/sh\n")
	cmds = append(cmds, printed("a2")...)
	swapExecutable(t, func() (string, error) { return "/opt/it's here/abcd", nil })
	cmds = append(cmds, printed("a3")...)
	// The script abcd's own conventions file names.
	cmds = append(cmds, `scripts/abcd.sh rules --prompt "x"`, "./scripts/abcd.sh rules SHELL")
	for _, c := range cmds {
		for _, line := range []string{c, "cd /w && " + c, c + " | head"} {
			if got := judgeSubagentTranscript([]byte(shellCallLine(t, line)), "AGENTS.md"); !got.RanRules {
				t.Errorf("the judge does not know the printed command %q", line)
			}
		}
	}
	if got := judgeSubagentTranscript([]byte(shellCallLine(t, "cat /opt/abcd/rules")), "AGENTS.md"); got.RanRules {
		t.Error("another command is taken for the rules command")
	}
}

// TestHistoryStagedListsSubagentLines: the kept lines are listed newest first,
// with what the host sent sanitised, and under --json they go to stderr beside
// an array that stays the array it is.
func TestHistoryStagedListsSubagentLines(t *testing.T) {
	repo, sha := sessionEndRepo(t)
	t.Chdir(repo)
	if so, _ := runHook(t, "", "history", "staged"); strings.Contains(so, "sub-agent") {
		t.Errorf("with no line kept: %q", so)
	}
	tellSubagent(t, "s1", "first", "AGENTS.md", rules.SubagentToldByStart)
	tellSubagent(t, "s1", "second", "AGENTS.md", rules.SubagentToldByStart)
	if err := rules.WriteSubagentFact("s1", "second", rules.SubagentHeld, rules.SubagentRecord{}); err != nil {
		t.Fatal(err)
	}
	for _, agent := range []string{"first", "second"} {
		tp := writeAgentTranscript(t, t.TempDir(), agent, shellCallLine(t, "ls"))
		runHook(t, subagentPayload(t, "s1", repo, agent, tp, "kind\x1b[31m"), "hook", "subagent-stop")
	}
	check := func(which, text string) {
		t.Helper()
		first, second := strings.Index(text, "sub-agent first"), strings.Index(text, "sub-agent second")
		if first < 0 || second < 0 || second > first {
			t.Errorf("%s does not list the lines newest first: %q", which, text)
		}
		for _, want := range []string{"2 sub-agent line(s)", "had not read the conventions file or run the rules command", "its writes were held"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s does not say %q: %q", which, want, text)
			}
		}
		if strings.Contains(text, "\x1b") {
			t.Errorf("%s prints an escape sequence: %q", which, text)
		}
	}
	so, _ := runHook(t, "", "history", "staged")
	check("the listing", so)
	so, se := runHook(t, "", "history", "staged", "--json")
	check("stderr under --json", se)
	var arr []json.RawMessage
	if err := json.Unmarshal([]byte(so), &arr); err != nil || len(arr) != 2 {
		t.Errorf("stdout under --json is not the array of 2: %v %s", err, so)
	}
	// A marker that cannot be read is named, and the listing stands.
	path := filepath.Join(abcdhome.Path(os.Getenv("HOME"), "transcripts"), sha, "subagent-readings.json")
	if err := os.WriteFile(path, []byte("{torn"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, se := runHook(t, "", "history", "staged"); !strings.Contains(se, "unreadable") {
		t.Errorf("a torn marker is not named: %q", se)
	}
}

// TestSubagentLineNoticesCountEachLineOnce: what the session-start hook says of
// the lines kept since the last time it said so.
func TestSubagentLineNoticesCountEachLineOnce(t *testing.T) {
	repo, sha := sessionEndRepo(t)
	if n := subagentLineNotices(repo, sha); len(n) != 0 {
		t.Errorf("with no line kept: %q", n)
	}
	for _, agent := range []string{"a1", "a2"} {
		tellSubagent(t, "s1", agent, "AGENTS.md", rules.SubagentToldByStart)
		stopSubagent(t, repo, "s1", agent, shellCallLine(t, "ls"))
	}
	n := subagentLineNotices(repo, sha)
	if len(n) != 1 || !strings.Contains(n[0], "2 sub-agent line(s)") || !strings.Contains(n[0], "abcd history staged") {
		t.Errorf("notices = %q", n)
	}
	if n := subagentLineNotices(repo, sha); len(n) != 0 {
		t.Errorf("a line is counted twice: %q", n)
	}
	path := filepath.Join(abcdhome.Path(os.Getenv("HOME"), "transcripts"), sha, "subagent-readings.json")
	if err := os.WriteFile(path, []byte("{torn"), 0o600); err != nil {
		t.Fatal(err)
	}
	if n := subagentLineNotices(repo, sha); len(n) != 1 || !strings.Contains(n[0], "could not be read") {
		t.Errorf("a torn marker: %q", n)
	}
}

// TestSessionStartCountsNewSubagentLinesOnce: the session-start hook says how
// many lines were kept since it last said so, and then says nothing of them.
func TestSessionStartCountsNewSubagentLinesOnce(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	start := func(session string) string {
		t.Helper()
		_, se, code := runSessionStart(fmt.Sprintf(`{"session_id":%q,"hook_event_name":"SessionStart","cwd":%s}`, session, mustJSON(t, repo)), "hook", "session-start")
		if code != 0 {
			t.Fatalf("session-start exited %d: %s", code, se)
		}
		return se
	}
	const notice = "sub-agent line(s) kept since the last notice"
	if se := start("s0"); strings.Contains(se, notice) {
		t.Errorf("with no line kept: %q", se)
	}
	for _, agent := range []string{"a1", "a2"} {
		tellSubagent(t, "s1", agent, "AGENTS.md", rules.SubagentToldByStart)
		stopSubagent(t, repo, "s1", agent, shellCallLine(t, "ls"))
	}
	if se := start("s2"); !strings.Contains(se, "2 "+notice) {
		t.Errorf("the two lines are not counted: %q", se)
	}
	if se := start("s3"); strings.Contains(se, notice) {
		t.Errorf("a line is counted twice: %q", se)
	}
	tellSubagent(t, "s4", "a3", "AGENTS.md", rules.SubagentToldByStart)
	stopSubagent(t, repo, "s4", "a3", shellCallLine(t, "ls"))
	if se := start("s5"); !strings.Contains(se, "1 "+notice) {
		t.Errorf("the new line is not counted alone: %q", se)
	}
}
