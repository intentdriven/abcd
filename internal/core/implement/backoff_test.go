package implement

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestALockedRunStateIsABackoffWithItsReasonAndMinutes: contention of any kind
// the second session meets is a backoff the log names, with the reason and the
// minutes spent (itd-2609221656373558 criterion 6, iss-2609231206196407). A run
// state locked by another session's change is contention the verb itself sees,
// so the verb writes the line, with the minutes it waited for the lock.
func TestALockedRunStateIsABackoffWithItsReasonAndMinutes(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleSecond)

	held := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- fsutil.WithFileLock(filepath.Join(r.Dir, lockFileName), lockTimeout, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	_, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-1", Lane: "one"})
	close(release)
	if lerr := <-done; lerr != nil {
		t.Fatalf("holding the lock: %v", lerr)
	}
	if !errors.Is(err, ErrContention) {
		t.Fatalf("a claim against a locked run state = %v; want contention", err)
	}
	b := lastEvent(t, r, EventBackoff)
	if b.Session != "beta" || b.String("on") != "run_state" || b.String("reason") == "" {
		t.Fatalf("backoff line = %+v; want the second session's run_state backoff naming its reason", b.Fields)
	}
	if m, ok := b.Number("minutes"); !ok || m <= 0 {
		t.Fatalf("backoff minutes = %v (present %v); want the measured wait for the lock", m, ok)
	}
}

// TestAHandLoggedBackoffNamesItsReasonAndMinutes: contention the verb cannot
// see (the merge queue, a peer's worktree) is logged on the session's word, and
// that line must name the reason and the minutes too, or it is refused with
// nothing written (iss-2609231206196407).
func TestAHandLoggedBackoffNamesItsReasonAndMinutes(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "beta", RoleSecond)
	for _, fields := range []map[string]string{
		{"on": "queue", "minutes": "4"},
		{"on": "queue", "reason": "queue busy"},
		{"on": "queue", "reason": "queue busy", "minutes": "a while"},
		{"on": "queue", "reason": "queue busy", "minutes": "-1"},
	} {
		if _, err := r.Log("beta", EventBackoff, fields); !errors.Is(err, ErrRefused) {
			t.Errorf("backoff %v = %v; want refused", fields, err)
		}
	}
	for _, n := range eventNames(t, r) {
		if n == EventBackoff {
			t.Fatal("a refused backoff was written")
		}
	}
	if _, err := r.Log("beta", EventBackoff, map[string]string{"on": "queue", "reason": "queue busy", "minutes": "4"}); err != nil {
		t.Fatalf("a backoff naming its reason and minutes: %v", err)
	}
}
