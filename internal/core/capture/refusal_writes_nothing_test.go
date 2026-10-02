package capture

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRefusalOfAnUnheldIDWritesNothing holds every ledger verb that names an
// existing record to the capture chapter's promise for an id the ledger does
// not hold: exit 2, nothing written. The verbs located the record only under
// the ledger lock, after the mutation preamble had provisioned the ledger's
// directories and the lock file, so each refusal left
// .abcd/work/issues/.iss-alloc.lock (and the directory chain to it) behind in
// a checkout that had no ledger (iss-2609302305500526).
func TestRefusalOfAnUnheldIDWritesNothing(t *testing.T) {
	const (
		iss = "iss-2601010000000001"
		rdi = "rdi-2601010000000001"
	)
	grounds := "pursued: the fix holds, and a failing regression test would show it wrong"
	cases := []struct {
		verb string
		run  func(root string) error
	}{
		{"resolve", func(root string) error {
			_, err := Resolve(ResolveRequest{RepoRoot: root, ID: iss, Resolution: "fixed", Impact: "fix", Grounds: grounds})
			return err
		}},
		{"wontfix", func(root string) error {
			_, err := Wontfix(WontfixRequest{RepoRoot: root, ID: iss, Reason: "not acted on"})
			return err
		}},
		{"link", func(root string) error {
			_, err := Link(LinkRequest{RepoRoot: root, ID: iss, BlockedBy: []string{"iss-2601010000000002"}})
			return err
		}},
		{"defer", func(root string) error {
			_, err := Defer(DeferRequest{RepoRoot: root, ID: iss, After: "v0.1.0", Reason: "carried one cycle"})
			return err
		}},
		{"remedy", func(root string) error {
			_, err := SetRemedy(RemedyRequest{RepoRoot: root, ID: iss, Remedy: "fix the thing"})
			return err
		}},
		{"promote", func(root string) error {
			_, err := Promote(PromoteRequest{RepoRoot: root, ID: iss, Grounds: grounds})
			return err
		}},
		{"promote reading item", func(root string) error {
			_, err := Promote(PromoteRequest{RepoRoot: root, ID: rdi})
			return err
		}},
		{"disposition", func(root string) error {
			_, err := Disposition(DispositionRequest{RepoRoot: root, Item: rdi, State: "accepted", Grounds: grounds})
			return err
		}},
		{"admit", func(root string) error {
			_, err := Admit(AdmitRequest{RepoRoot: root, Item: rdi, Grounds: "the widened configuration is one the next release has to serve"})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.verb, func(t *testing.T) {
			root := t.TempDir()
			if err := tc.run(root); err == nil {
				t.Fatalf("%s of an id the ledger does not hold succeeded", tc.verb)
			}
			if _, err := os.Lstat(filepath.Join(root, ".abcd")); !os.IsNotExist(err) {
				var left []string
				_ = filepath.WalkDir(root, func(p string, _ os.DirEntry, _ error) error {
					if rel, _ := filepath.Rel(root, p); rel != "." {
						left = append(left, rel)
					}
					return nil
				})
				t.Errorf("%s refused but wrote %v; a refusal writes nothing", tc.verb, left)
			}
		})
	}
}
