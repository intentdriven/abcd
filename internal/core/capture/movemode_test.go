package capture

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStatusMoveKeepsRecordMode (iss-2610181517504705): a status move writes
// the record at its new path with the mode it had at its old one, so a record
// kept 0600 (as the embargo store keeps its private records) is still 0600
// once resolved or won't-fixed, and one at the ledger's 0644 stays 0644.
func TestStatusMoveKeepsRecordMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode os.FileMode
		move func(repo, ir, id string) (TransitionResult, error)
	}{
		{"resolve 0600", 0o600, func(repo, ir, id string) (TransitionResult, error) {
			return Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: id, Resolution: "patched", Impact: "fix"})
		}},
		{"wontfix 0600", 0o600, func(repo, ir, id string) (TransitionResult, error) {
			return Wontfix(WontfixRequest{Grounds: "declined: we expect this to stay out of scope for the foreseeable cycle",
				RepoRoot: repo, IssuesRoot: ir, ID: id, Reason: "platform constraint"})
		}},
		{"resolve 0644", 0o644, func(repo, ir, id string) (TransitionResult, error) {
			return Resolve(ResolveRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: id, Resolution: "patched", Impact: "fix"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, ir := ledger(t)
			res, err := testCapture(CaptureRequest{
				RepoRoot: repo, IssuesRoot: ir, Text: "body", Severity: SeverityMajor,
				Category: "bug", Source: "manual-test", Slug: "fixme", FoundDuring: "test",
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(filepath.Join(repo, res.Path), tc.mode); err != nil {
				t.Fatal(err)
			}
			tr, err := tc.move(repo, ir, res.ID)
			if err != nil {
				t.Fatal(err)
			}
			fi, err := os.Stat(filepath.Join(repo, tr.Path))
			if err != nil {
				t.Fatal(err)
			}
			if got := fi.Mode().Perm(); got != tc.mode {
				t.Fatalf("the moved record's mode is %v, want %v", got, tc.mode)
			}
		})
	}
}

// TestStatusMoveOntoExistingDestinationTakesSourceMode: a status move whose
// destination already holds a file (a copy a stopped move or a hand left
// there) writes the record with the source's mode, never the mode the file
// it replaces had: a record kept 0600 is not widened by a 0644 copy.
func TestStatusMoveOntoExistingDestinationTakesSourceMode(t *testing.T) {
	repo, ir := ledger(t)
	res, err := testCapture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "body", Severity: SeverityMajor,
		Category: "bug", Source: "manual-test", Slug: "fixme", FoundDuring: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(repo, res.Path)
	if err := os.Chmod(src, 0o600); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(ir, statusDirName[StateResolved], filepath.Base(src))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("a stale copy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	content, sum, err := readWithChecksum(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := commitTransition(repo, ir, src, dst, content, sum); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o600 {
		t.Fatalf("the moved record's mode is %v, want the source's 0600", got)
	}
	if data, err := os.ReadFile(dst); err != nil || string(data) != content {
		t.Fatalf("the destination holds %q, %v; want the record", data, err)
	}
}
