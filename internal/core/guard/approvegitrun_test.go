package guard

import "testing"

// withoutEntry is ld's registry with one entry left out, so a row the
// registry refuses on its own can show what the approver's reading answers.
func withoutEntry(ld Loaded, id string) Loaded {
	entries := map[string]Entry{}
	for k, e := range ld.Registry.Entries {
		if k != id {
			entries[k] = e
		}
	}
	ld.Registry.Entries = entries
	return ld
}

// TestApproveJudgesTheTransferProgramFlagsOfThePlumbing: fetch-pack's
// --upload-pack and --exec (git-fetch-pack(1)) and send-pack's --receive-pack
// and --exec (git-send-pack(1)) name the program git starts for the other
// side, as fetch's, pull's and push's do; fetch and pull are read for --exec
// too.
func TestApproveJudgesTheTransferProgramFlagsOfThePlumbing(t *testing.T) {
	refused := refusedAs(gitConfigWriteEntryID)
	r, _ := gitRig(t)
	r.run(t, []approveCase{
		{`git fetch --exec=./tool ../other`, refused},
		{`git pull --exec ./tool ../other main`, refused},
		{`git fetch-pack --upload-pack=./tool ../other`, refused},
		{`git fetch-pack --exec=./tool ../other`, refused},
		{`git send-pack --receive-pack=./tool ../other main`, refused},
		{`git send-pack --exec=./tool ../other main`, refused},
		{`git fetch-pack --depth=1 ../other`, nil},
		{`git send-pack --thin ../other main`, nil},
	})
}

// TestApproveLetsTheEditorAndAskpassBeANoOp: the editor and askpass
// variables and settings are approved when the value, trimmed, is exactly
// `true` or `:`, which exit at once whatever git hands them, in every form
// the approver reads; any other value is refused as before.
func TestApproveLetsTheEditorAndAskpassBeANoOp(t *testing.T) {
	refused := refusedAs(gitConfigWriteEntryID)
	r, _ := gitRig(t)
	r.run(t, []approveCase{
		{`GIT_EDITOR=true git rebase --continue`, nil},
		{`GIT_SEQUENCE_EDITOR=: git rebase -i HEAD~2`, nil},
		{`EDITOR=true git commit -m x`, nil},
		{`VISUAL=' true ' git commit -m x`, nil},
		{`GIT_ASKPASS=true git fetch`, nil},
		{`SSH_ASKPASS=: git fetch`, nil},
		{`env GIT_EDITOR=: git commit -m x`, nil},
		{`export GIT_EDITOR=true; git commit -m x`, nil},
		{`git -c core.editor=true commit -m x`, nil},
		{`git -c sequence.editor=: rebase -i HEAD~2`, nil},
		{`git -c core.askPass=true fetch`, nil},
		{`git config core.editor true`, nil},
		{`git config set sequence.editor :`, nil},
		{`GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=core.askpass GIT_CONFIG_VALUE_0=true git fetch`, nil},
		{`git clone -c core.editor=: src dst`, nil},
		{`GIT_EDITOR=./tool git rebase --continue`, refused},
		{`GIT_EDITOR='true; ./tool' git commit -m x`, refused},
		{`GIT_EDITOR=truex git commit -m x`, refused},
		{`GIT_ASKPASS=./tool git fetch`, refused},
		{`GIT_EDITOR="$V" git commit -m x`, refused},
		{`export GIT_EDITOR=./tool; git commit -m x`, refused},
		{`git -c core.editor=./tool commit -m x`, refused},
		{`git config core.askpass ./tool`, refused},
		{`GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=sequence.editor GIT_CONFIG_VALUE_0=./tool git rebase -i HEAD~2`, refused},
		{`git clone -c core.editor=./tool src dst`, refused},
	})
}

