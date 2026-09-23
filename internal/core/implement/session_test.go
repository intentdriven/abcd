package implement

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestJoiningNeedsNoWordFromTheFirstSession: a second session joins a run the
// first has been writing, and the only trace is its own record and its own
// session_open line — nothing is written anywhere the first must read.
func TestJoiningNeedsNoWordFromTheFirstSession(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	res, err := r.Join("beta", RoleSecond, "opus", "window 1")
	if err != nil || res.Rejoined {
		t.Fatalf("join = %+v, %v", res, err)
	}
	open := lastEvent(t, r, EventSessionOpen)
	if open.Session != "beta" || open.String("role") != "second" || open.String("model") != "opus" || open.String("reason") != "window 1" {
		t.Fatalf("session_open = %+v", open.Fields)
	}
	ss, err := r.Sessions()
	if err != nil || len(ss) != 2 || ss[1].Session != "beta" || ss[1].Role != RoleSecond {
		t.Fatalf("sessions = %+v, %v", ss, err)
	}
	// A resume keeps the role and says so.
	res, err = r.Join("beta", RoleSecond, "", "resume")
	if err != nil || !res.Rejoined {
		t.Fatalf("rejoin = %+v, %v", res, err)
	}
	if e := lastEvent(t, r, EventSessionOpen); e.String("rejoin") != "true" {
		t.Fatalf("rejoin line = %+v", e.Fields)
	}
}

// TestASessionKeepsItsRole: joining again under the other role is refused, so a
// second session cannot talk itself into the first's bounds mid-run.
func TestASessionKeepsItsRole(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "beta", RoleSecond)
	if _, err := r.Join("beta", RoleFirst, "", ""); !errors.Is(err, ErrRefused) {
		t.Fatalf("role change = %v; want a refusal", err)
	}
	ss, _ := r.Sessions()
	if ss[0].Role != RoleSecond {
		t.Fatalf("role after a refused change = %s", ss[0].Role)
	}
	for _, bad := range []string{"", "../x", ".hidden", strings.Repeat("a", 65)} {
		if _, err := r.Join(bad, RoleFirst, "", ""); !errors.Is(err, ErrRefused) {
			t.Errorf("join %q = %v; want a refusal", bad, err)
		}
	}
	if _, err := r.Join("gamma", Role("third"), "", ""); !errors.Is(err, ErrRefused) {
		t.Fatalf("unknown role = %v; want a refusal", err)
	}
}

// TestAWindowNamesItsMode: each mode's window line, and the second session
// refused the setting of it.
func TestAWindowNamesItsMode(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleSecond)
	for i, m := range Modes() {
		if _, err := r.SetMode("alpha", m, i+1); err != nil {
			t.Fatalf("mode %s: %v", m, err)
		}
		e := lastEvent(t, r, EventWindowMode)
		if e.String("mode") != string(m) || e.String("window") != string(rune('1'+i)) {
			t.Fatalf("window_mode line = %+v", e.Fields)
		}
		w, ok, err := r.CurrentMode()
		if err != nil || !ok || w.Mode != m {
			t.Fatalf("current mode = %+v %v %v", w, ok, err)
		}
	}
	if _, err := r.SetMode("beta", ModeClaim, 9); !errors.Is(err, ErrRefused) {
		t.Fatalf("second session setting the mode = %v; want a refusal", err)
	}
	if w, _, _ := r.CurrentMode(); w.Mode != ModeSplitRoles {
		t.Fatalf("a refused mode change moved the mode to %s", w.Mode)
	}
	if _, err := ParseMode("pairs"); !errors.Is(err, ErrRefused) {
		t.Fatalf("unknown mode = %v", err)
	}
}

// TestAHandWrittenWindowLineCounts: the run writes window_mode lines by hand, and
// the bounds read them exactly as they read the verb's.
func TestAHandWrittenWindowLineCounts(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "beta", RoleSecond)
	line := `{"ts":"` + c.now().Add(time.Minute).Format(time.RFC3339) + `","session":"A1","event":"window_mode","mode":"split-roles"}` + "\n"
	f, err := os.OpenFile(filepath.Join(r.Dir, logFileName(c.now())), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(line); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-1", Lane: "l"}); !errors.Is(err, ErrRefused) {
		t.Fatalf("claim under a hand-written split-roles window = %v; want a refusal", err)
	}
}

