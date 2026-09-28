//go:build unix

package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// symlinkedHome lays out a home whose ~/.abcd is a symlink to a directory
// elsewhere (the dotfiles shape) and returns the home and the link's target.
func symlinkedHome(t *testing.T) (home, target string) {
	t.Helper()
	home, target = t.TempDir(), t.TempDir()
	if err := os.Symlink(target, filepath.Join(home, ".abcd")); err != nil {
		t.Fatal(err)
	}
	return home, target
}

// A declaration behind a symlinked ~/.abcd is refused before a byte is read,
// with the refusal naming the link; the same file in a real ~/.abcd is read.
// This is the rule AGENTS.md states for the rules loader, held by the one
// primitive every home-scoped declaration reads through (iss-2609281017573862).
func TestReadHomeDeclarationRefusesAFileBehindASymlinkedAbcdHome(t *testing.T) {
	home, target := symlinkedHome(t)
	writeDeclaration(t, target, "trusted-roots", "/example\n")
	raw, refusal, err := ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024)
	if refusal != DeclarationBehindSymlink || !errors.Is(err, ErrHomeScopeSymlinked) || raw != nil {
		t.Fatalf("a declaration behind a symlinked ~/.abcd must be refused: refusal %d, err %v, raw %q", refusal, err, raw)
	}
	if !strings.Contains(err.Error(), "~/.abcd is a symlink") {
		t.Fatalf("the refusal must name the link in tilde form: %v", err)
	}

	real := t.TempDir()
	if err := os.Mkdir(filepath.Join(real, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeDeclaration(t, filepath.Join(real, ".abcd"), "trusted-roots", "/example\n")
	if raw, refusal, err := ReadHomeDeclaration(real, ".abcd/trusted-roots", 1024); refusal != DeclarationOK || err != nil || string(raw) != "/example\n" {
		t.Fatalf("the same declaration in a real ~/.abcd must be read: refusal %d, err %v, raw %q", refusal, err, raw)
	}
}

// A symlinked ~/.abcd holding no such file reads as absent — the half of the
// rule that spares a dotfiles home nothing is declared in.
func TestReadHomeDeclarationReadsAnEmptySymlinkedAbcdHomeAsAbsent(t *testing.T) {
	home, _ := symlinkedHome(t)
	_, refusal, err := ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024)
	if refusal != DeclarationAbsent || !os.IsNotExist(err) {
		t.Fatalf("a symlinked ~/.abcd with no declaration must read as absent: refusal %d, err %v", refusal, err)
	}
}

// Every directory below ~/.abcd is judged too, and home itself never is: a
// home reached through a link is the machine's layout, not a declaration.
func TestHomeScopeLinkJudgesTheDirectoriesBelowHomeOnly(t *testing.T) {
	realHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(realHome, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	linkedHome := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(realHome, linkedHome); err != nil {
		t.Fatal(err)
	}
	if err := HomeScopeLink(linkedHome, ".abcd/credentials.json"); err != nil {
		t.Fatalf("a home reached through a link is not refused: %v", err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(realHome, ".abcd", "sub")); err != nil {
		t.Fatal(err)
	}
	err := HomeScopeLink(realHome, ".abcd/sub/file")
	var le *HomeScopeLinkError
	if !errors.As(err, &le) || le.Link != "~/.abcd/sub" {
		t.Fatalf("a symlinked directory below ~/.abcd must be refused by name: %v", err)
	}
	if err := HomeScopeLink(t.TempDir(), ".abcd/credentials.json"); err != nil {
		t.Fatalf("an absent ~/.abcd is not a link: %v", err)
	}
}

// TestHomeDeclarationsReadThroughReadHomeDeclaration is the one-canonical-
// primitive detector for the rule above: every home-scoped declaration outside
// this package reads through ReadHomeDeclaration, never through the bare
// ReadDeclaration, whose path argument cannot say where the home ends and so
// cannot judge ~/.abcd. Before the rule was routed, nine readers called the
// bare form and every one of them followed a symlinked ~/.abcd.
func TestHomeDeclarationsReadThroughReadHomeDeclaration(t *testing.T) {
	var offenders []string
	err := filepath.WalkDir(filepath.Join(".."), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filepath.Base(p) == "fsutil" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "fsutil.ReadDeclaration(") {
			offenders = append(offenders, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("home-scoped declarations read through the bare fsutil.ReadDeclaration (route through fsutil.ReadHomeDeclaration):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// TestHomeScopeLinkRefusesARelThatIsNotARelativePath: rel names a place in
// the caller's home, so a rel that is not a clean relative path is refused
// rather than judged. Passing it would say "no link here" about a path the
// walk never covered: an escaping rel ("../x/f") left home before any
// directory was judged, and an absolute one judged HOME ITSELF, the one
// directory the rule deliberately leaves alone. ReadHomeDeclaration refuses
// it the same way, as unreadable rather than as a symlink it did not find.
func TestHomeScopeLinkRefusesARelThatIsNotARelativePath(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"",
		"../x/f",
		".abcd/../../x/f",
		"/etc/f",
		"./.abcd/f",
		".abcd//f",
		".abcd/f/",
	} {
		err := HomeScopeLink(home, rel)
		if !errors.Is(err, os.ErrInvalid) {
			t.Errorf("HomeScopeLink(%q) = %v, want a refusal wrapping os.ErrInvalid", rel, err)
		}
		if errors.Is(err, ErrHomeScopeSymlinked) {
			t.Errorf("HomeScopeLink(%q) reports a symlink that is not there: %v", rel, err)
		}
		if _, refusal, err := ReadHomeDeclaration(home, rel, 1<<10); refusal != DeclarationUnreadable || !errors.Is(err, os.ErrInvalid) {
			t.Errorf("ReadHomeDeclaration(%q) = refusal %d, err %v; want DeclarationUnreadable wrapping os.ErrInvalid", rel, refusal, err)
		}
	}
	if err := HomeScopeLink(home, ".abcd/credentials.json"); err != nil {
		t.Fatalf("a valid rel under a real ~/.abcd must pass: %v", err)
	}
}
