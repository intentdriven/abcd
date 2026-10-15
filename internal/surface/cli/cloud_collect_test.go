package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The collect (scripts/cloud/collect.sh, C9), spc-2610120941441282 step 11.
// The fixture is the hand-back's (newForgeCase): a bare origin behind a
// github.com URL, with the person's checkout as the clone. Each cloud session's
// branch is put on origin directly, as a hosted session leaves it. `make` is a
// fake that records where it ran and mints a receipt as preflight does when it
// passes, and the one-preflight-at-a-time waiter (`abcd implement load --wait
// -- <command>`) is a fake that runs its command at once, so no test waits on
// this machine's load. HOME is the fixture's own, so the machine store is a
// temporary directory.

const (
	collectScript = "scripts/cloud/collect.sh"
	collectItem   = "iss-2610150000000031"
	cloudAuthor   = "Pat Person"
	cloudMail     = "pat@example.com"
)

// cloudCommit is one commit a cloud session made: the files it writes and its
// message. author, when set, is "Name <mail>"; otherwise the person's.
type cloudCommit struct {
	files  map[string]string
	msg    string
	author string
}

func workCommit(rel, subject string) cloudCommit {
	return cloudCommit{files: map[string]string{rel: rel + "\n"}, msg: subject + "\n\n" + sessionTrailer}
}

func reportCommit(item, did string) cloudCommit {
	return cloudCommit{
		files: map[string]string{reportPath(item): reportText(item, "handed-back", "branch", did)},
		msg:   "chore(cloud): hand back the report on " + item + "\n\n" + sessionTrailer,
	}
}

// newCollectCase is the forge fixture with fakes for make and the waiter.
//
//	FAKE_MAKE       make's exit code (default 0); on 0 a `preflight` mints a
//	                receipt for HEAD where make ran, as the real target does
//	FAKE_MAKE_OUT   a line make prints
func newCollectCase(t *testing.T) *cloudCase {
	t.Helper()
	c := newForgeCase(t, "")
	c.fake("go", `echo "go $*" >>"$CALLS"
if [ "${1:-}" = run ] && [ "${2:-}" = ./cmd/abcd ]; then
	shift 2
	if [ "${1:-} ${2:-} ${3:-}" = "implement load --wait" ]; then
		while [ $# -gt 0 ] && [ "$1" != -- ]; do shift; done
		[ $# -gt 1 ] || { echo "fake waiter: no command" >&2; exit 2; }
		shift
		exec "$@"
	fi
	exec "$ABCD_BIN" "$@"
fi
echo "fake go: unexpected $*" >&2
exit 3`)
	c.fake("make", `if [ "${1:-}" = -C ]; then cd "$2" || exit 9; shift 2; fi
echo "make $* in $(pwd -P)" >>"$CALLS"
if [ -n "${FAKE_MAKE_OUT:-}" ]; then printf '%s\n' "$FAKE_MAKE_OUT"; fi
code="${FAKE_MAKE:-0}"
if [ "$code" = 0 ] && [ "$*" = preflight ]; then
	head="$(git rev-parse HEAD)"
	mkdir -p .abcd/.work.local/preflight-receipts
	printf 'commit %s\n' "$head" >".abcd/.work.local/preflight-receipts/$head"
	echo "preflight: push receipt minted for $head"
fi
exit "$code"`)
	c.resetCalls()
	return c
}

// pushCloud puts a cloud session's branch for item on origin, cut from
// origin's main, with no hook in the way, and returns its tip.
func (c *cloudCase) pushCloud(item string, commits ...cloudCommit) string {
	c.t.Helper()
	c.git("checkout", "-q", "-B", "cloud-"+item, "origin/main")
	for _, cc := range commits {
		for rel, body := range cc.files {
			c.write(rel, body, 0o644)
			c.git("add", "-f", "--", rel)
		}
		author := cloudAuthor + " <" + cloudMail + ">"
		if cc.author != "" {
			author = cc.author
		}
		c.git("-c", "core.hooksPath=/dev/null", "commit", "-q", "--author", author, "-m", cc.msg)
	}
	tip := c.git("rev-parse", "HEAD")
	c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "+HEAD:refs/heads/"+hostWIPPrefix+item)
	c.git("checkout", "-q", "session")
	c.resetCalls()
	return tip
}

