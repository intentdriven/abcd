package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// The itd-180 sixth-round nits (iss-2608300848049813).

// A run or item directory the walk cannot list is routed to Unsafe with its
// reason, as an unreadable file is, rather than aborting the whole report (and,
// with the rule enabled, the whole lint run).
func TestAnUnlistableRunOrItemDirectoryIsUnsafeNotAnAbort(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	root := readingLedger(t, run, item, "detection")
	dispositionRecord(t, root, item, "dsp-2608300000000003", "accepted")
	other := "rdg-2608300000000005"
	writeFile(t, root, ".abcd/work/issues/readings/"+other+"/rdi-2608300000000006.md", "x")
	unreadableDir(t, filepath.Join(root, ".abcd", "work", "issues", "readings", other))
	unreadableDir(t, filepath.Join(root, ".abcd", "work", "issues", "dispositions", item))

	report, err := ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatalf("an unlistable directory aborted the report: %v", err)
	}
	want := map[string]bool{
		".abcd/work/issues/readings/" + other:    false,
		".abcd/work/issues/dispositions/" + item: false,
	}
	for _, u := range report.Unsafe {
		if _, ok := want[u.Path]; ok {
			want[u.Path] = strings.Contains(u.Reason, "permission denied")
		}
	}
	for p, ok := range want {
		if !ok {
			t.Errorf("%s not routed to Unsafe with its reason: %+v", p, report.Unsafe)
		}
	}
	if _, err := Lint(readingOutstandingConfig(severityInfo), root); err != nil {
		t.Errorf("an enabled rule failed the lint run: %v", err)
	}
}

// A contest whose standing records include one no reader can read names it as
// illegible: the prescribed hand repair (write supersedes_disposition into the
// surplus record) is inert on a malformed record, whose supersession is
// discarded, so the reader has to know which one that is.
func TestAContestMarksItsIllegibleStandingRecords(t *testing.T) {
	run, item := "rdg-2608300000000001", "rdi-2608300000000002"
	good, bad := "dsp-2608300000000003", "dsp-2608300000000004"
	root := readingLedger(t, run, item, "detection")
	dispositionRecord(t, root, item, good, "accepted")
	writeFile(t, root, ".abcd/work/issues/dispositions/"+item+"/"+bad+".md",
		"---\nschema_version: 1\nid: \""+bad+"\"\nid: \""+bad+"\"\nitem: \""+item+"\"\n"+
			"state: \"accepted\"\ndisposition_grounds: \"a\"\n---\n\n")

	report, err := ReadReadingOutstanding(root, ".abcd/work/issues")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Contested) != 1 {
		t.Fatalf("want one contest, got %+v", report)
	}
	if got := report.Contested[0].Illegible; len(got) != 1 || got[0] != bad {
		t.Fatalf("Illegible = %v, want [%s]", got, bad)
	}
	fs, err := Lint(readingOutstandingConfig(severityInfo), root)
	if err != nil {
		t.Fatal(err)
	}
	var marked bool
	for _, f := range fs {
		if f.RuleID == ruleReadingOutstanding && strings.Contains(f.Message, bad+" (not well-formed") {
			marked = true
		}
	}
	if !marked {
		t.Errorf("the contest message does not mark %s illegible: %+v", bad, fs)
	}
}
