package cli

import (
	"os"
	"path/filepath"
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

// TestPrePushReadsPushedCommitsAsStored: a local replace ref that stands an
// empty-tree commit in for a commit carrying files does not make that commit
// pass the status row. The hook reads every commit as stored, as the push
// sends it, so it is gated, refused without a receipt, and origin is unmoved.
func TestPrePushReadsPushedCommitsAsStored(t *testing.T) {
	c := newPrePushCase(t)
	files := c.commit("feature.md", "a feature\n")
	c.git("replace", files, c.emptyTreeCommit("", "abcd announcement: merging #1012"))
	if tree := c.git("rev-parse", files+"^{tree}"); tree != c.git("hash-object", "-t", "tree", "/dev/null") {
		t.Fatalf("the replace ref does not stand in: %s reads as tree %s", files, tree)
	}
	out, err := c.push("origin", files+":"+statusRef)
	if err == nil || !strings.Contains(out, "no passing preflight") {
		t.Fatalf("a commit carrying files passed under the status prefix behind a replace ref (err %v)\n%s", err, out)
	}
	if got := c.remoteRef(statusRef); got != "" {
		t.Errorf("the remote moved to %s although the push was refused", got)
	}
}

// TestPrePushIgnoresAGraftFile: a .git/info/grafts line that gives an
// announcement no parents hides the commit carrying files beneath it from a
// history walk that honours grafts. The hook reads no graft file, so the walk
// meets that commit, the push is gated and refused without a receipt, and
// origin is unmoved.
func TestPrePushIgnoresAGraftFile(t *testing.T) {
	c := newPrePushCase(t)
	files := c.commit("feature.md", "a feature\n")
	empty := c.git("hash-object", "-w", "-t", "tree", "/dev/null")
	announcement := c.commitTree(empty, files, "abcd announcement: merging #1012")
	gitDir := c.git("rev-parse", "--absolute-git-dir")
	if err := os.WriteFile(filepath.Join(gitDir, "info", "grafts"), []byte(announcement+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if n := c.git("-c", "advice.graftFileDeprecated=false", "rev-list", "--count", announcement); n != "1" {
		t.Fatalf("the graft does not cut the history: %s commits reachable from the announcement", n)
	}
	out, err := c.push("origin", announcement+":"+statusRef)
	if err == nil || !strings.Contains(out, "no passing preflight") {
		t.Fatalf("a commit carrying files passed under the status prefix behind a graft (err %v)\n%s", err, out)
	}
	if got := c.remoteRef(statusRef); got != "" {
		t.Errorf("the remote moved to %s although the push was refused", got)
	}
}

// TestPrePushGatesAStatusPushFromAShallowClone: a .git/shallow line naming an
// announcement stops a history walk at it, hiding the commit carrying files
// beneath it, even with no graft file read. The range the hook walks lists
// that commit, whose parents it cannot read, so the status row does not pass
// it: the push is gated, refused without a receipt, and origin is unmoved.
// githook_prepush_shallow_test.go pins the same from a real shallow clone.
func TestPrePushGatesAStatusPushFromAShallowClone(t *testing.T) {
	c := newPrePushCase(t)
	files := c.commit("feature.md", "a feature\n")
	empty := c.git("hash-object", "-w", "-t", "tree", "/dev/null")
	announcement := c.commitTree(empty, files, "abcd announcement: merging #1012")
	gitDir := c.git("rev-parse", "--absolute-git-dir")
	if err := os.WriteFile(filepath.Join(gitDir, "shallow"), []byte(announcement+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c.env = append(c.env, "GIT_NO_REPLACE_OBJECTS=1", "GIT_GRAFT_FILE=/dev/null/none")
	n := c.git("rev-list", "--count", announcement)
	c.env = c.env[:len(c.env)-2]
	if n != "1" {
		t.Fatalf("the shallow file does not cut the history: %s commits reachable from the announcement", n)
	}
	out, err := c.push("origin", announcement+":"+statusRef)
	if err == nil || !strings.Contains(out, "no passing preflight") {
		t.Fatalf("a commit carrying files passed under the status prefix behind a shallow file (err %v)\n%s", err, out)
	}
	if got := c.remoteRef(statusRef); got != "" {
		t.Errorf("the remote moved to %s although the push was refused", got)
	}
}

// TestPrePushGatesAMergeThatBringsInFiles: a merge with the empty tree over an
// announcement, whose second parent carries files, moves those files' commits
// onto the branch, so it is gated as any push is.
func TestPrePushGatesAMergeThatBringsInFiles(t *testing.T) {
	c := newPrePushCase(t)
	first := c.emptyTreeCommit("", "abcd announcement: merging #1012")
	if out, err := c.push("origin", first+":"+statusRef); err != nil {
		t.Fatalf("the announcement itself was refused\n%s", out)
	}
	files := c.commit("feature.md", "a feature\n")
	empty := c.git("hash-object", "-w", "-t", "tree", "/dev/null")
	merge := c.git("commit-tree", empty, "-p", first, "-p", files, "-m", "abcd announcement: merged #1012")
	out, err := c.push("origin", merge+":"+statusRef)
	if err == nil || !strings.Contains(out, "no passing preflight") {
		t.Fatalf("a merge bringing in files passed under the status prefix without a receipt (err %v)\n%s", err, out)
	}
	if got := c.remoteRef(statusRef); got != first {
		t.Errorf("the remote holds %q, want the announcement %s untouched", got, first)
	}
}

// TestPrePushAnyRefMerelyLikeTheStatusPrefixNeedsAReceipt: only refs/heads/
// abcd-status/ is receipt-less. An empty-tree commit under a prefix that only
// begins like it, differs in case, or is a tag needs a receipt as any push does.
func TestPrePushAnyRefMerelyLikeTheStatusPrefixNeedsAReceipt(t *testing.T) {
	for _, ref := range []string{
		"refs/heads/abcd-statusX/octo",
		"refs/heads/ABCD-STATUS/octo",
		"refs/tags/abcd-status/octo",
	} {
		t.Run(ref, func(t *testing.T) {
			c := newPrePushCase(t)
			first := c.emptyTreeCommit("", "abcd announcement: merging #1012")
			out, err := c.push("origin", first+":"+ref)
			if err == nil || !strings.Contains(out, "no passing preflight") {
				t.Fatalf("an empty-tree commit at %s passed without a receipt (err %v)\n%s", ref, err, out)
			}
			if got := c.remoteRef(ref); got != "" {
				t.Errorf("the remote holds %s at %s", got, ref)
			}
		})
	}
}
