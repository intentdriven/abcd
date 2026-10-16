package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
// message. author, when set, is "Name <mail>"; otherwise the person's. exec
// commits its files executable.
type cloudCommit struct {
	files  map[string]string
	msg    string
	author string
	exec   bool
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
//	COLLECT_STDOUT  the file collect's standard output goes to, when a test
//	                sends it to one: make copies what it holds so far to
//	                $CALLS.stdout-at-make, the ordering marker of what collect
//	                printed before preflight ran
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
if [ -n "${COLLECT_STDOUT:-}" ]; then cat "$COLLECT_STDOUT" >"$CALLS.stdout-at-make"; fi
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
			mode := os.FileMode(0o644)
			if cc.exec {
				mode = 0o755
			}
			c.write(rel, body, mode)
			if err := os.Chmod(filepath.Join(c.dir, filepath.FromSlash(rel)), mode); err != nil {
				c.t.Fatal(err)
			}
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

// toolingPaths are the paths whose change on a cloud branch would let the
// cloud's own code decide how collect's preflight is built and gated as the
// person: the build file preflight runs, the hooks, the scripts, the
// workflows, the module and workspace files, a vendored dependency, the gates'
// settings and baselines at the top of .abcd/ (and its config/), and any
// top-level hidden directory, a host's settings among them (matched by shape,
// so the made-up host's directory stands in for every one).
var toolingPaths = []string{
	"Makefile", "GNUmakefile", ".githooks/pre-push", "scripts/preflight-receipt.sh", ".github/workflows/ci.yml",
	".testhost/settings.json", "go.mod", "go.sum", "go.work", "go.work.sum", "vendor/example.com/m/m.go",
	".abcd/record-lint.json", ".abcd/docs-lint.json", ".abcd/site.json", ".abcd/site-baseline.json",
	".abcd/citations-baseline.json", ".abcd/prose-citations-baseline.json", ".abcd/rules.json",
	".abcd/config/pii.json",
}

// TestCollectRefusesToolingChangesByDefault: before anything is made, collect
// refuses a branch whose commits change the tooling, printing the diffstat of
// those paths, unless --allow-tooling-changes is passed. The refusal keeps the
// build and gate machinery as the default branch has it; the records under
// .abcd/development/ and .abcd/work/ are no tooling and stay collectable.
func TestCollectRefusesToolingChangesByDefault(t *testing.T) {
	t.Run("records stay allowed", func(t *testing.T) {
		c := newCollectCase(t)
		c.pushCloud(collectItem, workCommit(".abcd/development/specs/open/spc-1-a.md", "docs: a spec"),
			workCommit(".abcd/work/issues/open/iss-1-a.md", "chore: an issue"), reportCommit(collectItem, "Did it."))
		out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
		if code != 0 {
			t.Fatalf("collect refused records under .abcd/development/ and .abcd/work/ (exit %d)\n%s%s", code, out, errOut)
		}
	})
	for _, rel := range toolingPaths {
		t.Run(rel, func(t *testing.T) {
			c := newCollectCase(t)
			c.pushCloud(collectItem, workCommit("one.md", "fix: one"), workCommit(rel, "chore: tooling"),
				reportCommit(collectItem, "Did it."))
			out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
			if code == 0 {
				t.Fatalf("collect accepted a branch changing %s\n%s%s", rel, out, errOut)
			}
			if !strings.Contains(errOut, "--allow-tooling-changes") {
				t.Errorf("the refusal does not name --allow-tooling-changes\n%s", errOut)
			}
			if !strings.Contains(errOut, rel) || !strings.Contains(errOut, "changed") {
				t.Errorf("the refusal does not print the diffstat naming %s\n%s", rel, errOut)
			}
			c.madeNothing(t, collectItem)
		})
	}
}

// TestCollectAllowsToolingChangesWithFlag: --allow-tooling-changes is the
// person's deliberate yes, and the collect goes ahead.
func TestCollectAllowsToolingChangesWithFlag(t *testing.T) {
	c := newCollectCase(t)
	c.pushCloud(collectItem, workCommit("one.md", "fix: one"), workCommit("scripts/new.sh", "chore: tooling"),
		reportCommit(collectItem, "Did it."))
	out, errOut, code := c.scriptSplit(nil, collectScript, collectItem, "--allow-tooling-changes")
	if code != 0 {
		t.Fatalf("collect --allow-tooling-changes exited %d\n%s%s", code, out, errOut)
	}
	if got := c.git("diff", "--name-only", "origin/main", "collect/"+collectItem); got != "one.md\nscripts/new.sh" {
		t.Errorf("the landing branch changes %q, want the work and the tooling change", got)
	}
}

// TestCollectSkipsEarlierReportCommits: a session that stopped and was later
// handed back leaves two report commits for its item. Each commit changing
// only this item's report is left out, and the work around them is collected.
func TestCollectSkipsEarlierReportCommits(t *testing.T) {
	c := newCollectCase(t)
	stopped := reportCommit(collectItem, "Stopped.")
	stopped.files[reportPath(collectItem)] = reportText(collectItem, "stopped", "branch", "Stopped.")
	c.pushCloud(collectItem, workCommit("one.md", "fix: one"), stopped, workCommit("two.md", "fix: two"),
		reportCommit(collectItem, "Did it."))
	out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
	if code != 0 {
		t.Fatalf("collect of a branch with two report commits exited %d\n%s%s", code, out, errOut)
	}
	if got := c.git("log", "--reverse", "--format=%s", "origin/main..collect/"+collectItem); got != "fix: one\nfix: two" {
		t.Errorf("the landing branch carries %q, want the two work commits and no report commit", got)
	}
}

// TestCollectRefusesAnotherItemsReportCommit: only this item's report commits
// are left out; a commit changing another item's report alone is refused.
func TestCollectRefusesAnotherItemsReportCommit(t *testing.T) {
	c := newCollectCase(t)
	other := reportCommit("iss-2610150000000099", "Not this item.")
	c.pushCloud(collectItem, workCommit("one.md", "fix: one"), other, reportCommit(collectItem, "Did it."))
	out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
	if code == 0 || !strings.Contains(errOut, "cloud-reports") {
		t.Fatalf("collect accepted a commit changing another item's report (exit %d)\n%s%s", code, out, errOut)
	}
	c.madeNothing(t, collectItem)
}

// TestCollectRefusesSymlinkedMachineStore: the machine store is never read or
// written through a symlink, at any level: ~/.abcd.noindex, its worktrees/,
// and worktrees/<root-sha>/.
func TestCollectRefusesSymlinkedMachineStore(t *testing.T) {
	for _, level := range []string{".abcd.noindex", ".abcd.noindex/worktrees", ".abcd.noindex/worktrees/<root>"} {
		t.Run(level, func(t *testing.T) {
			c := newCollectCase(t)
			c.pushCloud(collectItem, workCommit("one.md", "fix: one"), reportCommit(collectItem, "Did it."))
			root := c.git("rev-list", "-n", "1", "--max-parents=0", "HEAD")
			link := filepath.Join(c.home(), filepath.FromSlash(strings.Replace(level, "<root>", root, 1)))
			if err := os.MkdirAll(filepath.Dir(link), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), link); err != nil {
				t.Fatal(err)
			}
			out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
			if code == 0 {
				t.Fatalf("collect went ahead through a symlink at %s\n%s%s", level, out, errOut)
			}
			if !strings.Contains(errOut, "symlink") {
				t.Errorf("the refusal does not name the symlink\n%s", errOut)
			}
			if strings.Contains(c.callLog(), "make ") || strings.Contains(c.callLog(), "worktree add") {
				t.Errorf("a refused collect made a worktree or ran make:\n%s", c.callLog())
			}
		})
	}
}

