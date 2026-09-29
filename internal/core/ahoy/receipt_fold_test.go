package ahoy

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestUnderFoldsCaseOnFoldingFS proves receipt classification's containment is
// case-folding on a case-folding filesystem: a case-variant repo/home root still
// classifies as "under", so the receipt is redacted rather than leaking the
// absolute developer-identity path. With the filesystem case-SENSITIVE the
// compare stays byte-exact, so two genuinely distinct directories are never
// merged (iss-2608270908340925).
func TestUnderFoldsCaseOnFoldingFS(t *testing.T) {
	restore := caseFoldingFS
	t.Cleanup(func() { caseFoldingFS = restore })

	root := filepath.FromSlash("/Users/alice/repo")
	variant := filepath.FromSlash("/Users/alice/REPO/.abcd/config.json")
	sibling := filepath.FromSlash("/Users/alice/other/x")

	t.Run("folding FS classifies a case-variant path as under", func(t *testing.T) {
		caseFoldingFS = func() bool { return true }
		if !under(root, variant) {
			t.Errorf("under(%q, %q) = false on a folding FS, want true", root, variant)
		}
		if under(root, sibling) {
			t.Errorf("under(%q, %q) = true, want false — a true sibling is not under", root, sibling)
		}
	})

	t.Run("case-sensitive FS keeps byte-exact semantics", func(t *testing.T) {
		caseFoldingFS = func() bool { return false }
		if under(root, variant) {
			t.Errorf("under(%q, %q) = true with fold off, want false", root, variant)
		}
		exact := filepath.FromSlash("/Users/alice/repo/.abcd/config.json")
		if !under(root, exact) {
			t.Errorf("under(%q, %q) = false with fold off, want true (exact case is still under)", root, exact)
		}
	})
}

// TestReceiptPathRedactsCaseVariantRepoRoot is the end-to-end half: a write whose
// absolute path spells the repo root in a case variant must not surface the
// developer-identity root on the receipt on a case-folding filesystem — the
// classifier folds, so the entry comes out relative rather than as an absolute
// /Users/<name>/… path. (The exact repo-relative rendering is fsutil.RepoRel's
// unfolded concern, outside this classifier's scope; the contract asserted here
// is the iss-177 identity guarantee.)
func TestReceiptPathRedactsCaseVariantRepoRoot(t *testing.T) {
	restore := caseFoldingFS
	t.Cleanup(func() { caseFoldingFS = restore })

	repo := filepath.FromSlash("/Users/alice/code/proj")
	identityRoot := filepath.FromSlash("/Users/alice")
	in := filepath.FromSlash("/Users/alice/code/PROJ/.abcd/config.json")

	caseFoldingFS = func() bool { return true }
	got := receiptPath(repo, in)
	if filepath.IsAbs(got) {
		t.Errorf("receiptPath(%q, %q) = %q, want a redacted (non-absolute) entry", repo, in, got)
	}
	if strings.Contains(got, identityRoot) {
		t.Errorf("receiptPath(%q, %q) = %q, leaks the developer-identity root %q", repo, in, got, identityRoot)
	}
}