// TestApproveJudgesGitConfigWithGitConfigSet: GIT_CONFIG names the file git
// config reads and writes in place of the repository's (git-config(1)), so
// a write with it set, on the command or carried, is refused as a --file
// write is; a read is not.
func TestApproveJudgesGitConfigWithGitConfigSet(t *testing.T) {
	refused := refusedAs(gitConfigWriteEntryID)
	r, _ := gitRig(t)
	r.run(t, []approveCase{
		{`GIT_CONFIG=./other.cfg git config user.name x`, refused},
		{`env GIT_CONFIG=./other.cfg git config user.name x`, refused},
		{`export GIT_CONFIG=./other.cfg; git config user.name x`, refused},
		{`GIT_CONFIG=./other.cfg git config --get user.name`, nil},
		{`git config user.name x`, nil},
	})
}

// TestApproveRefusesACarriedConfigFile: a configuration file a carried
// GIT_CONFIG_GLOBAL or GIT_CONFIG_SYSTEM names is one the guard does not
// read, as it is when set on the command itself.
func TestApproveRefusesACarriedConfigFile(t *testing.T) {
	refused := refusedAs(gitConfigWriteEntryID)
	r, _ := gitRig(t)
	r.run(t, []approveCase{
		{`export GIT_CONFIG_GLOBAL=./other.cfg; git log -1`, refused},
		{`export GIT_CONFIG_SYSTEM=./other.cfg && git status`, refused},
		{`git log -1`, nil},
	})
}

// TestApproveCarriesANameTheLineAssignsWithoutAValue: a program-naming or
// GIT_CONFIG_* name a command of the line assigns other than by NAME=value
// (read, printf -v, a for loop, ${NAME:=…}) is carried as a value the line
// does not fix, and so is refused at a later git command; a bare export of a
// name keeps the value it holds.
func TestApproveCarriesANameTheLineAssignsWithoutAValue(t *testing.T) {
	refused := refusedAs(gitConfigWriteEntryID)
	r, _ := gitRig(t)
	r.run(t, []approveCase{
		{`read -r GIT_EXTERNAL_DIFF <<< ./tool; export GIT_EXTERNAL_DIFF; git diff`, refused},
		{`printf -v GIT_PAGER %s ./tool; export GIT_PAGER; git log -1`, refused},
		{`for GIT_EXTERNAL_DIFF in ./tool; do export GIT_EXTERNAL_DIFF; git diff; done`, refused},
		{`: ${GIT_EXTERNAL_DIFF:=./tool}; export GIT_EXTERNAL_DIFF; git diff`, refused},
		{`set -a; read -r GIT_PAGER <<< ./tool; git log -1`, refused},
		{`read -r GIT_CONFIG_GLOBAL <<< ./other.cfg; export GIT_CONFIG_GLOBAL; git log -1`, refused},
		{`export GIT_PAGER=cat; read -r GIT_PAGER <<< ./tool; git log -1`, refused},
		{`GIT_PAGER=x read -r line <<< y; git log -1`, refused},
		{`GIT_PAGER=cat; export GIT_PAGER; git log -1`, nil},
		{`read -r line <<< x; git log -1`, nil},
		{`GIT_PAGER=cat git log -1; git status`, nil},
		{`GIT_EDITOR=true git rebase --continue && git log -1`, nil},
		// A prefix on a command that runs a line of its own (eval, a shell)
		// reaches the git commands of that line, so it is carried.
		{`GIT_PAGER='$T --flag' eval 'git log -1'`, refused},
		{`GIT_PAGER='$T --flag' bash -c 'git log -1'`, refused},
	})
}

// TestApproveCarriesByRunOrder: where a line runs a command again or later
// than its text places it (a loop, a function, an eval body, a ${NAME:=…}),
// the environment the whole line leaves is carried to every git command in
// it; any other line keeps text order.
func TestApproveCarriesByRunOrder(t *testing.T) {
	refused := refusedAs(gitConfigWriteEntryID)
	r, _ := gitRig(t)
	r.run(t, []approveCase{
		{`for i in 1 2; do git diff; export GIT_EXTERNAL_DIFF=./tool; done`, refused},
		{`while true; do git diff; export GIT_EXTERNAL_DIFF=./tool; break; done`, refused},
		{`f() { git diff; export GIT_EXTERNAL_DIFF=./tool; }; f; f`, refused},
		{`eval 'export GIT_EXTERNAL_DIFF=./tool'; git diff`, refused},
		{`git diff "${GIT_EXTERNAL_DIFF:=./tool}"`, refused},
		{`for i in 1 2; do git log -1; done`, nil},
		{`f() { git status; }; f`, nil},
		{`git status; export GIT_PAGER='$T --flag'`, nil},
		// A line read whole still judges each git command by what text
		// order carries to it as well, so a value set before the git
		// command and changed after it is not judged by the later one alone.
		{`export GIT_PAGER='$T --flag'; git log -1; export GIT_PAGER=cat; for i in 1; do :; done`, refused},
		{`export GIT_PAGER=cat; git log -1; for i in 1; do :; done`, nil},
	})
}

