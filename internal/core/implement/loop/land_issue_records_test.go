package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/gitutil"
)

const (
	anotherIssueRel = ".abcd/work/issues/open/iss-77-another.md"
	thirdIssueRel   = ".abcd/work/issues/open/iss-78-third.md"
)

// editIntentProse adds a sentence to the lane's own intent record in its worktree: an
// edit inside a store that changes nothing a close decides by.
func editIntentProse(t *testing.T, l Lane) {
	t.Helper()
	landWrite(t, l.Worktree, plannedIntentRel, landRead(t, l.Worktree, plannedIntentRel)+"\nA sentence added by hand.\n")
}

// issueRecordChanges are the changes to issue records the landing did not write: a
// record added, another moved to resolved/, another removed, and another's text edited.
// Each names the path a refusal names.
func issueRecordChanges() map[string]struct {
	change func(t *testing.T, l Lane)
	names  string
} {
	return map[string]struct {
		change func(t *testing.T, l Lane)
		names  string
	}{
		"a record added": {func(t *testing.T, l Lane) {
			t.Helper()
			landWrite(t, l.Worktree, thirdIssueRel, strings.NewReplacer(`"iss-77"`, `"iss-78"`, `"another"`, `"third"`).Replace(landRead(t, l.Worktree, anotherIssueRel)))
		}, "iss-78-third.md"},
		"another record resolved": {func(t *testing.T, l Lane) {
			t.Helper()
			if _, err := capture.Resolve(capture.ResolveRequest{RepoRoot: l.Worktree, ID: "iss-77", Resolution: "settled by hand", Impact: "fix",
				ByCommit: l.HeadSHA}); err != nil {
				t.Fatalf("resolving another issue: %v", err)
			}
		}, "iss-77-another.md"},
		"another record removed": {func(t *testing.T, l Lane) {
			t.Helper()
			if err := os.Remove(filepath.Join(l.Worktree, filepath.FromSlash(anotherIssueRel))); err != nil {
				t.Fatal(err)
			}
		}, "iss-77-another.md"},
		"another record edited": {func(t *testing.T, l Lane) {
			t.Helper()
			landWrite(t, l.Worktree, anotherIssueRel, landRead(t, l.Worktree, anotherIssueRel)+"\nA line added by hand.\n")
		}, "iss-77-another.md"},
	}
}

// preparedWithAnotherIssue is a closing lane whose judged head holds a second open issue
// that no receipt of the lane resolves and the intent does not name, with the landing's
// first step taken.
func preparedWithAnotherIssue(t *testing.T) (*landFixture, Lane) {
	t.Helper()
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	f.commitAtTheJudgedHead(t, l, func() { f.fileAnotherIssue(t, l) })
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if l.Landing == nil || !l.Landing.Closes || len(l.Resolves) != 1 || l.Resolves[0].Issue != f.issue {
		t.Fatalf("premise: a closing lane resolving its own issue: %+v %+v", l.Landing, l.Resolves)
	}
	return f, l
}

// wantsIssueRecordRefused takes the records step, finds it refuse naming the path, and
// finds no records commit taken.
func (f *landFixture) wantsIssueRecordRefused(t *testing.T, judged, names string) *Refusal {
	t.Helper()
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, names) {
		t.Fatalf("the refusal names %s: %+v", names, r)
	}
	if got := currentLane(t, f.repo, f.runID); got.Landing.RecordsDone || got.HeadSHA != judged {
		t.Fatalf("no records commit is taken: %+v", got.Landing)
	}
	return r
}

// The landing commits an issue record it does not resolve only as its own moves leave
// it: one added, moved, removed or edited in the lane's worktree before the records step
// stops that step, and the lane's own resolution lands once the record is put back.
func TestTheRecordsStepCommitsNoIssueRecordTheLandingDidNotWrite(t *testing.T) {
	for name, c := range issueRecordChanges() {
		t.Run(name, func(t *testing.T) {
			f, l := preparedWithAnotherIssue(t)
			judged := l.HeadSHA
			c.change(t, l)
			f.wantsIssueRecordRefused(t, judged, c.names)

			f.repo.Git("-C", l.Worktree, "checkout", "-q", judged, "--", ".abcd/work/issues")
			f.repo.Git("-C", l.Worktree, "clean", "-q", "-f", "--", ".abcd/work/issues")
			cur := f.wantsRecordsCommit(t)
			files := f.repo.Git("-C", cur.Worktree, "ls-tree", "-r", "--name-only", cur.HeadSHA, "--", ".abcd/work/issues")
			if !strings.Contains(files, "resolved/"+f.issue+"-") || !strings.Contains(files, anotherIssueRel) || strings.Contains(files, "iss-78") {
				t.Fatalf("the records commit resolves the lane's own issue and leaves the other as the head has it:\n%s", files)
			}
		})
	}
}

