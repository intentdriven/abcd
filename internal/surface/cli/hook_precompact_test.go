package cli

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/checklist"
)

// hook_precompact_test.go — the PreCompact event sidecar (spc-2610230805429429
// step 6): the compaction's time and trigger are appended to the working
// copy's checklist.events.jsonl, and the hook never blocks the compaction.

// preCompact runs the PreCompact hook's verb in root with trigger, as the host
// sends it, and answers both streams and the exit code.
func preCompact(t *testing.T, root, trigger string) (stdout, stderr string, code int) {
	t.Helper()
	b, err := json.Marshal(map[string]string{
		"session_id": "s-compact", "cwd": root, "hook_event_name": "PreCompact",
		"trigger": trigger, "custom_instructions": "keep the plan",
	})
	if err != nil {
		t.Fatal(err)
	}
	return runSessionStart(string(b), "hook", "prompt-router-reset")
}

// compactionEvents is every line of root's event sidecar, decoded.
func compactionEvents(t *testing.T, root string) []map[string]any {
	t.Helper()
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(checklist.EventsRelPath)))
	if err != nil {
		t.Fatalf("the event sidecar: %v", err)
	}
	defer f.Close()
	var out []map[string]any
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e map[string]any
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("a sidecar line is not JSON: %v: %q", err, sc.Text())
		}
		out = append(out, e)
	}
	return out
}

// TestPreCompactRecordsTheCompaction: each compaction appends one line with its
// time and trigger. A trigger the host does not document is recorded as
// "other", and nothing else from the payload is copied in.
func TestPreCompactRecordsTheCompaction(t *testing.T) {
	root := checklistCheckout(t)
	before := time.Now().Add(-time.Second)
	for _, trigger := range []string{"auto", "manual", "something-new"} {
		if stdout, stderr, code := preCompact(t, root, trigger); code != 0 || stdout != "" || strings.Contains(stderr, "not recorded") {
			t.Fatalf("PreCompact %s: exit %d, stdout %q, stderr %q", trigger, code, stdout, stderr)
		}
	}
	events := compactionEvents(t, root)
	if len(events) != 3 {
		t.Fatalf("want one line per compaction, 3; got %d: %v", len(events), events)
	}
	for i, want := range []string{"auto", "manual", "other"} {
		e := events[i]
		if e["schema"] != checklist.EventsSchema || e["event"] != "compact" || e["trigger"] != want {
			t.Errorf("line %d = %v; want schema %q, event compact, trigger %q", i, e, checklist.EventsSchema, want)
		}
		at, err := time.Parse(time.RFC3339Nano, e["at"].(string))
		if err != nil || at.Before(before) || at.After(time.Now().Add(time.Second)) {
			t.Errorf("line %d: at %v is not the compaction's time (%v)", i, e["at"], err)
		}
		if len(e) != 4 {
			t.Errorf("line %d carries more than its schema, time, event and trigger: %v", i, e)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(checklist.EventsRelPath)))
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"keep the plan", "something-new", "s-compact"} {
		if strings.Contains(string(data), leak) {
			t.Errorf("the sidecar copies %q from the payload: %s", leak, data)
		}
	}
	fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(checklist.EventsRelPath)))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("the sidecar's mode = %v (%v); want 0600, as the tier's other stores", fi.Mode().Perm(), err)
	}
}

// TestPreCompactNeverBlocks: a compaction that cannot be recorded still exits 0
// and writes nothing to stdout, so the host compacts, and the hook says on
// stderr that the compaction went unrecorded and goes ahead.
func TestPreCompactNeverBlocks(t *testing.T) {
	root := checklistCheckout(t)
	// A folder where the sidecar would be: the append cannot open it.
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(checklist.EventsRelPath)), 0o700); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := preCompact(t, root, "auto")
	if code != 0 || stdout != "" {
		t.Fatalf("an unrecorded compaction must not block: exit %d, stdout %q", code, stdout)
	}
	for _, want := range []string{"compaction was not recorded", "checklist.events.jsonl", "the compaction goes ahead"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr must say %q; stderr = %q", want, stderr)
		}
	}
	if strings.Contains(stderr, root) {
		t.Errorf("stderr names the checkout's path: %q", stderr)
	}
}

