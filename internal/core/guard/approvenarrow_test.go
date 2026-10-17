package guard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestApproveReadsAPagerOrSSHCommandAsTheCommandItRuns: a git command that
// sets core.pager or core.sshCommand for itself to a command the line fixes
// runs that command, so the setting is read as that command, the way a bang
// alias is, and approved when the command is clean.
func TestApproveReadsAPagerOrSSHCommandAsTheCommandItRuns(t *testing.T) {
	r, _ := gitRig(t)
	refused := refusedAs(gitConfigWriteEntryID)
	r.run(t, []approveCase{
		{`git -c core.pager=cat log -1`, nil},
		{`git -c core.pager='less -R' log`, nil},
		{`git -c core.pager= log`, nil},
		{`git -c core.sshCommand='ssh -i ~/.ssh/deploy_key -o IdentitiesOnly=yes' fetch origin`, nil},
		{`git -c core.pager='rm -rf ~' log`, refused},
		{`git -c core.pager='python3 -c "import os"' log`, refused},
		{`git -c core.sshCommand="$SSH" fetch`, refused},
		{`echo 'ssh "$@"' > s.sh && git -c core.sshCommand='sh s.sh' fetch`, refused},
		{`git -c core.sshCommand='ssh -v' fetch && git -c core.pager=cat log`, nil},
		{`git -c core.pager='git -c core.pager=cat log' log`, refused},
		{`git config core.pager cat`, refused},
		{`cd "$D" && git -c core.pager=cat log`, refused},
	})
}