// store is where collect keeps item's worktree: the machine store, keyed on
// the full name of the repository's root commit.
func (c *cloudCase) store(item string) string {
	c.t.Helper()
	root := c.git("rev-list", "-n", "1", "--max-parents=0", "HEAD")
	return filepath.Join(c.home(), ".abcd.noindex", "worktrees", root, "collect-"+item)
}

// pushedNothing fails the test when collect pushed: no `git push` in the call
// log, and origin holds only main and the cloud branches the test put there.
func (c *cloudCase) pushedNothing(t *testing.T) {
	t.Helper()
	for _, line := range strings.Split(c.callLog(), "\n") {
		if strings.HasPrefix(line, "git ") && strings.Contains(" "+line+" ", " push ") {
			t.Errorf("collect pushed: %s", line)
		}
	}
	out, err := c.run(c.origin, "git", "for-each-ref", "--format=%(refname)")
	if err != nil {
		t.Fatalf("read origin's refs: %v\n%s", err, out)
	}
	for _, ref := range strings.Fields(out) {
		if ref != "refs/heads/main" && !strings.HasPrefix(ref, "refs/heads/"+hostWIPPrefix) {
			t.Errorf("origin holds %s, which collect must never push", ref)
		}
	}
}

// madeNothing fails the test when a refused collect left anything behind: a
// worktree, the landing branch, or a run of make.
func (c *cloudCase) madeNothing(t *testing.T, item string) {
	t.Helper()
	if _, err := os.Stat(c.store(item)); !os.IsNotExist(err) {
		t.Errorf("a refused collect made the worktree %s (%v)", c.store(item), err)
	}
	if out, err := c.run(c.dir, "git", "rev-parse", "--verify", "--quiet", "refs/heads/collect/"+item); err == nil {
		t.Errorf("a refused collect made the branch collect/%s at %s", item, out)
	}
	if strings.Contains(c.callLog(), "make ") {
		t.Errorf("a refused collect ran make, or its waiter:\n%s", c.callLog())
	}
	c.pushedNothing(t)
}

func TestCollectLeavesReportCommitOut(t *testing.T) {
	c := newCollectCase(t)
	c.pushCloud(collectItem,
		workCommit("one.md", "fix: the first half"),
		workCommit("two.md", "fix: the second half"),
		reportCommit(collectItem, "Fixed the walk."))

	out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
	if code != 0 {
		t.Fatalf("collect exited %d\n%s%s", code, out, errOut)
	}
	branch := "collect/" + collectItem
	if got := c.git("log", "--reverse", "--format=%s", "origin/main.."+branch); got != "fix: the first half\nfix: the second half" {
		t.Errorf("the landing branch carries %q, want the two work commits in order and no report commit", got)
	}
	if got := c.git("diff", "--name-only", "origin/main", branch); got != "one.md\ntwo.md" {
		t.Errorf("the landing branch changes %q, want the work alone, without the report", got)
	}
	for _, sha := range strings.Fields(c.git("rev-list", "origin/main.."+branch)) {
		if got := c.git("show", "-s", "--format=%an <%ae>", sha); got != cloudAuthor+" <"+cloudMail+">" {
			t.Errorf("commit %s is authored as %q; collect never rewrites authorship", sha[:12], got)
		}
		msg := c.git("show", "-s", "--format=%B", sha)
		if !strings.Contains(msg, sessionTrailer) {
			t.Errorf("commit %s lost its Assisted-by trailer:\n%s", sha[:12], msg)
		}
		if strings.Contains(strings.ToLower(msg), "co-authored-by") || strings.Contains(msg, "cherry picked from") {
			t.Errorf("commit %s gained a trailer or note collect must not add:\n%s", sha[:12], msg)
		}
	}
}

