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
	for _, rel := range []string{".abcd/.work.local", "sub", "build", "x/staging/review-receipts", "x/review-receipts", "x/old"} {
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

// TestAStageCopyIsReadAsEachPlatformParsesItsWords: a clustered or abbreviated
// target directory, -T spelled long or short, a mode or owner abbreviated, an
// exact --strip, macOS's -S that takes no value, and a word no line fixes
// that may be -t, each read as the commands read them, so the source still
// lands where it lands; a move that names the local tier as a source removes
// it however -t is spelled.
func TestAStageCopyIsReadAsEachPlatformParsesItsWords(t *testing.T) {
	tree := landingTree(t)
	stage := NewFiles(tree).WithStage(filepath.Join(t.TempDir(), "stage"), nil, nil)
	for _, line := range []string{
		`cp -S x/f/review-receipts .abcd/.work.local/`,
		`install -S x/f/review-receipts .abcd/.work.local/`,
		`cp --t .abcd/.work.local x/f/review-receipts`,
		`cp -R --no-t x/staging .abcd/.work.local`,
		`install x/f/review-receipts .abcd/.work.local --m 600`,
		`install --strip-program=true --strip x/f/review-receipts .abcd/.work.local/`,
		`cp -R $X .abcd/.work.local x/f/review-receipts`,
		`mv -ft x/old .abcd/.work.local`,
		`mv --target-dir=x/old .abcd/.work.local`,
		`mv --targ=x/old .abcd/.work.local`,
		`mv --t x/old .abcd/.work.local`,
		`mv -ft x/old .abcd/.work.local && mv x/staging .abcd/.work.local`,
		`patch -x 0 -p1 -d .abcd/.work.local -i x/a.diff`,
		`patch --debug 0 -p1 -d .abcd/.work.local -i x/a.diff`,
		`patch --read-only ignore -p1 -d .abcd/.work.local -i x/a.diff`,
		`patch --reject-format unified -p1 -d .abcd/.work.local -i x/a.diff`,
	} {
		if a := approveShell(clean(), stage, line); a.Allow {
			t.Errorf("a stage's line was approved: %s", line)
		}
	}
	for _, line := range []string{
		`cp -S x/notes.txt .abcd/.work.local/`,
		`cp --t .abcd/.work.local x/notes.txt`,
		`install -m 644 x/notes.txt build/`,
		`mv -ft x/old x/notes.txt`,
		`patch -x 0 -p1 -d build -i x/a.diff`,
	} {
		if a := approveShell(clean(), stage, line); !a.Allow {
			t.Errorf("%q is refused: %s", line, a.Reason)
		}
	}
	bare := t.TempDir()
	if err := os.MkdirAll(filepath.Join(bare, "x", "staging", "review-receipts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(bare, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	fresh := NewFiles(bare).WithStage(filepath.Join(t.TempDir(), "stage"), nil, nil)
	for _, line := range []string{`mv x/staging .abcd/.work.local/`, `cp -R x/staging .abcd/.work.local/`} {
		if a := approveShell(clean(), fresh, line); a.Allow {
			t.Errorf("with no local tier yet, a stage's line was approved: %s", line)
		}
	}
}

// TestAnArchiveIsReadForWhereItsMembersCanLand: tar's and unzip's options are
// read cluster by cluster, so a glued file name is not a mode letter, a
// clustered -C or -d names the directory, a member kept with its absolute or
// parent path can land anywhere, and a listing, a test or a dry run writes
// nothing.
func TestAnArchiveIsReadForWhereItsMembersCanLand(t *testing.T) {
	tree := landingTree(t)
	stage := NewFiles(tree).WithStage(filepath.Join(t.TempDir(), "stage"), nil, nil)
	for _, line := range []string{
		`tar -xPf x/a.tar -C sub`,
		`tar xPf x/a.tar -C sub`,
		`tar --absolute-names -xf x/a.tar -C sub`,
		`tar -xfOld -C .abcd/.work.local`,
		`cd sub && tar -xC ../.abcd/.work.local -f ../x/a.tar`,
		`tar xCf .abcd/.work.local x/a.tar`,
		`unzip -: x/a.zip -d sub`,
		`unzip -o: x/a.zip -d sub`,
		`unzip -qd .abcd/.work.local x/a.zip`,
		`unzip -q x/a.zip -d.abcd/.work.local`,
	} {
		if a := approveShell(clean(), stage, line); a.Allow {
			t.Errorf("a stage's line was approved: %s", line)
		}
	}
	for _, line := range []string{
		`tar -xf x/a.tar -C sub`,
		`tar xCf sub x/a.tar`,
		`unzip -qd sub x/a.zip`,
	} {
		if a := approveShell(clean(), stage, line); !a.Allow {
			t.Errorf("%q is refused: %s", line, a.Reason)
		}
	}
	f := NewFiles(tree)
	for _, line := range []string{`tar -xfOld`, `tar -xPf x/a.tar -C build`, `unzip -: x/a.zip -d build`} {
		if a := approveShell(clean(), f, line); a.Allow {
			t.Errorf("a line was approved: %s", line)
		}
	}
	for _, line := range []string{
		`unzip -l x/a.zip`,
		`unzip -t x/a.zip`,
		`unzip -p x/a.zip notes.txt`,
		`tar -tf x/a.tar`,
		`tar -xOf x/a.tar notes.txt`,
		`tar xOf x/a.tar notes.txt`,
		`patch --dry-run -p1 -i x/a.diff`,
	} {
		if a := approveShell(clean(), f, line); !a.Allow {
			t.Errorf("%q is refused: %s", line, a.Reason)
		}
	}
}

// TestAStageCopyFedByXargsIsJudgedForAnySource: the words xargs adds, or puts
// in place of its replacement string, are names no line fixes, so what they
// carry lands under any name, and a destination they may be is not placed.
func TestAStageCopyFedByXargsIsJudgedForAnySource(t *testing.T) {
	tree := landingTree(t)
	stage := NewFiles(tree).WithStage(filepath.Join(t.TempDir(), "stage"), nil, nil)
	for _, line := range []string{
		`printf '%s\n' x/review-receipts | xargs mv -t .abcd/.work.local`,
		`printf '%s\n' x/review-receipts | xargs cp -R -t .abcd/.work.local`,
		`printf '%s\n' x/review-receipts | xargs -I{} cp -R {} .abcd/.work.local/`,
		`printf '%s\n' x/review-receipts | xargs -I % cp -R % .abcd/.work.local/`,
		`printf '%s\n' .abcd/.work.local | xargs cp -R x/review-receipts`,
		`printf '%s\n' x/review-receipts | xargs ln -s -t .abcd/.work.local`,
	} {
		if a := approveShell(clean(), stage, line); a.Allow {
			t.Errorf("a stage's line was approved: %s", line)
		}
	}
	for _, line := range []string{
		`printf '%s\n' x/notes.txt | xargs cp -t build`,
		`printf '%s\n' x/notes.txt | xargs -I{} cp {} build/`,
	} {
		if a := approveShell(clean(), stage, line); !a.Allow {
			t.Errorf("%q is refused: %s", line, a.Reason)
		}
	}
}

// TestALinkMadeEarlierOnTheLineIsFollowed: a later write, move or copy through
// a link the line makes first is judged where the link leads, whichever
// command made it; the link alone, and a link to an ordinary directory, stay
// open.
func TestALinkMadeEarlierOnTheLineIsFollowed(t *testing.T) {
	tree := landingTree(t)
	stage := NewFiles(tree).WithStage(filepath.Join(t.TempDir(), "stage"), nil, nil)
	for _, line := range []string{
		`ln -s .abcd/.work.local L && mv x/review-receipts L/`,
		`ln -s .abcd/.work.local L && mkdir -p L/approvals`,
		`ln -s .abcd/.work.local L; printf x > L/approvals/a`,
		`ln -s "$PWD/.abcd/.work.local" L && mkdir L/approvals`,
		`ln -s ../.abcd/.work.local x/L && cp -R x/review-receipts x/L/`,
		`ln -s .abcd/.work.local A && mv A B && mkdir B/approvals`,
		`cp -s ` + filepath.Join(tree, ".abcd", ".work.local") + ` L && mkdir L/approvals`,
		`ln -s .abcd/.work.local -t sub && mkdir sub/.work.local/approvals`,
	} {
		if a := approveShell(clean(), stage, line); a.Allow {
			t.Errorf("a stage's line was approved: %s", line)
		}
	}
	for _, line := range []string{
		`ln -s .abcd/.work.local L`,
		`ln -s build L && printf x > L/a`,
		`ln -s .abcd/.work.local L && printf x > L/scratch.txt`,
	} {
		if a := approveShell(clean(), stage, line); !a.Allow {
			t.Errorf("%q is refused: %s", line, a.Reason)
		}
	}
}
