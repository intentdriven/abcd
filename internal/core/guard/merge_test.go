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

// TestTheApprovalActIsThePersons: recording the technical facilitator's
// approval is refused to an agent in any mode, through the binary (the
// registry entry) and through go run (the guard's own), and every other land
// spelling is allowed.
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
			{`go run ./cmd/abcd land --approve`, approveGoRunEntryID},
			{`go run ./cmd/abcd/ land feat/x --approve=true`, approveGoRunEntryID},
			{`go run -race github.com/intentdriven/abcd/cmd/abcd@latest land --approve`, approveGoRunEntryID},
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
		} {
			if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
				t.Fatalf("human %v: Check(%q) = %+v, %v; want it allowed", human, cmd, d, err)
			}
		}
	}
	if d, err := Defaults().Check(`go run ./cmd/abcd land --approve`); err != nil || d.Verdict != VerdictBlock {
		t.Fatalf("the approval act is refused with no mode handed in too: %+v, %v", d, err)
	}
}