// TestCollectCherryPickUsesThePersonsHooks: the person's checkout names its
// hooks by a relative core.hooksPath, which git resolves inside whichever
// worktree a command runs in. The cherry-pick into the collect worktree runs
// with the person's own checkout's .githooks, by absolute path, never the
// hooks a collected commit put in the worktree.
func TestCollectCherryPickUsesThePersonsHooks(t *testing.T) {
	c := newCollectCase(t)
	cloudHook := cloudCommit{
		files: map[string]string{".githooks/post-commit": "#!/bin/sh\necho 'cloud hook ran' >>\"$CALLS\"\n"},
		msg:   "chore: a hook\n\n" + sessionTrailer,
		exec:  true,
	}
	c.pushCloud(collectItem, cloudHook, workCommit("one.md", "fix: one"), reportCommit(collectItem, "Did it."))
	c.write(".githooks/post-commit", "#!/bin/sh\necho 'person hook ran' >>\"$CALLS\"\n", 0o755)
	out, errOut, code := c.scriptSplit(nil, collectScript, collectItem, "--allow-tooling-changes")
	if code != 0 {
		t.Fatalf("collect exited %d\n%s%s", code, out, errOut)
	}
	log := c.callLog()
	if strings.Contains(log, "cloud hook ran") {
		t.Errorf("the cherry-pick ran a hook a collected commit carries:\n%s", log)
	}
	if !strings.Contains(log, "person hook ran") {
		t.Errorf("the cherry-pick did not run the person's own hooks:\n%s", log)
	}
}

