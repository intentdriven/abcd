package guard

import (
	"strings"
	"testing"
)

// TestHistoryRewritesAreRefused: a repository's history is not rewritten. Every
// command that rewrites it, moves or deletes a ref by hand, stands other objects
// in for it, overwrites a remote, or destroys the reflog and the unreachable
// objects a rewrite could be recovered from is refused, whichever spelling,
// prefix or wrapper carries it.
func TestHistoryRewritesAreRefused(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		// git filter-repo, any arguments, and the program git runs for it.
		{`git filter-repo --path secret.txt --invert-paths`, VerdictBlock, "git-filter-repo"},
		{`git filter-repo`, VerdictBlock, "git-filter-repo"},
		{`git -C /repo filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`git -c user.name=x filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`git --no-pager filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`/usr/bin/git filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`FILTER_BRANCH_SQUELCH_WARNING=1 git filter-repo`, VerdictBlock, "git-filter-repo"},
		{`env A=1 git filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`command git filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`sudo git filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`sh -c 'git filter-repo --force'`, VerdictBlock, "git-filter-repo"},
		{`bash -lc 'git filter-repo --force'`, VerdictBlock, "git-filter-repo"},
		{`eval 'git filter-repo --force'`, VerdictBlock, "git-filter-repo"},
		{`cd repo && git filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`git -c alias.fr=filter-repo fr --force`, VerdictBlock, "git-filter-repo"},
		{`git -c alias.fr='!git filter-repo' fr`, VerdictBlock, "git-filter-repo"},
		{`git filter-rep? --force`, VerdictBlock, "git-filter-repo"},
		{`git-filter-repo --path secret.txt --invert-paths`, VerdictBlock, "git-filter-repo"},
		{`/opt/homebrew/bin/git-filter-repo --force --path secret.txt`, VerdictBlock, "git-filter-repo"},
		{`sh -c 'git-filter-repo --path secret.txt'`, VerdictBlock, "git-filter-repo"},

		// git filter-branch, any arguments, and the program git runs for it.
		{`git filter-branch --tree-filter 'rm x' HEAD`, VerdictBlock, "git-filter-branch"},
		{`git filter-branch`, VerdictBlock, "git-filter-branch"},
		{`git -C /repo filter-branch -f HEAD`, VerdictBlock, "git-filter-branch"},
		{`git -c core.pager=cat filter-branch HEAD`, VerdictBlock, "git-filter-branch"},
		{`FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch HEAD`, VerdictBlock, "git-filter-branch"},
		{`env FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch HEAD`, VerdictBlock, "git-filter-branch"},
		{`sh -c 'git filter-branch HEAD'`, VerdictBlock, "git-filter-branch"},
		{`git-filter-branch HEAD`, VerdictBlock, "git-filter-branch"},
		{`"$(git --exec-path)"/git-filter-branch HEAD`, VerdictBlock, "git-filter-branch"},

		// git update-ref: no form of it only reads.
		{`git update-ref refs/heads/main abc1234`, VerdictBlock, "git-update-ref"},
		{`git update-ref refs/heads/main abc1234 def5678`, VerdictBlock, "git-update-ref"},
		{`git update-ref -d refs/heads/main`, VerdictBlock, "git-update-ref"},
		{`git update-ref --no-deref -d refs/heads/main abc1234`, VerdictBlock, "git-update-ref"},
		{`git update-ref --stdin`, VerdictBlock, "git-update-ref"},
		{`git update-ref -m "move it" refs/heads/main abc1234`, VerdictBlock, "git-update-ref"},
		{`git -C /repo update-ref refs/heads/main abc1234`, VerdictBlock, "git-update-ref"},
		{`git -c a.b=c update-ref -d refs/tags/v1`, VerdictBlock, "git-update-ref"},
		{`GIT_DIR=/repo/.git git update-ref -d refs/heads/main`, VerdictBlock, "git-update-ref"},
		{`env GIT_DIR=/repo/.git git update-ref -d refs/heads/main`, VerdictBlock, "git-update-ref"},
		{`sh -c 'git update-ref -d refs/heads/main'`, VerdictBlock, "git-update-ref"},
		{`git -c alias.ur=update-ref ur -d refs/heads/main`, VerdictBlock, "git-update-ref"},
		{`printf 'delete refs/heads/main\n' | git update-ref --stdin`, VerdictBlock, "git-update-ref"},

		// git replace: every form that stands an object in for another.
		{`git replace abc1234 def5678`, VerdictBlock, "git-replace"},
		{`git -C /repo replace abc1234 def5678`, VerdictBlock, "git-replace"},
		{`sh -c 'git replace abc1234 def5678'`, VerdictBlock, "git-replace"},
		{`git replace --graft abc1234 def5678`, VerdictBlock, ""},
		{`git replace --graft abc1234`, VerdictBlock, "git-replace-rewrite"},
		{`git replace -g abc1234`, VerdictBlock, "git-replace-rewrite"},
		{`git replace --gra abc1234`, VerdictBlock, "git-replace-rewrite"},
		{`git replace --edit abc1234`, VerdictBlock, "git-replace-rewrite"},
		{`git replace -e abc1234`, VerdictBlock, "git-replace-rewrite"},
		{`git replace --convert-graft-file`, VerdictBlock, "git-replace-rewrite"},
		{`git replace -f abc1234 def5678`, VerdictBlock, ""},

		// The reflog and the unreachable objects, what a rewrite is undone from.
		{`git reflog expire --expire=now --all`, VerdictBlock, "git-reflog-expire"},
		{`git -C /repo reflog expire --all`, VerdictBlock, "git-reflog-expire"},
		{`sh -c 'git reflog expire --all'`, VerdictBlock, "git-reflog-expire"},
		{`git reflog delete HEAD@{1}`, VerdictBlock, "git-reflog-delete"},
		{`git reflog drop main`, VerdictBlock, "git-reflog-drop"},
		{`git gc --prune=now`, VerdictBlock, "git-gc-prune"},
		{`git gc --prune=all --aggressive`, VerdictBlock, "git-gc-prune"},
		{`git gc --prune`, VerdictBlock, "git-gc-prune"},
		{`git gc --pru=now`, VerdictBlock, "git-gc-prune"},
		{`git -c gc.auto=0 gc --prune=now`, VerdictBlock, "git-gc-prune"},
		{`sh -c 'git gc --prune=now'`, VerdictBlock, "git-gc-prune"},
		{`git prune`, VerdictBlock, "git-prune"},
		{`git prune --expire=now`, VerdictBlock, "git-prune"},
		{`git -C /repo prune`, VerdictBlock, "git-prune"},

		// A remote overwritten: the spellings already refused, and those that
		// were not.
		{`git push --force-with-lease origin main`, VerdictBlock, "git-push-force"},
		{`git -c http.proxy=x push --force-with-lease origin main`, VerdictBlock, "git-push-force"},
		{`GIT_SSH_COMMAND=ssh git push --force origin main`, VerdictBlock, "git-push-force"},
		{`env GIT_SSH_COMMAND=ssh git push -f origin main`, VerdictBlock, "git-push-force"},
		{`sh -c 'git push --force-with-lease origin main'`, VerdictBlock, "git-push-force"},
		{`git push origin +main`, VerdictBlock, "git-push-force-refspec"},
		{`git push origin +HEAD:main`, VerdictBlock, "git-push-force-refspec"},
		{`git -c a.b=c push origin +main:main`, VerdictBlock, "git-push-force-refspec"},
		{`env A=1 git push origin +main:main`, VerdictBlock, "git-push-force-refspec"},
		{`sh -c 'git push origin +main:main'`, VerdictBlock, "git-push-force-refspec"},
		{`git push --mirror`, VerdictBlock, "git-push-mirror"},
		{`git push --mirror git@example.com:o/r.git`, VerdictBlock, "git-push-mirror"},
		{`git push --mir origin`, VerdictBlock, "git-push-mirror"},
		{`git -C /repo push --mirror origin`, VerdictBlock, "git-push-mirror"},
		{`sh -c 'git push --mirror origin'`, VerdictBlock, "git-push-mirror"},
		{`git send-pack --force origin main`, VerdictBlock, "git-send-pack-force"},
		{`git send-pack -f origin main`, VerdictBlock, "git-send-pack-force"},
		{`git send-pack --mirror origin`, VerdictBlock, "git-send-pack-force"},
		{`git send-pack --force-with-lease origin main`, VerdictBlock, "git-send-pack-force"},
		{`git send-pack origin +main:main`, VerdictBlock, "git-send-pack-force-refspec"},
		{`git fast-import --force`, VerdictBlock, "git-fast-import-force"},
		{`git fast-export --all | git fast-import --force`, VerdictBlock, "git-fast-import-force"},
		{`gh api -X PATCH repos/o/r/git/refs/heads/main -f sha=abc1234 -F force=true`, VerdictBlock, "gh-api-ref-force"},
		{`gh api --method PATCH repos/o/r/git/refs/heads/main -F sha=abc1234 --field force=true`, VerdictBlock, "gh-api-ref-force"},
		{`gh api repos/o/r/git/refs/heads/main --method=PATCH --field=force=true`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH repos/o/r/git/refs/heads/main -f force=true`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH repos/o/r/git/refs/heads/main -Fforce=true`, VerdictBlock, "gh-api-ref-force"},
		{`sh -c 'gh api -X PATCH repos/o/r/git/refs/heads/main -F force=true'`, VerdictBlock, "gh-api-ref-force"},
	})
}

