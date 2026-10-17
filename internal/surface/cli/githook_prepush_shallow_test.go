package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The committed pre-push hook's receipt-less rows pushed from a shallow clone
// (iss-2610171249310666). A hosted cloud session starts from one, so `abcd
// announce` and `abcd reserve` push their empty-tree commits from it. The
// shape check reads the commits the push adds and proves each has the empty
// tree; a .git/shallow line hides only the history beneath the commit it
// names, so the check refuses a range only when one of the commits it lists
// is such a commit, whose parents it cannot read. Every refusal the hook makes
// from a full clone it still makes from a shallow one.

// shallowClone clones origin's main at depth one into a new working clone,
// with the committed hook running, as a hosted session's setup does. It first
// puts a second commit on origin's main, so the clone is shallow.
func (c *prePushCase) shallowClone() *prePushCase {
	c.t.Helper()
	c.git("checkout", "-q", "main")
	c.commit("second.md", "second\n")
	c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "main")
	c.git("checkout", "-q", "feature")

	s := &prePushCase{t: c.t, dir: filepath.Join(filepath.Dir(c.dir), "shallow"), origin: c.origin, env: c.env}
	if out, err := c.run(filepath.Dir(c.dir), "git", "clone", "-q", "--depth", "1", "--branch", "main", "file://"+c.origin, s.dir); err != nil {
		c.t.Fatalf("git clone --depth 1: %v\n%s", err, out)
	}
	s.git("config", "user.name", "Alice Example")
	s.git("config", "user.email", "alice@example.com")
	s.git("config", "core.hooksPath", ".githooks")
	s.mustBeShallow()
	return s
}

// mustBeShallow fails the test unless the clone is still shallow.
func (c *prePushCase) mustBeShallow() {
	c.t.Helper()
	if got := c.git("rev-parse", "--is-shallow-repository"); got != "true" {
		c.t.Fatalf("the clone is not shallow (--is-shallow-repository %q)", got)
	}
}

// fetchRef fetches one of origin's refs into its remote-tracking ref, as the
// announce and reserve verbs fetch their namespaces, and returns its tip.
func (c *prePushCase) fetchRef(ref string) string {
	c.t.Helper()
	tracking := "refs/remotes/origin/" + strings.TrimPrefix(ref, "refs/heads/")
	c.git("fetch", "-q", "--no-tags", "origin", "+"+ref+":"+tracking)
	c.mustBeShallow()
	return c.git("rev-parse", tracking)
}

// TestPrePushPassesAnnouncementsFromAShallowClone: from a shallow clone, a
// person's first announcement (a parentless create), the next (a child of the
// tip), and an announcement over a status branch origin already held and the
// clone fetched, each pass with no preflight receipt, and the clone stays
// shallow: the hook deepens nothing.
func TestPrePushPassesAnnouncementsFromAShallowClone(t *testing.T) {
	t.Run("create, then update", func(t *testing.T) {
		s := newPrePushCase(t).shallowClone()
		tip := ""
		for _, subject := range []string{"abcd announcement: merging #1012", "abcd announcement: merged #1012"} {
			next := s.emptyTreeCommit(tip, subject)
			if out, err := s.push("--porcelain", "origin", next+":"+statusRef); err != nil {
				t.Fatalf("%q was refused from a shallow clone although it changes no file\n%s", subject, out)
			}
			if got := s.remoteRef(statusRef); got != next {
				t.Fatalf("after %q the remote holds %q, want %s", subject, got, next)
			}
			tip = next
		}
		s.mustBeShallow()
	})
	t.Run("update over a fetched status branch", func(t *testing.T) {
		c := newPrePushCase(t)
		first := c.emptyTreeCommit("", "abcd announcement: merging #1012")
		if out, err := c.push("origin", first+":"+statusRef); err != nil {
			t.Fatalf("the first announcement was refused from the full clone\n%s", out)
		}
		s := c.shallowClone()
		if got := s.fetchRef(statusRef); got != first {
			t.Fatalf("the fetched status branch is at %s, want %s", got, first)
		}
		next := s.emptyTreeCommit(first, "abcd announcement: merged #1012")
		if out, err := s.push("--porcelain", "origin", next+":"+statusRef); err != nil {
			t.Fatalf("an announcement over a fetched status branch was refused from a shallow clone\n%s", out)
		}
		if got := s.remoteRef(statusRef); got != next {
			t.Fatalf("the remote holds %q, want %s", got, next)
		}
		s.mustBeShallow()
	})
}

