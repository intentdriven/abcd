package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The landing of a run with members (itd-2611201450052326,
// spc-2611201455051897 step 2, criterion 4): one `Resolves:` per member the
// lane resolves, a `Refs:` per member handed back alone, a records commit
// whose body names the bundle in prose, and a pull request titled for the
// area and the count; a run without members lands with the text it had.

// unbundledLandingFixture is a run without members at its landing, for the
// byte-for-byte comparison of its records commit and pull request text.
func unbundledLandingFixture() (State, Lane) {
	st := State{RunID: "run-1", Key: eligibleIssue, Admitted: &Admitted{Kind: "drain", Record: "adr-2609300000000001", Path: ".abcd/development/decisions/adrs/2609300000000001-drain-rule.md"}}
	lane := Lane{ID: "lane-1", Key: eligibleIssue, SpecStep: 1, StepTitle: "The renderer panics", Branch: "build/run-1-lane-1",
		BaseSHA: strings.Repeat("a", 40), HeadSHA: strings.Repeat("b", 40),
		Resolves: []Resolution{{Issue: eligibleIssue, Commit: strings.Repeat("c", 40), Note: "guarded", Impact: "fix", Grounds: "pursued: x"}}}
	return st, lane
}

// landedText is the records commit's message and the pull request's title
// and body for st's lane, as the landing composes them, separated as the
// golden file holds them.
func landedText(st State, lane Lane, done string) string {
	return recordsMessage(st, lane, []string{done}, resolutionTrailers(st, lane)) +
		"\n----\n" + prTitle(st, lane) + "\n----\n" + prBody(st, lane, []string{"Assisted-by: Claude:claude-mythos-5"})
}

// TestARunWithoutMembersLandsWithTheTextItHad: the records commit, the pull
// request's title and its body of a run without members equal what the
// landing wrote before a run could carry members.
func TestARunWithoutMembersLandsWithTheTextItHad(t *testing.T) {
	want, err := os.ReadFile("testdata/issue-landing-unbundled.golden")
	if err != nil {
		t.Fatal(err)
	}
	st, lane := unbundledLandingFixture()
	if got := landedText(st, lane, "resolved "+eligibleIssue+" with aaaaaaa"); got != string(want) {
		t.Fatalf("a run without members lands as before:\n%s", got)
	}
}

// bundleLanding is a run of the three bundle issues at its landing, the
// third handed back alone in a fix round after round 1 resolved it: the
// lane resolves the lead and the second.
func bundleLanding() (State, Lane) {
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	st := State{RunID: "run-1", Key: bundleLead,
		Admitted: &Admitted{Kind: "drain", Record: "adr-2609300000000001", Path: ".abcd/development/decisions/adrs/2609300000000001-drain-rule.md"},
		Bundle: []Member{
			{Issue: bundleTwo, Footprint: []string{bundleFiles[bundleTwo]}, ReservedAt: &now},
			{Issue: bundleThree, Footprint: []string{bundleFiles[bundleThree]}, ReservedAt: &now,
				HandBack: &HandBack{At: now, Kind: HandBackUserVisible, Reason: "the correction changes what a reader is told"}},
		}}
	lane := Lane{ID: "lane-1", Key: bundleLead, SpecStep: 1, StepTitle: "The page's sentence is wrong", Branch: "build/run-1-lane-1",
		BaseSHA: strings.Repeat("a", 40), HeadSHA: strings.Repeat("b", 40),
		Resolves: []Resolution{resolving(bundleLead, strings.Repeat("c", 40)), resolving(bundleTwo, strings.Repeat("d", 40)),
			resolving(bundleThree, strings.Repeat("e", 40))}}
	// The fix round's receipt hands the third back (recordReceipt).
	recordReceipt(&lane, "fix/receipt.json", LaneReceipt{Resolves: lane.Resolves[:2], HandBacks: []IssueHandBack{handingBack(bundleThree)}})
	return st, lane
}

// TestABundlesRecordsCommitResolvesEachMemberItFixed: the records commit
// carries one `Resolves:` per member the lane resolves, none for the member
// handed back in a fix round and a `Refs:` for it, and its body names the
// bundle's lead and every member, the one handed back included, in prose
// that no trailer reader takes for a trailer.
func TestABundlesRecordsCommitResolvesEachMemberItFixed(t *testing.T) {
	st, lane := bundleLanding()
	trailers := resolutionTrailers(st, lane)
	want := []string{"Resolves: " + bundleLead, "Resolves: " + bundleTwo, "Refs: " + bundleThree, "Admitted-by: drain adr-2609300000000001"}
	if !slices.Equal(trailers, want) {
		t.Fatalf("one Resolves: per member resolved and a Refs: per member handed back:\n got %v\nwant %v", trailers, want)
	}
	msg := recordsMessage(st, lane, []string{"resolved " + bundleLead + " with ccccccc", "resolved " + bundleTwo + " with ddddddd"}, append(trailers, composedAssistedBy()))
	body, _, _ := strings.Cut(msg, "\nResolves: ")
	for _, want := range []string{
		"The implement loop's landing for the bundle led by " + bundleLead,
		"This is the landing of a bundle of 3 issues of one kind in one place: its lead is " + bundleLead + ", and its members are " + bundleTwo + " and " + bundleThree + ".",
		bundleThree + " was handed back alone and is not resolved here.",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("the records commit's body says %q:\n%s", want, msg)
		}
	}
	read := landingTrailers(msg)
	wantRead := append(slices.Clone(want), composedAssistedBy())
	slices.Sort(wantRead)
	if !slices.Equal(read, wantRead) {
		t.Fatalf("the prose adds no trailer:\n got %v\nwant %v", read, wantRead)
	}
	// Once the admission declares the member, it is declared once.
	st.Admitted = &Admitted{Kind: "build", Record: bundleThree}
	if got := resolutionTrailers(st, lane); strings.Count(strings.Join(got, "\n"), "Refs: "+bundleThree) != 1 {
		t.Fatalf("an issue the admission declares is declared once: %v", got)
	}
}

