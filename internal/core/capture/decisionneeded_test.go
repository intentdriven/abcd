package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The decision-record marker on the issue (itd-82 decision 10;
// spc-2609212015054359 step 6): a drain's trust-rule hand-back writes
// `decision_needed:` holding its question onto the open issue and changes
// nothing else in the record, and the eligibility rule hands a marked record
// back with the marker as its reason.

// recordOf is the one file a fixture's record lives in.
func (f drainFixture) recordOf(id string) string {
	f.t.Helper()
	m, err := filepath.Glob(filepath.Join(f.ir, "*", id+"-*.md"))
	if err != nil || len(m) != 1 {
		f.t.Fatalf("one record of %s, got %v (%v)", id, m, err)
	}
	return m[0]
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestTheMarkerIsTheOnlyChangeToTheRecord: the marker is one line added to
// the frontmatter, every other byte of the record kept; the reader carries it
// on the Issue; marking again with the same question writes nothing, and a
// new question replaces the old one, naming it.
func TestTheMarkerIsTheOnlyChangeToTheRecord(t *testing.T) {
	f := newDrainFixture(t)
	f.file("iss-1", SeverityMinor, "bug", "guard the owner check in the hook")
	path := f.recordOf("iss-1")
	before := readFile(t, path)

	const q = "may a guard ever skip the owner check?\nOnly for the owner's own session?"
	res, err := MarkDecisionNeeded(DecisionNeededRequest{RepoRoot: f.repo, IssuesRoot: f.ir, ID: "iss-1", Question: q})
	if err != nil {
		t.Fatalf("MarkDecisionNeeded: %v", err)
	}
	const folded = "may a guard ever skip the owner check? Only for the owner's own session?"
	if res.Question != folded || res.Unchanged || res.Previous != "" || res.Status != StateOpen {
		t.Fatalf("the result names the folded question written: %+v", res)
	}
	after := readFile(t, path)
	marker := ""
	for _, ln := range strings.SplitAfter(after, "\n") {
		if strings.HasPrefix(ln, "decision_needed: ") {
			marker = ln
		}
	}
	if marker == "" || strings.Replace(after, marker, "", 1) != before {
		t.Fatalf("the record gains the marker line and nothing else:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if fm, _, err := parseFrontmatterAndBody(after); err != nil || fm["decision_needed"] != folded {
		t.Fatalf("the marker holds the question: %v %v", fm["decision_needed"], err)
	}
	lr, err := List(ListRequest{RepoRoot: f.repo, IssuesRoot: f.ir, State: StateOpen})
	if err != nil || len(lr.Issues) != 1 || len(lr.Skipped) != 0 || lr.Issues[0].DecisionNeeded != folded {
		t.Fatalf("the reader carries the marker: %+v (skipped %v) %v", lr.Issues, lr.Skipped, err)
	}
	if held, err := DecisionNeededHolds(f.repo, "iss-1", q); err != nil || !held {
		t.Fatalf("the record holds the question as written: %v %v", held, err)
	}

	again, err := MarkDecisionNeeded(DecisionNeededRequest{RepoRoot: f.repo, IssuesRoot: f.ir, ID: "iss-1", Question: q})
	if err != nil || !again.Unchanged || readFile(t, path) != after {
		t.Fatalf("the same question again writes nothing: %+v %v", again, err)
	}
	other, err := MarkDecisionNeeded(DecisionNeededRequest{RepoRoot: f.repo, IssuesRoot: f.ir, ID: "iss-1", Question: "who may read the token?"})
	if err != nil || other.Previous != folded || other.Unchanged {
		t.Fatalf("a new question replaces the old one, naming it: %+v %v", other, err)
	}
	if got := readFile(t, path); strings.Count(got, "decision_needed: ") != 1 || !strings.Contains(got, "who may read the token?") {
		t.Fatalf("one marker, the new question:\n%s", got)
	}
}

// TestTheMarkerIsRefusedWithNothingWritten: an empty question, a malformed
// id, an id the ledger does not hold and a record that is not open are each
// refused, and the record is left as it was.
func TestTheMarkerIsRefusedWithNothingWritten(t *testing.T) {
	f := newDrainFixture(t)
	f.file("iss-1", SeverityMinor, "bug", "guard the owner check in the hook")
	path := f.recordOf("iss-1")
	before := readFile(t, path)
	for _, req := range []DecisionNeededRequest{
		{ID: "iss-1", Question: " \n\t "},
		{ID: "iss-1/../x", Question: "a question"},
		{ID: "iss-9", Question: "a question"},
	} {
		req.RepoRoot, req.IssuesRoot = f.repo, f.ir
		if _, err := MarkDecisionNeeded(req); err == nil {
			t.Errorf("%+v is not refused", req)
		}
	}
	if readFile(t, path) != before {
		t.Fatal("a refused marker changed the record")
	}
	if _, err := Resolve(ResolveRequest{RepoRoot: f.repo, IssuesRoot: f.ir, ID: "iss-1", Resolution: "fixed", Impact: "fix"}); err != nil {
		t.Fatalf("resolving the fixture: %v", err)
	}
	_, err := MarkDecisionNeeded(DecisionNeededRequest{RepoRoot: f.repo, IssuesRoot: f.ir, ID: "iss-1", Question: "a question"})
	if !errors.Is(err, ErrTransitionConflict) {
		t.Fatalf("a record that is not open is refused as a transition conflict: %v", err)
	}
}

// TestAMarkedIssueIsHandedBackWithTheMarkerAsItsReason: an issue every other
// field rule would take is handed back by the rule decision-needed, its
// reason stating the question and what answers it; the marker comes after the
// severity rule, whose answer it would not change, and before a remedy
// waiting on a ruling.
func TestAMarkedIssueIsHandedBackWithTheMarkerAsItsReason(t *testing.T) {
	f := newDrainFixture(t)
	f.file("iss-1", SeverityMinor, "bug", "guard the owner check in the hook")
	f.file("iss-2", SeverityMajor, "bug", "guard the owner check in the hook")
	f.file("iss-3", SeverityMinor, "bug", "Waits on ruling H4: then guard it")
	f.file("iss-4", SeverityMinor, "bug", "guard the owner check in the hook")
	for _, id := range []string{"iss-1", "iss-2", "iss-3"} {
		if _, err := MarkDecisionNeeded(DecisionNeededRequest{RepoRoot: f.repo, IssuesRoot: f.ir, ID: id, Question: "may a guard skip the owner check?"}); err != nil {
			t.Fatalf("marking %s: %v", id, err)
		}
	}
	p := f.plan()
	for _, id := range []string{"iss-1", "iss-3"} {
		v := verdictOf(t, p, id)
		if v.Outcome != DrainHandBack || v.Rule != RuleDecisionNeeded ||
			!strings.Contains(v.Reason, "the question: may a guard skip the owner check?") || !strings.Contains(v.Reason, "decision_needed:") {
			t.Errorf("%s is handed back by its marker, naming the question: %+v", id, v)
		}
	}
	if v := verdictOf(t, p, "iss-2"); v.Rule != RuleSeverity {
		t.Errorf("a marked major issue is handed back by its severity first: %+v", v)
	}
	if v := verdictOf(t, p, "iss-4"); v.Outcome != DrainEligible {
		t.Errorf("an unmarked issue stays eligible: %+v", v)
	}
}
