package guard

import "testing"

func assertAllowed(t *testing.T, cmds []string) {
	t.Helper()
	for _, cmd := range cmds {
		t.Run(cmd, func(t *testing.T) {
			if d := readingOf(t, cmd); d.Verdict != VerdictAllow {
				t.Errorf("Check(%q) = %q via %q (%v), want allow", cmd, d.Verdict, d.EntryID, d.Matches)
			}
		})
	}
}

// TestStandDownIsReadAfterTheSubcommand: a global option's value is not the
// subcommand's flag, nor is an option's own value, so neither stands an entry
// down; and only `git prune` and `git replace` stand down at all.
func TestStandDownIsReadAfterTheSubcommand(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`git --namespace -n prune --expire=now`, VerdictBlock, "git-prune"},
		{`git -C -n prune`, VerdictBlock, "git-prune"},
		{`git --namespace -d replace A B`, VerdictBlock, "git-replace"},
		{`git --namespace -l replace A B`, VerdictBlock, "git-replace"},
		{`git --namespace -n reflog expire --all`, VerdictBlock, "git-reflog-expire"},
		{`git prune -n --no-dr`, VerdictBlock, "git-prune"},
		{`git prune -n --no-dry-run`, VerdictBlock, "git-prune"},
		{`git prune -n "$X"`, VerdictBlock, "git-prune"},
		{`git "$(true)" prune -n`, VerdictBlock, ""},
		{`git reflog expire -n --no-dr --expire=now --all`, VerdictBlock, "git-reflog-expire"},
		{`git reflog expire --dry-run --all`, VerdictBlock, "git-reflog-expire"},
		{`git reflog delete -n HEAD@{1}`, VerdictBlock, "git-reflog-delete"},
	})
	assertAllowed(t, []string{
		`git prune -n`,
		`git prune --dry-run`,
		`git prune --dry-r`,
		`git -C /repo prune -n`,
		`git --namespace x prune -n`,
		`git replace -d A B`,
		`git replace A -d B`,
		`git -C /repo replace -l A B`,
	})
}

// TestDashedGitProgramSpelledAsAGlob: a pattern in the program's word is read as
// the names it can expand to, the `git-` of it included.
func TestDashedGitProgramSpelledAsAGlob(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`/opt/homebrew/bin/g?t-filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`g?t-filter-repo --force`, VerdictBlock, "git-filter-repo"},
		{`g*-filter-branch HEAD`, VerdictBlock, "git-filter-branch"},
		{`[g]it-update-ref -d refs/heads/main`, VerdictBlock, "git-update-ref"},
		{`g?t-push --force origin main`, VerdictBlock, "git-push-force"},
	})
	assertAllowed(t, []string{
		`g?t-status --short`,
		`ls *`,
		`cat g?t-filter-repo`,
	})
}

// TestForgeRefPathForms: a ref path is one wherever the API is mounted and
// however the repository is named, and a path whose fixed text names another
// resource is not one, however many substitutions it holds.
func TestForgeRefPathForms(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`gh api -X PATCH repositories/123/git/refs/heads/main -F force=true`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH https://ghe.example/api/v3/repos/o/r/git/refs/heads/main -F force=true`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH api/v3/repos/o/r/git/refs/heads/main --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH repositories/123/git/refs/heads/main --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH repos/o/r/git//refs/heads/main -F force=true`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH repos/o/r/git/refs%2Fheads%2Fmain -F force=true`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH repos/$R/git/$X --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH repositories/$ID/$X --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH "$BASE/$REST" --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH repos/$R --input body.json`, VerdictBlock, "gh-api-ref-input"},
	})
	assertAllowed(t, []string{
		`gh api -X PATCH repos/$OWNER/$REPO/pulls/$N --input body.json`,
		`gh api -X PATCH repos/$R/pulls/1 --input body.json`,
		`gh api -X PATCH repos/o/r/pulls/$N --input body.json`,
		`gh api -X PATCH repos/o/r/pul$N --input body.json`,
		`gh api -X PATCH "$BASE/repos/o/r/pulls/1" --input body.json`,
		`gh api -X PATCH notifications/$ID --input body.json`,
		`gh api -X PATCH repositories/123/pulls/1 --input body.json`,
		`gh api -X PATCH https://ghe.example/api/v3/repos/o/r/pulls/1 --input body.json`,
		`gh api -X PATCH repos/o/r/git/commits/abc --input body.json`,
	})
}