// TestHistoryRewriteNearMissesPass: the words alone are no rewrite. A command
// that reads a ref, names one of these commands in its text, or does the
// ordinary work of a local branch stays allowed, an unpushed rebase and amend
// among it.
func TestHistoryRewriteNearMissesPass(t *testing.T) {
	cases := []string{
		// The words in text, not in command position.
		`git log --grep=filter-branch`,
		`git log --oneline --grep='git filter-repo'`,
		`git log --oneline -- scripts/filter-branch.sh`,
		`git commit -m "docs: say why git filter-branch is refused"`,
		`git commit -m 'guard: refuse filter-repo and update-ref'`,
		`abcd capture "an agent ran git filter-branch on main"`,
		`abcd capture "git update-ref -d moved the branch"`,
		`grep -rn filter-branch internal/`,
		`grep -rn 'git update-ref' .`,
		`git help filter-branch`,
		`git help update-ref`,
		`man git-filter-branch`,
		`which git-filter-repo`,
		"cat > notes.md <<'EOF'\ngit filter-repo rewrites history; git update-ref -d deletes a ref.\nEOF",

		// Reading the refs update-ref would write.
		`git rev-parse refs/heads/main`,
		`git show-ref --verify refs/heads/main`,
		`git for-each-ref refs/heads/`,
		`git symbolic-ref --short HEAD`,

		// Reading stand-ins, and taking one away.
		`git replace`,
		`git replace -l`,
		`git replace --list 'abc*'`,
		`git replace --format=short -l`,
		`git replace -d abc1234`,

		// Reading the reflog.
		`git reflog`,
		`git reflog show main`,
		`git reflog exists refs/heads/main`,
		`git reflog --all`,
		`git log -g`,

		// Housekeeping that keeps the grace period, and other things called prune.
		`git gc`,
		`git gc --auto`,
		`git gc --aggressive`,
		`git gc --no-prune`,
		`git worktree prune`,
		`git remote prune origin`,
		`git fetch --prune`,
		`git prune-packed`,
		`git count-objects -v`,
		`git fsck --unreachable`,

		// Ordinary pushes and imports.
		`git push origin main`,
		`git push origin HEAD:refs/heads/feat/x`,
		`git push --no-mirror origin main`,
		`git clone --mirror https://example.com/o/r.git`,
		`git send-pack origin main`,
		`git send-pack --dry-run origin main`,
		`git fast-import --quiet`,
		`git fast-export --all`,

		// The forge API on anything but a forced ref move.
		`gh api repos/o/r/git/refs/heads/main`,
		`gh api -X PATCH repos/o/r/git/refs/heads/feat -f sha=abc1234`,
		`gh api -X PATCH repos/o/r -F allow_merge_commit=true`,
		`gh api -X PATCH repos/o/r/pulls/1 -f body="$(cat body.md)"`,
		`gh api graphql -F force=true -f query=x`,
		`gh api -X "$METHOD" repos/o/r/pulls/1 -f title=x`,
		`gh api -X PATCH repos/o/r/git/refs/heads/feat -F sha="$SHA"`,

		// Ordinary local work on a commit not yet pushed.
		`git rebase main`,
		`git rebase origin/main`,
		`git rebase -i HEAD~3`,
		`git rebase --continue`,
		`git commit --amend --no-edit`,
		`git commit --amend -m "better words"`,
		`git reset --soft HEAD~1`,
		`git branch -f feat/x origin/main`,
		`git revert abc1234`,
		`git cherry-pick abc1234`,
	}
	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			if d := readingOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("Check(%q) = %q via %q (%v), want allow", cmd, d.Verdict, d.EntryID, d.Matches)
			}
		})
	}
}

