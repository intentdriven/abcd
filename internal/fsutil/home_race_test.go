//go:build unix

package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// raceableHome lays out a home with a real ~/.abcd and, inside the same home, a
// "dotfiles checkout" directory, each holding its own copy of name. It returns
// the home and the dotfiles directory.
func raceableHome(t *testing.T, name string) (home, dotfiles string) {
	t.Helper()
	home = t.TempDir()
	dotfiles = filepath.Join(home, "dotfiles", "abcd")
	for _, dir := range []string{filepath.Join(home, ".abcd"), dotfiles} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeDeclaration(t, filepath.Join(home, ".abcd"), name, "/real\n")
	writeDeclaration(t, dotfiles, name, "/dotfiles\n")
	return home, dotfiles
}

// swapAbcdForLinkOnce installs the vetting hook so that, the first time
// ~/.abcd has been judged, the real directory is moved aside and ~/.abcd
// becomes a symlink into the dotfiles checkout — the same-uid race that falls
// between a check by path and a use by path.
func swapAbcdForLinkOnce(t *testing.T, home, dotfiles string) {
	t.Helper()
	swapped := false
	restore := SwapHomeScopeVettedForTest(func(dir string) {
		if swapped || dir != filepath.Join(home, ".abcd") {
			return
		}
		swapped = true
		if err := os.Rename(dir, filepath.Join(home, "moved-aside")); err != nil {
			t.Fatalf("swap: %v", err)
		}
		if err := os.Symlink(dotfiles, dir); err != nil {
			t.Fatalf("swap: %v", err)
		}
	})
	t.Cleanup(restore)
}

// TestReadHomeDeclarationRefusesAnAbcdHomeSwappedForALinkAfterItsCheck is
// iss-2609281310017733: ~/.abcd was a real directory when it was judged and a
// symlink into a dotfiles checkout by the time the declaration was opened. A
// check by path followed by a read by path reads the dotfiles copy; the read
// through the descriptor of the directory that was judged refuses it, naming
// the link, and returns no bytes.
func TestReadHomeDeclarationRefusesAnAbcdHomeSwappedForALinkAfterItsCheck(t *testing.T) {
	home, dotfiles := raceableHome(t, "trusted-roots")
	swapAbcdForLinkOnce(t, home, dotfiles)
	raw, refusal, err := ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024)
	if raw != nil || refusal != DeclarationBehindSymlink || !errors.Is(err, ErrHomeScopeSymlinked) {
		t.Fatalf("a ~/.abcd swapped for a link after its check must be refused as the link: refusal %d, err %v, raw %q", refusal, err, raw)
	}
}

// The file's own guards still hold on the descriptor route: a leaf that is a
// symlink is refused as not regular, and a leaf that is swapped again after
// every judgement is refused as swapped, even inside a real ~/.abcd. (A single
// benign swap is re-judged and read: TestReadHomeDeclarationReadsABenign-
// ReplacementAfterRevetting, iss-2609291157309818.)
func TestReadHomeDeclarationStillRefusesAHostileLeaf(t *testing.T) {
	home, dotfiles := raceableHome(t, "trusted-roots")
	if err := os.Symlink(filepath.Join(dotfiles, "trusted-roots"), filepath.Join(home, ".abcd", "linked")); err != nil {
		t.Fatal(err)
	}
	if raw, refusal, err := ReadHomeDeclaration(home, ".abcd/linked", 1024); refusal != DeclarationNotRegular || err == nil || raw != nil {
		t.Fatalf("a symlinked leaf must be refused as not regular: refusal %d, err %v, raw %q", refusal, err, raw)
	}

	prev := declarationVetted
	t.Cleanup(func() { declarationVetted = prev })
	declarationVetted = func(p string) {
		other := writeDeclaration(t, filepath.Join(home, ".abcd"), "other", "/swapped\n")
		if err := os.Rename(other, p); err != nil {
			t.Fatalf("swap: %v", err)
		}
	}
	raw, refusal, err := ReadHomeDeclaration(home, ".abcd/trusted-roots", 1024)
	if raw != nil || refusal != DeclarationUnreadable || !errors.Is(err, ErrDeclarationSwapped) {
		t.Fatalf("a leaf swapped after its judgement must be refused as swapped: refusal %d, err %v, raw %q", refusal, err, raw)
	}
}

