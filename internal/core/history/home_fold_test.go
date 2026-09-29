//go:build unix

package history

import (
	"path/filepath"
	"testing"
)

// TestLocalTranscriptRootsCaseVariantFollowsTheFoldPredicate: a
// local-transcript-roots entry spelled in a case variant of the checkout pulls
// it in exactly when the filesystem folds case. The branch cannot be provoked
// on a case-sensitive host, so the predicate is forced both ways
// (iss-2609090951297149). The checkout does not exist, so neither spelling
// resolves to the other.
func TestLocalTranscriptRootsCaseVariantFollowsTheFoldPredicate(t *testing.T) {
	parent := t.TempDir()
	repo := filepath.Join(parent, "checkout-absent")
	home := t.TempDir()
	t.Setenv("HOME", home)
	declareLocal(t, home, filepath.Join(parent, "CHECKOUT-ABSENT"), 0o600)
	real := caseFoldingFS
	t.Cleanup(func() { caseFoldingFS = real })

	caseFoldingFS = func() bool { return true }
	if ok, note := localDeclared(repo); !ok {
		t.Errorf("fold on: a case-variant declaration of the checkout must pull it in; note %q", note)
	}
	caseFoldingFS = func() bool { return false }
	if ok, note := localDeclared(repo); ok {
		t.Errorf("fold off: a case-variant declaration names a different checkout and must pull nothing in; note %q", note)
	}
}
