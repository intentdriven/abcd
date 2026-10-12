package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The committed pre-push hook's reservation exemption (spc-2610031753359849
// step 4, itd-2609091034175565). A reservation is a chain of empty-tree commits
// on refs/heads/abcd-reserved/<record-id>, pushed by `abcd reserve` with the
// repository's hooks running and never with --no-verify. No preflight can vouch
// for a commit that changes no file, so the hook lets an update under that
// prefix through without a receipt, but only when every commit the push moves
// has the empty tree: ordinary history pushed under the prefix is gated as any
// other branch is, and a non-fast-forward is refused as on any other branch.

// reservedRef is the reservation branch the fixtures push to.
const reservedRef = "refs/heads/abcd-reserved/iss-2609020716570699"

// emptyTreeCommit writes an empty-tree commit with git commit-tree, as the
// reservation writer does: parent "" makes a parentless one.
func (c *prePushCase) emptyTreeCommit(parent, subject string) string {
	c.t.Helper()
	tree := c.git("hash-object", "-w", "-t", "tree", "/dev/null")
	return c.commitTree(tree, parent, subject)
}

// commitTree writes a commit of tree over parent with git commit-tree.
func (c *prePushCase) commitTree(tree, parent, subject string) string {
	c.t.Helper()
	args := []string{"commit-tree", tree}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	return c.git(append(args, "-m", subject)...)
}

// remoteRef is the commit the bare remote holds for a full ref name, or "".
func (c *prePushCase) remoteRef(ref string) string {
	c.t.Helper()
	out, err := c.run(c.origin, "git", "rev-parse", "--verify", "--quiet", ref)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// TestPrePushPassesEveryReservationActWithoutAReceipt is C24: a reserve (a
// parentless create), a renewal, a takeover and a release (each a child of the
// tip) all pass the committed hook with no preflight receipt and no
// --no-verify, and the hook never runs the preflight.
func TestPrePushPassesEveryReservationActWithoutAReceipt(t *testing.T) {
	c := newPrePushCase(t)
	tip := ""
	for _, act := range []string{"abcd reservation: iss-2609020716570699", "abcd reservation: renewed",
		"abcd reservation: taken over", "abcd reservation released: iss-2609020716570699"} {
		next := c.emptyTreeCommit(tip, act)
		if out, err := c.push("--porcelain", "origin", next+":"+reservedRef); err != nil {
			t.Fatalf("%q was refused by the pre-push hook although every commit it moves has the empty tree\n%s", act, out)
		}
		if got := c.remoteRef(reservedRef); got != next {
			t.Fatalf("after %q the remote holds %q, want %s", act, got, next)
		}
		tip = next
	}
	if _, err := os.Stat(filepath.Join(c.dir, ".abcd", ".work.local", "preflight-ran")); err == nil {
		t.Errorf("the hook ran the preflight for a reservation push")
	}
}

// TestPrePushGatesANonEmptyTreeUnderTheReservationPrefix is C25: a push under
// the prefix whose range carries a commit with a non-empty tree needs a
// receipt like any other push, on a create (the range walks into it) and on an
// update over a reservation tip.
func TestPrePushGatesANonEmptyTreeUnderTheReservationPrefix(t *testing.T) {
	t.Run("create carrying ordinary history", func(t *testing.T) {
		c := newPrePushCase(t)
		c.commit("feature.md", "a feature\n")
		out, err := c.push("origin", "HEAD:"+reservedRef)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("ordinary history pushed under the reservation prefix passed without a receipt (err %v)\n%s", err, out)
		}
		if got := c.remoteRef(reservedRef); got != "" {
			t.Errorf("the remote moved to %s although the push was refused", got)
		}
	})
	t.Run("create whose empty-tree tip sits on ordinary history", func(t *testing.T) {
		c := newPrePushCase(t)
		head := c.commit("feature.md", "a feature\n")
		over := c.emptyTreeCommit(head, "abcd reservation: iss-2609020716570699")
		out, err := c.push("origin", over+":"+reservedRef)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("an empty-tree tip over a non-empty commit passed without a receipt (err %v)\n%s", err, out)
		}
	})
	t.Run("update carrying a non-empty tree", func(t *testing.T) {
		c := newPrePushCase(t)
		first := c.emptyTreeCommit("", "abcd reservation: iss-2609020716570699")
		if out, err := c.push("origin", first+":"+reservedRef); err != nil {
			t.Fatalf("the reservation itself was refused\n%s", out)
		}
		tree := c.git("rev-parse", "HEAD^{tree}")
		child := c.commitTree(tree, first, "not a reservation")
		out, err := c.push("origin", child+":"+reservedRef)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("a non-empty tree pushed over a reservation passed without a receipt (err %v)\n%s", err, out)
		}
		if got := c.remoteRef(reservedRef); got != first {
			t.Errorf("the remote holds %q, want the reservation %s untouched", got, first)
		}
	})
}

