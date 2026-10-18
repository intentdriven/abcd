package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
)

// hook_away_test.go — step 2 of spc-2610180914469489: the away record repeated
// to the agent at every session start and on every prompt, and the one line
// the person sees on every prompt.

func awayStartPayload(session, cwd, source string) string {
	b, _ := json.Marshal(map[string]string{"session_id": session, "cwd": cwd, "source": source, "hook_event_name": "SessionStart"})
	return string(b)
}

func awayPromptPayload(session, cwd, prompt string) string {
	b, _ := json.Marshal(map[string]string{"session_id": session, "cwd": cwd, "prompt": prompt, "hook_event_name": "UserPromptSubmit"})
	return string(b)
}

// promptHookOut is the prompt hook's stdout decoded: the plain text when it is
// not an object, else the object's fields.
type promptHookOut struct {
	Plain   string
	Message string
	Context string
}

func decodePromptHook(t *testing.T, stdout string) promptHookOut {
	t.Helper()
	trimmed := strings.TrimSpace(stdout)
	if !strings.HasPrefix(trimmed, "{") {
		return promptHookOut{Plain: stdout, Context: stdout}
	}
	var v struct {
		SystemMessage      string `json:"systemMessage"`
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		t.Fatalf("the prompt hook's stdout is neither plain text nor one object: %v\n%s", err, stdout)
	}
	if v.HookSpecificOutput.HookEventName != "" && v.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
		t.Errorf("hookEventName = %q", v.HookSpecificOutput.HookEventName)
	}
	return promptHookOut{Message: v.SystemMessage, Context: v.HookSpecificOutput.AdditionalContext}
}

func setAway(t *testing.T, args ...string) {
	t.Helper()
	if _, stderr, err := runSplit(t, "", append([]string{"mode", "autonomous"}, args...)...); err != nil {
		t.Fatalf("mode autonomous: %v\n%s", err, stderr)
	}
}

// assertAwayBlock checks the away block's load-bearing sentences.
func assertAwayBlock(t *testing.T, where, text string) {
	t.Helper()
	for _, want := range []string{
		"## AWAY",
		"grants no rights",
		"abcd decide",
		"abcd mode queue",
		"--approve",
		"embargo triage verdict",
		"drain's hand-back",
		"release sign-off",
		"without a standing instruction",
		"adding a dependency",
		"force push",
		"spends money",
		"Run `abcd mode back` only on the person's word in this session",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("%s: the away block misses %q:\n%s", where, want, text)
		}
	}
}

// TestSessionStartAwayBlockEverySource: startup, resume and compact print the
// away block while the record holds, and nothing of it when it does not.
func TestSessionStartAwayBlockEverySource(t *testing.T) {
	root := awayCheckout(t)
	for _, source := range []string{"startup", "resume", "compact"} {
		so, _, err := runSplit(t, awayStartPayload("s-"+source, root, source), "hook", "session-start")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(so, "## AWAY") {
			t.Errorf("%s while present printed the away block:\n%s", source, so)
		}
	}
	setAway(t, "--until", "09:00")
	for _, source := range []string{"startup", "resume", "compact"} {
		so, _, err := runSplit(t, awayStartPayload("s-"+source, root, source), "hook", "session-start")
		if err != nil {
			t.Fatal(err)
		}
		assertAwayBlock(t, "session-start "+source, so)
		if !strings.Contains(so, "Alice") {
			t.Errorf("%s: the block does not name who set it:\n%s", source, so)
		}
	}
	// Ended: nothing again.
	if _, _, err := runSplit(t, "", "mode", "back"); err != nil {
		t.Fatal(err)
	}
	so, _, _ := runSplit(t, awayStartPayload("s-after", root, "startup"), "hook", "session-start")
	if strings.Contains(so, "## AWAY") {
		t.Errorf("after mode back the session start still prints the block:\n%s", so)
	}
}

// TestPromptRouterAwayBlockEveryPrompt: the block rides every prompt of a
// session, the second as well as the first, outside the loader's dedup ledger.
func TestPromptRouterAwayBlockEveryPrompt(t *testing.T) {
	root := awayCheckout(t)
	so, _, _ := runSplit(t, awayPromptPayload("p1", root, "hello"), "hook", "prompt-router")
	if strings.Contains(so, "## AWAY") {
		t.Fatalf("present: the prompt carries the away block:\n%s", so)
	}
	setAway(t)
	for i, prompt := range []string{"hello", "and again"} {
		so, _, err := runSplit(t, awayPromptPayload("p1", root, prompt), "hook", "prompt-router")
		if err != nil {
			t.Fatal(err)
		}
		out := decodePromptHook(t, so)
		assertAwayBlock(t, "prompt "+string(rune('1'+i)), out.Context)
	}
	// The --json envelope carries it in text too.
	so, _, _ = runSplit(t, awayPromptPayload("p1", root, "third"), "hook", "prompt-router", "--json")
	var v routerView
	if err := json.Unmarshal([]byte(so), &v); err != nil {
		t.Fatalf("--json: %v\n%s", err, so)
	}
	assertAwayBlock(t, "--json text", v.Text)
	if !strings.Contains(v.SystemMessage, "abcd mode back") {
		t.Errorf("--json system_message = %q", v.SystemMessage)
	}
}