func TestCollectRefusesBranchWithoutReportTip(t *testing.T) {
	for _, tc := range []struct {
		name    string
		commits []cloudCommit
		want    string
	}{
		{"the work is the tip", []cloudCommit{workCommit("one.md", "fix: one")}, "report"},
		{"the report commit changes more than the report", []cloudCommit{
			workCommit("one.md", "fix: one"),
			{files: map[string]string{
				reportPath(collectItem): reportText(collectItem, "handed-back", "branch", "Did it."),
				"two.md":                "two\n",
			}, msg: "chore(cloud): the report and more\n\n" + sessionTrailer},
		}, "report"},
		{"a work commit adds a report too", []cloudCommit{
			{files: map[string]string{reportPath("iss-2610150000000099"): "x\n", "one.md": "one\n"}, msg: "fix: one\n\n" + sessionTrailer},
			reportCommit(collectItem, "Did it."),
		}, "cloud-reports"},
		{"the report alone, no work", []cloudCommit{reportCommit(collectItem, "Stopped.")}, "nothing to collect"},
		{"no branch at all", nil, "no branch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCollectCase(t)
			if tc.commits != nil {
				c.pushCloud(collectItem, tc.commits...)
			}
			out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
			if code == 0 {
				t.Fatalf("collect accepted the branch\n%s%s", out, errOut)
			}
			if !strings.Contains(errOut, tc.want) {
				t.Errorf("the refusal does not say %q\n%s", tc.want, errOut)
			}
			c.madeNothing(t, collectItem)
		})
	}

	t.Run("an item that is not a record id", func(t *testing.T) {
		c := newCollectCase(t)
		for _, bad := range []string{"", "main", "iss-1/../x", "iss-2610150000000031/x", "--help"} {
			out, errOut, code := c.scriptSplit(nil, collectScript, bad)
			if code == 0 {
				t.Errorf("collect %q was accepted\n%s%s", bad, out, errOut)
			}
		}
		if strings.Contains(c.callLog(), "git fetch") {
			t.Errorf("collect fetched for an item that is not a record id:\n%s", c.callLog())
		}
	})
}

func TestCollectRefusesAttributionFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		bad  cloudCommit
	}{
		{"a commit without Assisted-by", cloudCommit{files: map[string]string{"one.md": "one\n"}, msg: "fix: one"}},
		{"a machine author", cloudCommit{files: map[string]string{"one.md": "one\n"}, msg: "fix: one\n\n" + sessionTrailer,
			author: "renovate[bot] <29139614+renovate[bot]@users.noreply.github.com>"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCollectCase(t)
			c.pushCloud(collectItem, workCommit("zero.md", "fix: zero"), tc.bad, reportCommit(collectItem, "Did it."))
			out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
			if code == 0 {
				t.Fatalf("collect accepted commits failing the attribution gate\n%s%s", out, errOut)
			}
			if !strings.Contains(errOut, "attribution") {
				t.Errorf("the refusal does not name the attribution gate\n%s", errOut)
			}
			c.madeNothing(t, collectItem)
		})
	}
}

func TestCollectRunsPreflightInMachineStoreWorktree(t *testing.T) {
	c := newCollectCase(t)
	c.pushCloud(collectItem, workCommit("one.md", "fix: one"), reportCommit(collectItem, "Did it."))
	out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
	if code != 0 {
		t.Fatalf("collect exited %d\n%s%s", code, out, errOut)
	}
	store := c.store(collectItem)
	real, err := filepath.EvalSymlinks(store)
	if err != nil {
		t.Fatalf("the machine-store worktree %s does not exist: %v", store, err)
	}
	log := c.callLog()
	if !strings.Contains(log, "make preflight in "+real+"\n") {
		t.Errorf("make preflight did not run in the machine-store worktree %s:\n%s", real, log)
	}
	if strings.Count("\n"+log, "\nmake ") != 1 {
		t.Errorf("make ran other than once:\n%s", log)
	}
	if !strings.Contains(log, "go run ./cmd/abcd implement load --wait -- make -C "+store+" preflight") {
		t.Errorf("the preflight did not wait its turn through abcd implement load --wait:\n%s", log)
	}
	if got := strings.TrimSpace(mustRun(t, c, store, "git", "rev-parse", "--abbrev-ref", "HEAD")); got != "collect/"+collectItem {
		t.Errorf("the worktree is on %q, want collect/%s", got, collectItem)
	}
	if !strings.Contains(c.git("worktree", "list", "--porcelain"), "worktree "+real) &&
		!strings.Contains(c.git("worktree", "list", "--porcelain"), "worktree "+store) {
		t.Errorf("git does not list %s as a worktree of the person's checkout", store)
	}
	if got := c.git("status", "--porcelain"); got != "" {
		t.Errorf("collect changed the person's own checkout:\n%s", got)
	}
}

