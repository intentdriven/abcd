//go:build unix

package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// declareIn writes a home-scoped path declaration under home/.abcd/name with
// mode, and returns the rel HomeDeclarationNames reads it by.
func declareIn(t *testing.T, home, name, body string, mode os.FileMode) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(home, ".abcd", name)
	if err := os.WriteFile(p, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
	return ".abcd/" + name
}

// The fold branch of the declaration match is decided by the fold argument,
// never by the host: with it set a case-variant spelling of the target is the
// same path, and with it clear it is a different one. Both halves run on every
// host because the paths do not exist, so neither spelling can resolve to the
// other (iss-2609090951297149).
func TestHomeDeclarationNamesFoldsCaseOnlyWhenTold(t *testing.T) {
	home := t.TempDir()
	rel := declareIn(t, home, "roots", "/Example/Checkout-Absent\n", 0o600)
	target := "/example/checkout-absent"
	if declared, ignored := HomeDeclarationNames(home, rel, 1<<10, target, true); !declared || ignored != "" {
		t.Errorf("fold on: a case-variant declaration of the target must be honoured; declared %v, ignored %q", declared, ignored)
	}
	if declared, ignored := HomeDeclarationNames(home, rel, 1<<10, target, false); declared || ignored != "" {
		t.Errorf("fold off: a case-variant declaration names a different path; declared %v, ignored %q", declared, ignored)
	}
}

// An entry and the target are each compared as written and symlink-resolved,
// so a declaration spelled through a link names the resolved target and a
// resolved declaration names a target spelled through a link. The two callers
// once differed here: one resolved only the entry (iss-2609090951283654).
func TestHomeDeclarationNamesComparesBothSpellingsOfEachSide(t *testing.T) {
	realDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "via-link")
	if err := os.Symlink(realDir, link); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, entry, target string }{
		{"entry through a link, target resolved", link, realDir},
		{"entry resolved, target through a link", realDir, link},
	} {
		home := t.TempDir()
		rel := declareIn(t, home, "roots", tc.entry+"\n", 0o600)
		if declared, ignored := HomeDeclarationNames(home, rel, 1<<10, tc.target, false); !declared {
			t.Errorf("%s: not honoured (ignored %q)", tc.name, ignored)
		}
	}
}

// Only an absolute, uncommented line is an entry: a relative line names a
// different directory per caller, and a comment names nothing.
func TestHomeDeclarationNamesIgnoresRelativeAndCommentedLines(t *testing.T) {
	home := t.TempDir()
	rel := declareIn(t, home, "roots", "# /example/checkout-absent\nexample/checkout-absent\n\n", 0o600)
	if declared, _ := HomeDeclarationNames(home, rel, 1<<10, "/example/checkout-absent", false); declared {
		t.Fatal("a commented or relative line re-admitted the target")
	}
}

// A declaration that is present and not honoured says why, in a clause the
// caller renders in its own voice; an absent one is the ordinary case and
// says nothing. A present, unvouched-for declaration names nothing even when
// its text names the target.
func TestHomeDeclarationNamesSaysWhyAPresentDeclarationWasIgnored(t *testing.T) {
	home := t.TempDir()
	if declared, ignored := HomeDeclarationNames(home, ".abcd/roots", 1<<10, "/example/checkout-absent", false); declared || ignored != "" {
		t.Fatalf("an absent declaration: declared %v, ignored %q; want false and no diagnostic", declared, ignored)
	}
	rel := declareIn(t, home, "roots", "/example/checkout-absent\n", 0o622)
	declared, ignored := HomeDeclarationNames(home, rel, 1<<10, "/example/checkout-absent", false)
	if declared {
		t.Fatal("a declaration writable by others re-admitted the target")
	}
	if !strings.Contains(ignored, "writable by others") {
		t.Fatalf("the ignored declaration must say why: %q", ignored)
	}
}
