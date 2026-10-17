package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestALandingAReleaseHoldsExitsThree: a landing held by a release being cut
// (itd-2609151838312703 C7) is contention, exit 3, as every verb that backs
// off exits; any other refusal stays exit 2.
func TestALandingAReleaseHoldsExitsThree(t *testing.T) {
	for _, c := range []struct {
		held bool
		code int
	}{{true, 3}, {false, 2}} {
		err := landRefusal(landing.Result{}, &landing.Refusal{Reason: "release v0.16.0 is being cut by quinn", Fix: "run abcd land again", Held: c.held})
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != c.code || !strings.Contains(ee.Msg, "release v0.16.0 is being cut by quinn") {
			t.Fatalf("held %v: %v; want exit %d naming the hold", c.held, err, c.code)
		}
	}
}

// landHoldWorld is landWorld with a forge that answers the origin check and
// lists no merged pull request, for a landing run with --no-enqueue to reach
// the release hold and the push.
func landHoldWorld(t *testing.T) (dir, origin string) {
	t.Helper()
	dir, origin = landWorld(t)
	landSeams.Forge = func(_ string, args ...string) (string, error) {
		if len(args) > 1 && args[0] == "repo" && args[1] == "view" {
			return `{"nameWithOwner":"o/r","url":"` + origin + `","sshUrl":"","defaultBranchRef":{"name":"main"}}`, nil
		}
		if slices.Contains(args, "--state=merged") {
			return "[]", nil
		}
		return "", errors.New("no other forge call in this test")
	}
	return dir, origin
}

// pushDatedRelease pushes quinn's release reservation for subject onto
// origin, claimed and committed at at, from a scratch clone.
func pushDatedRelease(t *testing.T, origin, subject string, at time.Time) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "ahead")
	cmd := exec.Command("git", "clone", "-q", origin, dir)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clone: %v\n%s", err, out)
	}
	msg := filepath.Join(t.TempDir(), "msg")
	text := "abcd reservation: " + subject + "\n\nAbcd-Reservation: " + subject + "\nAbcd-Holder: quinn\n" +
		"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Claimed-At: " + at.UTC().Format(time.RFC3339) +
		"\nAbcd-Expires-At: " + at.UTC().Add(24*time.Hour).Format(time.RFC3339) + "\n"
	if err := os.WriteFile(msg, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := gitCmd(t, dir, "hash-object", "-w", "-t", "tree", "/dev/null")
	date := "@" + strconv.FormatInt(at.Unix(), 10) + " +0000"
	cmd = exec.Command("git", "-C", dir, "-c", "user.name=Quinn", "-c", "user.email=quinn@example.invalid", "-c", "commit.gpgsign=false",
		"commit-tree", tree, "-F", msg)
	cmd.Env = append(gittest.Env(t), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "push", "-q", "origin", strings.TrimSpace(string(out))+":"+claim.RefPrefix+subject)
}

// TestLandWithNoLocalTierNamesAReleaseDatedAheadAndLands: the checkout has no
// local tier (its gate is skipped on a receipt another worktree minted), so it
// cannot keep when it first saw a release reservation. One dated five years
// ahead is not counted: land exits 0, pushes, and prints one notice line
// naming the release. One dated a minute ahead holds it: exit 3.
func TestLandWithNoLocalTierNamesAReleaseDatedAheadAndLands(t *testing.T) {
	for _, c := range []struct {
		name string
		at   time.Duration
		code int
	}{{"dated five years ahead", 5 * 365 * 24 * time.Hour, 0}, {"dated a minute ahead", time.Minute, 3}} {
		t.Run(c.name, func(t *testing.T) {
			dir, origin := landHoldWorld(t)
			head := gitCmd(t, dir, "rev-parse", "HEAD")
			other := filepath.Join(t.TempDir(), "other")
			gitCmd(t, dir, "worktree", "add", "-q", "-b", "side", other, "main")
			writeRel(t, other, landing.ReceiptsRelDir+"/"+head, "")
			pushDatedRelease(t, origin, "rel-v0.16.0", time.Now().UTC().Truncate(time.Second).Add(c.at))
			code, _, stderr := reserveRun(t, dir, "land", "--no-enqueue")
			notice := "abcd land: release v0.16.0 is reserved by quinn with a date more than 5 minutes ahead of this clock, and this checkout cannot keep when it first saw it"
			pushed := gitCmd(t, origin, "for-each-ref", "--format=%(objectname)", "refs/heads/feat/x")
			switch {
			case code != c.code:
				t.Fatalf("exit %d, stderr %q; want exit %d", code, stderr, c.code)
			case c.code == 0 && (strings.Count(stderr, notice) != 1 || pushed != head):
				t.Fatalf("stderr %q, origin feat/x %q; want one notice line naming the release and %s pushed", stderr, pushed, head)
			case c.code == 3 && (strings.Contains(stderr, notice) || pushed != "" || !strings.Contains(stderr, "being cut by quinn")):
				t.Fatalf("stderr %q, origin feat/x %q; want quinn's hold named and nothing pushed", stderr, pushed)
			}
			if _, err := os.Stat(filepath.Join(dir, ".abcd", ".work.local")); err == nil {
				t.Fatal("land made the checkout's local tier")
			}
		})
	}
}

// TestLandExitsTwoWhenTheReservationsCannotBeRead: origin stops answering
// after the gate, so no release hold can be judged and nothing is pushed.
// Nobody is known to hold the release, so it is a refusal, exit 2, not the
// back-off of exit 3 that waits on a holder.
func TestLandExitsTwoWhenTheReservationsCannotBeRead(t *testing.T) {
	dir, origin := landHoldWorld(t)
	gate := landSeams.Gate
	landSeams.Gate = func(d string) error {
		if err := gate(d); err != nil {
			return err
		}
		return os.Rename(origin, origin+".gone")
	}
	code, _, stderr := reserveRun(t, dir, "land", "--no-enqueue")
	if code != 2 || !strings.Contains(stderr, "could not read the reservations from origin") ||
		!strings.Contains(stderr, "run abcd land again once origin answers") {
		t.Fatalf("exit %d, stderr %q: want exit 2 naming the failed read", code, stderr)
	}
}
