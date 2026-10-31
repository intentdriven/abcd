package claim

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Snapshot.Contention, CopyKey and OwnCopyKey (spc-2610161044461448 step 2):
// what a drain reads to pass over held work with no fetch and no push, and to
// draw its own order from this copy's key.

// TestContentionNamesWhatHoldsARecordOffInTheView: a record another person
// reserves live, one this person's other copy reserves live, and one with an
// unreadable reservation within its hold each hold it off, naming why; a
// footprint path shared with another person's live reservation of another
// record holds it off naming that record; this copy's own reservations, an
// unreadable one past its hold and an expired one hold nothing off.
func TestContentionNamesWhatHoldsARecordOffInTheView(t *testing.T) {
	now := t0
	until := now.Add(time.Hour)
	s := Snapshot{FetchedAt: now.Add(-time.Minute), Reservations: []Reservation{
		{Record: "iss-1", Holder: "quinn", Footprint: []string{"a.go"}, EffectiveExpiry: until, State: StateLive},
		{Record: "iss-2", Holder: "octo", Footprint: []string{"b.go"}, EffectiveExpiry: until, State: StateLive},
		{Record: "iss-3", Unreadable: "no trailers", EffectiveExpiry: until, State: StateUnreadable},
		{Record: "iss-4", Unreadable: "no trailers", EffectiveExpiry: now.Add(-time.Minute), State: StateUnreadable},
		{Record: "iss-5", Holder: "octo", Copy: CopyThis, Footprint: []string{"internal/x"}, EffectiveExpiry: until, State: StateLive},
		{Record: "iss-6", Holder: "quinn", Footprint: []string{"internal/y"}, EffectiveExpiry: now.Add(-time.Hour), State: StateExpired},
		{Record: "iss-7", Holder: "quinn", Footprint: []string{"internal/z"}, EffectiveExpiry: until, State: StateLive},
	}}
	for _, tc := range []struct {
		id        string
		footprint []string
		want      string
	}{
		{"iss-1", []string{"c.go"}, "iss-1 is reserved by quinn until"},
		{"iss-2", []string{"c.go"}, "iss-2 is reserved by octo until"},
		{"iss-3", []string{"c.go"}, "iss-3 has an unreadable reservation holding it until"},
		{"iss-4", []string{"c.go"}, ""},
		{"iss-5", []string{"internal/x/a.go"}, ""},
		{"iss-6", []string{"internal/y/a.go"}, ""},
		{"iss-8", []string{"internal/z/a.go"}, `"internal/z/a.go" is in the footprint of iss-7, reserved by quinn until`},
		{"iss-8", []string{"internal"}, `"internal/z" is in the footprint of iss-7, reserved by quinn until`},
		{"iss-8", []string{"internal/x/a.go", "internal/y/b.go"}, ""},
	} {
		got := s.Contention(tc.id, tc.footprint, now, Self{})
		if tc.want == "" && got != "" || tc.want != "" && !strings.HasPrefix(got, tc.want) {
			t.Errorf("Contention(%s, %v) = %q; want %q", tc.id, tc.footprint, got, tc.want)
		}
		if got != "" && !strings.Contains(got, "as of") {
			t.Errorf("Contention(%s, %v) = %q; want the view's age", tc.id, tc.footprint, got)
		}
	}
}

// TestOwnCopyKeyReadsTheKeyAndNeverMintsOne: with no key kept, OwnCopyKey is
// "" and writes nothing; CopyKey mints one, and both read it back after; a
// kept file that is not a key reads as none.
func TestOwnCopyKeyReadsTheKeyAndNeverMintsOne(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(root, filepath.FromSlash(copyKeyRel))
	if got := OwnCopyKey(root); got != "" {
		t.Fatalf("OwnCopyKey with no key kept = %q", got)
	}
	if _, err := os.Stat(filepath.Dir(key)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("OwnCopyKey wrote under the local tier: %v", err)
	}
	minted, err := CopyKey(root)
	if err != nil || !validCopyKey(minted) {
		t.Fatalf("CopyKey = %q, %v; want a minted key", minted, err)
	}
	if again, err := CopyKey(root); err != nil || again != minted {
		t.Fatalf("CopyKey again = %q, %v; want %q", again, err, minted)
	}
	if got := OwnCopyKey(root); got != minted {
		t.Fatalf("OwnCopyKey = %q; want the minted %q", got, minted)
	}
	if err := os.WriteFile(key, []byte("not a key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := OwnCopyKey(root); got != "" {
		t.Fatalf("OwnCopyKey over a file that is not a key = %q", got)
	}
	if _, err := CopyKey(t.TempDir()); err == nil {
		t.Fatal("CopyKey in a checkout with no local tier minted a key")
	}
}
