package fsutil

import "testing"

// TestRedactRootFoldsRootSpellingWhenFS proves the developer-identity redactor is
// case-folding on a case-folding filesystem: a message that names $HOME in a case
// variant (as the shell or a syscall may echo it back on macOS/APFS) is still
// redacted, so the username does not leak. With the filesystem case-SENSITIVE the
// redactor keeps exact-match semantics, so two genuinely distinct paths that
// differ only in case are never merged.
func TestRedactRootFoldsRootSpellingWhenFS(t *testing.T) {
	restore := caseFoldingFS
	t.Cleanup(func() { caseFoldingFS = restore })

	root := "/Users/alice/repo"

	// A case-variant spelling of the root, both as a bare mention at a boundary
	// and as a prefix of a path under it.
	variantBare := "cannot access /Users/alice/REPO"
	variantUnder := "wrote /Users/alice/REPO/dist/out.bin"

	t.Run("folding FS redacts a case-variant root", func(t *testing.T) {
		caseFoldingFS = func() bool { return true }
		if got := RedactRoot(variantBare, root, "."); got != "cannot access ." {
			t.Errorf("RedactRoot(%q) = %q, want %q", variantBare, got, "cannot access .")
		}
		if got := RedactRoot(variantUnder, root, "."); got != "wrote ./dist/out.bin" {
			t.Errorf("RedactRoot(%q) = %q, want %q", variantUnder, got, "wrote ./dist/out.bin")
		}
	})

	t.Run("case-sensitive FS keeps exact-match semantics", func(t *testing.T) {
		caseFoldingFS = func() bool { return false }
		if got := RedactRoot(variantBare, root, "."); got != variantBare {
			t.Errorf("RedactRoot(%q) with fold off = %q, want it unchanged", variantBare, got)
		}
		// An exact-case root is still redacted with folding off.
		exact := "cannot access /Users/alice/repo"
		if got := RedactRoot(exact, root, "."); got != "cannot access ." {
			t.Errorf("RedactRoot(%q) with fold off = %q, want %q", exact, got, "cannot access .")
		}
	})
}