// TestForgeValueFlagsTakeTheirValue: a word an option takes is no operand, so a
// variable holding a file or a field is not read as the ref path.
func TestForgeValueFlagsTakeTheirValue(t *testing.T) {
	assertAllowed(t, []string{
		`gh api -X PATCH repos/o/r/pulls/1 --input "$BODY"`,
		`gh api -X PATCH repos/o/r/pulls/1 -F "$FIELD"`,
		`gh api -X PATCH repos/o/r/pulls/1 -f "$FIELD" --input "$BODY"`,
		`gh api -X PATCH repos/o/r/pulls/1 -H "$HEADER" --input body.json`,
	})
	runVerdictCases(t, []verdictCase{
		{`gh api -X PATCH repos/o/r/git/refs/heads/main --input "$BODY"`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH repos/o/r/git/refs/heads/main -F "$FIELD"`, VerdictBlock, "gh-api-ref-force"},
	})
}

// TestGraphQLForceIsReadInTheQueryAndItsFields: only a query field is a query,
// and force set beside one that names updateRef sets force on it.
func TestGraphQLForceIsReadInTheQueryAndItsFields(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`gh api graphql -f query='mutation{updateRef(input:{refId:"x",oid:"y",force:true}){clientMutationId}}'`, VerdictBlock, "gh-api-graphql-ref-force"},
		{`gh api graphql -f query='mutation($i:UpdateRefInput!){updateRef(input:$i){clientMutationId}}' -F 'i[force]=true'`, VerdictBlock, "gh-api-graphql-ref-force-field"},
		{`gh api graphql -F 'input[force]=true' -f query='mutation($input:UpdateRefInput!){updateRef(input:$input){clientMutationId}}'`, VerdictBlock, "gh-api-graphql-ref-force-field"},
		{`gh api graphql -f query='mutation($i:UpdateRefInput!){updateRef(input:$i){clientMutationId}}' -F force=true`, VerdictBlock, "gh-api-graphql-ref-force-field"},
		{`gh api graphql -f query='mutation($i:UpdateRefInput!){updateRef(input:$i){clientMutationId}}' -F 'i[force]='"$V"`, VerdictBlock, "gh-api-graphql-ref-force-field"},
	})
	assertAllowed(t, []string{
		`gh api repos/o/r/issues/1/comments -f body='updateRef with force: true is refused'`,
		`gh api repos/o/r/issues -f title=x -f body='mutation{updateRef(input:{force:true})}'`,
		`gh api graphql -f query='mutation($i:UpdateRefInput!){updateRef(input:$i){clientMutationId}}' -F 'i[oid]=abc'`,
		`gh api graphql -f query='mutation($i:UpdateRefInput!){updateRef(input:$i){clientMutationId}}' -F 'i[force]=false'`,
		`gh api graphql -f query='query{viewer{login}}' -F 'i[force]=true'`,
	})
}

// TestForgeRefPathIsRuledOutOnlyByANamedRoute: a word a ref path can hold proves
// nothing about a path a substitution helps to write. Only fixed text naming a
// REST route that is not a ref rules it out.
func TestForgeRefPathIsRuledOutOnlyByANamedRoute(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`gh api -X PATCH "$BASE/refs/heads/$B" --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH "$BASE/refs/heads/$B" -F force=true`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH "$BASE/refs/heads/pulls/1" --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH "$BASE/heads/$B" --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH "$BASE/unheard/of" --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH repos/o/r/git/re$(x)fs/heads/main --input body.json`, VerdictBlock, "gh-api-ref-input"},
		{`gh api -X PATCH repos/o/r/git/re$(x)fs/heads/main -F force=true`, VerdictBlock, "gh-api-ref-force"},
		{`gh api -X PATCH repos/o/r/g$(x)it/refs/heads/main -F force=true`, VerdictBlock, "gh-api-ref-force"},
	})
	assertAllowed(t, []string{
		`gh api -X PATCH repos/$OWNER/$REPO/pulls/$N --input body.json`,
		`gh api -X PATCH "$BASE/repos/o/r/issues/1" --input body.json`,
		`gh api -X PATCH repos/$R/releases/1 --input body.json`,
	})
}

// TestGraphQLFieldWithAnUnresolvedKey: on a GraphQL call a field whose key the
// line does not fix can be the query, so its value is read as one.
func TestGraphQLFieldWithAnUnresolvedKey(t *testing.T) {
	const forced = `mutation{updateRef(input:{refId:\"x\",oid:\"y\",force:true}){clientMutationId}}`
	runVerdictCases(t, []verdictCase{
		{`gh api graphql -f "$K=` + forced + `"`, VerdictBlock, "gh-api-graphql-any-key-ref-force"},
		{`gh api --hostname github.com graphql -F "$K=` + forced + `"`, VerdictBlock, "gh-api-graphql-any-key-ref-force"},
		{`gh api graphql --raw-field="$K=` + forced + `"`, VerdictBlock, "gh-api-graphql-any-key-ref-force"},
	})
	assertAllowed(t, []string{
		`gh api repos/o/r/issues/1/comments -f "$K=` + forced + `"`,
		`gh api graphql -f "$K=query{viewer{login}}"`,
		`gh api graphql -f "body=` + forced + `"`,
	})
}

