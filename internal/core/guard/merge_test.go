package guard

import (
	"strings"
	"testing"
)

// modeOf is a MergeMode that reports human or auto, counting its reads.
func modeOf(human bool, reads *int) MergeMode {
	return func() (bool, string) {
		*reads++
		if human {
			return true, "facilitator.mode is human (bundled layer, bundled)"
		}
		return false, "facilitator.mode is auto (repo layer, .abcd/config.json at origin/main)"
	}
}

// TestHumanModeRefusesTheMergeCommands is spc-2610161609350616 step 5: while
// facilitator.mode is human, `gh pr merge --auto`, a `gh pr merge` that
// merges, and the GraphQL enqueue and auto-merge mutations are refused,
// naming the setting that allows each; withdrawing an arming is not a merge.
// In auto every one is allowed, and the mode is read only when a merge
// command asks.
func TestHumanModeRefusesTheMergeCommands(t *testing.T) {
	for _, c := range []struct {
		cmd   string
		entry string
	}{
		{`gh pr merge 123 --auto`, mergeAutoEntryID},
		{`gh pr merge 123 --auto --squash --match-head-commit=abc`, mergeAutoEntryID},
		{`gh pr merge --auto=true 123`, mergeAutoEntryID},
		{`gh -R o/r pr merge 123 --auto`, mergeAutoEntryID},
		{`sh -c 'gh pr merge 123 --auto'`, mergeAutoEntryID},
		{`gh pr merge 123`, mergeNowEntryID},
		{`gh pr merge 123 --squash`, mergeNowEntryID},
		{`gh api graphql -f query='mutation($id:ID!){enqueuePullRequest(input:{pullRequestId:$id}){clientMutationId}}' -f id=X`, mergeEnqueueEntryID},
		{`gh api graphql -f query='mutation{enablePullRequestAutoMerge(input:{pullRequestId:"X"}){clientMutationId}}'`, mergeEnqueueEntryID},
		{`gh api graphql -f query='mutation{mergePullRequest(input:{pullRequestId:"X"}){clientMutationId}}'`, mergeEnqueueEntryID},
		{`gh api graphql -F query=@q.graphql`, mergeUnreadEntryID},
		{`gh api graphql -Fquery=@q.graphql -F id=X`, mergeUnreadEntryID},
		{`gh api graphql --field=query=@- < q.graphql`, mergeUnreadEntryID},
		{`gh api graphql --input body.json`, mergeUnreadEntryID},
		{`gh api graphql -f query="$(cat q.graphql)"`, mergeUnreadEntryID},
		{`gh api graphql -f "$Q"`, mergeUnreadEntryID},
		{`gh api -X PUT repos/{owner}/{repo}/pulls/123/merge`, mergeRESTEntryID},
		{`gh api --method=put /repos/{owner}/{repo}/pulls/123/merge -f merge_method=squash`, mergeRESTEntryID},
		{`gh api -XPUT repos/{owner}/{repo}/pulls/123/merge`, mergeRESTEntryID},
		{`gh api --method "$M" repos/{owner}/{repo}/pulls/123/merge`, mergeRESTEntryID},
		{`gh api --method=$M repos/{owner}/{repo}/pulls/123/merge`, mergeRESTEntryID},
		{`gh api -X"$M" repos/{owner}/{repo}/pulls/123/merge`, mergeRESTEntryID},
		{`gh api -X PUT "repos/$REPO/pulls/$N/merge"`, mergeRESTEntryID},
		{`gh api -X PUT "$URL"`, mergeRESTEntryID},
		{`gh api -H 'Accept: application/vnd.github+json' graphql -f query='mutation{mergePullRequest(input:{pullRequestId:"X"}){clientMutationId}}'`, mergeEnqueueEntryID},
		{`gh api --hostname h.example graphql -F query=@q.graphql`, mergeUnreadEntryID},
		{`gh api --method POST graphql --input body.json`, mergeUnreadEntryID},
		{`gh api "$EP" -f query='mutation{mergePullRequest(input:{pullRequestId:"X"}){clientMutationId}}'`, mergeEnqueueEntryID},
		{`gh api /graphql -f query='mutation{mergePullRequest(input:{pullRequestId:"X"}){clientMutationId}}'`, mergeEnqueueEntryID},
		{`gh api -f query='mutation{enqueuePullRequest(input:{pullRequestId:"X"}){clientMutationId}}' graphql`, mergeEnqueueEntryID},
		{`gh api graphql -F=query=@q.graphql`, mergeUnreadEntryID},
		{`gh api -X=PUT repos/{owner}/{repo}/pulls/123/merge`, mergeRESTEntryID},
		{`gh api -X PUT repos/:owner/:repo/pulls/123/merge`, mergeRESTEntryID},
	} {
		t.Run(c.cmd, func(t *testing.T) {
			reads := 0
			d, err := Defaults().WithMergeMode(modeOf(true, &reads)).Check(c.cmd)
			if err != nil {
				t.Fatal(err)
			}
			if d.Verdict != VerdictBlock || d.EntryID != c.entry || !strings.Contains(d.Message, "facilitator.mode: auto") {
				t.Fatalf("Check(%q) = %q via %q (%s), want a block via %q naming the setting", c.cmd, d.Verdict, d.EntryID, d.Message, c.entry)
			}
			d, err = Defaults().WithMergeMode(modeOf(false, &reads)).Check(c.cmd)
			if err != nil || d.Verdict == VerdictBlock {
				t.Fatalf("in auto, Check(%q) = %+v, %v; want it not blocked", c.cmd, d, err)
			}
		})
	}
	for _, cmd := range []string{
		`gh pr merge 123 --disable-auto`,
		`gh pr view 123 --json state`,
		`gh api graphql -f query='query{viewer{login}}'`,
		`gh api graphql -f query='query($id:ID!){node(id:$id){id}}' -F id="$ID"`,
		`gh api repos/{owner}/{repo}/pulls/123/merge`,
		`gh api -X GET repos/{owner}/{repo}/pulls/123/merge`,
		`gh api -X PUT repos/{owner}/{repo}/pulls/123/requested_reviewers`,
		`gh api "repos/$REPO/pulls/$N"`,
		`gh api -X PUT "repos/$REPO/collaborators/$U"`,
		`gh api -H 'Accept: application/vnd.github+json' repos/{owner}/{repo}/pulls/123`,
		`gh api -F file=@notes.md repos/{owner}/{repo}/issues`,
		`git merge main`,
		`abcd capture "an agent ran gh pr merge 123 --auto"`,
	} {
		reads := 0
		d, err := Defaults().WithMergeMode(modeOf(true, &reads)).Check(cmd)
		if err != nil || d.Verdict == VerdictBlock {
			t.Fatalf("in human, Check(%q) = %+v, %v; want it not blocked", cmd, d, err)
		}
		if strings.HasPrefix(cmd, "git") && reads != 0 {
			t.Fatalf("Check(%q) read the mode %d times; a line with no merge command reads none", cmd, reads)
		}
	}
	// With no resolver handed in, the guard cannot tell the mode and judges
	// no merge command.
	if d, err := Defaults().Check(`gh pr merge 123 --auto`); err != nil || d.Verdict == VerdictBlock {
		t.Fatalf("a registry with no mode = %+v, %v", d, err)
	}
}

