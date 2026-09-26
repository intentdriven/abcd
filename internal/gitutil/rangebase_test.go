package gitutil_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestRefIsSafe pins the argument-injection guard every positional ref goes
// through: a ref beginning with '-' reaches git as an option, and no legitimate
// ref name starts with one.
func TestRefIsSafe(t *testing.T) {
	for ref, want := range map[string]bool{
		"main":                      true,
		"origin/main":               true,
		"HEAD":                      true,
		"HEAD~2":                    true,
		"0123456789abcdef":          true,
		"":                          false,
		"-x":                        false,
		"--output=/tmp/owned":       false,
		"--end-of-options":          false,
		"-":                         false,
		strings.Repeat("a", 40):     true,
		"--upload-pack=touch owned": false,
	} {
		if got := gitutil.RefIsSafe(ref); got != want {
			t.Errorf("RefIsSafe(%q) = %v, want %v", ref, got, want)
		}
	}
}

// TestIsNullOID pins the all-zeroes placeholder a forge hands a range gate when
// a push has no predecessor to name. It resolves to nothing, so a range gate
// must see it for what it is before handing it to git.
func TestIsNullOID(t *testing.T) {
	for s, want := range map[string]bool{
		strings.Repeat("0", 40):       true,
		strings.Repeat("0", 64):       true,
		strings.Repeat("0", 39):       false,
		strings.Repeat("0", 41):       false,
		"":                            false,
		strings.Repeat("0", 39) + "1": false,
	} {
		if got := gitutil.IsNullOID(s); got != want {
			t.Errorf("IsNullOID(%q) = %v, want %v", s, got, want)
		}
	}
}

// TestResolveRangeBase is the one derivation of a range gate's base: an empty
// value or the null placeholder is "no usable base for this event" (skipped,
// said aloud by the caller), an option-shaped value is refused, a value that
// names no commit is a fault, and anything else resolves to a full object name.
func TestResolveRangeBase(t *testing.T) {
	repo := newRepo(t, "")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, repo)
	head, err := gitutil.Run(repo, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	for _, raw := range []string{"", "   ", strings.Repeat("0", 40), strings.Repeat("0", 64)} {
		sha, usable, err := gitutil.ResolveRangeBase(repo, raw)
		if err != nil || usable || sha != "" {
			t.Errorf("ResolveRangeBase(%q) = (%q, %v, %v), want (\"\", false, nil): no usable base is a skip, not a fault", raw, sha, usable, err)
		}
	}

	sha, usable, err := gitutil.ResolveRangeBase(repo, "HEAD")
	if err != nil || !usable || sha != head {
		t.Errorf("ResolveRangeBase(HEAD) = (%q, %v, %v), want (%q, true, nil)", sha, usable, err, head)
	}

	for _, raw := range []string{"-x", "--output=owned", "no-such-branch", "7c2a4e6b8d0f1937a5c3e9b1d7f5a2c4e6b8d0f2"} {
		if sha, usable, err := gitutil.ResolveRangeBase(repo, raw); err == nil {
			t.Errorf("ResolveRangeBase(%q) = (%q, %v, nil), want an error: a base that names no commit is a fault the gate cannot judge past", raw, sha, usable)
		}
	}
}

// TestRequireFullHistory refuses a shallow checkout by name: past the graft a
// commit's parent is absent, so a gate comparing each commit with its parents
// would read every change as a whole-file add and cover nothing.
func TestRequireFullHistory(t *testing.T) {
	repo := newRepo(t, "")
	for i, body := range []string{"one\n", "two\n"} {
		if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		commitAll(t, repo)
		_ = i
	}
	if err := gitutil.RequireFullHistory(repo); err != nil {
		t.Fatalf("RequireFullHistory(full clone) = %v, want nil", err)
	}

	shallow := filepath.Join(t.TempDir(), "shallow")
	if out, err := runGit(t, filepath.Dir(shallow), "clone", "-q", "--depth", "1", "file://"+repo, shallow); err != nil {
		t.Fatalf("git clone --depth 1: %v: %s", err, out)
	}
	err := gitutil.RequireFullHistory(shallow)
	if !errors.Is(err, gitutil.ErrShallowCheckout) {
		t.Fatalf("RequireFullHistory(shallow clone) = %v, want ErrShallowCheckout", err)
	}

	if err := gitutil.RequireFullHistory(t.TempDir()); err == nil || errors.Is(err, gitutil.ErrShallowCheckout) {
		t.Fatalf("RequireFullHistory(not a repository) = %v, want a probe failure that is not mistaken for shallow or for full", err)
	}
}

// TestRunCappedBytesKeepsTheBytes pins the one difference from RunCapped: the
// output comes back exactly as git wrote it. A file's trailing blank lines and a
// last line's trailing whitespace are content to a caller that counts lines, and
// a trim changes the count.
func TestRunCappedBytesKeepsTheBytes(t *testing.T) {
	repo := newRepo(t, "")
	body := "  leading\ntrailing   \n\n\n"
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, repo)
	out, err := gitutil.RunCappedBytes(repo, 1<<20, "cat-file", "blob", "HEAD:a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != body {
		t.Fatalf("RunCappedBytes = %q, want the blob verbatim %q", out, body)
	}
	if _, err := gitutil.RunCappedBytes(repo, 4, "cat-file", "blob", "HEAD:a.txt"); err == nil {
		t.Fatal("RunCappedBytes over the cap returned no error; a truncated blob is a wrong one")
	}
}