// TestPrePushPassesReservationsFromAShallowClone: from a shallow clone, a
// reserve (a parentless create), a renewal and a release (each a child of the
// tip) pass with no preflight receipt.
func TestPrePushPassesReservationsFromAShallowClone(t *testing.T) {
	s := newPrePushCase(t).shallowClone()
	tip := ""
	for _, act := range []string{"abcd reservation: iss-2609020716570699", "abcd reservation: renewed",
		"abcd reservation released: iss-2609020716570699"} {
		next := s.emptyTreeCommit(tip, act)
		if out, err := s.push("--porcelain", "origin", next+":"+reservedRef); err != nil {
			t.Fatalf("%q was refused from a shallow clone although it changes no file\n%s", act, out)
		}
		if got := s.remoteRef(reservedRef); got != next {
			t.Fatalf("after %q the remote holds %q, want %s", act, got, next)
		}
		tip = next
	}
	s.mustBeShallow()
}

// TestPrePushStaysStrictFromAShallowClone: every push under the status prefix
// the hook gates from a full clone it gates from a shallow one. A commit that
// changes a file needs a receipt, on a create and over an announcement; a
// rewrite is refused as a non-fast-forward; and a range whose commits cannot
// all be read, because the walk reaches a .git/shallow commit whose parents
// this clone never fetched, needs a receipt even when every commit it can read
// has the empty tree.
func TestPrePushStaysStrictFromAShallowClone(t *testing.T) {
	t.Run("create that changes a file", func(t *testing.T) {
		s := newPrePushCase(t).shallowClone()
		s.commit("feature.md", "a feature\n")
		out, err := s.push("origin", "HEAD:"+statusRef)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("a commit that changes a file passed under the status prefix from a shallow clone (err %v)\n%s", err, out)
		}
		if got := s.remoteRef(statusRef); got != "" {
			t.Errorf("the remote moved to %s although the push was refused", got)
		}
	})
	t.Run("update that changes a file", func(t *testing.T) {
		s := newPrePushCase(t).shallowClone()
		first := s.emptyTreeCommit("", "abcd announcement: merging #1012")
		if out, err := s.push("origin", first+":"+statusRef); err != nil {
			t.Fatalf("the announcement itself was refused\n%s", out)
		}
		child := s.commitTree(s.git("rev-parse", "HEAD^{tree}"), first, "not an announcement")
		out, err := s.push("origin", child+":"+statusRef)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("a file change over an announcement passed from a shallow clone (err %v)\n%s", err, out)
		}
		if got := s.remoteRef(statusRef); got != first {
			t.Errorf("the remote holds %q, want the announcement %s untouched", got, first)
		}
	})
	t.Run("empty-tree commit over the shallow tip", func(t *testing.T) {
		s := newPrePushCase(t).shallowClone()
		over := s.emptyTreeCommit(s.git("rev-parse", "HEAD"), "abcd announcement: merging #1012")
		out, err := s.push("origin", over+":"+statusRef)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("an announcement over the default branch's history passed from a shallow clone (err %v)\n%s", err, out)
		}
		if got := s.remoteRef(statusRef); got != "" {
			t.Errorf("the remote moved to %s although the push was refused", got)
		}
	})
	t.Run("rewrite", func(t *testing.T) {
		c := newPrePushCase(t)
		first := c.emptyTreeCommit("", "abcd announcement: merging #1012")
		if out, err := c.push("origin", first+":"+statusRef); err != nil {
			t.Fatalf("the first announcement was refused from the full clone\n%s", out)
		}
		s := c.shallowClone()
		s.fetchRef(statusRef)
		rival := s.emptyTreeCommit("", "abcd announcement: a rewritten history")
		out, err := s.push("origin", "+"+rival+":"+statusRef)
		if err == nil || !strings.Contains(out, "non-fast-forward") {
			t.Fatalf("a forced rewrite of a status branch passed from a shallow clone (err %v)\n%s", err, out)
		}
		if got := s.remoteRef(statusRef); got != first {
			t.Errorf("the remote holds %q, want %s", got, first)
		}
	})
	t.Run("range reaching past the shallow line", func(t *testing.T) {
		// origin holds files under an empty-tree commit on another branch;
		// fetched at depth one, that commit is the shallow line and the commit
		// carrying files beneath it is never fetched. A new announcement over
		// it lists only empty-tree commits, but the walk cannot read past the
		// line, so the push needs a receipt.
		c := newPrePushCase(t)
		files := c.commit("feature.md", "a feature\n")
		cover := c.emptyTreeCommit(files, "abcd announcement: merging #1012")
		c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", cover+":refs/heads/cover")
		s := c.shallowClone()
		s.git("fetch", "-q", "--no-tags", "--depth", "1", "origin", "+refs/heads/cover:refs/remotes/origin/cover")
		if n := s.git("rev-list", "--count", cover); n != "1" {
			t.Fatalf("the shallow line does not cut the history: %s commits reachable from the cover", n)
		}
		if _, err := s.run(s.dir, "git", "cat-file", "-e", files); err == nil {
			t.Fatalf("the commit carrying files was fetched into the shallow clone")
		}
		next := s.emptyTreeCommit(cover, "abcd announcement: merged #1012")
		out, err := s.push("origin", next+":"+statusRef)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("a range reaching past the shallow line passed under the status prefix (err %v)\n%s", err, out)
		}
		if got := s.remoteRef(statusRef); got != "" {
			t.Errorf("the remote moved to %s although the push was refused", got)
		}
	})
}