// TestApproveHandsTheCarryToANestedLine: a command line a git setting or
// option runs is judged with the environment carried to the git command
// that runs it, and a bash -c body sees the carry by where it is placed.
func TestApproveHandsTheCarryToANestedLine(t *testing.T) {
	refused := refusedAs(gitConfigWriteEntryID)
	r, _ := gitRig(t)
	r.run(t, []approveCase{
		{`export GIT_PAGER=cat; git -c core.pager='git log -1' log -1`, refused},
		{`git -c core.pager='git log -1' log -1`, nil},
		{`export GIT_EXTERNAL_DIFF=./tool; bash -c 'git diff'`, refused},
		{`export GIT_PAGER=cat; bash -c 'git log -1'`, nil},
	})
}

// TestApproveJudgesTheCommandLinesGitOptionsRun: an option that hands git a
// command line to run through a shell is approved only when the line spells
// it out and the guard, reading it as a line of its own, clears it:
// git-rebase(1)'s -x/--exec, git-submodule(1)'s foreach, git-bisect(1)'s
// run, git-difftool(1)'s -x/--extcmd, git-filter-branch(1)'s --setup and
// command filters, and git-send-email(1)'s --to-cmd, --cc-cmd, --header-cmd
// and --sendmail-cmd. git-grep(1)'s -O/--open-files-in-pager value is judged
// as a pager is.
func TestApproveJudgesTheCommandLinesGitOptionsRun(t *testing.T) {
	refused := refusedAs(gitConfigWriteEntryID)
	r, _ := gitRig(t)
	r.run(t, []approveCase{
		{`git rebase -x 'tool --flag' HEAD~2`, nil},
		{`git rebase --exec='tool --flag' HEAD~2`, nil},
		{`git rebase -Sx HEAD~2`, nil},
		{`git rebase -x '$T --flag' HEAD~2`, refused},
		{`git rebase --exec "$C" HEAD~2`, refused},
		{`git rebase -ix '$T' HEAD~2`, refused},
		{`git rebase HEAD~2 --ex='$T'`, refused},
		{`git rebase -x'$T' HEAD~2`, refused},

		{`git submodule foreach 'tool --flag'`, nil},
		{`git submodule --quiet foreach --recursive tool --flag`, nil},
		{`git submodule foreach '$T --flag'`, refused},
		{`git submodule foreach tool "$A"`, refused},
		{`git submodule status`, nil},

		{`git bisect run tool --flag`, nil},
		{`git bisect run '$T' --flag`, refused},
		{`git bisect run "$C"`, refused},
		{`git bisect good`, nil},

		{`git difftool -x 'tool --flag' HEAD`, nil},
		{`git difftool --extcmd='tool --flag' HEAD`, nil},
		{`git difftool -x '$T' HEAD`, refused},
		{`git difftool --ext='$T' HEAD`, refused},
		{`git difftool -dx'$T' HEAD`, refused},

		{`git send-email --to-cmd='tool --flag' x.patch`, nil},
		{`git send-email --to=a@example.com --cc=b@example.com x.patch`, nil},
		{`git send-email --to-cmd='$T' x.patch`, refused},
		{`git send-email --cc-cmd '$T' x.patch`, refused},
		{`git send-email --header-cmd='$T' x.patch`, refused},
		{`git send-email --sendmail-cmd='$T' x.patch`, refused},
		{`git send-email -Sendmail-Cmd='$T' x.patch`, refused},

		{`git grep -O x`, nil},
		{`git grep -Ocat x`, nil},
		{`git grep --open-files-in-pager=cat x`, nil},
		{`git grep -e -O'$T' x`, nil},
		{`git grep -O'$T' x`, refused},
		{`git grep -iO'$T' x`, refused},
		{`git grep --open-files-in-pager='$T' x`, refused},
		{`git grep -O"$P" x`, refused},
	})
	// The registry refuses git filter-branch on its own; without that entry
	// the approver's reading answers.
	ld := withoutEntry(clean(), "git-filter-branch")
	for _, c := range []approveCase{
		{`git filter-branch --tree-filter 'tool --flag' HEAD`, nil},
		{`git filter-branch -f --msg-filter 'tool --flag' --subdirectory-filter '$D' HEAD`, nil},
		{`git filter-branch --tree-filter '$T' HEAD`, refused},
		{`git filter-branch --setup '$T' HEAD`, refused},
		{`git filter-branch --env-filter "$C" HEAD`, refused},
		{`git filter-branch --index-filter '$T' HEAD`, refused},
		{`git filter-branch --parent-filter '$T' HEAD`, refused},
		{`git filter-branch --commit-filter '$T' HEAD`, refused},
		{`git filter-branch --tag-name-filter '$T' HEAD`, refused},
		{`git filter-branch --msg-filter '$T' HEAD`, refused},
	} {
		a := approveShell(ld, r.files(), c.line)
		switch {
		case len(c.ids) == 0 && !a.Allow:
			t.Errorf("%q: refused (%s): %s", c.line, a.EntryID, a.Reason)
		case len(c.ids) > 0 && (a.Allow || a.EntryID != gitConfigWriteEntryID):
			t.Errorf("%q: allow %v under %q, want a refusal under %v (%s)", c.line, a.Allow, a.EntryID, c.ids, a.Reason)
		}
	}
}