// TestCaseLabelIsNoDashedGitProgram: a `case` label is a pattern in command
// position, and one that fixes no letter of a name is not read as a program.
func TestCaseLabelIsNoDashedGitProgram(t *testing.T) {
	assertAllowed(t, []string{
		`case "$x" in *[!0-9]*) echo bad ;; esac`,
		`case "$x" in [A-Za-z0-9]*) y=1 ;; esac`,
		`case "$x" in [Ii][Tt][Dd]-*) y=1 ;; esac`,
	})
	for _, line := range []string{
		`*[!0-9]*) record_refuse "$x" "y" ;;`,
		`[A-Za-z0-9]*) key="$first"; pat="$rest" ;;`,
		`[Ii][Tt][Dd]-*) ;;`,
	} {
		t.Run(line, func(t *testing.T) {
			d := readingOf(t, line)
			for _, id := range []string{"git-filter-branch", "git-filter-repo", "git-update-ref", "git-prune"} {
				if contains(d.Matches, id) {
					t.Errorf("Check(%q) matches %s (%v): a label that fixes no letter is no program", line, id, d.Matches)
				}
			}
		})
	}
}

// TestARefMovedWithForceIsReadFromEverySpellingGHSends: gh's -F converts true, whole numbers
// and null and reads @file, its -f sends text, and the rule asks only whether force is off.
func TestARefMovedWithForceIsReadFromEverySpellingGHSends(t *testing.T) {
	const ref = `gh api -X PATCH repos/o/r/git/refs/heads/main `
	bad := []verdictCase{}
	for _, field := range []string{
		`-F force=true`, `-f force=true`, `-F force=TRUE`, `-f force=True`,
		`-F force=1`, `-f force=1`, `-F force=2`, `-F force=-1`, `-F force=+1`, `-F force=01`,
		`-f force=t`, `-f force=T`, `-f force=y`, `-f force=yes`, `-f force=on`, `-f force=no`, `-f force=off`,
		`-F force=@flag.txt`, `-F force=@-`, `-F force={owner}`,
		`-F 'force[]=1'`, `-F 'force[x]=1'`, `-f 'force[]=true'`,
		`-F Force=1`, `-F FORCE=1`, `-F 'force= false'`,
		`--field force=1`, `--field=force=1`, `--raw-field force=1`, `--raw-field=force=1`, `-Fforce=1`, `-fforce=1`,
		`-F "force=1"`, `-F force=f"$X"`, `-F force=false"$X"`, `-F force=1"$X"`, `-F force=$(cat f)`,
		`-F "$K=1"`, `-F "f$K=1"`, `-F "force$K=1"`, `-F "force[$K]=1"`,
		`-F forc?=1`, `-F force=tru?`, `-$F`, `-$F sha=abc`,
		`-F force=false -F force=1`, `-F sha=abc -f force=1`,
		`-iF force=true`, `-if force=1`, `-iFforce=1`, `-ifforce=true`, `-iF=force=1`, `-qiF force=1`,
	} {
		bad = append(bad, verdictCase{ref + field, VerdictBlock, "gh-api-ref-force"})
	}
	runVerdictCases(t, bad)
	good := []string{}
	for _, field := range []string{
		`-F force=false`, `-f force=false`, `--field force=false`, `--raw-field=force=false`, `-Fforce=false`,
		`-F force=False`, `-F force=FALSE`, `-F force=0`, `-f force=0`, `-F force=null`, `-F force=`, `-f force=`,
		`-f sha=abc1234`, `-F sha="$SHA"`, `-F forced=1`, `-F 'x[force]=1'`, `-f notforce=1`, `-f message=force=1`,
		`-F forc?=false`, `-iF force=false`, `-iFforce=false`, `-iF sha=abc`, `-if sha=abc`,
	} {
		good = append(good, ref+field)
	}
	bad = bad[:0]
	for _, line := range []string{
		`gh api -iX PATCH repos/o/r/git/refs/heads/main -F force=1`,
		`gh api -iXPATCH repos/o/r/git/refs/heads/main -F force=1`,
	} {
		bad = append(bad, verdictCase{line, VerdictBlock, "gh-api-ref-force"})
	}
	runVerdictCases(t, bad)
	good = append(good,
		`gh api -X PATCH repos/o/r/pulls/1 -F force=1`,
		`gh api repos/o/r/git/refs/heads/main -F force=1 -X GET`,
	)
	assertAllowed(t, good)
}