// TestCheckHoldsTheSecondSessionsBounds: the release step and a corpus lane are
// refused to the second session and logged; review, audit and land are open;
// the first session may take every step.
func TestCheckHoldsTheSecondSessionsBounds(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleSecond)
	for _, st := range Steps() {
		v, err := r.Check("alpha", st, []string{"internal/core/lint/x.go"})
		if err != nil || !v.Allowed {
			t.Fatalf("first session, %s: %+v %v", st, v, err)
		}
	}
	for _, st := range []Step{StepReview, StepAudit, StepLand} {
		if v, err := r.Check("beta", st, nil); err != nil || !v.Allowed {
			t.Fatalf("second session, %s: %+v %v", st, v, err)
		}
	}
	if v, err := r.Check("beta", StepLane, []string{"internal/core/implement/x.go"}); err != nil || !v.Allowed {
		t.Fatalf("second session, non-corpus lane: %+v %v", v, err)
	}
	before := len(eventNames(t, r))
	if _, err := r.Check("beta", StepRelease, nil); !errors.Is(err, ErrRefused) {
		t.Fatalf("second session, release = %v; want a refusal", err)
	}
	if e := lastEvent(t, r, EventRefusal); e.String("condition") != "second_session_release" || e.Session != "beta" {
		t.Fatalf("refusal line = %+v", e.Fields)
	}
	if _, err := r.Check("beta", StepLane, []string{"commands/capture.md"}); !errors.Is(err, ErrRefused) {
		t.Fatalf("second session, corpus lane = %v; want a refusal", err)
	}
	if after := len(eventNames(t, r)); after != before+2 {
		t.Fatalf("two refusals logged %d lines; allowed checks must write nothing", after-before)
	}
	if _, err := r.Check("beta", Step("ship"), nil); !errors.Is(err, ErrRefused) {
		t.Fatalf("unknown step = %v; want a refusal", err)
	}
	// In a split-roles window the second session opens no lane, whatever its
	// paths, and still reviews.
	if _, err := r.SetMode("alpha", ModeSplitRoles, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Check("beta", StepLane, []string{"internal/core/implement/x.go"}); !errors.Is(err, ErrRefused) {
		t.Fatalf("second session, lane in split-roles = %v; want a refusal", err)
	}
	if e := lastEvent(t, r, EventRefusal); e.String("condition") != "split_roles_second_builds_nothing" {
		t.Fatalf("refusal line = %+v", e.Fields)
	}
	if v, err := r.Check("beta", StepReview, nil); err != nil || !v.Allowed || v.Mode != ModeSplitRoles {
		t.Fatalf("second session, review in split-roles: %+v %v", v, err)
	}
}

// TestLogWritesTheRunsOwnEventsAndRefusesTheVerbs: a hand-kept event is written
// with typed fields; a verb-owned or unknown event, a reserved or malformed
// field, and an unjoined session are refused with nothing written.
func TestLogWritesTheRunsOwnEventsAndRefusesTheVerbs(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "beta", RoleSecond)
	e, err := r.Log("beta", EventBackoff, map[string]string{"on": "queue", "reason": "queue busy", "minutes": "12", "ratio": "0.5", "critical": "false"})
	if err != nil {
		t.Fatal(err)
	}
	if e.Event != EventBackoff {
		t.Fatalf("logged %+v", e)
	}
	got := lastEvent(t, r, EventBackoff)
	if string(got.Fields["minutes"]) != "12" || string(got.Fields["ratio"]) != "0.5" || string(got.Fields["critical"]) != "false" || got.String("on") != "queue" {
		t.Fatalf("fields = %+v", got.Fields)
	}
	before := len(eventNames(t, r))
	// A verb-owned event is refused for what it is, not merely as unknown.
	if _, err := r.Log("beta", EventClaimDenied, nil); err == nil || !strings.Contains(err.Error(), "never by hand") {
		t.Fatalf("hand-logged claim_denied = %v; want the verb-owned refusal", err)
	}
	for name, c := range map[string]struct {
		session, event string
		fields         map[string]string
	}{
		"verb-owned claim": {"beta", EventClaim, nil},
		"verb-owned mode":  {"beta", EventWindowMode, map[string]string{"mode": "single"}},
		"unknown event":    {"beta", "lane_opne", nil},
		"reserved field":   {"beta", EventStop, map[string]string{"session": "alpha"}},
		"bad field name":   {"beta", EventStop, map[string]string{"Bad-Key": "x"}},
		"oversized value":  {"beta", EventStop, map[string]string{"why": strings.Repeat("x", 2000)}},
		"unjoined":         {"gamma", EventStop, nil},
	} {
		if _, err := r.Log(c.session, c.event, c.fields); !errors.Is(err, ErrRefused) {
			t.Errorf("%s: %v; want a refusal", name, err)
		}
	}
	if after := len(eventNames(t, r)); after != before {
		t.Fatalf("refused log calls wrote %d lines", after-before)
	}
}

// TestPeekCreatesNothing: the read-only renders open a run that does not exist
// as empty, and leave no directory behind.
func TestPeekCreatesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	r, err := Peek(testSHA)
	if err != nil {
		t.Fatal(err)
	}
	ss, err := r.Sessions()
	if err != nil || len(ss) != 0 {
		t.Fatalf("sessions = %v, %v", ss, err)
	}
	cs, err := r.Claims()
	if err != nil || len(cs) != 0 {
		t.Fatalf("claims = %v, %v", cs, err)
	}
	if _, ok, err := r.CurrentMode(); ok || err != nil {
		t.Fatalf("mode on an empty run: %v %v", ok, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Peek created ~/.abcd: %v", err)
	}
	if _, err := Peek("not-a-sha"); !errors.Is(err, ErrRefused) {
		t.Fatalf("Peek of a malformed key = %v; want a refusal", err)
	}
}

// TestOpenRefusesASymlinkedRunsDirectory: the store is never created through a
// planted redirect.
func TestOpenRefusesASymlinkedRunsDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(home, ".abcd", "runs")); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(testSHA); err == nil {
		t.Fatal("Open through a symlinked runs directory succeeded")
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("Open wrote through the symlink: %v", entries)
	}
}
