package loop

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

// The brief of a run with members (itd-2611201450052326, spc-2611201455051897
// step 2): every issue's record and remedy from the lane's base, the lead
// first and each member after it in the run's order, one commit asked for each
// issue, and the receipt's `handbacks` named; a run without members is briefed
// byte for byte as before.

// unbundledBriefFixture is a run without members and the sources its brief is
// rendered from, fixed, for the byte-for-byte comparison with the brief as it
// was rendered before a run could carry members.
func unbundledBriefFixture() (State, Lane, issueBriefSources) {
	st := State{RunID: "run-1", Key: eligibleIssue}
	lane := Lane{ID: "lane-1", Branch: "build/run-1-lane-1", BaseSHA: strings.Repeat("a", 40), Worktree: "/wt/lane-1"}
	src := issueBriefSources{
		issuePath: ".abcd/work/issues/open/" + eligibleIssue + "-renderer.md", issueText: "---\nid: " + eligibleIssue + "\n---\n\nThe renderer panics.\n",
		remedy: issueRemedy, conventions: "Working conventions: test first.", conventionsFrom: "its marked section",
		decisions: namedDecisions{hasFrozen: true, frozen: []string{"- 2026-09-30 — " + eligibleIssue + " waits for the drain."}},
	}
	return st, lane, src
}

// TestARunWithoutMembersIsBriefedByteForByteAsBefore: the brief of a run
// without members equals the one rendered before a run could carry members.
func TestARunWithoutMembersIsBriefedByteForByteAsBefore(t *testing.T) {
	want, err := os.ReadFile("testdata/issue-brief-unbundled.golden")
	if err != nil {
		t.Fatal(err)
	}
	st, lane, src := unbundledBriefFixture()
	if got := renderIssueBrief(st, lane, "lane-dir", src); !bytes.Equal(got, want) {
		t.Fatalf("a run without members is briefed as before:\n%s", got)
	}
}

// TestABundleBriefRendersEveryIssueFromTheLanesBase: the brief of a run with
// members quotes each issue's record and remedy as the lane's base holds it,
// the lead first and the members after it in the run's order, asks for one
// commit per issue named in its resolution, and names `handbacks`; a record
// changed on the default branch after the base does not reach it.
func TestABundleBriefRendersEveryIssueFromTheLanesBase(t *testing.T) {
	repo, _ := bundleRepo(t)
	now := time.Now().UTC().Truncate(time.Second)
	runID := bundleRun(t, repo, now)
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSpace(repo.Git("rev-parse", "main"))
	lane := Lane{ID: "lane-1", Branch: "build/" + runID + "-lane-1", BaseSHA: base, Worktree: "/wt/lane-1"}

	src, err := readIssueBriefSources(repo.Root(), st, &lane)
	if err != nil {
		t.Fatalf("the members' records are read at the base: %v", err)
	}
	if len(src.members) != 2 || src.members[0].id != bundleTwo || src.members[1].id != bundleThree {
		t.Fatalf("each member is read, in the run's order: %+v", src.members)
	}
	// A member's record edited on the default branch after the base.
	edited := src.members[0].path
	repo.Write(edited, strings.Replace(src.members[0].text, "Correct the page's sentence.", "A REMEDY WRITTEN AFTER THE BASE.", 1))
	repo.Commit("an edit after the lane's base")
	if src, err = readIssueBriefSources(repo.Root(), st, &lane); err != nil {
		t.Fatal(err)
	}
	brief := string(renderIssueBrief(st, lane, "lane-dir", src))
	if strings.Contains(brief, "A REMEDY WRITTEN AFTER THE BASE.") {
		t.Fatalf("a member's record is read at the lane's base, not the default branch's tip:\n%s", brief)
	}
	var at []int
	for _, id := range bundleIssues {
		i := strings.Index(brief, "### "+id+"'s remedy")
		if i < 0 {
			t.Fatalf("the brief quotes %s's remedy:\n%s", id, brief)
		}
		at = append(at, i)
		if !strings.Contains(brief, "## The issue: "+id) {
			t.Fatalf("the brief quotes %s's record:\n%s", id, brief)
		}
		if !strings.Contains(brief, `{"issue": "`+id+`"`) {
			t.Fatalf("the receipt's shape names %s:\n%s", id, brief)
		}
	}
	if !slices.IsSorted(at) || !strings.Contains(brief, "### "+bundleLead+"'s remedy (the lead)") {
		t.Fatalf("the lead is first and the members follow in the run's order: %v", at)
	}
	for _, want := range []string{
		"Make one commit for each issue, and name that commit in the issue's resolution.",
		`"handbacks": []`,
		"Name every issue of the run, " + bundleLead + ", " + bundleTwo + " and " + bundleThree + ", exactly once in every receipt",
		"or carries a top-level `handback` is refused",
		"Correct the page's sentence.",
	} {
		if !strings.Contains(brief, want) {
			t.Errorf("the bundle's brief carries %q:\n%s", want, brief)
		}
	}
	if open, close := fenceMarkers(brief); open != 2*len(bundleIssues)+len(bundleIssues)*2+2 || close != open {
		t.Fatalf("the brief writes only its own fence markers, one pair per remedy, record and the conventions (%d openers, %d closers)", open, close)
	}
}