// TestApproveLetsAGlobRemovalInsideTheTree: a glob removal whose fixed
// directory is inside the working tree and unprotected is approved, unless a
// protected name could match the glob: one it spells, one a leading dot it
// spells lets it reach, or one already standing in that directory.
func TestApproveLetsAGlobRemovalInsideTheTree(t *testing.T) {
	r, repo := gitRig(t)
	repo.Git("config", "core.hooksPath", "tools/hooks")
	for _, d := range []string{"build/obj", "tools/hooks", "vendor/lib", "dots"} {
		if err := os.MkdirAll(filepath.Join(r.dir, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(r.dir, "dots", ".envrc"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	r.run(t, []approveCase{
		{`rm -rf build/*`, nil},
		{`rm -f build/*.o`, nil},
		{`rm -f build/obj/*.o build/obj/*.d`, nil},
		{`rm -rf vendor/lib/*`, nil},
		{`cd build && rm -f *.o`, nil},
		{`rm -rf tools/*`, protectedOrRegistry},
		{`rm -rf tools/hook?`, protectedOrRegistry},
		{`rm -f build/.git*`, protectedOrRegistry},
		{`rm -f vendor/lib/.env*`, protectedOrRegistry},
		{`rm -rf dots/*`, protectedOrRegistry},
		{`rm -rf .git/*`, protectedOrRegistry},
		{`rm -rf ./*`, refusedAs(criticalRemovalEntryID)},
		{`rm -rf ` + filepath.Join(filepath.Dir(r.dir), "*"), refusedAs(criticalRemovalEntryID)},
		{`rm -f $D/*.o`, refusedAs(unfixedPathEntryID, criticalRemovalEntryID)},
		{`cd "$D" && rm -f *.o`, refusedAs(unfixedPathEntryID, criticalRemovalEntryID)},
	})
}

// TestApproveLeavesAScriptsFlagsToTheScript: the words after a script's
// name, or after python's -m module, are the script's, not the
// interpreter's, so a flag there that the interpreter would read as inline
// code is not inline code; a flag before the script still is.
func TestApproveLeavesAScriptsFlagsToTheScript(t *testing.T) {
	r := plainRig(t)
	r.run(t, []approveCase{
		{`node app.js -p 3000`, nil},
		{`node --enable-source-maps app.js -e prod`, nil},
		{`node -r ./setup.js app.js -p 3000`, nil},
		{`python3 -m pytest -c pytest.ini`, nil},
		{`python3 -u tools/run.py -c config.toml`, nil},
		{`python3 -W ignore script.py -c x`, nil},
		{`ruby script.rb -e staging`, nil},
		{`perl -Ilib t/run.pl -e x`, nil},
		{`node -p 3000 app.js`, refusedAs(inlineCodeEntryID)},
		{`node --title app.js -e 'process.exit(1)'`, refusedAs(inlineCodeEntryID)},
		{`python3 -W ignore -c 'import os'`, refusedAs(inlineCodeEntryID)},
		{`python3 -X dev -c 'import os'`, refusedAs(inlineCodeEntryID)},
		{`ruby -r json -e 'puts 1'`, refusedAs(inlineCodeEntryID)},
	})
}

// TestApproveJudgesAGlobRemovalsDirectoryThroughSymlinks: the directory a
// glob removal lists is judged where it leads, so one that is a symlink to a
// directory outside the working tree is refused, while one leading to a
// directory inside the tree is approved. And the glob is approved only when
// no earlier part of the line writes or removes a file, since an earlier
// part may make the directory a link (ln, cp -s, cp -l) after the line was
// judged.
func TestApproveJudgesAGlobRemovalsDirectoryThroughSymlinks(t *testing.T) {
	r, _ := gitRig(t)
	outside := filepath.Join(r.home, "Library")
	for _, d := range []string{filepath.Join(r.dir, "build"), outside} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(r.dir, "linkd")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(r.dir, "build"), filepath.Join(r.dir, "inlink")); err != nil {
		t.Fatal(err)
	}
	unfixed := refusedAs(unfixedPathEntryID)
	r.run(t, []approveCase{
		{`rm -rf build/*`, nil},
		{`rm -rf inlink/*`, nil},
		{`rm -f build/*.o inlink/*.o`, nil},
		{`cd build && rm -f *.o`, nil},
		{`rm -rf build/* && touch done`, nil},
		{`rm -rf linkd/*`, unfixed},
		{`rm -f linkd/*.md`, unfixed},
		{`cd linkd && rm -f *.md`, unfixed},
		{`mkdir fresh && ln -s ~/Library fresh/l && rm -rf fresh/l/*`, unfixed},
		{`ln -s ~/Library l2 && rm -rf l2/*`, unfixed},
		{`ln -s ~/Library l2; rm -rf l2/*`, unfixed},
		{`cp -s ~/Library/x build/x && rm -rf build/*`, unfixed},
		{`cp -l ~/Library/x build/x && rm -rf build/*`, unfixed},
		{`touch build/x && rm -rf build/*`, unfixed},
	})
}

// TestApproveRefusesAGlobWithAWildcardInADirectoryPart: a glob whose wildcard
// stands in a directory part (build/*/*), or that ends in a /, matches names
// in directories the line never names, and any of them may be a symlink out
// of the working tree, so the shell would expand it into the link's target.
// Such a removal is refused as a target the line does not fix; a glob whose
// wildcards stand only in its last name stays approved.
func TestApproveRefusesAGlobWithAWildcardInADirectoryPart(t *testing.T) {
	r, _ := gitRig(t)
	outside := filepath.Join(r.home, "Library")
	for _, d := range []string{filepath.Join(r.dir, "build", "obj"), outside} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(r.dir, "build", "l")); err != nil {
		t.Fatal(err)
	}
	unfixed := refusedAs(unfixedPathEntryID)
	r.run(t, []approveCase{
		{`rm -rf build/*`, nil},
		{`rm -f build/obj/*.o`, nil},
		{`rm -rf build/*/*`, unfixed},
		{`rm -rf ./build/*/*`, unfixed},
		{`rm -f build/*/x.o`, unfixed},
		{`rm -rf build/*/`, unfixed},
		{`rm -rf build/o?j/*`, unfixed},
		{`cd build && rm -rf */*`, unfixed},
	})
}

// TestApproveRefusesATargetHoldingAnUnquotedParen: zsh reads an unquoted ( in
// a word as the start of glob qualifiers or modifiers, which can change the
// path the word names: build/l(:A) is the real path a symlink leads to, out of
// the working tree, though the line spells a name inside it, and the words
// after the ( are not read as the command's own. So a command with an unquoted
// ( in a word, whatever the command and whatever follows the (, is refused as
// one whose targets the line does not fix; bash refuses such a line. A quoted
// or escaped ( is text, and a function definition's () is not a word, so both
// stay approved.
func TestApproveRefusesATargetHoldingAnUnquotedParen(t *testing.T) {
	r, _ := gitRig(t)
	outside := filepath.Join(r.home, "Library")
	for _, d := range []string{filepath.Join(r.dir, "build"), filepath.Join(r.dir, "a", "b"), outside} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(r.dir, "build", "l")); err != nil {
		t.Fatal(err)
	}
	if zsh, err := exec.LookPath("zsh"); err == nil {
		cmd := exec.Command(zsh, "-fc", "print -r -- build/l(:A)")
		cmd.Dir = r.dir
		out, err := cmd.Output()
		want, _ := filepath.EvalSymlinks(outside)
		if got := strings.TrimSpace(string(out)); err != nil || got != want {
			t.Fatalf("zsh read build/l(:A) as %q (%v), want %q, so this case tests nothing", got, err, want)
		}
	}
	unfixed := refusedAs(unfixedPathEntryID)
	r.run(t, []approveCase{
		{`rm -f 'a(b)'`, nil},
		{`rm -f "a(b)"`, nil},
		{`rm -f a\(b\)`, nil},
		{`touch 'x(.)'`, nil},
		{`f() { rm -f x; }; f`, nil},
		{`f () { rm -f x; }; f`, nil},
		{`function g() { true; }; g`, nil},
		{`function g () { true; }; g`, nil},
		{`rm -rf build/l(:A)`, unfixed},
		{`rm -f x(.)`, unfixed},
		{`rm -rf a/b(/)`, unfixed},
		{`rm -rf (build/l)(:A)`, unfixed},
		{`rm -f 'x'(.)`, unfixed},
		{`touch x(.)`, unfixed},
		{`echo hi > x(:A)`, unfixed},
		{`> x(.)`, unfixed},
		{`find a(:A) -delete`, unfixed},
		{`sed x(.) -i`, unfixed},
		{`cp a(.) -t dir`, unfixed},
		{`echo hi(:A)`, unfixed},
		{`ls x(/)`, unfixed},
	})
}

// TestApproveAllowsTheParensBashNeeds: two ( a command line needs in bash
// itself name no path and are not read as qualifiers: the (( of an
// arithmetic for-loop, and the optional ( before a case pattern. Each is
// approved where the keyword before it stands unquoted, and the words around
// it are still read; a ( anywhere else in a word is still refused.
func TestApproveAllowsTheParensBashNeeds(t *testing.T) {
	r := plainRig(t)
	unfixed := refusedAs(unfixedPathEntryID)
	r.run(t, []approveCase{
		{`for ((i=0;i<3;i++)); do echo $i; done`, nil},
		{`for((i=0;i<3;i++)); do echo $i; done`, nil},
		{"for ((i=0; i<3; i++))\ndo\n  echo $i\ndone", nil},
		{`if true; then for ((i=0;i<2;i++)); do echo $i; done; fi`, nil},
		{`case a in (a) echo x;; esac`, nil},
		{`case a in (a) echo x;; (b) echo y;; esac`, nil},
		{`case "$1" in (a|b) echo x;; (c) echo y;; esac`, nil},
		{"case a in\n  (a) echo x ;;\nesac", nil},
		{`if true; then case a in (a) echo x;; esac; fi`, nil},
		{`for ((i=0;i<3;i++)); do rm -f x(.); done`, unfixed},
		{`case a in (a) rm -f x(.);; esac`, unfixed},
		{`"for" ((i=0;i<3;i++)); do echo; done`, unfixed},
		{`echo for ((i=0))`, unfixed},
		{`'case' a in (a) echo x;; esac`, unfixed},
		{`case a "in" (a) echo x;; esac`, unfixed},
		{`echo case a in (a)`, unfixed},
	})
}
