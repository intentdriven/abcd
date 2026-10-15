package implement

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// TestTheClaimBoundsAreTheSharedLeases holds the in-run claim's lease bounds to
// the shared lease package's: one set of numbers, spelled once.
func TestTheClaimBoundsAreTheSharedLeases(t *testing.T) {
	if DefaultLease != lease.Default || MinLease != lease.Min || MaxLease != lease.Max {
		t.Fatalf("DefaultLease, MinLease, MaxLease = %s, %s, %s; want lease.Default, Min, Max = %s, %s, %s",
			DefaultLease, MinLease, MaxLease, lease.Default, lease.Min, lease.Max)
	}
}

// TestClaimJudgesThroughTheSharedLease is C32: the in-run claim judges a claim
// file through lease.Judge, so a file whose expires_at runs thirty days past
// its claimed_at — which no claim verb writes, but a hand edit can — holds for
// a day and no longer, in the status and at the claim alike.
func TestClaimJudgesThroughTheSharedLease(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleFirst)
	now := c.now()
	body := `{"record":"iss-7","session":"alpha","lane":"fix","claimed_at":"` + now.Format(time.RFC3339) +
		`","expires_at":"` + now.Add(30*24*time.Hour).Format(time.RFC3339) + `"}` + "\n"
	if err := os.WriteFile(filepath.Join(r.Dir, claimsDirName, "iss-7.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c.advance(lease.Max - time.Second)
	if states, err := r.Claims(); err != nil || len(states) != 1 || !states[0].Live {
		t.Fatalf("a second before the day's clamp: claims = %+v, %v; want alpha's claim live", states, err)
	}
	c.advance(time.Second)
	states, err := r.Claims()
	if err != nil || len(states) != 1 || states[0].Live {
		t.Fatalf("at claimed_at plus a day: claims = %+v, %v; want alpha's claim lapsed", states, err)
	}
	res, err := r.Claim(ClaimRequest{Session: "beta", Record: "iss-7", Lane: "fix"})
	if err != nil || res.Lapsed == nil || res.Lapsed.Session != "alpha" {
		t.Fatalf("beta's claim at the clamp = %+v, %v; want it granted over alpha's lapsed claim", res, err)
	}
}

// TestAClaimFileWithoutAClaimTimeIsUnreadable: the clamp is counted from
// claimed_at, so a file that drops it cannot be judged and reads as unreadable,
// held only for the grace, rather than as a claim with no ceiling.
func TestAClaimFileWithoutAClaimTimeIsUnreadable(t *testing.T) {
	r, c := newRun(t)
	now := c.now()
	path := filepath.Join(r.Dir, claimsDirName, "iss-8.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"record":"iss-8","session":"alpha","lane":"fix","expires_at":"` + now.Add(time.Hour).Format(time.RFC3339) + `"}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatal(err)
	}
	states, err := r.Claims()
	if err != nil || len(states) != 1 || !states[0].Unreadable || !states[0].ExpiresAt.Equal(now.Add(UnreadableClaimGrace)) {
		t.Fatalf("claims = %+v, %v; want one unreadable claim held until %s", states, err, now.Add(UnreadableClaimGrace))
	}
}