// TestAMergeNamingItsRepositoryCountsAsHuman: the mode read is this
// checkout's, so a merge command that names its repository (-R, --repo, a
// GH_REPO prefix, a REST route spelling owner and name) is refused in auto
// too, saying why; with gh's placeholders the REST route is this checkout's
// and auto allows it.
func TestAMergeNamingItsRepositoryCountsAsHuman(t *testing.T) {
	for _, c := range []struct {
		cmd   string
		entry string
	}{
		{`gh -R o/r pr merge 123 --auto`, mergeAutoEntryID},
		{`gh pr merge 123 --repo o/r`, mergeNowEntryID},
		{`gh pr merge 123 --repo=o/r --squash`, mergeNowEntryID},
		{`gh pr merge -Ro/r 123 --auto`, mergeAutoEntryID},
		{`GH_REPO=o/r gh pr merge 123 --auto`, mergeAutoEntryID},
		{`gh api -X PUT repos/o/r/pulls/123/merge`, mergeRESTEntryID},
		{`gh api -X PUT "repos/$REPO/pulls/$N/merge"`, mergeRESTEntryID},
		{`GH_REPO="$R" gh pr merge 1 --auto`, mergeAutoEntryID},
		{`gh pr merge 1 --repo="$R" --auto`, mergeAutoEntryID},
		{`gh pr merge 1 -R"$R" --auto`, mergeAutoEntryID},
		{`export GH_REPO=o/r; gh pr merge 1 --auto`, mergeAutoEntryID},
		{`cd ../other && gh pr merge 1 --auto`, mergeAutoEntryID},
	} {
		reads := 0
		d, err := Defaults().WithMergeMode(modeOf(false, &reads)).Check(c.cmd)
		if err != nil || d.Verdict != VerdictBlock || d.EntryID != c.entry || !strings.Contains(d.Message, "names the repository it merges in") || strings.Contains(d.Message, "facilitator.mode: auto") {
			t.Fatalf("in auto, Check(%q) = %q via %q (%s), %v; want a block via %q saying it names its repository, offering no setting", c.cmd, d.Verdict, d.EntryID, d.Message, err, c.entry)
		}
	}
	reads := 0
	for _, cmd := range []string{
		`gh api -X PUT repos/{owner}/{repo}/pulls/123/merge`,
		`gh api -X PUT repos/:owner/:repo/pulls/123/merge`,
		`gh pr merge 123 --auto`,
	} {
		if d, err := Defaults().WithMergeMode(modeOf(false, &reads)).Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Fatalf("in auto, Check(%q) = %+v, %v; want it allowed", cmd, d, err)
		}
	}
}

