package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Each account's order within a group (spc-2610161044461448 step 2;
// iss-2610161015387318; dcn-2610161040368890): within each group of one
// category and one severity, the issues naming a file come before those naming
// a directory, and each part is taken in a rendezvous-hash order of the
// account's spread key and the issue id, so several drains spread across a
// group instead of racing for its oldest issue.

// fileAt captures one open record whose found_at is foundAt, making the path
// in the checkout first (a directory when it ends in "/"), since capture
// refuses a found_at naming nothing there.
func (f drainFixture) fileAt(id string, sev Severity, cat Category, foundAt string) {
	f.t.Helper()
	p := filepath.Join(f.repo, filepath.FromSlash(strings.TrimSuffix(foundAt, "/")))
	if strings.HasSuffix(foundAt, "/") {
		if err := os.MkdirAll(p, 0o755); err != nil {
			f.t.Fatal(err)
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			f.t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
	if _, err := Capture(CaptureRequest{
		RepoRoot: f.repo, IssuesRoot: f.ir, Text: "text of " + id, Severity: sev,
		Category: cat, Source: "manual-test", Slug: "s", FoundDuring: "t", ForceID: id,
		Remedy: "r", FoundAt: strings.TrimSuffix(foundAt, "/"),
	}); err != nil {
		f.t.Fatalf("capture %s: %v", id, err)
	}
}

// eligibleOrder is the eligible ids of a plan for spread, in order.
func (f drainFixture) eligibleOrder(spread string) []string {
	f.t.Helper()
	p, err := PlanDrain(DrainPlanRequest{RepoRoot: f.repo, IssuesRoot: f.ir, Spread: spread})
	if err != nil {
		f.t.Fatalf("PlanDrain: %v", err)
	}
	var got []string
	for _, v := range p.Dispositions {
		if v.Outcome == DrainEligible {
			got = append(got, v.ID)
		}
	}
	return got
}

// TestAGroupTakesTheIssuesNamingAFileFirst: within one category and severity,
// an issue whose found_at names a file (one whose name has no extension
// included) is taken before one naming a directory, whatever their
// age; an issue in a more urgent group still comes before both.
func TestAGroupTakesTheIssuesNamingAFileFirst(t *testing.T) {
	f := newDrainFixture(t)
	f.fileAt("iss-20", SeverityMinor, "bug", "internal/pkg/")
	f.fileAt("iss-21", SeverityMinor, "bug", "internal/pkg/a.go")
	f.fileAt("iss-22", SeverityMinor, "bug", "tools/Makefile")
	f.fileAt("iss-23", SeverityMinor, "bug", "docs/")
	f.fileAt("iss-24", SeverityMinor, "tech-debt", "docs/")
	want := []string{"iss-24", "iss-21", "iss-22", "iss-20", "iss-23"}
	if got := f.eligibleOrder(""); !slices.Equal(got, want) {
		t.Fatalf("eligible order = %v\nwant           %v", got, want)
	}
	p := f.plan()
	if !strings.Contains(p.Order, "naming a file before") {
		t.Fatalf("the order rule states the file-first part: %q", p.Order)
	}
}

// TestTwoHoldersOrderAGroupDifferently: two accounts' spread keys take one
// ten-issue group in different orders, each the same at every plan, while
// both take the more urgent group first; with no spread the group is oldest
// first, as before.
func TestTwoHoldersOrderAGroupDifferently(t *testing.T) {
	f := newDrainFixture(t)
	var group []string
	for i := 30; i < 40; i++ {
		id := fmt.Sprintf("iss-%d", i)
		group = append(group, id)
		f.file(id, SeverityMinor, "bug", "r")
	}
	f.file("iss-50", SeverityNitpick, "bug", "r")
	octo, quinn := "octo\x00"+strings.Repeat("a", 32), "quinn\x00"+strings.Repeat("b", 32)
	a, b := f.eligibleOrder(octo), f.eligibleOrder(quinn)
	for _, got := range [][]string{a, b} {
		if len(got) != 11 || got[0] != "iss-50" {
			t.Fatalf("the nitpick group comes first for every account: %v", got)
		}
		part := slices.Clone(got[1:])
		slices.Sort(part)
		if !slices.Equal(part, group) {
			t.Fatalf("the group is every one of its issues once: %v", got)
		}
	}
	if slices.Equal(a, b) {
		t.Fatalf("two holders take the group in one order: %v", a)
	}
	if again := f.eligibleOrder(octo); !slices.Equal(again, a) {
		t.Fatalf("one holder's order moved between plans: %v then %v", a, again)
	}
	if plain := f.eligibleOrder(""); !slices.Equal(plain[1:], group) {
		t.Fatalf("with no spread the group is oldest first: %v", plain)
	}
	p, err := PlanDrain(DrainPlanRequest{RepoRoot: f.repo, IssuesRoot: f.ir, Spread: octo})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.Order, "rendezvous hash") || strings.Contains(p.Order, "oldest first") {
		t.Fatalf("the order rule states the account's own order: %q", p.Order)
	}
}

// TestAPathAbsentWhereTheLedgerWasReadIsNotFileScoped: an issue whose found_at
// names a path the ledger's commit does not hold (deleted since, or never
// there) is not file-scoped: it sorts with the issues naming a directory or
// no path, after one naming a file the commit holds, whatever their age.
func TestAPathAbsentWhereTheLedgerWasReadIsNotFileScoped(t *testing.T) {
	f := newDrainFixture(t)
	f.fileAt("iss-60", SeverityMinor, "bug", "internal/gone.go")
	f.fileAt("iss-61", SeverityMinor, "bug", "internal/kept.go")
	files := func(paths []string) (map[string]bool, error) {
		return map[string]bool{"internal/kept.go": true}, nil
	}
	p, err := PlanDrain(DrainPlanRequest{RepoRoot: f.repo, IssuesRoot: f.ir, Files: files})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range p.Dispositions {
		if v.Outcome == DrainEligible {
			got = append(got, v.ID)
		}
	}
	if want := []string{"iss-61", "iss-60"}; !slices.Equal(got, want) {
		t.Fatalf("eligible order = %v; want %v: a path the ledger's commit lacks is not file-scoped", got, want)
	}
}

// TestTheWorkingTreeReadsASymlinkAsGitStoresIt: with no commit to ask, the
// working tree is looked in, and a path is classified as git would store it:
// a symbolic link to a directory is a file entry, as git stores a link as a
// blob, a path through that link is not in the tree, and one the tree no
// longer holds is not file-scoped either; so the link's issue sorts with the
// file's, oldest first, and the other two after them.
func TestTheWorkingTreeReadsASymlinkAsGitStoresIt(t *testing.T) {
	f := newDrainFixture(t)
	if err := os.MkdirAll(filepath.Join(f.repo, "internal", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("pkg", filepath.Join(f.repo, "internal", "link")); err != nil {
		t.Fatal(err)
	}
	f.fileAt("iss-70", SeverityMinor, "bug", "internal/link/")
	f.fileAt("iss-71", SeverityMinor, "bug", "internal/gone.go")
	f.fileAt("iss-72", SeverityMinor, "bug", "internal/kept.go")
	f.fileAt("iss-73", SeverityMinor, "bug", "internal/link/a.go")
	if err := os.Remove(filepath.Join(f.repo, "internal", "gone.go")); err != nil {
		t.Fatal(err)
	}
	if got, want := f.eligibleOrder(""), []string{"iss-70", "iss-72", "iss-71", "iss-73"}; !slices.Equal(got, want) {
		t.Fatalf("eligible order = %v; want %v", got, want)
	}
}

// TestTheWorkingTreeAndACommitAgreeOnASymlink: the working-tree fallback and
// the commit's own answer (gitutil.TreePathTypes, as a drain on origin asks
// it) classify the same paths alike: a link to a directory and a link to a
// file are files, a path through a link is not, a directory is not, and a
// file is.
func TestTheWorkingTreeAndACommitAgreeOnASymlink(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("dir/a.go", "package dir\n")
	for link, to := range map[string]string{"link": "dir", "flink": "dir/a.go"} {
		if err := os.Symlink(to, filepath.Join(r.Root(), link)); err != nil {
			t.Fatal(err)
		}
	}
	r.Commit("links")
	head := strings.TrimSpace(r.Git("rev-parse", "HEAD"))
	paths := []string{"link", "link/a.go", "flink", "dir", "dir/a.go"}
	types, err := gitutil.TreePathTypes(r.Root(), head, paths)
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := worktreeFiles(r.Root())(paths)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"link": true, "flink": true, "dir/a.go": true}
	for i, p := range paths {
		if got := types[i] == "blob"; got != want[p] {
			t.Errorf("the commit: %s is a file %v; want %v", p, got, want[p])
		}
		if worktree[p] != want[p] {
			t.Errorf("the working tree: %s is a file %v; want %v", p, worktree[p], want[p])
		}
	}
}
