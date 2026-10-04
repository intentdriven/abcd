package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// linkBareAbcdHome makes home's ~/.abcd.noindex a symlink to an EMPTY directory
// elsewhere (the dotfiles shape before abcd has written anything) and returns
// that directory, so a test can see whether anything was written through the
// link.
func linkBareAbcdHome(t *testing.T, home string) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), "dotfiles-abcd")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, abcdhome.Path(home)); err != nil {
		t.Fatal(err)
	}
	return target
}

// TestInstallRegistersNothingThroughASymlinkedAbcdHome is iss-2609281129171021:
// every reader and writer of a file abcd trusts in ~/.abcd.noindex refuses a symlinked
// ~/.abcd.noindex, and the history registry was the one writer left following it —
// `ahoy install` created ~/.abcd.noindex/history/index.json and the per-repo meta.json
// wherever the link pointed (a dotfiles checkout). The registration is skipped,
// nothing lands behind the link, and the install says why, naming the link and
// the remedy.
func TestInstallRegistersNothingThroughASymlinkedAbcdHome(t *testing.T) {
	home, _ := setupHermetic(t)
	target := linkBareAbcdHome(t, home)
	repo := committedRepo(t)
	t.Chdir(repo)

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("install wrote through the symlinked ~/.abcd.noindex: the link target holds %v", names)
	}
	var registry string
	for _, n := range res.Notes {
		if strings.Contains(n, "registration") && strings.Contains(n, "~/.abcd.noindex is a symlink") {
			registry = n
		}
	}
	if registry == "" {
		t.Fatalf("no note names the skipped registration and the symlinked ~/.abcd.noindex; notes = %v", res.Notes)
	}
	if !strings.Contains(registry, "replace the link with a real directory") {
		t.Errorf("the registration note must carry the remedy; got %q", registry)
	}
	if !strings.Contains(registry, "nothing was written") {
		t.Errorf("the registration note must say nothing was written; got %q", registry)
	}
}

// TestDetectReportsASymlinkedHistoryStoreAsADiagnostic: the detector must not
// answer a symlinked ~/.abcd.noindex with "~/.abcd.noindex/history/ not bootstrapped", a
// required gap install would then try, and refuse, to close on every run. It
// raises one diagnostic instead — not required, not resolvable — naming the
// link and the remedy.
func TestDetectReportsASymlinkedHistoryStoreAsADiagnostic(t *testing.T) {
	home, _ := setupHermetic(t)
	linkBareAbcdHome(t, home)

	det, err := Detect(managedRepoWithCommit(t))
	if err != nil {
		t.Fatal(err)
	}
	var diag *Gap
	for i, g := range det.Gaps {
		switch g.ID {
		case "history.bootstrap_missing", "history.meta_missing":
			t.Errorf("a symlinked ~/.abcd.noindex raised the actionable gap %q", g.ID)
		case "history.home_symlinked":
			diag = &det.Gaps[i]
		}
	}
	if diag == nil {
		t.Fatalf("no history.home_symlinked diagnostic; gaps = %v", gapIDs(det.Gaps))
	}
	if diag.Required || diag.Resolvable {
		t.Errorf("the diagnostic must be neither required nor resolvable: %+v", *diag)
	}
	if !strings.Contains(diag.Detail, "~/.abcd.noindex is a symlink") {
		t.Errorf("the diagnostic must name the symlinked ~/.abcd.noindex; got %q", diag.Detail)
	}
}

// TestInstallRegistersThroughAHomeThatIsItselfALink is the other half: home
// itself reached through a link (/home -> /usr/home, a relocated account) is
// the machine's layout, not a declaration, and the fsutil rule never judges
// it. The registry's non-following create must not refuse it either.
func TestInstallRegistersThroughAHomeThatIsItselfALink(t *testing.T) {
	realHome, _ := setupHermetic(t)
	linkedHome := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(realHome, linkedHome); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", linkedHome)
	repo := committedRepo(t)
	t.Chdir(repo)

	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(abcdhome.Path(realHome, "history", "index.json")); err != nil {
		t.Fatalf("a home reached through a link must still be registered: %v; notes = %v", err, res.Notes)
	}
}

// managedRepoWithCommit is a managed repository (a marker block) with a root
// commit, so the detector reaches the history-store checks with a sha.
func managedRepoWithCommit(t *testing.T) string {
	t.Helper()
	repo := committedRepo(t)
	body := "# Project\n\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n"
	if err := os.WriteFile(filepath.Join(repo, "CLAUDE.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}