// TestABundlesPullRequestNamesItsAreaAndCount: the pull request's title names
// the area and how many issues it resolves, and its body lists every member,
// the one handed back with its `Refs:`.
func TestABundlesPullRequestNamesItsAreaAndCount(t *testing.T) {
	st, lane := bundleLanding()
	if got, want := prTitle(st, lane), "fix(docs): 2 issues, "+bundleLead+" and 1 more"; got != want {
		t.Fatalf("the title names the area and the count:\n got %q\nwant %q", got, want)
	}
	body := prBody(st, lane, nil)
	for _, want := range []string{
		"This pull request fixes the bundle led by " + bundleLead + ", 3 issues of one kind in docs, each by its own remedy.",
		"- It resolves " + bundleLead + ", fixed by cccccccccccc.",
		"- It resolves " + bundleTwo + ", fixed by dddddddddddd.",
		"- " + bundleThree + " was handed back alone by the lane, for a person; it is not resolved here.",
		"\nResolves: " + bundleLead + "\nResolves: " + bundleTwo + "\nRefs: " + bundleThree + "\n",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("the body carries %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Resolves: "+bundleThree) {
		t.Fatalf("no Resolves: for the member handed back:\n%s", body)
	}
	// The lead handed back alone: the title names the first issue resolved.
	st.LeadHandBack = &HandBack{Kind: HandBackUserVisible, Reason: "x"}
	st.Bundle[1].HandBack = nil
	lane.Resolves = []Resolution{resolving(bundleTwo, strings.Repeat("d", 40)), resolving(bundleThree, strings.Repeat("e", 40))}
	if got, want := prTitle(st, lane), "fix(docs): 2 issues, "+bundleTwo+" and 1 more"; got != want {
		t.Fatalf("with the lead handed back, the title names the first issue resolved:\n got %q\nwant %q", got, want)
	}
}

// TestABundlesAreaReadsADirectoryFoundAtAsItsOwnArea: the pull request's area
// takes a member's found_at that is a directory in the lane's checkout as the
// area itself, as the rule for who joins does (issueArea), and a file's
// directory otherwise.
func TestABundlesAreaReadsADirectoryFoundAtAsItsOwnArea(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "guide"), 0o755); err != nil {
		t.Fatal(err)
	}
	st := State{Key: bundleLead, Bundle: []Member{
		{Issue: bundleTwo, Footprint: []string{"docs/guide"}},
		{Issue: bundleThree, Footprint: []string{"docs/guide"}},
	}}
	if got := bundleArea(st, root); got != "docs/guide" {
		t.Fatalf("a directory found_at is its own area: %q", got)
	}
	st.Bundle[1].Footprint = []string{"docs/guide/start.md"}
	if got := bundleArea(st, root); got != "docs/guide" {
		t.Fatalf("a file's area is its directory: %q", got)
	}
}
