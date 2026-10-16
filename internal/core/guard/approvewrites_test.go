package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The approver's own checks on writes and removals read the guard's parse of
// the line: the targets its write model resolves (redirections and the
// listed writers), the operands of the removers, and `git config`'s scope and
// key. A target the line does not fix is refused; a fixed one is compared
// with the protected paths after its existing directories are resolved
// through symlinks, folding case where the filesystem folds it.

// approveCase is one line and the ids it may be refused under; none means it
// is approved.
type approveCase struct {
	line string
	ids  []string
}

// approveRig is a working tree, a home and the files the approver reads.
type approveRig struct {
	dir, home string
}

func (r approveRig) files() *Files {
	f := NewFiles(r.dir)
	f.home = r.home
	return f
}

// run judges each case and fails on any answer the case does not expect.
func (r approveRig) run(t *testing.T, cases []approveCase) {
	t.Helper()
	for _, c := range cases {
		a := approveShell(clean(), r.files(), c.line)
		switch {
		case len(c.ids) == 0 && !a.Allow:
			t.Errorf("%q: refused (%s): %s", c.line, a.EntryID, a.Reason)
		case len(c.ids) > 0 && a.Allow:
			t.Errorf("%q was approved; want a refusal under %v", c.line, c.ids)
		case len(c.ids) > 0 && !containsString(c.ids, a.EntryID) && !containsString(Defaults().ids(), a.EntryID):
			// Where the guard's own registry refuses the line first, its
			// entry speaks; either way it is refused.
			t.Errorf("%q: refused under %q, want %v (%s)", c.line, a.EntryID, c.ids, a.Reason)
		}
	}
}