// TestTheApprovalActIsThePersons: recording the technical facilitator's
// approval is refused to an agent in any mode, through the binary (the
// registry entry) and under any other program name (the guard's own): go run
// of the module or of `.`, a cross-compiled build, a copy renamed, a program
// a variable names, the flag before the verb. An approving review posted on
// the forge is refused the same way; a review that comments or requests
// changes is not. Every other land spelling is allowed.
func TestTheApprovalActIsThePersons(t *testing.T) {
	for _, human := range []bool{true, false} {
		reads := 0
		reg := Defaults().WithMergeMode(modeOf(human, &reads))
		for _, c := range []struct {
			cmd   string
			entry string
		}{
			{`abcd land --approve`, "abcd-land-approve"},
			{`abcd land feat/x --approve`, "abcd-land-approve"},
			{`"$CLAUDE_PLUGIN_ROOT"/abcd land --approve`, "abcd-land-approve"},
			{`go run ./cmd/abcd land --approve`, approveAnyEntryID},
			{`go run ./cmd/abcd/ land feat/x --approve=true`, approveAnyEntryID},
			{`go run -race github.com/intentdriven/abcd/cmd/abcd@latest land --approve`, approveAnyEntryID},
			{`cd cmd/abcd && go run . land --approve`, approveAnyEntryID},
			{`bin/abcd-darwin-arm64 land --approve`, approveAnyEntryID},
			{`/tmp/x land --approve`, approveAnyEntryID},
			{`"$B" land feat/x --approve`, approveAnyEntryID},
			{`env FOO=1 ./tool land --approve`, approveAnyEntryID},
			{`./tool --approve land`, approveAnyEntryID},
			{`bin/x --approve=true land feat/x`, approveAnyEntryID},
			{`gh pr review 12 --approve`, approveForgeEntryID},
			{`gh pr review -a 12 -b ok`, approveForgeEntryID},
			{`gh -R o/r pr review 12 --approve=true`, approveForgeEntryID},
			{`gh api repos/{owner}/{repo}/pulls/12/reviews -f event=APPROVE`, approveForgeEntryID},
			{`gh api -X POST "repos/$R/pulls/$N/reviews/$ID/events" -f event=APPROVE`, approveForgeEntryID},
			{`gh api graphql -f query='mutation{addPullRequestReview(input:{pullRequestId:"X",event:APPROVE}){clientMutationId}}'`, approveForgeEntryID},
		} {
			d, err := reg.Check(c.cmd)
			if err != nil || d.Verdict != VerdictBlock || d.EntryID != c.entry {
				t.Fatalf("human %v: Check(%q) = %q via %q, %v; want a block via %q", human, c.cmd, d.Verdict, d.EntryID, err, c.entry)
			}
		}
		for _, cmd := range []string{
			`abcd land`,
			`go run ./cmd/abcd land`,
			`go run ./cmd/abcd land --record-review a.md --record-review b.md`,
			`go run ./cmd/abcd land --approve=false`,
			`go run ./cmd/abcd capture "abcd land --approve"`,
			`bin/abcd-darwin-arm64 land`,
			`bin/abcd-darwin-arm64 land --approve=false`,
			`bin/abcd-darwin-arm64 capture "land --approve"`,
			`gh pr review 12 --comment -b looks-fine`,
			`gh pr review 12 --request-changes -b "see the approve path"`,
			`gh api repos/{owner}/{repo}/pulls/12/reviews`,
			`gh api repos/{owner}/{repo}/pulls/12/reviews -f event=COMMENT -f body=ok`,
		} {
			if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
				t.Fatalf("human %v: Check(%q) = %+v, %v; want it allowed", human, cmd, d, err)
			}
		}
	}
	for _, cmd := range []string{`go run ./cmd/abcd land --approve`, `gh pr review 12 --approve`} {
		if d, err := Defaults().Check(cmd); err != nil || d.Verdict != VerdictBlock {
			t.Fatalf("the approval act %q is refused with no mode handed in too: %+v, %v", cmd, d, err)
		}
	}
}
