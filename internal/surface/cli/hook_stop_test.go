package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/autonomy"
)

// hook_stop_test.go — step 5 of spc-2610180914469489: `abcd hook stop` and
// `abcd hook stop-failure`, the front doors onto the stop verdict.

// fakeOwed replaces the owed reader for one test with one answering owed, and
// counts the calls and what each was asked.
func fakeOwed(t *testing.T, owed *autonomy.Owed) *[]stopReadOptions {
	t.Helper()
	var asked []stopReadOptions
	prev := stopOwedReader
	stopOwedReader = func(_ context.Context, q stopReadOptions) (autonomy.Owed, error) {
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

type stopOut struct {
	Decision      string `json:"decision"`
	Reason        string `json:"reason"`
	SystemMessage string `json:"systemMessage"`
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