// pushCloudTree is pushCloud for paths a working tree cannot hold as written:
// a case variant of a path the tree already has, on a file system that folds
// case, or bytes the file system would normalise. Each commit is made with
// git's plumbing in a temporary index, so its paths are exactly the test's.
// It returns the tip.
func (c *cloudCase) pushCloudTree(item string, commits ...cloudCommit) string {
	c.t.Helper()
	tmp := c.t.TempDir()
	index := filepath.Join(tmp, "index")
	env := []string{"GIT_INDEX_FILE=" + index,
		"GIT_AUTHOR_NAME=" + cloudAuthor, "GIT_AUTHOR_EMAIL=" + cloudMail,
		"GIT_COMMITTER_NAME=" + cloudAuthor, "GIT_COMMITTER_EMAIL=" + cloudMail}
	plumb := func(args ...string) string {
		c.t.Helper()
		out, err := c.runEnv(c.dir, env, "git", args...)
		if err != nil {
			c.t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(out)
	}
	parent := c.git("rev-parse", "origin/main")
	plumb("read-tree", parent)
	for n, cc := range commits {
		for rel, body := range cc.files {
			src := filepath.Join(tmp, "blob")
			if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
				c.t.Fatal(err)
			}
			blob := plumb("hash-object", "-w", "--", src)
			plumb("update-index", "--add", "--cacheinfo", "100644,"+blob+","+rel)
		}
		tree := plumb("write-tree")
		msgFile := filepath.Join(tmp, "msg"+strconv.Itoa(n))
		if err := os.WriteFile(msgFile, []byte(cc.msg), 0o644); err != nil {
			c.t.Fatal(err)
		}
		parent = plumb("commit-tree", tree, "-p", parent, "-F", msgFile)
	}
	c.git("-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "+"+parent+":refs/heads/"+hostWIPPrefix+item)
	c.resetCalls()
	return parent
}

// TestCollectRefusesCaseVariantToolingPath: the tooling rule matches a path
// whatever its case, as check-wip-source.sh matches its segment, since a file
// system that folds case writes SCRIPTS/x.sh into scripts/.
func TestCollectRefusesCaseVariantToolingPath(t *testing.T) {
	for _, rel := range []string{"SCRIPTS/x.sh", "MAKEFILE", "Go.mod", ".GitHub/workflows/x.yml", ".ABCD/record-lint.json"} {
		t.Run(rel, func(t *testing.T) {
			c := newCollectCase(t)
			c.pushCloudTree(collectItem, workCommit("one.md", "fix: one"), workCommit(rel, "chore: tooling"),
				reportCommit(collectItem, "Did it."))
			out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
			if code == 0 {
				t.Fatalf("collect accepted a branch changing %s\n%s%s", rel, out, errOut)
			}
			if !strings.Contains(errOut, "--allow-tooling-changes") || !strings.Contains(errOut, rel) {
				t.Errorf("the refusal does not name %s as tooling\n%s", rel, errOut)
			}
			c.madeNothing(t, collectItem)
		})
	}
	t.Run("the local tier in another case", func(t *testing.T) {
		c := newCollectCase(t)
		c.pushCloudTree(collectItem, workCommit("one.md", "fix: one"),
			workCommit(".ABCD/.Work.Local/preflight-receipts/x", "chore: a receipt"), reportCommit(collectItem, "Did it."))
		out, errOut, code := c.scriptSplit(nil, collectScript, collectItem, "--allow-tooling-changes")
		if code == 0 || !strings.Contains(errOut, "local tier") {
			t.Fatalf("collect accepted the local tier in another case (exit %d)\n%s%s", code, out, errOut)
		}
		c.madeNothing(t, collectItem)
	})
}

// TestCollectRefusesCaseCollidingAddedPath: an added path whose case-folded
// form is another path's, in the default branch's tree or in the copy's, would
// overwrite that path on a file system that folds case. It is refused whatever
// the flags say.
func TestCollectRefusesCaseCollidingAddedPath(t *testing.T) {
	for _, tc := range []struct {
		name    string
		commits []cloudCommit
		path    string
	}{
		{"a variant of the default branch's file", []cloudCommit{workCommit("SEED.md", "fix: seed")}, "SEED.md"},
		{"a variant of the default branch's script", []cloudCommit{workCommit("SCRIPTS/check-attribution.sh", "fix: gate")},
			"SCRIPTS/check-attribution.sh"},
		{"two variants in the copy", []cloudCommit{workCommit("notes.md", "fix: notes"), workCommit("Notes.md", "fix: Notes")}, "Notes.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCollectCase(t)
			c.pushCloudTree(collectItem, append(tc.commits, reportCommit(collectItem, "Did it."))...)
			out, errOut, code := c.scriptSplit(nil, collectScript, collectItem, "--allow-tooling-changes")
			if code == 0 {
				t.Fatalf("collect accepted %s\n%s%s", tc.path, out, errOut)
			}
			if !strings.Contains(errOut, "case") || !strings.Contains(errOut, tc.path) {
				t.Errorf("the refusal does not name %s as a case collision\n%s", tc.path, errOut)
			}
			c.madeNothing(t, collectItem)
		})
	}
	t.Run("a path in its own case collides with nothing", func(t *testing.T) {
		c := newCollectCase(t)
		c.pushCloudTree(collectItem, workCommit("Readme.md", "docs: a readme"), reportCommit(collectItem, "Did it."))
		out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
		if code != 0 {
			t.Fatalf("collect refused a path no other path folds to (exit %d)\n%s%s", code, out, errOut)
		}
	})
}