// TestApproveJudgesTheProgramsGitOptionsPick: an option that names the
// program git starts, or picks the tool it runs, is refused as the setting
// it stands for is: git-send-email(1)'s --smtp-server given a path,
// git-instaweb(1)'s --httpd, --browser and --module-path, and the -t/--tool
// of git-difftool(1) and git-mergetool(1). A host name for the mail server,
// and -g/--gui alone, are approved.
func TestApproveJudgesTheProgramsGitOptionsPick(t *testing.T) {
	refused := refusedAs(gitConfigWriteEntryID)
	r, _ := gitRig(t)
	r.run(t, []approveCase{
		{`git send-email --smtp-server=smtp.example.com x.patch`, nil},
		{`git send-email --smtp-server-port=587 x.patch`, nil},
		{`git send-email --smtp-server=./tool x.patch`, refused},
		{`git send-email --smtp-server /opt/tool x.patch`, refused},
		{`git send-email --smtp-server="$S" x.patch`, refused},

		{`git instaweb --start`, nil},
		{`git instaweb -p 1234 --start`, nil},
		{`git instaweb --httpd=tool --start`, refused},
		{`git instaweb -d tool --start`, refused},
		{`git instaweb --browser tool`, refused},
		{`git instaweb -b tool`, refused},
		{`git instaweb --module-path=./lib`, refused},
		{`git instaweb -lm ./lib`, refused},

		{`git difftool -g HEAD`, nil},
		{`git difftool --tool-help`, nil},
		{`git difftool -Stext HEAD`, nil},
		{`git difftool -t tool HEAD`, refused},
		{`git difftool --tool=tool HEAD`, refused},
		{`git difftool -ttool HEAD`, refused},
		{`git -c diff.guitool=tool difftool -g HEAD`, refused},

		{`git mergetool -g`, nil},
		{`git mergetool --tool-help`, nil},
		{`git mergetool --tool-help=all`, nil},
		{`git mergetool --tool-helpx tool`, refused},
		{`git mergetool -t tool`, refused},
		{`git mergetool --tool=tool`, refused},
		{`git -c merge.guitool=tool mergetool -g`, refused},
	})
}
