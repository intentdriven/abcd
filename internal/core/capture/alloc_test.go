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
	if err := cleanOrphanPlaceholders(repo, ir); err != nil {
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

// lockStatusDir captures one record into open/, resolves a second into
// resolved/, and then sets the named status directory's mode, restoring it when
// the test ends. It returns the open record's id.
func lockStatusDir(t *testing.T, repo, ir, status string, mode os.FileMode) string {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root reads a directory whatever its mode")
	}
	var ids []string
	for _, slug := range []string{"kept-open", "then-resolved"} {
		res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a finding", Severity: SeverityMinor,
			Category: "bug", Source: "manual-test", Slug: slug, FoundDuring: "t"})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, res.ID)
	}
	if _, err := Resolve(ResolveRequest{RepoRoot: repo, IssuesRoot: ir, ID: ids[1], Resolution: "fixed", Impact: "fix"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(ir, status)
	if err := os.Chmod(dir, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	return ids[0]
}

// TestAnUnreadableStatusDirectoryIsAFaultNotAnUnknownID is
// iss-2609261241121312: a status directory that cannot be read was skipped, so
// an id it may well hold was reported as not found in any status directory —
// ErrUnknownIssueID, a refusal of the caller's input (exit 2) — when the ledger
// could not be read, which is a fault. An absent status directory is still
// tolerated; any other read error is returned, never read as an empty folder.
func TestAnUnreadableStatusDirectoryIsAFaultNotAnUnknownID(t *testing.T) {
	for _, status := range []string{"open", "resolved"} {
		t.Run("findIssue with "+status+"/ unreadable", func(t *testing.T) {
			repo, ir := ledger(t)
			id := lockStatusDir(t, repo, ir, status, 0)
			_, _, err := findIssue(ir, id)
			if err == nil || errors.Is(err, ErrUnknownIssueID) || !errors.Is(err, fs.ErrPermission) {
				t.Fatalf("findIssue with %s/ unreadable: %v, want the read fault", status, err)
			}
			if !strings.Contains(err.Error(), status+"/") || strings.Contains(err.Error(), repo) {
				t.Errorf("the fault must name the status directory repo-relatively: %v", err)
			}
		})
	}
	t.Run("resolve with open/ unreadable", func(t *testing.T) {
		repo, ir := ledger(t)
		id := lockStatusDir(t, repo, ir, "open", 0)
		_, err := Resolve(ResolveRequest{RepoRoot: repo, IssuesRoot: ir, ID: id, Resolution: "fixed", Impact: "fix"})
		if err == nil || errors.Is(err, ErrUnknownIssueID) || errors.Is(err, ErrRequestRefused) {
			t.Fatalf("resolve with open/ unreadable: %v, want a fault", err)
		}
	})
	t.Run("wontfix with resolved/ unreadable", func(t *testing.T) {
		repo, ir := ledger(t)
		id := lockStatusDir(t, repo, ir, "resolved", 0)
		_, err := Wontfix(WontfixRequest{RepoRoot: repo, IssuesRoot: ir, ID: id, Reason: "not worth it"})
		if err == nil || errors.Is(err, ErrUnknownIssueID) {
			t.Fatalf("wontfix with resolved/ unreadable: %v, want a fault (the id may sit there too)", err)
		}
	})
	t.Run("list and status with resolved/ unreadable", func(t *testing.T) {
		repo, ir := ledger(t)
		lockStatusDir(t, repo, ir, "resolved", 0)
		if lr, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll}); err == nil {
			t.Errorf("list read an unreadable resolved/ as empty: %d issues", len(lr.Issues))
		}
		if st, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir}); err == nil {
			t.Errorf("status read an unreadable resolved/ as empty: %d resolved", st.ResolvedCount)
		}
	})
	t.Run("forced mint with resolved/ unreadable", func(t *testing.T) {
		repo, ir := ledger(t)
		lockStatusDir(t, repo, ir, "resolved", 0)
		if _, _, err := reservePath(repo, ir, "forced", "iss-42"); err == nil || errors.Is(err, ErrDuplicateIssueID) {
			t.Fatalf("a forced mint that could not read resolved/ reserved the id: %v", err)
		}
		if _, err := os.Stat(filepath.Join(ir, "open", "iss-42-forced.md")); !os.IsNotExist(err) {
			t.Errorf("a refused mint left a placeholder behind (stat err=%v)", err)
		}
	})
	t.Run("capture with open/ searchable but unreadable", func(t *testing.T) {
		repo, ir := ledger(t)
		lockStatusDir(t, repo, ir, "open", 0o300)
		if res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "another", Severity: SeverityMinor,
			Category: "bug", Source: "manual-test", Slug: "another", FoundDuring: "t"}); err == nil {
			t.Fatalf("a capture minted %s into an open/ its sweep and occupancy check could not read", res.ID)
		}
	})
	t.Run("an absent status directory is tolerated", func(t *testing.T) {
		repo, ir := ledger(t)
		id := lockStatusDir(t, repo, ir, "open", 0o755)
		if err := os.Remove(filepath.Join(ir, "wontfix")); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if _, _, err := findIssue(ir, id); err != nil {
			t.Fatalf("findIssue with wontfix/ absent: %v", err)
		}
		if _, _, err := findIssue(ir, "iss-42"); !errors.Is(err, ErrUnknownIssueID) {
			t.Fatalf("an id no present directory holds: %v, want ErrUnknownIssueID", err)
		}
		if _, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll}); err != nil {
			t.Fatalf("list with wontfix/ absent: %v", err)
		}
	})
}