// TestHistoryRewriteSuccessorsAskForANewCommit: each of these refusals says
// what to run instead, and that is a new commit, a revert or a fix on top,
// never another route to the same rewrite.
func TestHistoryRewriteSuccessorsAskForANewCommit(t *testing.T) {
	ids := []string{
		"git-filter-repo", "git-filter-branch",
		"git-update-ref", "git-replace", "git-replace-rewrite",
		"git-reflog-expire", "git-reflog-delete", "git-reflog-drop", "git-gc-prune", "git-prune",
		"git-push-mirror", "git-send-pack-force", "git-send-pack-force-refspec",
		"git-fast-import-force", "gh-api-ref-force", "gh-api-ref-input",
		"gh-api-graphql-ref-force", "gh-api-graphql-any-key-ref-force", "gh-api-graphql-ref-force-field", "gh-repo-sync-force",
	}
	entries := Defaults().Entries
	for _, id := range ids {
		e, ok := entries[id]
		if !ok {
			t.Errorf("the bundled registry has no entry %s", id)
			continue
		}
		if e.Tier != TierBlocker {
			t.Errorf("entry %s is tier %q: a history rewrite is refused, not warned about", id, e.Tier)
		}
		if !strings.Contains(e.Successor, "new commit") || !strings.Contains(e.Successor, "git revert") {
			t.Errorf("entry %s: successor %q does not ask for a new commit, a revert or a fix on top", id, e.Successor)
		}
	}
}

