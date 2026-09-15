package capture

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestOrphanStillRemovableRejectsCommittedFile (B26) proves the pre-unlink guard
// refuses to remove a placeholder that a capture commit has replaced/filled in
// the sweep's TOCTOU window: a zero-byte inode classified as an orphan that
// becomes a non-empty committed file must no longer be removable.
func TestOrphanStillRemovableRejectsCommittedFile(t *testing.T) {
	dir := t.TempDir()
	cand := filepath.Join(dir, "iss-1-note.md")
	if err := os.WriteFile(cand, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	seen, err := os.Lstat(cand)
	if err != nil {
		t.Fatal(err)
	}
	// Baseline: a still-empty, still-same inode is removable.
	if !orphanStillRemovable(cand, seen) {
		t.Fatal("a genuine zero-byte orphan must still be removable")
	}
	// A concurrent commit replaces the placeholder with a full issue file via
	// atomic rename (temp file + rename -> new, non-empty inode).
	tmp := filepath.Join(dir, "iss-1-note.md.tmp")
	if err := os.WriteFile(tmp, []byte("---\nid: \"iss-1\"\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, cand); err != nil {
		t.Fatal(err)
	}
	if orphanStillRemovable(cand, seen) {
		t.Fatal("the sweep must not remove a placeholder a commit has since filled")
	}
}

// TestCleanOrphanPlaceholdersStillSweepsAgedOrphan guards against the B26 guard
// regressing normal sweep behaviour: a genuinely aged zero-byte placeholder is
// still removed.
func TestCleanOrphanPlaceholdersStillSweepsAgedOrphan(t *testing.T) {
	repo := t.TempDir()
	ir := filepath.Join(repo, "issues")
	if err := ensureLedgerDirs(repo, ir); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(ir, "open", "iss-1-note.md")
	if err := os.WriteFile(orphan, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * orphanAgeThreshold)
	if err := os.Chtimes(orphan, old, old); err != nil {
		t.Fatal(err)
	}
	if err := cleanOrphanPlaceholders(ir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(orphan); !os.IsNotExist(err) {
		t.Fatalf("aged zero-byte orphan should have been swept, Lstat err=%v", err)
	}
}

// TestLedgerAncestorSymlinkRefused pins EVERY directory on the way to the
// ledger — and the ledger's own directories — to the rule its record leaves
// already obey. ensureLedgerDirs refused a symlinked issues/ or status
// directory on the WRITE path, but provisioned the parents by following
// whatever was there, so a committed `.abcd` or `.abcd/work` symlink in a
// hostile clone put the whole store (the allocator lock included) outside the
// checkout while the result still reported a repo-relative path.
//
// The READ path is the same boundary from the other side, and the first fix
// only walked as far as the ledger's parent: a committed symlink at issues/ or
// at a status directory was still followed, because ReadDir follows a symlink
// and O_NOFOLLOW guards only a record's final component — so `list --json` and
// the bare status render serialized a sibling checkout's records from a fresh
// clone. Every verb resolves its roots first, so the guard belongs there: all
// four link sites are refused, whether the target sits inside the checkout or
// outside it, nothing behind the link is read, and nothing is written there.
func TestLedgerAncestorSymlinkRefused(t *testing.T) {
	const marker = "MARKER-MUST-NOT-SERIALIZE"
	_, _, realID, fm := hostileRecordLedger(t)
	decoy := reshapeRecord(t, fm, realID, "iss-9", "decoy", "a body holding "+marker+"\n")

	for _, tc := range []struct {
		link    string // repo-relative directory a hostile clone commits as a symlink
		records string // where the ledger's open records then sit under the target
	}{
		{".abcd", "work/issues/open"},
		{".abcd/work", "issues/open"},
		{".abcd/work/issues", "open"},
		{".abcd/work/issues/open", "."},
	} {
		for _, where := range []string{"outside the checkout", "inside the checkout"} {
			t.Run(tc.link+" is a symlink "+where, func(t *testing.T) {
				repo := t.TempDir()
				dest := filepath.Join(repo, "decoy")
				if where == "outside the checkout" {
					dest = t.TempDir()
				}
				if err := os.MkdirAll(filepath.Join(dest, tc.records), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dest, tc.records, "iss-9-decoy.md"), []byte(decoy), 0o644); err != nil {
					t.Fatal(err)
				}
				link := filepath.Join(repo, filepath.FromSlash(tc.link))
				if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(dest, link); err != nil {
					t.Skipf("symlink unsupported: %v", err)
				}
				before := treeEntries(t, dest)
				ir := filepath.Join(repo, LedgerRelPath)

				_, err := Capture(CaptureRequest{
					RepoRoot: repo, IssuesRoot: ir, Text: "b", Severity: SeverityMinor,
					Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "t",
				})
				if !errors.Is(err, ErrPathUnsafe) {
					t.Fatalf("Capture through a symlinked ledger directory: want ErrPathUnsafe, got %v", err)
				}
				if after := treeEntries(t, dest); !slices.Equal(before, after) {
					t.Fatalf("Capture wrote behind the link: before %v, after %v", before, after)
				}

				list, listErr := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
				refuseLeakedRecord(t, "List", list, marker)
				if !errors.Is(listErr, ErrPathUnsafe) {
					t.Fatalf("List through a symlinked ledger directory: want ErrPathUnsafe, got %v", listErr)
				}

				status, statusErr := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
				refuseLeakedRecord(t, "Status", status, marker)
				if !errors.Is(statusErr, ErrPathUnsafe) {
					t.Fatalf("Status through a symlinked ledger directory: want ErrPathUnsafe, got %v", statusErr)
				}

				if _, err := Promote(PromoteRequest{
					RepoRoot: repo, IssuesRoot: ir, ID: "iss-9", Grounds: "pursued: t",
				}); !errors.Is(err, ErrPathUnsafe) {
					t.Fatalf("Promote through a symlinked ledger directory: want ErrPathUnsafe, got %v", err)
				}
			})
		}
	}
}

// refuseLeakedRecord fails if what a read verb returned carries the decoy
// record's body. The refusal alone is not the claim under test: the point is
// that the records behind the link never reached a serialized result.
func refuseLeakedRecord(t *testing.T, what string, result any, marker string) {
	t.Helper()
	out, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), marker) {
		t.Fatalf("%s serialized a record from behind the link:\n%s", what, out)
	}
}

// treeEntries lists dir's contents recursively, relative and sorted, so a
// refusal can be held to having written nothing behind the link.
func treeEntries(t *testing.T, dir string) []string {
	t.Helper()
	var found []string
	if err := filepath.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		found = append(found, rel)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(found)
	return found
}
