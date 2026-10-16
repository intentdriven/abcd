package guard

import (
	"os"
	"path/filepath"
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