// TestSubcommandNamedReadsAWholeSubstitutionAsNoSubcommand: an entry that sets
// subcommand_named fires on a subcommand the line's own text names, however
// much of it a substitution helps to write, and never on a word that is only a
// substitution, which is every subcommand there is. Without it a blocker naming
// only a subcommand refuses `git "$@"`, and the wrapper functions that run it.
func TestSubcommandNamedReadsAWholeSubstitutionAsNoSubcommand(t *testing.T) {
	const body = `{"schema_version":1,"entries":{"no-frob":{"tier":"blocker",
		"pattern":{"command":"tool","subcommand":"frob","subcommand_named":true},
		"successor":"s","why":"w"}}}`
	r, err := Load(writeCommittedOverride(t, body))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		cmd  string
		want Verdict
	}{
		{`tool frob`, VerdictBlock},
		{`tool fr$(x)`, VerdictBlock},
		{`tool $(true) frob`, VerdictBlock},
		{`tool $(x)`, VerdictAllow},
		{`tool "$@"`, VerdictAllow},
		{`tool "$SUB" a`, VerdictAllow},
		{`tool other`, VerdictAllow},
	} {
		d, err := r.Check(c.cmd)
		if err != nil {
			t.Fatalf("Check(%q): %v", c.cmd, err)
		}
		if d.Verdict != c.want {
			t.Errorf("Check(%q) = %q via %q, want %q", c.cmd, d.Verdict, d.EntryID, c.want)
		}
	}

	if _, err := Load(writeCommittedOverride(t, `{"schema_version":1,"entries":{"no-frob":{"tier":"blocker",
		"pattern":{"command":"tool","subcommand_named":true},"successor":"s","why":"w"}}}`)); err == nil {
		t.Error("an entry setting subcommand_named without a subcommand loaded; it says it reads a word it does not have")
	}

	lifted, err := Load(writeCommittedOverride(t, `{"schema_version":1,"entries":{"git-prune":{"pattern":{"subcommand_named":false}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	d, err := lifted.Check(`git "$SUB"`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(d.Matches, "git-prune") {
		t.Errorf("a repository lifting subcommand_named: matches %v, want git-prune among them", d.Matches)
	}
}