// TestCollectMatchesNonASCIIPaths: git quotes a path holding a non-ASCII byte
// ("scripts/\303\251.sh") unless it is read NUL-separated, and a quoted path
// matches no pattern. The paths are read with -z, so such a path is matched as
// it is.
func TestCollectMatchesNonASCIIPaths(t *testing.T) {
	c := newCollectCase(t)
	c.pushCloudTree(collectItem, workCommit("one.md", "fix: one"), workCommit("scripts/\u00e9.sh", "chore: tooling"),
		reportCommit(collectItem, "Did it."))
	out, errOut, code := c.scriptSplit(nil, collectScript, collectItem)
	if code == 0 {
		t.Fatalf("collect accepted a branch changing scripts/\u00e9.sh\n%s%s", out, errOut)
	}
	if !strings.Contains(errOut, "--allow-tooling-changes") {
		t.Errorf("the refusal is not the tooling refusal\n%s", errOut)
	}
	c.madeNothing(t, collectItem)
}

// TestCollectShowsWhatWillRunBeforePreflight: preflight on the collected copy
// runs that copy's code as the person, its tests and its Go-written gates
// included. Before it runs, collect says so on standard output, with the full
// diffstat of the collected commits, so the person sees what code will run.
// The fake make copies what standard output holds when it starts.
func TestCollectShowsWhatWillRunBeforePreflight(t *testing.T) {
	run := func(t *testing.T, c *cloudCase, args ...string) string {
		t.Helper()
		f, err := os.Create(filepath.Join(t.TempDir(), "stdout"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		cmd := exec.Command("bash", append([]string{filepath.Join(c.dir, filepath.FromSlash(collectScript))}, args...)...)
		cmd.Dir = c.dir
		cmd.Env = append(append([]string{}, c.env...), "COLLECT_STDOUT="+f.Name())
		var errb strings.Builder
		cmd.Stdout, cmd.Stderr = f, &errb
		if err := cmd.Run(); err != nil {
			t.Fatalf("collect: %v\n%s", err, errb.String())
		}
		snap, err := os.ReadFile(c.calls + ".stdout-at-make")
		if err != nil {
			t.Fatalf("make never ran, or wrote no marker: %v\n%s", err, errb.String())
		}
		return string(snap)
	}

	t.Run("the default", func(t *testing.T) {
		c := newCollectCase(t)
		c.pushCloud(collectItem, workCommit("one.md", "fix: one"), workCommit("docs/two.md", "fix: two"),
			reportCommit(collectItem, "Did it."))
		before := run(t, c, collectItem)
		for _, want := range []string{
			"runs the collected copy's code on this computer, as you",
			"its tests and its Go-written gates",
			"decision 9",
			"build and gate machinery",
			"one.md", "docs/two.md", "2 files changed",
		} {
			if !strings.Contains(before, want) {
				t.Errorf("before make ran, standard output did not say %q:\n%s", want, before)
			}
		}
		if strings.Contains(before, ".abcd/work/cloud-reports") {
			t.Errorf("the diffstat counts the report, which is not collected:\n%s", before)
		}
	})

	t.Run("with tooling changes allowed", func(t *testing.T) {
		c := newCollectCase(t)
		c.pushCloud(collectItem, workCommit("scripts/new.sh", "chore: tooling"), reportCommit(collectItem, "Did it."))
		before := run(t, c, collectItem, "--allow-tooling-changes")
		for _, want := range []string{"runs the collected copy's code on this computer, as you", "scripts/new.sh",
			"the build and gate machinery is the copy's too"} {
			if !strings.Contains(before, want) {
				t.Errorf("before make ran, standard output did not say %q:\n%s", want, before)
			}
		}
	})
}