// plainRig is a working tree outside any repository, with its own home.
func plainRig(t *testing.T) approveRig {
	t.Helper()
	root := t.TempDir()
	r := approveRig{dir: filepath.Join(root, "repo"), home: filepath.Join(root, "home")}
	for _, d := range []string{r.dir, r.home, filepath.Join(r.dir, ".abcd"), filepath.Join(r.dir, "sub")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// gitRig is a repository's working tree, with its own home.
func gitRig(t *testing.T) (approveRig, *gittest.Repo) {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write("README", "x\n")
	repo.Commit("seed")
	home := t.TempDir()
	return approveRig{dir: repo.Root(), home: home}, repo
}

func refusedAs(ids ...string) []string { return ids }

var protectedOrRegistry = refusedAs(protectedPathEntryID)

// TestApproveReadsQuotingAndBackslashesFromTheParse: a protected target is the
// same file however the line quotes it or puts backslashes in it, and a line
// that only names a protected path, without writing it, is not a write.
func TestApproveReadsQuotingAndBackslashesFromTheParse(t *testing.T) {
	r := plainRig(t)
	r.run(t, []approveCase{
		{`echo '{}' > '.mcp.json'`, protectedOrRegistry},
		{`echo '{}' > ".m"'cp'.json`, protectedOrRegistry},
		{`echo '{}' > \.mcp\.json`, protectedOrRegistry},
		{`echo '{}' > ./sub/../.mcp.json`, protectedOrRegistry},
		{`echo x >> ~/.zsh\rc`, protectedOrRegistry},
		{`printf x | tee -a "$HOME/.bashrc"`, refusedAs(protectedPathEntryID, unfixedPathEntryID)},
		{`echo '{}' > .abcd/guard.json`, protectedOrRegistry},
		{`echo '{}' > ./.abcd/./guard.json`, protectedOrRegistry},
		{`cp x.json ".abcd/guard.json"`, protectedOrRegistry},
		{`sed -i.bak s/a/b/ .abcd/guard.json`, protectedOrRegistry},
		{`curl -o ~/.profile https://example.com/p`, protectedOrRegistry},
		{`dd if=x of=.claude/settings.json`, protectedOrRegistry},
		{`touch ~/.zshenv`, protectedOrRegistry},
		{`mkdir -p .husky && touch .husky/pre-push`, protectedOrRegistry},
		{`chmod +x ~/.zprofile`, protectedOrRegistry},
		{`echo 1 > .envrc`, protectedOrRegistry},
		{`rm -f .mcp.json`, protectedOrRegistry},
		{`rm -rf .abcd`, protectedOrRegistry},
		{`mv .abcd/guard.json /tmp/guard.json`, protectedOrRegistry},
		{`ln -s ~/.zshrc rc`, protectedOrRegistry},
		{`sh -c 'echo x >> ~/.bash_profile'`, protectedOrRegistry},
		// Named, not written.
		{`cat .mcp.json`, nil},
		{`git log -- .abcd/guard.json .claude/settings.json`, nil},
		{`echo "the .zshrc is read at start-up"`, nil},
		{`ls -la ~/.zshrc .husky`, nil},
		{`grep -n hooks .git/config > hooks.txt`, nil},
		{`cp .mcp.json mcp.copy`, nil},
		{`echo x > sub/notes.txt 2>/dev/null`, nil},
		{`echo x > sub/notes.txt 2>&1`, nil},
		{`touch sub/new.txt && chmod 600 sub/new.txt`, nil},
	})
}

// TestApproveFoldsCaseWhereTheFilesystemDoes: on a filesystem that folds
// case, .MCP.json is .mcp.json; on one that does not, it is another file.
func TestApproveFoldsCaseWhereTheFilesystemDoes(t *testing.T) {
	r := plainRig(t)
	folding := approveFoldsCase
	t.Cleanup(func() { approveFoldsCase = folding })
	lines := []string{`echo '{}' > .MCP.json`, `echo x > .ABCD/Guard.json`, `echo x >> ~/.ZSHRC`, `rm -rf .Abcd`}
	approveFoldsCase = func() bool { return true }
	var cases []approveCase
	for _, l := range lines {
		cases = append(cases, approveCase{l, protectedOrRegistry})
	}
	r.run(t, cases)
	approveFoldsCase = func() bool { return false }
	cases = cases[:0]
	for _, l := range lines {
		cases = append(cases, approveCase{l, nil})
	}
	r.run(t, cases)
}

// TestApproveRefusesTargetsTheLineDoesNotFix: a write or a removal whose
// target is a substitution's output, a variable's value or a glob, or a
// relative path after a change of directory the line does not fix, may be
// any file, a protected one included.
func TestApproveRefusesTargetsTheLineDoesNotFix(t *testing.T) {
	r := plainRig(t)
	notFixed := refusedAs(unfixedPathEntryID, criticalRemovalEntryID)
	r.run(t, []approveCase{
		{`echo x > $F`, notFixed},
		{`echo x > "$F"`, notFixed},
		{`echo x > "${OUT:-out.txt}"`, notFixed},
		{`echo x > "$(cat target)"`, notFixed},
		{"echo x > `cat target`", notFixed},
		{`echo x > out.*`, notFixed},
		{`cp a.txt $DEST`, notFixed},
		{`tee $(ls -d sub)/x.txt < in`, notFixed},
		{`cd "$D" && echo x > out.txt`, notFixed},
		{`cd $(mktemp -d) && touch x`, notFixed},
		{`touch $(date +%s).txt`, notFixed},
		{`rm "$x"/y`, notFixed},
		{`mv $a b.txt`, notFixed},
		{`ln -s "$T" sub/l`, notFixed},
		// Fixed by the line.
		{`cd sub && echo x > out.txt`, nil},
		{`echo x > sub/out.txt`, nil},
		{`rm -f sub/out.txt`, nil},
		// A glob removal inside the tree that no protected name can match.
		{`rm sub/*.o`, nil},
	})
}

// TestApproveResolvesSymlinkedDirectories: a target reached through a link,
// in a directory or as the file itself, is judged as the file it reaches.
func TestApproveResolvesSymlinkedDirectories(t *testing.T) {
	r, _ := gitRig(t)
	for link, to := range map[string]string{
		"g":       filepath.Join(r.dir, ".git"),
		"cfg":     filepath.Join(r.dir, ".abcd"),
		"rc":      filepath.Join(r.home, ".zshrc"),
		"homedir": r.home,
	} {
		if err := os.MkdirAll(filepath.Join(r.dir, ".abcd"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(r.home, ".zshrc"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(to, filepath.Join(r.dir, link)); err != nil {
			t.Fatal(err)
		}
	}
	r.run(t, []approveCase{
		{`cp hook.sh g/hooks/pre-commit`, protectedOrRegistry},
		{`echo x >> g/config`, protectedOrRegistry},
		{`echo '{}' > cfg/guard.json`, protectedOrRegistry},
		{`echo x >> rc`, protectedOrRegistry},
		{`printf x > homedir/.profile`, protectedOrRegistry},
		{`rm g/hooks/post-checkout`, protectedOrRegistry},
		// The link itself, removed, is not the file it reaches.
		{`rm rc`, nil},
		{`cat rc > copy.txt`, nil},
	})
}

// TestApproveProtectsTheRepositorysOwnPaths: the git common directory, a
// linked worktree's included, the hooks directory core.hooksPath names, and
// the worktree's own .git are protected wherever the line reaches them from.
func TestApproveProtectsTheRepositorysOwnPaths(t *testing.T) {
	main, repo := gitRig(t)
	repo.Git("config", "core.hooksPath", "tools/hooks")
	wt := filepath.Join(t.TempDir(), "lane")
	repo.Git("worktree", "add", "-q", "-b", "lane", wt)
	r := approveRig{dir: wt, home: main.home}
	common := filepath.Join(main.dir, ".git")
	r.run(t, []approveCase{
		{`cp hook.sh ` + filepath.Join(common, "hooks", "pre-commit"), protectedOrRegistry},
		{`echo x >> ` + filepath.Join(common, "config"), protectedOrRegistry},
		{`cp hook.sh tools/hooks/pre-commit`, protectedOrRegistry},
		{`cp hook.sh ` + filepath.Join(main.dir, "tools", "hooks", "pre-push"), protectedOrRegistry},
		{`echo gitdir: /tmp/x > .git`, protectedOrRegistry},
		{`echo x >> ~/.gitconfig`, protectedOrRegistry},
		{`echo x > ~/.config/git/config`, protectedOrRegistry},
		{`mkdir -p ~/.claude && echo '{}' > ~/.claude/settings.json`, protectedOrRegistry},
		{`echo '{}' > .claude/settings.local.json`, protectedOrRegistry},
		{`cp hook.sh tools/other/x.sh`, nil},
	})
}

// TestApproveRefusesGitConfigThatReachesBeyondTheRepository: `git config`
// writing outside the repository's own file, or setting a key that starts a
// program, is refused, as is a git command that sets such a key for itself;
// reading configuration, and setting an ordinary key, is not.
func TestApproveRefusesGitConfigThatReachesBeyondTheRepository(t *testing.T) {
	r, _ := gitRig(t)
	refused := refusedAs(gitConfigWriteEntryID)
	r.run(t, []approveCase{
		{`git config --global core.editor vim`, refused},
		{`git config --global user.name x`, refused},
		{`git config --system a.b c`, refused},
		{`git config --file x.cfg a.b c`, refused},
		{`git config -f x.cfg a.b c`, refused},
		{`git config --file=x.cfg a.b c`, refused},
		{`git config set --global user.name x`, refused},
		{`git config --global --unset user.name`, refused},
		{`git config core.hooksPath .hooks`, refused},
		{`git config core.fsmonitor true`, refused},
		{`git config CORE.HOOKSPATH .hooks`, refused},
		{`git config "core.pager" less`, refused},
		{`git config core.'sshCommand' 'ssh -v'`, refused},
		{`git config alias.x '!sh x.sh'`, refused},
		{`git config filter.lfs.clean 'x %f'`, refused},
		{`git config diff.pdf.textconv pdftotext`, refused},
		{`git config set core.editor vim`, refused},
		{`git config --add credential.helper store`, refused},
		{`git config commit.gpgSign true`, refused},
		{`git config $K v`, refused},
		{`git config -e`, refused},
		{`git -c core.fsmonitor=./x.sh status`, refused},
		{`git -c core.sshCommand='python3 -c x' fetch`, refused},
		{`git -c core.hooksPath=/tmp/h checkout main`, refused},
		{`GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=core.pager GIT_CONFIG_VALUE_0=./x.sh git log`, refused},
		{`git config user.name x`, nil},
		{`git config --local user.email x@example.com`, nil},
		{`git config --get core.hooksPath`, nil},
		{`git config core.hooksPath`, nil},
		{`git config --global --get user.email`, nil},
		{`git config --list`, nil},
		{`git config --global -l`, nil},
		{`git config --file .gitmodules --get submodule.x.url`, nil},
		{`git config get --global user.name`, nil},
		{`git config --unset core.pager`, nil},
		{`git -c color.ui=never log -1`, nil},
	})
}

// TestApproveProtectedNamesStayRelativeToTheTree: a working tree that itself
// sits below a protected name (a worktree kept under .claude, say) still
// writes its own files.
func TestApproveProtectedNamesStayRelativeToTheTree(t *testing.T) {
	root := t.TempDir()
	r := approveRig{dir: filepath.Join(root, ".claude", "worktrees", "lane"), home: filepath.Join(root, "home")}
	for _, d := range []string{r.dir, r.home} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	r.run(t, []approveCase{
		{`echo x > notes.txt`, nil},
		{`echo '{}' > .claude/settings.json`, protectedOrRegistry},
		{`echo '{}' > ` + filepath.Join(root, ".claude", "settings.json"), protectedOrRegistry},
	})
	if strings.Contains(r.dir, "..") {
		t.Fatal("unclean rig")
	}
}
