package implement

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAHandAppendedLogFileIsNotFollowedThroughASymlink: the run log's leaf is
// never appended through a symlink, so a log symlinked onto a claim file
// cannot make the claim unreadable (iss-2609230720193756).
func TestAHandAppendedLogFileIsNotFollowedThroughASymlink(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	if _, err := r.Claim(ClaimRequest{Session: "alpha", Record: "itd-7", Lane: "l7"}); err != nil {
		t.Fatal(err)
	}
	claimPath := filepath.Join(r.Dir, "claims", "itd-7.json")
	before, err := os.ReadFile(claimPath)
	if err != nil {
		t.Fatal(err)
	}
	day := filepath.Join(r.Dir, logFileName(r.now()))
	if err := os.Remove(day); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("claims", "itd-7.json"), day); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Log("alpha", EventLaneOpen, map[string]string{"lane": "l7"}); err == nil {
		t.Fatal("appending through a symlinked log leaf succeeded")
	}
	after, err := os.ReadFile(claimPath)
	if err != nil || string(after) != string(before) {
		t.Fatalf("the claim changed under a symlinked log: %q, %v", after, err)
	}
}

// TestAClaimNamingAnInvalidSessionOrLaneIsUnreadable: a hand-edited claim whose
// session or lane is not a name — terminal escapes, a path — reads as an
// unreadable claim, so it never reaches a refusal message printed to the
// operator's screen (iss-2609230720193756).
func TestAClaimNamingAnInvalidSessionOrLaneIsUnreadable(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	for name, c := range map[string]Claim{
		"escape in session": {Record: "itd-8", Session: "evil\x1b[2J", Lane: "l8"},
		"escape in lane":    {Record: "itd-8", Session: "beta", Lane: "l8\x1b]0;x\x07"},
		"path as lane":      {Record: "itd-8", Session: "beta", Lane: "../../x"},
	} {
		c.ClaimedAt = r.now()
		c.ExpiresAt = r.now().Add(time.Hour)
		data, err := encodeClaim(c)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(r.Dir, "claims", "itd-8.json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		root, err := r.root()
		if err != nil {
			t.Fatal(err)
		}
		_, err = r.readClaim(root, "itd-8")
		root.Close()
		var bad *UnreadableClaimError
		if !errors.As(err, &bad) {
			t.Errorf("%s: readClaim = %v; want an unreadable claim", name, err)
		}
		if _, err := r.Release("alpha", "itd-8"); err == nil || strings.ContainsAny(err.Error(), "\x1b\x07") {
			t.Errorf("%s: release = %v; want a refusal carrying no escape", name, err)
		}
	}
}

// TestTheRunLockIsTheOwnersAlone: the run's lock file is created 0600 like
// every other file in the run state (iss-2609230720193756).
func TestTheRunLockIsTheOwnersAlone(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	fi, err := os.Stat(filepath.Join(r.Dir, lockFileName))
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != fileMode {
		t.Fatalf(".lock mode = %o, want %o", perm, fileMode)
	}
}
