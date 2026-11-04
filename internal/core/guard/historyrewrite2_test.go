package guard

import "testing"

const execPath = "/opt/homebrew/opt/git/libexec/git-core/"

// TestDashedGitProgramReadsAsGit: `git-<sub>` is `git <sub>` by another door, at
// any path, so every git entry reads it without one entry per program. A word
// that only names the program, as an operand of a command that does not run it,
// stays a word.
func TestDashedGitProgramReadsAsGit(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`git-filter-repo --path=secret.txt --invert-paths --force`, VerdictBlock, "git-filter-repo"},
		{`git-filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`/opt/homebrew/bin/git-filter-repo --path=x`, VerdictBlock, "git-filter-repo"},
		{`git-filter-branch --tree-filter=x`, VerdictBlock, "git-filter-branch"},
		{`"$(git --exec-path)"/git-filter-branch HEAD`, VerdictBlock, "git-filter-branch"},
		{execPath + `git-push --force origin main`, VerdictBlock, "git-push-force"},
		{execPath + `git-push origin +main:main`, VerdictBlock, "git-push-force-refspec"},
		{execPath + `git-push --mirror`, VerdictBlock, "git-push-mirror"},
		{execPath + `git-update-ref -d refs/heads/main`, VerdictBlock, "git-update-ref"},
		{execPath + `git-reflog expire --all`, VerdictBlock, "git-reflog-expire"},
		{execPath + `git-gc --prune=now`, VerdictBlock, "git-gc-prune"},
		{`git-commit --no-verify -m x`, VerdictBlock, "git-commit-no-verify"},
		{`git-pus? --force origin main`, VerdictBlock, "git-push-force"},
		{`FOO=1 git-push --force origin main`, VerdictBlock, "git-push-force"},
		{`env FOO=1 ` + execPath + `git-push -f origin main`, VerdictBlock, "git-push-force"},
		{`sh -c '` + execPath + `git-push --force origin main'`, VerdictBlock, "git-push-force"},
		{`cd repo && git-update-ref -d refs/heads/main`, VerdictBlock, "git-update-ref"},
		{`myrunner git-push --force origin main`, VerdictWarn, ""},
	})
	for _, cmd := range []string{
		`command -v git-filter-repo`,
		`which git-filter-repo`,
		`type git-update-ref`,
		`man git-filter-repo`,
		`man git-push`,
		`ls ` + execPath + `git-push`,
		`git-push origin main`,
		`git-lfs track '*.psd'`,
		`git-crypt status`,
		`./scripts/git-push-all.sh --force`,
		`brew install git-filter-repo`,
	} {
		t.Run(cmd, func(t *testing.T) {
			if d := readingOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("Check(%q) = %q via %q (%v), want allow", cmd, d.Verdict, d.EntryID, d.Matches)
			}
		})
	}
}

// TestConfigOnTheLineIsReadAsWhatItDoes: a setting handed to git on its own
// command line is the flag or the refspec it stands for, so the entry for that
// flag reads it.
func TestConfigOnTheLineIsReadAsWhatItDoes(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`git -c remote.origin.push=+HEAD:refs/heads/main push origin`, VerdictBlock, "git-push-force-refspec"},
		{`git -c REMOTE.origin.PUSH=+HEAD:main push origin`, VerdictBlock, "git-push-force-refspec"},
		{`git -C /repo -c remote.origin.push=+HEAD:main push`, VerdictBlock, "git-push-force-refspec"},
		{`git -c remote.origin.push=+HEAD:main -c alias.p=push p origin`, VerdictBlock, "git-push-force-refspec"},
		{`GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=remote.origin.push GIT_CONFIG_VALUE_0=+HEAD:main git push origin`, VerdictBlock, "git-push-force-refspec"},
		{`sh -c 'git -c remote.origin.push=+HEAD:main push origin'`, VerdictBlock, "git-push-force-refspec"},
		{`git -c remote.origin.mirror=true push origin`, VerdictBlock, "git-push-mirror"},
		{`git -c remote.origin.mirror=1 push origin`, VerdictBlock, "git-push-mirror"},
		{`git -c remote.origin.mirror push origin`, VerdictBlock, "git-push-mirror"},
		{`git -c gc.pruneExpire=now gc`, VerdictBlock, "git-gc-prune"},
		{`git -c gc.pruneexpire=all gc --aggressive`, VerdictBlock, "git-gc-prune"},
		{`git -c gc.reflogExpire=now gc`, VerdictBlock, "git-gc-prune"},
		{`git -c gc.reflogExpireUnreachable=now gc`, VerdictBlock, "git-gc-prune"},
	})
	for _, cmd := range []string{
		`git -c remote.origin.push=main:main push origin`,
		`git -c remote.origin.push=+HEAD:main fetch origin`,
		`git -c remote.origin.mirror=false push origin`,
		`git -c remote.origin.mirror=true fetch origin`,
		`git -c gc.pruneExpire=never gc`,
		`git -c gc.auto=0 gc`,
		`git -c gc.pruneExpire=now log`,
		`git -c user.name=x push origin main`,
	} {
		t.Run(cmd, func(t *testing.T) {
			if d := readingOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("Check(%q) = %q via %q (%v), want allow", cmd, d.Verdict, d.EntryID, d.Matches)
			}
		})
	}
}

// TestForcedRefMovesThroughPlumbingAndTheForge: the routes to a force push that
// no flag of `git push` names.
func TestForcedRefMovesThroughPlumbingAndTheForge(t *testing.T) {
	const ref = "repos/o/r/git/refs/heads/main"
	runVerdictCases(t, []verdictCase{
		{`printf '+HEAD:refs/heads/main\n' | git send-pack --stdin ../r3.git`, VerdictBlock, "git-send-pack-force"},
		{`git send-pack --stdin ../r3.git < refs.txt`, VerdictBlock, "git-send-pack-force"},

		{`gh api -X PATCH ` + ref + ` --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH ` + ref + ` --input -`, VerdictBlock, "gh-api-ref-input"},
		{`gh api --method PATCH /` + ref + ` --input=body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH https://api.github.com/` + ref + ` --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH repos/$R/git/refs/heads/$B --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH "$URL" --input body.json`, VerdictBlock, "gh-api-ref-input"},

		{`gh api -X PATCH ` + ref + ` -F force="$V"`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH ` + ref + ` -F force=$(cat f)`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH ` + ref + ` --field=force="$V"`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH ` + ref + ` -F "$FIELD"`, VerdictBlock, "gh-api-ref-force"},

		{`gh api graphql -f query='mutation{updateRef(input:{refId:"x",oid:"y",force:true}){clientMutationId}}'`, VerdictBlock, "gh-api-graphql-ref-force"},
		{`gh api graphql -f 'query=mutation { updateRef(input: { refId: "x", oid: "y", force: true }) { clientMutationId } }'`, VerdictBlock, "gh-api-graphql-ref-force"},
		{`gh api graphql --raw-field=query='mutation{updateRef(input:{refId:"x",oid:"y",force:true}){clientMutationId}}'`, VerdictBlock, "gh-api-graphql-ref-force"},
		{`gh api graphql -f query='mutation($f:Boolean!){updateRef(input:{refId:"x",oid:"y",force:$f}){clientMutationId}}' -F f=true`, VerdictBlock, "gh-api-graphql-ref-force"},
		{`gh api graphql -f query='mutation{updateRefs(input:{repositoryId:"x",refUpdates:[{name:"n",afterOid:"y",force:true}]}){clientMutationId}}'`, VerdictBlock, "gh-api-graphql-ref-force"},
		{`gh api --hostname github.com graphql -f query='mutation{updateRef(input:{refId:"x",oid:"y",force:true}){clientMutationId}}'`, VerdictBlock, "gh-api-graphql-ref-force"},

		{`gh repo sync --force`, VerdictBlock, "gh-repo-sync-force"},
		{`gh repo sync o/fork --source o/r --force`, VerdictBlock, "gh-repo-sync-force"},
		{`sh -c 'gh repo sync --force'`, VerdictBlock, "gh-repo-sync-force"},
	})
	for _, cmd := range []string{
		`git send-pack origin main`,
		`gh api -X PATCH repos/o/r --input body.json`,
		`gh api -X PATCH repos/o/r/pulls/1 --input -`,
		`gh api -X PATCH repos/o/r/pulls/$N --input body.json`,
		`gh api repos/o/r/git/refs --input body.json`,
		`gh api -X PATCH ` + ref + ` -F force=false`,
		`gh api -X PATCH ` + ref + ` -F sha="$SHA"`,
		`gh api -X PATCH repos/o/r/pulls/1 -F force=true`,
		`gh api graphql -f query='query{repository(owner:"o",name:"r"){name}}'`,
		`gh api graphql -f query='query{repository(owner:"o",name:"r"){ref(qualifiedName:"main"){name}}}'`,
		`gh api graphql -f query='mutation{updateRef(input:{refId:"x",oid:"y"}){clientMutationId}}'`,
		`gh api graphql -f query='mutation{updateRef(input:{refId:"x",oid:"y",force:false}){clientMutationId}}'`,
		`gh repo sync`,
		`gh repo sync o/fork --source o/r`,
		`gh repo view --json name`,
		`abcd capture "gh repo sync --force moved the branch"`,
	} {
		t.Run(cmd, func(t *testing.T) {
			if d := readingOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("Check(%q) = %q via %q (%v), want allow", cmd, d.Verdict, d.EntryID, d.Matches)
			}
		})
	}
}

// TestReadOnlyAndRemovingFormsPass: a form that only looks, or only takes a
// stand-in away, is no rewrite, and the entries for the forms beside it do not
// reach it; a line that cancels the dry run is the real thing.
func TestReadOnlyAndRemovingFormsPass(t *testing.T) {
	for _, cmd := range []string{
		`git replace -d abc def`,
		`git replace --delete abc def`,
		`git replace --del abc def`,
		`git replace -l abc def`,
		`git prune -n`,
		`git prune --dry-run`,
		`git prune --dry-r`,
		`git prune -nv`,
		`git prune -n --expire=now`,
	} {
		t.Run(cmd, func(t *testing.T) {
			if d := readingOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("Check(%q) = %q via %q (%v), want allow", cmd, d.Verdict, d.EntryID, d.Matches)
			}
		})
	}
	runVerdictCases(t, []verdictCase{
		{`git replace abc def`, VerdictBlock, "git-replace"},
		{`git prune`, VerdictBlock, "git-prune"},
		{`git prune -v`, VerdictBlock, "git-prune"},
		{`git prune --dry-run --no-dry-run`, VerdictBlock, "git-prune"},
		{`git prune -$(x)`, VerdictBlock, "git-prune"},
		{`git prune -- -n`, VerdictBlock, "git-prune"},
	})
}
