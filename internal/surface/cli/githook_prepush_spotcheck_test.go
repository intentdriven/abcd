package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The committed pre-push hook's spot-check row (itd-2610301750162657,
// adr-2611251806496719): refs/heads/abcd-spotcheck, the spot-check's store, is
// an orphan of small JSON files under the store's folders, pushed by abcd
// with the hooks running and no receipt. The row admits a fast-forward whose
// every commit holds only regular files at the store's paths; every other
// push to the branch is gated as any branch is, a branch whose name merely
// begins like it is no branch of the row, and the store is never deleted.

const spotcheckRef = "refs/heads/abcd-spotcheck"

// storeFile is one file of a store commit: its path, content and mode.
type storeFile struct {
	path, body, mode string
}

// storeCommit writes a commit over parent ("" for none) whose tree holds
// files alone, made in an index of its own, as the store writes one.
func (c *prePushCase) storeCommit(parent string, files ...storeFile) string {
	c.t.Helper()
	index := filepath.Join(c.t.TempDir(), "index")
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = c.dir
		cmd.Env = append(append([]string{}, c.env...), "GIT_INDEX_FILE="+index)
		out, err := cmd.CombinedOutput()
		if err != nil {
			c.t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	if parent != "" {
		git("read-tree", parent+"^{tree}")
	} else {
		git("read-tree", "--empty")
	}
	for i, f := range files {
		p := filepath.Join(c.t.TempDir(), "f"+string(rune('a'+i)))
		if err := os.WriteFile(p, []byte(f.body), 0o644); err != nil {
			c.t.Fatal(err)
		}
		sha := git("hash-object", "-w", p)
		mode := f.mode
		if mode == "" {
			mode = "100644"
		}
		git("update-index", "--add", "--cacheinfo", mode+","+sha+","+f.path)
	}
	return c.commitTree(git("write-tree"), parent, "spotcheck: records")
}

var (
	windowOpen = storeFile{path: "window-open.json", body: "{}\n"}
	aHandBack  = storeFile{path: "handbacks/20261125T010203Z-iss-12-0123456789abcdef.json", body: "{}\n"}
	anItem     = storeFile{path: "items/20261125T010203Z-0123456789ab/001-drain-landed-" + strings.Repeat("a", 40) + ".json", body: "{}\n"}
)

// TestPrePushPassesStoreCommitsWithoutAReceipt: the store's first push (an
// orphan create) and the next (a fast-forward over it) pass with no receipt.
func TestPrePushPassesStoreCommitsWithoutAReceipt(t *testing.T) {
	c := newPrePushCase(t)
	first := c.storeCommit("", windowOpen)
	if out, err := c.push("--porcelain", "origin", first+":"+spotcheckRef); err != nil {
		t.Fatalf("the store's first commit was refused\n%s", out)
	}
	next := c.storeCommit(first, aHandBack, anItem)
	if out, err := c.push("--porcelain", "origin", next+":"+spotcheckRef); err != nil {
		t.Fatalf("a fast-forward of store records was refused\n%s", out)
	}
	if got := c.remoteRef(spotcheckRef); got != next {
		t.Fatalf("the remote holds %q, want %s", got, next)
	}
}

// TestPrePushGatesEveryOtherPushToTheStore: a file outside the store's paths,
// an executable or a link at one, a tree entry named "..", the product's
// history pushed to the branch, and a commit carrying files over a store
// commit each need a receipt; a rewrite is refused as a non-fast-forward; a
// deletion is refused naming the ruling; and a branch whose name begins like
// the store's is no branch of the row.
func TestPrePushGatesEveryOtherPushToTheStore(t *testing.T) {
	refused := func(t *testing.T, c *prePushCase, commit, ref string) {
		t.Helper()
		out, err := c.push("origin", commit+":"+ref)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("the push passed without a receipt (err %v)\n%s", err, out)
		}
		if got := c.remoteRef(ref); got != "" && got == commit {
			t.Errorf("the remote moved to %s although the push was refused", got)
		}
	}
	for name, f := range map[string]storeFile{
		"a file outside the store":   {path: "main.go", body: "package main\n"},
		"a file at the root":         {path: "notes.json", body: "{}\n"},
		"not json":                   {path: "handbacks/x.txt", body: "x\n"},
		"an executable":              {path: aHandBack.path, body: "#!/bin/sh\n", mode: "100755"},
		"a link":                     {path: aHandBack.path, body: "../../etc/passwd", mode: "120000"},
		"another folder":             {path: "scripts/x.json", body: "{}\n"},
		"a name with a space":        {path: "handbacks/a b.json", body: "{}\n"},
		"a store folder file at top": {path: "handbacks.json", body: "{}\n"},
	} {
		t.Run(name, func(t *testing.T) {
			c := newPrePushCase(t)
			refused(t, c, c.storeCommit("", windowOpen, f), spotcheckRef)
		})
	}
	t.Run("a tree entry named ..", func(t *testing.T) {
		// git's index refuses such a path, so the tree is written as raw
		// entries, as a hand-made object could be.
		c := newPrePushCase(t)
		mktree := func(entry string) string {
			cmd := exec.Command("git", "mktree")
			cmd.Dir, cmd.Env, cmd.Stdin = c.dir, c.env, strings.NewReader(entry+"\n")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("git mktree: %v\n%s", err, out)
			}
			return strings.TrimSpace(string(out))
		}
		p := filepath.Join(t.TempDir(), "x.json")
		if err := os.WriteFile(p, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		blob := c.git("hash-object", "-w", p)
		inner := mktree("100644 blob " + blob + "\tx.json")
		dots := mktree("040000 tree " + inner + "\t..")
		root := mktree("040000 tree " + dots + "\titems")
		if got := c.git("ls-tree", "-r", "--full-tree", "--name-only", root); got != "items/../x.json" {
			t.Fatalf("the hand-made tree lists %q", got)
		}
		refused(t, c, c.commitTree(root, "", "spotcheck: records"), spotcheckRef)
	})
	t.Run("the product's history", func(t *testing.T) {
		c := newPrePushCase(t)
		refused(t, c, c.commit("feature.md", "a feature\n"), spotcheckRef)
	})
	t.Run("a store commit over the product's history", func(t *testing.T) {
		c := newPrePushCase(t)
		base := c.commit("feature.md", "a feature\n")
		tree := c.git("hash-object", "-w", "-t", "tree", "/dev/null")
		refused(t, c, c.commitTree(tree, base, "spotcheck: records"), spotcheckRef)
	})
	t.Run("files over a store commit", func(t *testing.T) {
		c := newPrePushCase(t)
		first := c.storeCommit("", windowOpen)
		if out, err := c.push("origin", first+":"+spotcheckRef); err != nil {
			t.Fatalf("the store's first commit was refused\n%s", out)
		}
		out, err := c.push("origin", c.storeCommit(first, storeFile{path: "Makefile", body: "all:\n"})+":"+spotcheckRef)
		if err == nil || !strings.Contains(out, "no passing preflight") {
			t.Fatalf("a file outside the store passed over a store commit (err %v)\n%s", err, out)
		}
		if got := c.remoteRef(spotcheckRef); got != first {
			t.Errorf("the remote holds %q, want %s", got, first)
		}
	})
	t.Run("rewrite", func(t *testing.T) {
		c := newPrePushCase(t)
		first := c.storeCommit("", windowOpen)
		if out, err := c.push("origin", first+":"+spotcheckRef); err != nil {
			t.Fatalf("the store's first commit was refused\n%s", out)
		}
		out, err := c.push("origin", "+"+c.storeCommit("", aHandBack)+":"+spotcheckRef)
		if err == nil || !strings.Contains(out, "non-fast-forward") {
			t.Fatalf("a forced rewrite of the store passed (err %v)\n%s", err, out)
		}
	})
	t.Run("a branch named like the store", func(t *testing.T) {
		c := newPrePushCase(t)
		refused(t, c, c.storeCommit("", windowOpen), "refs/heads/abcd-spotcheck-x")
	})
	for name, del := range map[string][]string{
		"delete by colon refspec": {"origin", ":" + spotcheckRef},
		"delete by --delete":      {"--delete", "origin", "abcd-spotcheck"},
	} {
		t.Run(name, func(t *testing.T) {
			c := newPrePushCase(t)
			first := c.storeCommit("", windowOpen)
			if out, err := c.push("origin", first+":"+spotcheckRef); err != nil {
				t.Fatalf("the store's first commit was refused\n%s", out)
			}
			out, err := c.push(del...)
			if err == nil || !strings.Contains(out, "is kept for ever and never deleted") || !strings.Contains(out, "adr-2611251806496719") {
				t.Fatalf("a deletion of the store was not refused as one (err %v)\n%s", err, out)
			}
			if got := c.remoteRef(spotcheckRef); got != first {
				t.Errorf("the remote holds %q, want %s", got, first)
			}
		})
	}
	t.Run("a deletion of a branch named like the store", func(t *testing.T) {
		c := newPrePushCase(t)
		other := c.commit("feature.md", "a feature\n")
		c.preflight(c.dir)
		if out, err := c.push("origin", other+":refs/heads/abcd-spotcheck-x"); err != nil {
			t.Fatalf("a receipted push was refused\n%s", out)
		}
		if out, err := c.push("origin", ":refs/heads/abcd-spotcheck-x"); err != nil {
			t.Fatalf("the deletion of a branch only named like the store was refused\n%s", out)
		}
	})
}