// TestPrePushReadsAShallowLineAsGitDoes: git reads a .git/shallow line as its
// first object id's worth of hex, in either case, and ignores what follows, so
// the hook reads the line the same way. A shallow line written in upper case,
// with a carriage return, or with trailing text still cuts git's walk at the
// announcement, and the commit carrying files beneath it is still gated.
func TestPrePushReadsAShallowLineAsGitDoes(t *testing.T) {
	for name, line := range map[string]func(string) string{
		"upper case":      strings.ToUpper,
		"carriage return": func(id string) string { return id + "\r" },
		"trailing text":   func(id string) string { return id + " written by hand" },
	} {
		t.Run(name, func(t *testing.T) {
			c := newPrePushCase(t)
			files := c.commit("feature.md", "a feature\n")
			empty := c.git("hash-object", "-w", "-t", "tree", "/dev/null")
			announcement := c.commitTree(empty, files, "abcd announcement: merging #1012")
			gitDir := c.git("rev-parse", "--absolute-git-dir")
			if err := os.WriteFile(filepath.Join(gitDir, "shallow"), []byte(line(announcement)+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			c.env = append(c.env, "GIT_NO_REPLACE_OBJECTS=1", "GIT_GRAFT_FILE=/dev/null/none")
			n := c.git("rev-list", "--count", announcement)
			c.env = c.env[:len(c.env)-2]
			if n != "1" {
				t.Fatalf("git does not read the %s line as shallow: %s commits reachable from the announcement", name, n)
			}
			out, err := c.push("origin", announcement+":"+statusRef)
			if err == nil || !strings.Contains(out, "no passing preflight") {
				t.Fatalf("a commit carrying files passed under the status prefix behind a %s shallow line (err %v)\n%s", name, err, out)
			}
			if got := c.remoteRef(statusRef); got != "" {
				t.Errorf("the remote moved to %s although the push was refused", got)
			}
		})
	}
}
