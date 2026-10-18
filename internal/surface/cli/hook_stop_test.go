package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
)

// hook_stop_test.go — step 5 of spc-2610180914469489: `abcd hook stop` and
// `abcd hook stop-failure`, the front doors onto the stop verdict.

// fakeOwed replaces the owed reader for one test with one answering owed, and
// counts the calls and what each was asked.
func fakeOwed(t *testing.T, owed *autonomy.Owed) *[]autonomy.ReadOptions {
	t.Helper()
	var asked []autonomy.ReadOptions
	prev := stopOwedReader
	stopOwedReader = func(_ context.Context, q autonomy.ReadOptions) (autonomy.Owed, error) {
		asked = append(asked, q)
		return *owed, nil
	}
	t.Cleanup(func() { stopOwedReader = prev })
	return &asked
}

func stopHookPayload(t *testing.T, session, cwd string, active bool, last string, crons ...string) string {
	t.Helper()
	var cs []map[string]any
	for i, c := range crons {
		cs = append(cs, map[string]any{"id": "cron-" + string(rune('a'+i)), "schedule": c, "recurring": true, "prompt": "look"})
	}
	b, err := json.Marshal(map[string]any{
		"session_id": session, "cwd": cwd, "hook_event_name": "Stop",
		"stop_hook_active": active, "last_assistant_message": last,
		"background_tasks": []any{}, "session_crons": cs,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// stopOut is the Stop hook's answer as the host reads it. A block is the
// host's non-error feedback, hookSpecificOutput.additionalContext, which
// keeps the session going under the same loop protections as a decision
// block without the "Stop hook error" notice; Decision and Reason are filled
// from it, and Raw keeps what was printed.
type stopOut struct {
	Decision           string `json:"decision"`
	Reason             string `json:"reason"`
	SystemMessage      string `json:"systemMessage"`
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
	Raw map[string]any `json:"-"`
}

func runStop(t *testing.T, payload string) (stopOut, string, string) {
	t.Helper()
	so, se, err := runSplit(t, payload, "hook", "stop")
	if err != nil {
		t.Fatalf("hook stop exited non-zero: %v\nstderr: %s", err, se)
	}
	var out stopOut
	if strings.TrimSpace(so) != "" {
		if err := json.Unmarshal([]byte(so), &out); err != nil {
			t.Fatalf("hook stop stdout is not one object: %v\n%s", err, so)
		}
		_ = json.Unmarshal([]byte(so), &out.Raw)
		if c := out.HookSpecificOutput.AdditionalContext; c != "" && out.HookSpecificOutput.HookEventName == "Stop" {
			out.Decision, out.Reason = "block", c
		}
	}
	return out, so, se
}

var behindItem = autonomy.OwedItem{Kind: autonomy.OwedBehind, Capped: true, Branch: "feat/behind", PR: 12, Head: "aaaa"}

// TestHookStopOutput: `{"decision":"block","reason":…}` only on a block, a
// systemMessage while present with work owed; a malformed payload, an
// unreadable record or a held lock allows with a stderr line.
func TestHookStopOutput(t *testing.T) {
	root := awayCheckout(t)
	owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	asked := fakeOwed(t, &owed)

	// Present, work owed: allowed, flagged, the reader asked for disk only.
	out, so, _ := runStop(t, stopHookPayload(t, "s1", root, false, "Done."))
	if out.Decision != "" || !strings.Contains(out.SystemMessage, "feat/behind") {
		t.Errorf("present with work owed: %s", so)
	}
	if len(*asked) != 1 || (*asked)[0].Record.Holds(time.Now()) {
		t.Errorf("present: the reader was asked %+v, want once, not away", *asked)
	}
	// Present, nothing owed: silent.
	owed = autonomy.Owed{}
	if _, so, _ := runStop(t, stopHookPayload(t, "s1", root, false, "Done.")); so != "" {
		t.Errorf("present with nothing owed printed %q", so)
	}

	// Away, work owed: blocked.
	setAway(t)
	owed = autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	out, so, _ = runStop(t, stopHookPayload(t, "s1", root, false, "Done.", "*/10 * * * *"))
	if out.Decision != "block" || !strings.Contains(out.Reason, "feat/behind") || out.SystemMessage != "" {
		t.Errorf("away with work owed, first stop: %s", so)
	}
	if q := (*asked)[len(*asked)-1]; q.Root != root || !q.Record.Holds(time.Now()) || q.Home == "" {
		t.Errorf("away: the reader was asked %+v", q)
	}
	// Away, nothing owed, a qualifying wake: allowed, silent.
	owed = autonomy.Owed{}
	if _, so, _ := runStop(t, stopHookPayload(t, "s2", root, false, "Done.", "*/30 * * * *")); so != "" {
		t.Errorf("away, nothing owed, a wake: printed %q", so)
	}

	// A malformed payload allows, with a stderr line.
	if _, so, se := runStop(t, "{not json"); so != "" || !strings.Contains(se, "allow") {
		t.Errorf("malformed payload: stdout %q stderr %q", so, se)
	}
	// A bad session id while away allows, with a stderr line.
	if _, so, se := runStop(t, stopHookPayload(t, "../evil", root, false, "Done.")); so != "" || !strings.Contains(se, "session id") {
		t.Errorf("bad session id: stdout %q stderr %q", so, se)
	}

	// A held lock, when the stop must be recorded, allows with a stderr line.
	owed = autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	st := awayStore(t, root)
	if err := autonomy.SaveSession(root, "s3", autonomy.SessionState{
		Fingerprint: owed.Fingerprint(), WakeFingerprint: owed.CappedFingerprint(), Blocks: autonomy.DefaultBlockCap,
	}); err != nil {
		t.Fatal(err)
	}
	var held stopOut
	var heldOut, heldErr string
	if err := st.WithLock(func() error {
		held, heldOut, heldErr = runStop(t, stopHookPayload(t, "s3", root, true, "Done."))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if held.Decision != "" || heldOut != "" || !strings.Contains(heldErr, "lock") {
		t.Errorf("held lock: stdout %q stderr %q; want the stop allowed and the lock named", heldOut, heldErr)
	}

	// An unreadable record allows, with a stderr line.
	if err := os.WriteFile(st.RecordPath(), []byte("{garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, so, se := runStop(t, stopHookPayload(t, "s1", root, false, "Done.")); so != "" || !strings.Contains(se, "not away") {
		t.Errorf("unreadable record: stdout %q stderr %q", so, se)
	}
}

// TestHookStopCapFromEnvironment: CLAUDE_CODE_STOP_HOOK_BLOCK_CAP sets the
// cap, and the block passing it records the stop, labelled possibly forced by
// the cap, in the away record.
func TestHookStopCapFromEnvironment(t *testing.T) {
	root := awayCheckout(t)
	setAway(t)
	owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	fakeOwed(t, &owed)
	t.Setenv("CLAUDE_CODE_STOP_HOOK_BLOCK_CAP", "3")
	for i := 1; i <= 4; i++ {
		out, so, _ := runStop(t, stopHookPayload(t, "cap-1", root, i > 1, "Done."))
		if out.Decision != "block" {
			t.Fatalf("block %d: %s", i, so)
		}
		stops := readAway(t, root).Stops
		if want := map[bool]int{true: 1, false: 0}[i == 4]; len(stops) != want {
			t.Fatalf("after block %d the record holds %d stops, want %d", i, len(stops), want)
		}
	}
	s := readAway(t, root).Stops[0]
	if s.Kind != autonomy.StopWorkOwed || !s.PossiblyForcedByCap || !strings.Contains(s.Detail, "feat/behind") {
		t.Errorf("recorded %+v", s)
	}
}

// TestHookStopWakeCountAcrossRuns: the wake count and its fingerprint are
// read back from the session's state file by each hook run, kept in the
// session's start copy even when the hook runs from a worktree it made.
func TestHookStopWakeCountAcrossRuns(t *testing.T) {
	root := awayCheckout(t)
	if _, _, err := runSplit(t, awayStartPayload("wake-1", root, "startup"), "hook", "session-start"); err != nil {
		t.Fatal(err)
	}
	made := filepath.Join(t.TempDir(), "made")
	gitAt(t, root, "worktree", "add", "-q", "-b", "feat/made", made)
	made = realPath(t, made)
	setAway(t)
	owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	asked := fakeOwed(t, &owed)
	var blocks []bool
	for i := 0; i < 5; i++ {
		out, _, _ := runStop(t, stopHookPayload(t, "wake-1", made, false, "Done.", "*/10 * * * *"))
		blocks = append(blocks, out.Decision == "block")
	}
	want := []bool{true, false, false, true, true}
	for i := range want {
		if blocks[i] != want[i] {
			t.Fatalf("stops blocked %v, want %v (first block, two wakes, then blocked)", blocks, want)
		}
	}
	if q := (*asked)[0]; q.Root != root {
		t.Errorf("the reader was asked about %s, want the start copy %s", q.Root, root)
	}
	st, err := autonomy.LoadSession(root, "wake-1")
	if err != nil || st.Wakes != 2 || st.WakeFingerprint != owed.CappedFingerprint() {
		t.Errorf("the start copy's state file: %+v (%v)", st, err)
	}
	if _, err := os.Stat(filepath.Join(made, filepath.FromSlash(autonomy.SessionsDir))); err == nil {
		t.Errorf("the made worktree holds session state; it belongs to the start copy")
	}
}

// TestHookStopRecordsHandBack: a hand-back the verdict accepts is recorded
// in the away record, once per branch and head, and counted on the person's
// line.
func TestHookStopRecordsHandBack(t *testing.T) {
	root := awayCheckout(t)
	setAway(t, "--hold-pushes")
	item := autonomy.OwedItem{Kind: autonomy.OwedUnpushed, Capped: true, Branch: "feat/ready", Head: strings.Repeat("c", 40)}
	owed := autonomy.Owed{HandedBack: []autonomy.OwedItem{item}}
	fakeOwed(t, &owed)
	for i := 0; i < 2; i++ {
		if out, so, _ := runStop(t, stopHookPayload(t, "hb-1", root, false, "Done.", "*/30 * * * *")); out.Decision != "" {
			t.Fatalf("a handed-back item kept the stop blocked: %s", so)
		}
	}
	rec := readAway(t, root)
	if len(rec.HandedBack) != 1 || rec.HandedBack[0].Branch != "feat/ready" {
		t.Errorf("record holds hand-backs %+v, want feat/ready once", rec.HandedBack)
	}
	if !strings.Contains(rec.PersonLine(), "1 item handed back") {
		t.Errorf("the person's line: %q", rec.PersonLine())
	}
}

// TestHookStopFailureRecords: the error type is recorded while away, and
// nothing is written while present.
func TestHookStopFailureRecords(t *testing.T) {
	root := awayCheckout(t)
	payload := `{"session_id":"f1","cwd":"` + root + `","hook_event_name":"StopFailure","error":"rate_limit","error_details":"429 Too Many Requests","last_assistant_message":"API Error: Rate limit reached"}`
	so, se, err := runSplit(t, payload, "hook", "stop-failure")
	if err != nil || so != "" {
		t.Fatalf("present: err %v stdout %q stderr %q", err, so, se)
	}
	if _, err := os.Stat(awayStore(t, root).Dir()); err == nil {
		t.Errorf("present: the away store was created")
	}
	setAway(t)
	for i := 0; i < 2; i++ {
		if so, se, err := runSplit(t, payload, "hook", "stop-failure"); err != nil || so != "" {
			t.Fatalf("away: err %v stdout %q stderr %q", err, so, se)
		}
	}
	stops := readAway(t, root).Stops
	if len(stops) != 1 || stops[0].Kind != autonomy.StopAPIError || stops[0].ErrorType != "rate_limit" || stops[0].Count != 2 {
		t.Errorf("recorded %+v, want one api-error entry for rate_limit counted twice", stops)
	}
	// A malformed payload or an odd error type never fails the hook.
	if _, _, err := runSplit(t, "{nope", "hook", "stop-failure"); err != nil {
		t.Errorf("malformed payload: %v", err)
	}
	odd := strings.Replace(payload, "rate_limit", "rate limit\\u001b[31m", 1)
	if _, _, err := runSplit(t, odd, "hook", "stop-failure"); err != nil {
		t.Fatal(err)
	}
	last := readAway(t, root).Stops
	if got := last[len(last)-1].ErrorType; got != "unknown" {
		t.Errorf("an error type outside the host's shape was recorded as %q, want unknown", got)
	}
}

// gitCallLog puts a git on PATH that logs each call's arguments to the
// returned file and runs the real git.
func gitCallLog(t *testing.T) string {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	bin := t.TempDir()
	log := filepath.Join(t.TempDir(), "git-calls.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + log + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// unmanagedOwnUnpushed is a checkout abcd does not manage (no .abcd/ at its
// root) on an own branch, pushed without -u to nowhere: a commit of the
// checkout's author no remote holds.
func unmanagedOwnUnpushed(t *testing.T) string {
	t.Helper()
	repo := realPath(t, t.TempDir())
	gitInitAt(t, repo)
	gitAt(t, repo, "config", "--local", "user.email", "alice@example.invalid")
	gitAt(t, repo, "config", "--local", "user.name", "Alice")
	gitAt(t, repo, "config", "--local", "commit.gpgsign", "false")
	gitAt(t, repo, "commit", "-q", "--allow-empty", "-m", "seed")
	gitAt(t, repo, "checkout", "-q", "-b", "feat/elsewhere")
	gitAt(t, repo, "commit", "-q", "--allow-empty", "-m", "not pushed")
	return repo
}

// TestHookStopUnmanagedRepoSilent: in a checkout abcd does not manage the
// stop hook allows silently, never reads the owed work and runs no git past
// naming the checkout's root; StopFailure records nothing there, and the
// session-start and prompt hooks add no away block, even with an away record
// holding for the repository (set from a managed copy of it).
func TestHookStopUnmanagedRepoSilent(t *testing.T) {
	t.Run("stop", func(t *testing.T) {
		awayCheckout(t) // the sandboxed HOME
		repo := unmanagedOwnUnpushed(t)
		owed := autonomy.Owed{Items: []autonomy.OwedItem{{Kind: autonomy.OwedUnpushed, Capped: true, Branch: "feat/elsewhere", Head: "aaaa"}}}
		asked := fakeOwed(t, &owed)
		log := gitCallLog(t)
		_, so, se := runStop(t, stopHookPayload(t, "u1", repo, false, "Done."))
		if so != "" || se != "" {
			t.Errorf("an unmanaged checkout: stdout %q stderr %q, want nothing", so, se)
		}
		if len(*asked) != 0 {
			t.Errorf("an unmanaged checkout's owed work was read: %+v", *asked)
		}
		for _, line := range strings.Split(strings.TrimSpace(ghCalls(t, log)), "\n") {
			// `git version` is the client's version probe, read once a
			// process, not a read of the repository.
			if line != "" && line != "version" && !strings.Contains(line, "rev-parse --show-toplevel") {
				t.Errorf("an unmanaged checkout ran git past naming its root: %q", line)
			}
		}
	})
	t.Run("away record from a managed copy", func(t *testing.T) {
		root := awayCheckout(t)
		// A clone shares the repository's root commit, so its away store,
		// but has no .abcd/ at its root (the fixture's .abcd/ is untracked).
		sib := filepath.Join(t.TempDir(), "sib")
		gitAt(t, root, "clone", "-q", root, sib)
		sib = realPath(t, sib)
		setAway(t)
		owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
		asked := fakeOwed(t, &owed)
		if _, so, se := runStop(t, stopHookPayload(t, "u2", sib, false, "Done.")); so != "" || len(*asked) != 0 {
			t.Errorf("stop from an unmanaged copy: stdout %q stderr %q, reader asked %d times", so, se, len(*asked))
		}
		fail := `{"session_id":"u2","cwd":"` + sib + `","hook_event_name":"StopFailure","error":"rate_limit"}`
		if _, _, err := runSplit(t, fail, "hook", "stop-failure"); err != nil {
			t.Fatal(err)
		}
		if stops := readAway(t, root).Stops; len(stops) != 0 {
			t.Errorf("StopFailure from an unmanaged copy recorded %+v", stops)
		}
		if so, _, _ := runSplit(t, awayStartPayload("u2", sib, "startup"), "hook", "session-start"); strings.Contains(so, "## AWAY") {
			t.Errorf("session start in an unmanaged copy added the away block:\n%s", so)
		}
		if so, _, _ := runSplit(t, awayPromptPayload("u2", sib, "hello"), "hook", "prompt-router"); strings.Contains(so, "AWAY") || strings.Contains(so, "away") {
			t.Errorf("a prompt in an unmanaged copy carried the away block or line:\n%s", so)
		}
		// The managed copy is unchanged.
		if out, so, _ := runStop(t, stopHookPayload(t, "u3", root, false, "Done.")); out.Decision != "block" {
			t.Errorf("the managed copy no longer blocks: %s", so)
		}
	})
}

// TestHookStopBlockIsFeedback: a block is written as the host's non-error
// feedback, {"hookSpecificOutput":{"hookEventName":"Stop","additionalContext":…}},
// never as {"decision":"block"}, which the host shows as a hook error.
func TestHookStopBlockIsFeedback(t *testing.T) {
	root := awayCheckout(t)
	setAway(t)
	owed := autonomy.Owed{Items: []autonomy.OwedItem{behindItem}}
	fakeOwed(t, &owed)
	out, so, _ := runStop(t, stopHookPayload(t, "fb-1", root, false, "Done."))
	if _, ok := out.Raw["decision"]; ok {
		t.Errorf("a block is written as a decision, which the host shows as an error: %s", so)
	}
	if out.HookSpecificOutput.HookEventName != "Stop" || !strings.Contains(out.HookSpecificOutput.AdditionalContext, "feat/behind") {
		t.Errorf("a block is the Stop hook's additionalContext naming the item: %s", so)
	}
	if len(out.Raw) != 1 {
		t.Errorf("a block carries only hookSpecificOutput: %s", so)
	}
}

// homeFiles lists every path under dir, relative, for a before-and-after
// compare.
func homeFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(dir, func(p string, _ os.DirEntry, err error) error {
		if err == nil && p != dir {
			rel, _ := filepath.Rel(dir, p)
			out = append(out, rel)
		}
		return nil
	})
	return out
}

// TestAwayHooksLeaveUnmanagedRepoAlone: in a repository abcd does not manage
// every hook path of the away mode returns at once: the stop check (with the
// real owed reader), StopFailure and the branch recorder print nothing,
// create nothing under abcd's home folder, exit 0 and run no git past
// naming the checkout's root; the session-start and prompt hooks add no away
// block and create no away store, session note or state file. (The away
// store has no record here; a record shared with a managed copy is
// TestHookStopUnmanagedRepoSilent's.)
func TestAwayHooksLeaveUnmanagedRepoAlone(t *testing.T) {
	awayCheckout(t) // the sandboxed HOME
	home := os.Getenv("HOME")
	repo := unmanagedOwnUnpushed(t)
	abcdHome := filepath.Join(home, ".abcd.noindex")
	before := homeFiles(t, abcdHome)
	log := gitCallLog(t)
	onlyRoot := func(what string) {
		t.Helper()
		for _, line := range strings.Split(strings.TrimSpace(ghCalls(t, log)), "\n") {
			if line != "" && line != "version" && !strings.Contains(line, "rev-parse --show-toplevel") {
				t.Errorf("%s ran git past naming the root: %q", what, line)
			}
		}
		if err := os.WriteFile(log, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	so, se, err := runSplit(t, stopHookPayload(t, "un-1", repo, false, "Done."), "hook", "stop")
	if err != nil || so != "" {
		t.Errorf("stop: err %v stdout %q stderr %q", err, so, se)
	}
	onlyRoot("stop")
	fail := `{"session_id":"un-1","cwd":"` + repo + `","hook_event_name":"StopFailure","error":"rate_limit"}`
	if so, se, err := runSplit(t, fail, "hook", "stop-failure"); err != nil || so != "" {
		t.Errorf("stop-failure: err %v stdout %q stderr %q", err, so, se)
	}
	onlyRoot("stop-failure")
	baseline := func(command string) string {
		t.Helper()
		so, se, code := runGuard(preToolUse(t, "Bash", command, repo), "guard", "hook")
		if code != 0 || strings.Contains(so, "deny") {
			t.Fatalf("guard hook on %q: code %d stdout %q stderr %q", command, code, so, se)
		}
		return ghCalls(t, log)
	}
	// The guard's own reading of a command is not the recorder's: a command
	// with no branch act sets the baseline the branch acts must not exceed.
	_ = os.WriteFile(log, nil, 0o600)
	plain := baseline("git status --porcelain")
	for _, c := range []string{"git switch -c feat/new", "git worktree add -b feat/wt ../wt", "git checkout -b feat/co"} {
		_ = os.WriteFile(log, nil, 0o600)
		got := baseline(c)
		for _, line := range strings.Split(strings.TrimSpace(got), "\n") {
			if line != "" && line != "version" && !strings.Contains(line, "rev-parse --show-toplevel") && !strings.Contains(plain, line) {
				t.Errorf("the recorder ran git for %q in an unmanaged checkout: %q", c, line)
			}
		}
	}
	if after := homeFiles(t, abcdHome); !slices.Equal(before, after) {
		t.Errorf("abcd's home folder changed: before %v after %v", before, after)
	}

	so, _, _ = runSplit(t, awayStartPayload("un-1", repo, "startup"), "hook", "session-start")
	so2, _, _ := runSplit(t, awayPromptPayload("un-1", repo, "hello"), "hook", "prompt-router")
	if strings.Contains(so+so2, "AWAY") || strings.Contains(so2, "away until") {
		t.Errorf("an unmanaged checkout got the away block or line:\n%s\n%s", so, so2)
	}
	if _, err := os.Stat(filepath.Join(abcdHome, autonomy.StoreDir)); err == nil {
		t.Errorf("an unmanaged checkout created the away store")
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd")); err == nil {
		t.Errorf("an unmanaged checkout gained a .abcd/ folder")
	}
}

// TestHookStopPanicAllows: a panic inside the stop check allows the stop,
// exits 0 and names it on stderr; it never reaches the host as exit 2, which
// a Stop hook reads as a block.
func TestHookStopPanicAllows(t *testing.T) {
	root := awayCheckout(t)
	setAway(t)
	prev := stopOwedReader
	stopOwedReader = func(context.Context, autonomy.ReadOptions) (autonomy.Owed, error) { panic("boom") }
	t.Cleanup(func() { stopOwedReader = prev })
	var so, se string
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("the panic escaped the hook: %v", r)
			}
		}()
		so, se, err = runSplit(t, stopHookPayload(t, "p-1", root, false, "Done."), "hook", "stop")
	}()
	if err != nil || so != "" || !strings.Contains(se, "allowed") || !strings.Contains(se, "boom") {
		t.Errorf("a panic: err %v stdout %q stderr %q", err, so, se)
	}
}
