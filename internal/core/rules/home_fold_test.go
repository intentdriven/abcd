//go:build unix

package rules

import (
	"path/filepath"
	"testing"
)

// TestTrustedRootsCaseVariantFollowsTheFoldPredicate: a trusted-roots entry
// spelled in a case variant of the marker re-admits it exactly when the
// filesystem folds case. The branch cannot be provoked on a case-sensitive
// host, so the predicate is forced both ways (iss-2609090951297149). The marker
// does not exist, so neither spelling resolves to the other.
func TestTrustedRootsCaseVariantFollowsTheFoldPredicate(t *testing.T) {
	parent := t.TempDir()
	marker := filepath.Join(parent, "checkout-absent")
	home := t.TempDir()
	t.Setenv("HOME", home)
	declareTrusted(t, home, filepath.Join(parent, "CHECKOUT-ABSENT"))
	real := caseFoldingFS
	t.Cleanup(func() { caseFoldingFS = real })

	caseFoldingFS = func() bool { return true }
	if ok, note := trustedRootDeclared(marker); !ok {
		t.Errorf("fold on: a case-variant declaration of the marker must re-admit it; note %q", note)
	}
	caseFoldingFS = func() bool { return false }
	if ok, note := trustedRootDeclared(marker); ok {
		t.Errorf("fold off: a case-variant declaration names a different root and must re-admit nothing; note %q", note)
	}
}