// What is written after the landing's own check and before it stages is judged as staged,
// so the same changes made in that window stop the commit.
func TestAnIssueRecordChangedBeforeTheLandingStagesStopsItsCommit(t *testing.T) {
	for name, c := range issueRecordChanges() {
		t.Run(name, func(t *testing.T) {
			f, l := preparedWithAnotherIssue(t)
			judged := l.HeadSHA
			beforeLandingStage = func(string) { c.change(t, l) }
			t.Cleanup(func() { beforeLandingStage = nil })
			f.wantsIssueRecordRefused(t, judged, c.names)
		})
	}
}

// A commit found on the judged head that holds the landing's own records and one of the
// same changes besides is not the landing's records.
func TestACommitFoundHoldingAnIssueRecordTheLandingDidNotWriteIsRefused(t *testing.T) {
	for name, c := range issueRecordChanges() {
		t.Run(name, func(t *testing.T) {
			f, l := preparedWithAnotherIssue(t)
			judged := l.HeadSHA
			made := f.wantsRecordsCommit(t).HeadSHA
			msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", made)
			f.rollBackToJudged(t, judged)
			f.repo.Git("-C", l.Worktree, "reset", "-q", "--soft", judged)
			c.change(t, currentLane(t, f.repo, f.runID))
			f.repo.Git("-C", l.Worktree, "add", "-A", "--", ".abcd/work/issues")
			if out, err := gitutil.LocalGit(l.Worktree, "commit", "-q", "-m", msg); err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			r := f.wantsIssueRecordRefused(t, judged, c.names)
			if !strings.Contains(r.Reason, "holds") || !strings.Contains(r.Remedy, "restore the branch") {
				t.Fatalf("the refusal says what the commit holds and the way back: %+v", r)
			}
		})
	}
}

// An issue an intent the close ships names in related_issues lands only resolved: its
// record taken out of open/ and put nowhere is no resolution of it.
func TestAnIssueTheShippedIntentNamesIsCommittedOnlyAsResolved(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	f.fileAnotherIssue(t, l)
	f.laneNamesIssue(t, l, "iss-77")
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if err := os.Remove(filepath.Join(l.Worktree, filepath.FromSlash(anotherIssueRel))); err != nil {
		t.Fatal(err)
	}
	f.wantsIssueRecordRefused(t, l.HeadSHA, "iss-77-another.md")
}

// An issue an intent the close ships names lands resolved only as a resolve writes it:
// the record the lane's worktree moved from open/ to resolved/ is the open record with
// the fields a resolve sets and, with grounds, its one entry appended. One whose text or
// whose other fields were changed besides is refused, and nothing is committed.
func TestAResolutionOfANamedIssueLandsOnlyAsAResolveWritesIt(t *testing.T) {
	const resolvedRel = ".abcd/work/issues/resolved/iss-77-another.md"
	for name, c := range map[string]struct {
		grounds string
		edit    func(string) string
	}{
		"as the resolve wrote it":                   {"", nil},
		"as the resolve wrote it, with its grounds": {"pursued: the landing settles the link the intent names; shown wrong if the issue recurs", nil},
		"its text rewritten": {"", func(s string) string {
			return strings.Replace(s, "Another observation.", "A different observation.", 1)
		}},
		"its text rewritten above its grounds": {"pursued: the landing settles the link the intent names; shown wrong if the issue recurs", func(s string) string {
			return strings.Replace(s, "Another observation.", "A different observation.", 1)
		}},
		"a field a resolve does not set changed": {"", func(s string) string {
			return strings.Replace(s, `severity: "minor"`, `severity: "major"`, 1)
		}},
		"a field a resolve does not set added": {"", func(s string) string {
			return strings.Replace(s, "\nslug:", "\nrelated_specs: [spc-10]\nslug:", 1)
		}},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.validatedOnce(t)
			f.fileAnotherIssue(t, l)
			f.laneNamesIssue(t, l, "iss-77")
			f.step(t)
			l = currentLane(t, f.repo, f.runID)
			if _, err := capture.Resolve(capture.ResolveRequest{RepoRoot: l.Worktree, ID: "iss-77", Resolution: "settled for this landing", Impact: "fix",
				ByCommit: l.HeadSHA, Grounds: c.grounds}); err != nil {
				t.Fatalf("resolving the named issue: %v", err)
			}
			if c.edit == nil {
				f.wantsRecordsCommit(t)
				return
			}
			was := landRead(t, l.Worktree, resolvedRel)
			if changed := c.edit(was); changed == was {
				t.Fatalf("premise: the edit changes the record:\n%s", was)
			} else {
				landWrite(t, l.Worktree, resolvedRel, changed)
			}
			f.wantsIssueRecordRefused(t, l.HeadSHA, "iss-77-another.md")
		})
	}
}
