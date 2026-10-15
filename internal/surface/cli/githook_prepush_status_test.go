package cli

import (
	"strings"
	"testing"
)

// The committed pre-push hook's status row (itd-2609151838312703 C13 and C14,
// adr-2610150729333337). An announcement is an empty-tree commit on
// refs/heads/abcd-status/<username>, pushed by `abcd announce` with the hooks
// running. A session with no preflight receipt, a hosted one among them,
// pushes it through the reservation's own shape check: a fast-forward of
// commits that change no file. Every other push under the prefix is gated as
// any branch is, and a status branch is never deleted.

// statusRef is the status branch the fixtures push to.
const statusRef = "refs/heads/abcd-status/octo"

// TestPrePushPassesAnnouncementsWithoutAReceipt is C13: a person's first
// announcement (a parentless create) and the next (a child of the tip) pass
// the committed hook with no preflight receipt and no --no-verify.
func TestPrePushPassesAnnouncementsWithoutAReceipt(t *testing.T) {
	c := newPrePushCase(t)
	tip := ""
	for _, subject := range []string{"abcd announcement: merging #1012", "abcd announcement: merged #1012"} {
		next := c.emptyTreeCommit(tip, subject)
		if out, err := c.push("--porcelain", "origin", next+":"+statusRef); err != nil {
			t.Fatalf("%q was refused by the pre-push hook although it changes no file\n%s", subject, out)
		}
		if got := c.remoteRef(statusRef); got != next {
			t.Fatalf("after %q the remote holds %q, want %s", subject, got, next)
		}
		tip = next
	}
}

// TestPrePushGatesEveryOtherPushUnderTheStatusPrefix is C14: a push under the
// prefix that changes a file, on a create or over an announcement, needs a
// receipt like any push; one that rewrites the branch is refused as a
// non-fast-forward; and a deletion is refused, naming the ruling.
func TestPrePushGatesEveryOtherPushUnderTheStatusPrefix(t *testing.T) {
	t.Run("create that changes a file", func(t *testing.T) {
		c := newPrePushCase(t)
		c.commit("feature.md", "a feature\n")
		out, err := c.push("origin", "HEAD:"+statusRef)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("a commit that changes a file passed under the status prefix without a receipt (err %v)\n%s", err, out)
		}
		if got := c.remoteRef(statusRef); got != "" {
			t.Errorf("the remote moved to %s although the push was refused", got)
		}
	})
	t.Run("update that changes a file", func(t *testing.T) {
		c := newPrePushCase(t)
		first := c.emptyTreeCommit("", "abcd announcement: merging #1012")
		if out, err := c.push("origin", first+":"+statusRef); err != nil {
			t.Fatalf("the announcement itself was refused\n%s", out)
		}
		tree := c.git("rev-parse", "HEAD^{tree}")
		child := c.commitTree(tree, first, "not an announcement")
		out, err := c.push("origin", child+":"+statusRef)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("a file change over an announcement passed without a receipt (err %v)\n%s", err, out)
		}
		if got := c.remoteRef(statusRef); got != first {
			t.Errorf("the remote holds %q, want the announcement %s untouched", got, first)
		}
	})
	t.Run("rewrite", func(t *testing.T) {
		c := newPrePushCase(t)
		first := c.emptyTreeCommit("", "abcd announcement: merging #1012")
		if out, err := c.push("origin", first+":"+statusRef); err != nil {
			t.Fatalf("the announcement itself was refused\n%s", out)
		}
		rival := c.emptyTreeCommit("", "abcd announcement: a rewritten history")
		out, err := c.push("origin", "+"+rival+":"+statusRef)
		if err == nil || !strings.Contains(out, "non-fast-forward") {
			t.Fatalf("a forced rewrite of a status branch passed the hook (err %v)\n%s", err, out)
		}
		if got := c.remoteRef(statusRef); got != first {
			t.Errorf("the remote holds %q, want %s", got, first)
		}
	})
	for name, del := range map[string][]string{
		"delete by colon refspec": {"origin", ":" + statusRef},
		"delete by --delete":      {"--delete", "origin", "abcd-status/octo"},
	} {
		t.Run(name, func(t *testing.T) {
			c := newPrePushCase(t)
			first := c.emptyTreeCommit("", "abcd announcement: merging #1012")
			if out, err := c.push("origin", first+":"+statusRef); err != nil {
				t.Fatalf("the announcement itself was refused\n%s", out)
			}
			out, err := c.push(del...)
			if err == nil || !strings.Contains(out, "is kept for ever and never deleted") || !strings.Contains(out, "adr-2610150729333337") {
				t.Fatalf("a deletion of a status branch was not refused as one (err %v)\n%s", err, out)
			}
			if got := c.remoteRef(statusRef); got != first {
				t.Errorf("the remote holds %q, want %s", got, first)
			}
		})
	}
}