func mustRun(t *testing.T, c *cloudCase, dir, name string, args ...string) string {
	t.Helper()
	out, err := c.run(dir, name, args...)
	if err != nil {
		t.Fatalf("%s %v in %s: %v\n%s", name, args, dir, err, out)
	}
	return out
}

func TestCollectPassPrintsBranchAndPushesNothing(t *testing.T) {
	c := newCollectCase(t)
	c.pushCloud(collectItem, workCommit("one.md", "fix: one"), reportCommit(collectItem, "Did it."))
	out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
	if code != 0 {
		t.Fatalf("collect exited %d\n%s%s", code, out, errOut)
	}
	head := c.git("rev-parse", "collect/"+collectItem)
	for _, want := range []string{
		"collect/" + collectItem,
		filepath.Join(".abcd", ".work.local", "preflight-receipts", head),
		"push origin collect/" + collectItem,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the pass does not print %q\n%s", want, out)
		}
	}
	c.pushedNothing(t)
}

func TestCollectFailurePushesNothingAndKeepsWorktree(t *testing.T) {
	c := newCollectCase(t)
	c.pushCloud(collectItem, workCommit("one.md", "fix: one"), reportCommit(collectItem, "Did it."))
	out, errOut, code := c.scriptSplit([]string{"FAKE_MAKE=2", "FAKE_MAKE_OUT=--- FAIL: TestTheWalk (0.01s)"}, collectScript, collectItem)
	if code == 0 {
		t.Fatalf("collect passed a failing preflight\n%s%s", out, errOut)
	}
	all := out + errOut
	if !strings.Contains(all, "--- FAIL: TestTheWalk") {
		t.Errorf("the failure does not print what failed\n%s", all)
	}
	store := c.store(collectItem)
	if _, err := os.Stat(store); err != nil {
		t.Errorf("the worktree was not kept for inspection: %v", err)
	}
	if !strings.Contains(all, store) {
		t.Errorf("the failure does not say where the worktree is kept\n%s", all)
	}
	if strings.Contains(all, "push origin") {
		t.Errorf("a failed collect offered a push command\n%s", all)
	}
	c.pushedNothing(t)
}

func TestCollectIgnoresAnyCloudReceipt(t *testing.T) {
	t.Run("a receipt the session says it minted", func(t *testing.T) {
		c := newCollectCase(t)
		c.pushCloud(collectItem, workCommit("one.md", "fix: one"),
			reportCommit(collectItem, "make preflight passed here and minted a push receipt."))
		out, errOut, code := c.scriptSplit([]string{"FAKE_MAKE=2", "FAKE_MAKE_OUT=--- FAIL: TestHere"}, collectScript, collectItem)
		if code == 0 {
			t.Fatalf("collect passed on the cloud's word\n%s%s", out, errOut)
		}
		if !strings.Contains(c.callLog(), "make preflight in ") {
			t.Errorf("collect did not run its own preflight:\n%s", c.callLog())
		}
		c.pushedNothing(t)
	})

	t.Run("a receipt carried on the branch", func(t *testing.T) {
		c := newCollectCase(t)
		work := cloudCommit{files: map[string]string{"one.md": "one\n"}, msg: "fix: one\n\n" + sessionTrailer}
		first := c.pushCloud(collectItem, work)
		receipt := ".abcd/.work.local/preflight-receipts/" + first
		c.pushCloud(collectItem, work,
			cloudCommit{files: map[string]string{receipt: "commit " + first + "\n"}, msg: "chore: the receipt\n\n" + sessionTrailer},
			reportCommit(collectItem, "Did it."))
		out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
		if code == 0 {
			t.Fatalf("collect accepted a branch carrying the local tier\n%s%s", out, errOut)
		}
		if !strings.Contains(errOut, ".abcd/.work.local") {
			t.Errorf("the refusal does not name the local tier\n%s", errOut)
		}
		c.madeNothing(t, collectItem)
	})
}