// EnsureHomeScope creates each missing level, proves it and returns a root at
// the last; a home that is itself a link is the machine's layout and is
// walked; a symlinked level is refused by name with nothing created behind it;
// a non-directory level is not a real directory. OpenHomeScope creates
// nothing, so an absent level is os.IsNotExist.
func TestEnsureHomeScopeCreatesAndProvesEveryLevelBelowHome(t *testing.T) {
	realHome := t.TempDir()
	home := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(realHome, home); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenHomeScope(home, ".abcd/history"); !os.IsNotExist(err) {
		t.Fatalf("OpenHomeScope of an absent level must be not-exist and create nothing: %v", err)
	}
	root, err := EnsureHomeScope(home, ".abcd/history", 0o700)
	if err != nil {
		t.Fatalf("EnsureHomeScope under a home reached through a link: %v", err)
	}
	if err := root.WriteFile("index.json", []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root.Close()
	if _, err := os.Stat(filepath.Join(realHome, ".abcd", "history", "index.json")); err != nil {
		t.Fatalf("the root does not stand at ~/.abcd/history: %v", err)
	}
	if fi, err := os.Stat(filepath.Join(realHome, ".abcd")); err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatalf("a created level must carry perm: %v %v", fi, err)
	}

	linked, target := symlinkedHome(t)
	_, err = EnsureHomeScope(linked, ".abcd/history", 0o700)
	var le *HomeScopeLinkError
	if !errors.As(err, &le) || le.Link != "~/.abcd" {
		t.Fatalf("a symlinked ~/.abcd must be refused by name: %v", err)
	}
	if entries, _ := os.ReadDir(target); len(entries) != 0 {
		t.Fatalf("EnsureHomeScope created %q behind the link", entries[0].Name())
	}

	if err := os.Symlink(t.TempDir(), filepath.Join(realHome, ".abcd", "sub")); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenHomeScope(home, ".abcd/sub"); !errors.As(err, &le) || le.Link != "~/.abcd/sub" {
		t.Fatalf("a symlinked level below ~/.abcd must be refused by name: %v", err)
	}
	if err := os.WriteFile(filepath.Join(realHome, ".abcd", "file"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureHomeScope(home, ".abcd/file", 0o700); !errors.Is(err, ErrNotRealDir) {
		t.Fatalf("a non-directory level must be refused as not a real directory: %v", err)
	}
	for _, dir := range []string{"", "../x", "/etc", ".abcd/"} {
		if _, err := OpenHomeScope(home, dir); !errors.Is(err, os.ErrInvalid) {
			t.Errorf("OpenHomeScope(%q) = %v, want os.ErrInvalid", dir, err)
		}
	}
}

// A level swapped for a link after its judgement is refused by EnsureHomeScope
// as it is by the reader, and nothing is created behind the link.
func TestEnsureHomeScopeRefusesALevelSwappedForALinkAfterItsCheck(t *testing.T) {
	home, dotfiles := raceableHome(t, "path-entry")
	swapAbcdForLinkOnce(t, home, dotfiles)
	_, err := EnsureHomeScope(home, ".abcd/history", 0o700)
	if !errors.Is(err, ErrHomeScopeSymlinked) {
		t.Fatalf("a ~/.abcd swapped for a link after its check must be refused as the link: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dotfiles, "history")); !os.IsNotExist(err) {
		t.Fatalf("EnsureHomeScope created history behind the swapped link: %v", err)
	}
}

// TestReadHomeDeclarationDenyingJudgesTheModeOfTheFileItOpened: deny is judged
// on the opened file's fstat, not on the Lstat that vetted it. A 0600 file is
// read; the same file opened wider than 0600 — the one the Lstat judged, re-
// moded inside the window between that Lstat and the open — is refused as
// DeclarationExposed naming its mode, and no byte is returned. deny 0 (the
// plain ReadHomeDeclaration) keeps admitting a declaration others can read.
func TestReadHomeDeclarationDenyingJudgesTheModeOfTheFileItOpened(t *testing.T) {
	home, _ := raceableHome(t, "credentials.json")
	p := filepath.Join(home, ".abcd", "credentials.json")
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	if raw, refusal, err := ReadHomeDeclarationDenying(home, ".abcd/credentials.json", 1024, 0o077); refusal != DeclarationOK || err != nil || string(raw) != "/real\n" {
		t.Fatalf("a 0600 file must be read: refusal %d, err %v, raw %q", refusal, err, raw)
	}

	prev := declarationVetted
	t.Cleanup(func() { declarationVetted = prev })
	declarationVetted = func(string) {
		if err := os.Chmod(p, 0o644); err != nil {
			t.Fatalf("chmod: %v", err)
		}
	}
	raw, refusal, err := ReadHomeDeclarationDenying(home, ".abcd/credentials.json", 1024, 0o077)
	var mode *DeclarationModeError
	if raw != nil || refusal != DeclarationExposed || !errors.As(err, &mode) || mode.Perm != 0o644 {
		t.Fatalf("a file opened at 0644 must be refused on its fstat: refusal %d, err %v, raw %q", refusal, err, raw)
	}

	declarationVetted = prev
	if raw, refusal, err := ReadHomeDeclaration(home, ".abcd/credentials.json", 1024); refusal != DeclarationOK || err != nil || string(raw) != "/real\n" {
		t.Fatalf("with no mask a 0644 declaration is still read: refusal %d, err %v, raw %q", refusal, err, raw)
	}
}
