package guard

import (
	"os"
	"path/filepath"
	"testing"
)

// landingTree is a working tree whose local tier, a subdirectory and a build
// directory stand, so a destination that names one is a directory now.
func landingTree(t *testing.T) string {
	t.Helper()
	tree := t.TempDir()
	for _, rel := range []string{".abcd/.work.local", "sub", "build", "x/staging/review-receipts", "x/review-receipts"} {
		if err := os.MkdirAll(filepath.Join(tree, filepath.FromSlash(rel)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return tree
}

// TestAStageMoveCopyOrLinkIsJudgedWhereItsSourceLands: a source moved, copied
// or linked into a directory lands under its own name there, and what an
// archive, a copy of a directory's contents or a glob puts in a directory is
// judged by every name it can put there, so a gate reached that way is refused
// as a gate written directly is, and so is a removal of a directory holding
// one; a source landing under an ordinary name stays open.
func TestAStageMoveCopyOrLinkIsJudgedWhereItsSourceLands(t *testing.T) {
	tree := landingTree(t)
	stage := NewFiles(tree).WithStage(filepath.Join(t.TempDir(), "stage"), nil, nil)
	for _, line := range []string{
		`mv x/review-receipts .abcd/.work.local/`,
		`mv x/review-receipts .abcd/.work.local`,
		`cp -R x/review-receipts .abcd/.work.local/`,
		`mv -t .abcd/.work.local x/review-receipts`,
		`cp -R -t .abcd/.work.local x/review-receipts`,
		`mv x/review-receipts x/notes.txt .abcd/.work.local`,
		`ln -s ../../x/review-receipts .abcd/.work.local/`,
		`ln -s ../../x/review-receipts -t .abcd/.work.local`,
		`install -d .abcd/.work.local/approvals .abcd/.work.local/scratch`,
		`cp -R x/staging/* .abcd/.work.local/`,
		`cp -R x/staging/review-* .abcd/.work.local/`,
		`cp -R x/staging/. .abcd/.work.local/`,
		`cp -R x/staging/ .abcd/.work.local/`,
		`rsync -a x/staging/ .abcd/.work.local/`,
		`ditto x/staging .abcd/.work.local`,
		`tar -xf x/a.tar -C .abcd/.work.local`,
		`unzip x/a.zip -d .abcd/.work.local`,
		`patch -p1 -d .abcd/.work.local -i x/a.diff`,
		`mv x/staging/.abcd sub/`,
		`cp -R x/staging/.abcd sub`,
		`cp -R x/staging/. .`,
		`tar -xf x/a.tar`,
		`git clone x/repo.git .abcd/.work.local/approvals`,
		`rm -rf .abcd/.work.local`,
		`mv .abcd/.work.local x/old-tier`,
	} {
		if a := approveShell(clean(), stage, line); a.Allow {
			t.Errorf("a stage's line was approved: %s", line)
		}
	}
	for _, line := range []string{
		`mv x/notes.txt .abcd/.work.local/`,
		`cp -R x/staging .abcd/.work.local/`,
		`cp x/staging/*.md .abcd/.work.local/`,
		`ln -s ../../x/notes.txt .abcd/.work.local/`,
		`cp a.go ./`,
		`cp a.go b.go`,
		`mv x/notes.txt .`,
		`cp -R x/staging sub/`,
		`rsync -a x/staging/ build/`,
		`tar -xf x/a.tar -C build`,
		`unzip x/a.zip -d build`,
		`install -d build/a build/b`,
		`patch a.go x/a.diff`,
		`rm -rf .abcd/.work.local/scratch`,
		`rm -rf x/review-receipts`,
	} {
		if a := approveShell(clean(), stage, line); !a.Allow {
			t.Errorf("%q is refused: %s", line, a.Reason)
		}
	}
}

// TestACopyIntoADirectoryIsJudgedForTheProtectedPathsBelowIt: in any session, a
// source copied into a directory under a protected name, and a tree unpacked
// or copied whole into a directory that holds a protected path, are refused;
// the same writers aimed at an ordinary directory stay open.
func TestACopyIntoADirectoryIsJudgedForTheProtectedPathsBelowIt(t *testing.T) {
	tree := landingTree(t)
	f := NewFiles(tree)
	for _, line := range []string{
		`cp -R x/staging/.git ./`,
		`cp -R x/staging/.git .`,
		`cp -R x/staging/.vscode sub/`,
		`cp -R x/staging/.git* sub/`,
		`rsync -a x/staging/ ./`,
		`cp -R x/staging/. ./`,
		`tar -xf x/a.tar`,
		`unzip x/a.zip -d .`,
		`ditto x/staging .`,
	} {
		if a := approveShell(clean(), f, line); a.Allow {
			t.Errorf("a line was approved: %s", line)
		}
	}
	for _, line := range []string{
		`cp -R x/staging sub/`,
		`cp -R x/staging/. build/`,
		`rsync -a x/staging/ build/`,
		`tar -xf x/a.tar -C build`,
		`cp x/staging/*.go sub/`,
		`cp x/notes.txt .`,
		`patch a.go x/a.diff`,
	} {
		if a := approveShell(clean(), f, line); !a.Allow {
			t.Errorf("%q is refused: %s", line, a.Reason)
		}
	}
}