// TestPromptRouterAwaySystemMessage: one line the person sees on every prompt
// while the record holds: who set it, from which copy, the counts of queued
// steps and handed-back items, and the literal `abcd mode back`.
func TestPromptRouterAwaySystemMessage(t *testing.T) {
	root := awayCheckout(t)
	setAway(t, "--until", "09:00")
	for _, step := range []string{"push feat/x", "merge #12"} {
		if _, stderr, err := runSplit(t, "", "mode", "queue", step); err != nil {
			t.Fatalf("mode queue: %v\n%s", err, stderr)
		}
	}
	if _, err := awayStore(t, root).RecordHandBack("feat/x", strings.Repeat("a", 40), time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, prompt := range []string{"one", "two"} {
		so, _, err := runSplit(t, awayPromptPayload("m1", root, prompt), "hook", "prompt-router")
		if err != nil {
			t.Fatal(err)
		}
		msg := decodePromptHook(t, so).Message
		for _, want := range []string{"Alice", "2 steps queued", "1 item handed back", "`abcd mode back`", "away until"} {
			if !strings.Contains(msg, want) {
				t.Errorf("prompt %q: the person's line misses %q: %q", prompt, want, msg)
			}
		}
		if strings.Contains(msg, "\n") {
			t.Errorf("the person's line is more than one line: %q", msg)
		}
		if !strings.Contains(msg, filepathBase(root)) {
			t.Errorf("the person's line does not name the copy: %q", msg)
		}
	}
}

func filepathBase(p string) string { return filepath.Base(p) }

// TestSessionStartNotesCopy: the session's start worktree is noted per session
// id, and a later read from a worktree the session made resolves to the start
// copy; a session with no note resolves to its own worktree.
func TestSessionStartNotesCopy(t *testing.T) {
	root := awayCheckout(t)
	if _, _, err := runSplit(t, awayStartPayload("note-1", root, "startup"), "hook", "session-start"); err != nil {
		t.Fatal(err)
	}
	made := filepath.Join(t.TempDir(), "made")
	gitAt(t, root, "worktree", "add", "-q", "-b", "feat/made", made)
	made = realPath(t, made)

	got, _, err := autonomy.ResolveCopy(made, "note-1", autonomy.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Errorf("a hook run from the made worktree resolves to %s, want the start copy %s", got, root)
	}
	// A compaction from the made worktree keeps the first note.
	if _, _, err := runSplit(t, awayStartPayload("note-1", made, "compact"), "hook", "session-start"); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := autonomy.ResolveCopy(made, "note-1", autonomy.Options{}); got != root {
		t.Errorf("after a compaction from the made worktree the session resolves to %s, want %s", got, root)
	}
	// No note: the payload's own worktree.
	if got, _, _ := autonomy.ResolveCopy(made, "never-started", autonomy.Options{}); got != made {
		t.Errorf("a session with no note resolves to %s, want its own worktree %s", got, made)
	}
}

// TestPromptRouterLapsedWindowLine: a lapsed, unclosed record with a queued
// step or a recorded stop shows the person one line naming `abcd mode back`
// on every prompt until that verb closes it; one with neither shows nothing.
func TestPromptRouterLapsedWindowLine(t *testing.T) {
	root := awayCheckout(t)
	st := awayStore(t, root)
	lapse := func(queue, stop bool) {
		t.Helper()
		past := time.Now().Add(-3 * time.Hour)
		if _, err := st.Update(func(*autonomy.Record) (*autonomy.Record, error) {
			r, err := autonomy.Begin(nil, autonomy.Who{Name: "Alice", Copy: root}, past, past.Add(time.Hour), false)
			if err != nil {
				return nil, err
			}
			if queue {
				_ = r.AddStep("push feat/x", root, past)
			}
			if stop {
				r.AddStop(autonomy.Stop{Kind: autonomy.StopWorkOwed, Detail: "behind #12"}, past)
			}
			return r, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		name        string
		queue, stop bool
		want        bool
	}{
		{"queued step", true, false, true},
		{"recorded stop", false, true, true},
		{"neither", false, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			lapse(c.queue, c.stop)
			for _, prompt := range []string{"one", "two"} {
				so, _, _ := runSplit(t, awayPromptPayload("l1", root, prompt), "hook", "prompt-router")
				out := decodePromptHook(t, so)
				if strings.Contains(out.Context, "## AWAY") {
					t.Errorf("a lapsed record still sends the away block:\n%s", out.Context)
				}
				has := strings.Contains(out.Message, "`abcd mode back`")
				if has != c.want {
					t.Errorf("prompt %q: lapsed line shown=%v, want %v: %q", prompt, has, c.want, out.Message)
				}
				if c.want && strings.Contains(out.Message, "\n") {
					t.Errorf("the lapsed line is more than one line: %q", out.Message)
				}
			}
			if !c.want {
				return
			}
			if _, _, err := runSplit(t, "", "mode", "back"); err != nil {
				t.Fatal(err)
			}
			so, _, _ := runSplit(t, awayPromptPayload("l1", root, "after"), "hook", "prompt-router")
			if msg := decodePromptHook(t, so).Message; msg != "" {
				t.Errorf("after mode back the lapsed line still shows: %q", msg)
			}
		})
	}
}