// TestPrePushRefusesANonFastForwardReservationPush is C26: the exemption sits
// after the non-fast-forward guard, so a forced push that would replace a
// reservation tip with an unrelated chain is refused even though every commit
// in it has the empty tree.
func TestPrePushRefusesANonFastForwardReservationPush(t *testing.T) {
	c := newPrePushCase(t)
	first := c.emptyTreeCommit("", "abcd reservation: iss-2609020716570699")
	if out, err := c.push("origin", first+":"+reservedRef); err != nil {
		t.Fatalf("the reservation itself was refused\n%s", out)
	}
	rival := c.emptyTreeCommit("", "abcd reservation: a rival chain")
	out, err := c.push("origin", "+"+rival+":"+reservedRef)
	if err == nil || !strings.Contains(out, "non-fast-forward") {
		t.Fatalf("a forced non-fast-forward to a reservation branch passed the hook (err %v)\n%s", err, out)
	}
	if got := c.remoteRef(reservedRef); got != first {
		t.Errorf("the remote holds %q, want the first reservation %s", got, first)
	}
}

// TestPrePushRefusesDeletingAReservation: a deletion carries no commit, so the
// hook passed every one through, and `git push origin :<reservation>` erased
// another person's live reservation, after which a reserve meets an absent tip
// and takes the record with no lease judgement at all. A reservation ends with
// a tombstone pushed by `abcd reserve <id> --release`, never a deletion, so a
// delete of any ref under the prefix is refused, naming that path, in both of
// the spellings git sends the hook the same way.
func TestPrePushRefusesDeletingAReservation(t *testing.T) {
	for name, del := range map[string][]string{
		"colon refspec": {"origin", ":" + reservedRef},
		"--delete":      {"--delete", "origin", "abcd-reserved/iss-2609020716570699"},
	} {
		t.Run(name, func(t *testing.T) {
			c := newPrePushCase(t)
			first := c.emptyTreeCommit("", "abcd reservation: iss-2609020716570699")
			if out, err := c.push("origin", first+":"+reservedRef); err != nil {
				t.Fatalf("the reservation itself was refused\n%s", out)
			}
			out, err := c.push(del...)
			if err == nil {
				t.Fatalf("a deletion of a reservation passed the hook\n%s", out)
			}
			if !strings.Contains(out, "abcd reserve iss-2609020716570699 --release") {
				t.Errorf("the refusal does not name the release that ends a reservation\n%s", out)
			}
			if got := c.remoteRef(reservedRef); got != first {
				t.Errorf("the remote holds %q, want the reservation %s untouched", got, first)
			}
		})
	}
}

// TestPrePushStillPassesDeletingAnOrdinaryBranch pins the other side: only a
// receipt-less prefix refuses a deletion; an ordinary branch is deleted as
// before.
func TestPrePushStillPassesDeletingAnOrdinaryBranch(t *testing.T) {
	c := newPrePushCase(t)
	c.git("fetch", "-q", "origin")
	if out, err := c.push("origin", "origin/main:refs/heads/topic"); err != nil {
		t.Fatalf("a branch at a published commit was refused\n%s", out)
	}
	if out, err := c.push("origin", ":refs/heads/topic"); err != nil {
		t.Fatalf("deleting an ordinary branch was refused\n%s", out)
	}
	if got := c.remoteRef("refs/heads/topic"); got != "" {
		t.Errorf("the remote still holds topic at %s", got)
	}
}