// TestPreCompactRecordsNothingElsewhere: a session start runs the same verb
// and records no compaction, and a directory abcd does not manage gets no
// sidecar and no line.
func TestPreCompactRecordsNothingElsewhere(t *testing.T) {
	root := checklistCheckout(t)
	b, _ := json.Marshal(map[string]string{"session_id": "s-start", "cwd": root, "hook_event_name": "SessionStart", "source": "compact"})
	if _, _, code := runSessionStart(string(b), "hook", "prompt-router-reset"); code != 0 {
		t.Fatalf("the reset on a session start exited %d", code)
	}
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(checklist.EventsRelPath))); !os.IsNotExist(err) {
		t.Errorf("a session start recorded a compaction (%v)", err)
	}
	plain := t.TempDir()
	stdout, stderr, code := preCompact(t, plain, "auto")
	if code != 0 || stdout != "" || strings.Contains(stderr, "not recorded") {
		t.Errorf("outside a managed checkout: exit %d, stdout %q, stderr %q; want a quiet exit 0", code, stdout, stderr)
	}
	if _, err := os.Lstat(filepath.Join(plain, filepath.FromSlash(checklist.EventsRelPath))); !os.IsNotExist(err) {
		t.Errorf("a sidecar was written outside a managed checkout (%v)", err)
	}
}

// TestPreCompactLeavesAFileItDidNotMakeAlone: a second name for another file,
// or a FIFO, in the sidecar's place is left as it was, and the hook still exits
// 0 with nothing on stdout, so the compaction goes ahead unrecorded.
func TestPreCompactLeavesAFileItDidNotMakeAlone(t *testing.T) {
	t.Run("a second name for another file", func(t *testing.T) {
		root := checklistCheckout(t)
		other := filepath.Join(t.TempDir(), "log.txt")
		if err := os.WriteFile(other, []byte("someone else's file\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(other, filepath.Join(root, filepath.FromSlash(checklist.EventsRelPath))); err != nil {
			t.Skipf("no hard link across these directories: %v", err)
		}
		stdout, stderr, code := preCompact(t, root, "auto")
		if code != 0 || stdout != "" || !strings.Contains(stderr, "compaction was not recorded") {
			t.Fatalf("exit %d, stdout %q, stderr %q; want exit 0, nothing on stdout, and the unrecorded line", code, stdout, stderr)
		}
		if got, err := os.ReadFile(other); err != nil || string(got) != "someone else's file\n" {
			t.Fatalf("the other file was appended to: %q (%v)", got, err)
		}
	})
	t.Run("a FIFO", func(t *testing.T) {
		root := checklistCheckout(t)
		p := filepath.Join(root, filepath.FromSlash(checklist.EventsRelPath))
		if err := syscall.Mkfifo(p, 0o600); err != nil {
			t.Skipf("no FIFO here: %v", err)
		}
		type res struct {
			stdout, stderr string
			code           int
		}
		done := make(chan res, 1)
		go func() {
			so, se, c := preCompact(t, root, "auto")
			done <- res{so, se, c}
		}()
		select {
		case r := <-done:
			if r.code != 0 || r.stdout != "" || !strings.Contains(r.stderr, "compaction was not recorded") {
				t.Fatalf("exit %d, stdout %q, stderr %q; want exit 0, nothing on stdout, and the unrecorded line", r.code, r.stdout, r.stderr)
			}
		case <-time.After(60 * time.Second):
			t.Fatal("the hook waited on a FIFO in the sidecar's place")
		}
		if fi, err := os.Lstat(p); err != nil || fi.Mode()&os.ModeNamedPipe == 0 {
			t.Fatalf("the FIFO was replaced: %v %v", fi, err)
		}
	})
}
