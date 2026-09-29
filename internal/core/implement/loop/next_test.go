package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/gittest"
)

// pickIntent is planned intent itd-<n> linked to spc-<n>, every check passing
// unless extra frontmatter or body says otherwise, with the given criteria.
func pickIntent(n, extraFrontmatter, body, criteria string) string {
	return "---\nid: itd-" + n + "\nslug: i" + n + "\nspec_id: spc-" + n + "\nkind: standalone\n" + extraFrontmatter + "---\n# i" + n + "\n\n" +
		"## Mechanism\n\nWe expect it to work because it is small; shown wrong if it is not.\n\n" +
		"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n" + criteria + "\n" +
		body +
		"## Grounds\n\n- pursued: we expect the person's conjecture to stay the one the gate reports\n"
}

// pickSpec is the open spec spc-<n> of itd-<n>, with a `## Footprint` section
// when footprint is non-empty.
func pickSpec(n, footprint string) string {
	s := "---\nid: spc-" + n + "\nslug: i" + n + "\nintent: itd-" + n + "\n---\n# i" + n + "\n\n## Summary\n\nA written design record.\n"
	if footprint != "" {
		s += "\n## Footprint\n\n" + footprint
	}
	return s
}

const (
	gwt      = "- **Given** x, **when** y, **then** z.\n"
	fpSmall  = "- packages: internal/core/intent\n- tests: the score over fixtures\n"
	unsolved = "## Open Questions\n\n- Which runner?\n\n"
)

func pickRel(n string) (string, string) {
	return ".abcd/development/intents/planned/itd-" + n + "-i" + n + ".md", ".abcd/development/specs/open/spc-" + n + "-i" + n + ".md"
}

// pickRepo stands up a committed repository holding the given planned intents
// (number to intent and spec content), the local tier, a temporary HOME and a
// commit identity for the pick's record commit.
func pickRepo(t *testing.T, records map[string][2]string) *gittest.Repo {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "Pat Example")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "pat@example.com")
	}
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write("AGENTS.md", agentsMarked)
	for n, r := range records {
		ir, sr := pickRel(n)
		repo.Write(ir, r[0])
		repo.Write(sr, r[1])
	}
	repo.Commit("init")
	if err := os.MkdirAll(filepath.Join(repo.Root(), ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

// TestNextCandidatesAreWhatTheBuildWouldStart is criterion 1: only the planned
// intents that pass every check `abcd build <itd-N>` runs are candidates, and
// an empty candidate set is refused naming each excluded intent and the check
// that excluded it, writing nothing.
func TestNextCandidatesAreWhatTheBuildWouldStart(t *testing.T) {
	records := map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", "")},
		"11": {pickIntent("11", "", unsolved, gwt), pickSpec("11", "")},
		"12": {pickIntent("12", "held: \"awaiting the pacing ruling\"\n", settledQuestions, gwt), pickSpec("12", "")},
		"13": {pickIntent("13", "blocked_by: [itd-10]\n", settledQuestions, gwt), pickSpec("13", "")},
		"14": {strings.Replace(pickIntent("14", "", settledQuestions, gwt), "spec_id: spc-14", "spec_id: null", 1), pickSpec("14", "")},
		"15": {strings.Replace(pickIntent("15", "", settledQuestions, gwt), "We expect it to work because it is small; shown wrong if it is not.",
			intent.MechanismPrompt, 1), pickSpec("15", "")},
		"16": {pickIntent("16", "", settledQuestions, gwt), pickSpec("16", "")},
	}
	repo := pickRepo(t, records)
	// A peer holds itd-16: a local branch that shipped it.
	repo.Git("checkout", "-q", "-b", "lane-sixteen")
	ir, _ := pickRel("16")
	repo.Remove(ir)
	repo.Write(".abcd/development/intents/shipped/itd-16-i16.md", records["16"][0])
	repo.Commit("deliver sixteen")
	repo.Git("checkout", "-q", "main")

	set, err := candidates(repo.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Candidates) != 1 || set.Candidates[0].ID != "itd-10" {
		t.Fatalf("only the intent every check passes is a candidate: %+v", set.Candidates)
	}
	want := map[string]string{"itd-11": CheckOpenQuestions, "itd-12": CheckHold, "itd-13": CheckBlocked,
		"itd-14": CheckReady, "itd-15": CheckClaimSections, "itd-16": CheckPeers}
	if len(set.Excluded) != len(want) {
		t.Fatalf("every other planned intent is excluded: %+v", set.Excluded)
	}
	for _, e := range set.Excluded {
		if want[e.ID] != e.Check || e.Reason == "" {
			t.Fatalf("%s is excluded by %q with a reason, got %+v", e.ID, want[e.ID], e)
		}
	}

	// Without itd-10 nothing passes: the refusal names each and writes nothing.
	ir10, _ := pickRel("10")
	repo.Write(ir10, pickIntent("10", "held: \"not yet\"\n", settledQuestions, gwt))
	repo.Commit("hold ten")
	_, err = Next(repo.Root(), Options{}, NextOptions{})
	r := mustRefusal(t, err)
	if r.Step != StepPick || len(r.Excluded) != 7 {
		t.Fatalf("the empty set is refused at the pick naming every excluded intent: %+v", r)
	}
	for _, id := range []string{"itd-10 (hold", "itd-11 (open_questions", "itd-13 (blocked", "itd-16 (peers"} {
		if !strings.Contains(r.Reason, id) {
			t.Fatalf("the refusal names %q: %s", id, r.Reason)
		}
	}
	runTierAbsent(t, repo.Root())
}

// TestNextWritesTheReasonAsTheLaneFirstCommit is criteria 2 and 4: the pick
// starts the lane `abcd build <itd-N>` would start, the lane's worktree step
// commits exactly one run-marked `pursued:` entry onto the chosen intent as
// the branch's first commit, record-only, with no existing entry changed; the
// gate still reports the person's entry; and the receipt verifier does not
// count the pick's commit as the implementer's.
func TestNextWritesTheReasonAsTheLaneFirstCommit(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)},
		"11": {pickIntent("11", "", settledQuestions, gwt), pickSpec("11", "")},
	})
	ir, _ := pickRel("10")
	before, err := os.ReadFile(filepath.Join(repo.Root(), ir))
	if err != nil {
		t.Fatal(err)
	}

	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pick.Chosen.ID != "itd-10" || res.Pick.RunnerUp == nil || res.Pick.RunnerUp.ID != "itd-11" || res.Pick.TieBrokenByAge {
		t.Fatalf("the readiest is picked over the runner-up on score: %+v", res.Pick)
	}
	if !strings.HasPrefix(res.Entry, "picked by run "+res.Start.RunID+" on ") {
		t.Fatalf("the entry opens with the run's marker: %q", res.Entry)
	}
	for _, want := range []string{"itd-10 300", "itd-11 100", "rule: ", "runner-up: itd-11, which lost on score", "falsifier: "} {
		if !strings.Contains(res.Entry, want) {
			t.Fatalf("the entry names %q: %s", want, res.Entry)
		}
	}

	// The run is the one `abcd build itd-10` starts: same key, spec, lane and
	// first step; the pick rides beside it.
	st, err := ReadState(repo.Root(), res.Start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Key != "itd-10" || st.Intent != "itd-10" || st.Spec != "spc-10" || len(st.Lanes) != 1 ||
		st.Lanes[0].ID != "lane-1" || st.Lanes[0].Step != StepWorktree || st.Pick == nil || st.Pick.Lane != "lane-1" {
		t.Fatalf("the pick starts the build's own run: %+v", st)
	}

	if _, err := Advance(repo.Root(), res.Start.RunID, DefaultSteps(), Options{}); err != nil {
		t.Fatal(err)
	}
	st, err = ReadState(repo.Root(), res.Start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	l := st.Lanes[0]
	if !isFullHex(l.PickSHA) || l.HeadSHA != l.PickSHA {
		t.Fatalf("the worktree step records the pick commit as the lane's head: %+v", l)
	}
	past := strings.Fields(repo.Git("rev-list", l.BaseSHA+".."+l.Branch))
	if len(past) != 1 || past[0] != l.PickSHA {
		t.Fatalf("the pick's entry is the branch's first and only commit past its base: %v", past)
	}
	if files := strings.TrimSpace(repo.Git("diff-tree", "--no-commit-id", "--name-only", "-r", l.PickSHA)); files != ir {
		t.Fatalf("the commit is record-only, the intent alone: %q", files)
	}
	after := repo.Git("show", l.PickSHA+":"+ir)
	if !strings.HasPrefix(after+"\n", strings.TrimRight(string(before), "\n")) {
		t.Fatalf("no existing line of the record changes:\n%s", after)
	}
	added := strings.TrimLeft(strings.TrimPrefix(after+"\n", strings.TrimRight(string(before), "\n")), "\n")
	if strings.Count(added, "\n- ") != 0 || !strings.HasPrefix(added, "- pursued: picked by run "+st.RunID+" on ") {
		t.Fatalf("exactly one run-marked pursued entry is appended: %q", added)
	}
	if got, err := os.ReadFile(filepath.Join(repo.Root(), ir)); err != nil || string(got) != string(before) {
		t.Fatalf("the checkout the run started from is never written (%v)", err)
	}
	ready, err := intent.Ready(l.Worktree, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range ready.Checks {
		if c.Name == intent.CheckGrounds && !strings.Contains(c.Detail, "most recent pursued: we expect the person's conjecture") {
			t.Fatalf("the gate still reports the person's entry: %q", c.Detail)
		}
	}

	// The same lane from here: the brief, then the implementer's receipt, which
	// may not count the pick's commit.
	advanceTo(t, repo, st.RunID, StepImplement)
	if _, err := Advance(repo.Root(), st.RunID, DefaultSteps(), Options{}); err != nil {
		t.Fatal(err)
	}
	st, _ = ReadState(repo.Root(), st.RunID)
	l = st.Lanes[0]
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), st.RunID, "lane-1")
	work := laneCommit(t, repo, l, "work.txt")
	rel := writeReceipt(t, dir, goodReceipt(t, st.RunID, l, dir, l.PickSHA, work))
	_, err = Receipt(repo.Root(), st.RunID, rel, DefaultSteps(), Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "the pick's record-only commit, not the implementer's work") {
		t.Fatalf("the pick's commit is not the implementer's: %+v", r)
	}
	rel = writeReceipt(t, dir, goodReceipt(t, st.RunID, l, dir, work))
	if _, err := Receipt(repo.Root(), st.RunID, rel, DefaultSteps(), Options{}); err != nil {
		t.Fatalf("the implementer's own commit verifies: %v", err)
	}
}

// TestThePickCommitIsFoundNotRemade is the Handler contract for the pick: a
// worktree step run again after the pick commit adopts it, and one run again
// after the entry was written but not committed commits it once.
func TestThePickCommitIsFoundNotRemade(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)}})
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), res.Start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	c := Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st}
	first := st.Lanes[0]
	if _, err := worktreeStep(c, &first); err != nil {
		t.Fatal(err)
	}
	again := st.Lanes[0]
	if _, err := worktreeStep(c, &again); err != nil || again.PickSHA != first.PickSHA {
		t.Fatalf("the pick commit is found, not remade: %v, %s vs %s", err, again.PickSHA, first.PickSHA)
	}

	// Written but not committed: undo the commit, keep the entry.
	repo.Git("-C", first.Worktree, "reset", "-q", "--soft", first.BaseSHA)
	repo.Git("-C", first.Worktree, "restore", "--staged", ".")
	third := st.Lanes[0]
	if _, err := worktreeStep(c, &third); err != nil {
		t.Fatalf("an entry written and not committed is committed: %v", err)
	}
	if n := len(strings.Fields(repo.Git("rev-list", third.BaseSHA+".."+third.Branch))); n != 1 {
		t.Fatalf("one pick commit, got %d", n)
	}
	body := repo.Git("show", third.PickSHA+":"+strings.TrimSpace(repo.Git("diff-tree", "--no-commit-id", "--name-only", "-r", third.PickSHA)))
	if strings.Count(body, "picked by run") != 1 {
		t.Fatalf("the entry is written once:\n%s", body)
	}
}

// TestAPickCommitIsAdoptedByContentNotShape: a commit on the lane's branch that
// carries the pick's subject and ends a record with this run's entry is adopted
// only when it is the base's record of the picked intent plus exactly that one
// entry. One that also guts the intent's criteria, or lands the entry on a
// different planned intent, is refused and left where it is.
func TestAPickCommitIsAdoptedByContentNotShape(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)},
		"11": {pickIntent("11", "", settledQuestions, gwt), pickSpec("11", "")},
	})
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), res.Start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Intent != "itd-10" {
		t.Fatalf("the fixture's readiest is itd-10, got %s", st.Intent)
	}
	c := Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st}
	first := st.Lanes[0]
	if _, err := worktreeStep(c, &first); err != nil {
		t.Fatal(err)
	}
	ir10, _ := pickRel("10")
	ir11, _ := pickRel("11")
	wt := first.Worktree
	genuine, err := os.ReadFile(filepath.Join(wt, ir10))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(genuine), "\n"), "\n")
	entry := lines[len(lines)-1] + "\n"
	if !strings.HasPrefix(entry, "- pursued: picked by run "+st.RunID+" ") {
		t.Fatalf("the genuine record ends with the run's entry: %q", entry)
	}

	for name, craft := range map[string]func() (string, string){
		"guts the picked intent's criteria": func() (string, string) {
			return ir10, strings.Replace(string(genuine), gwt, "", 1)
		},
		"rewrites a different planned intent": func() (string, string) {
			return ir11, pickIntent("11", "", settledQuestions, gwt) + entry
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo.Git("-C", wt, "reset", "-q", "--hard", first.BaseSHA)
			rel, body := craft()
			if err := os.WriteFile(filepath.Join(wt, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			repo.Git("-C", wt, "commit", "-q", "-am", pickSubject(st))
			crafted := strings.TrimSpace(repo.Git("-C", wt, "rev-parse", "HEAD"))
			lane := st.Lanes[0]
			_, err := worktreeStep(c, &lane)
			r := mustRefusal(t, err)
			if !strings.Contains(r.Reason, "is not the pick's record commit") || lane.PickSHA == crafted {
				t.Fatalf("a crafted commit is refused, never adopted: %+v (pick %s)", r, lane.PickSHA)
			}
			if head := strings.TrimSpace(repo.Git("-C", wt, "rev-parse", "HEAD")); head != crafted {
				t.Fatalf("the refused commit is left where it is: %s vs %s", head, crafted)
			}
		})
	}

	// Written and not committed takes the same test: a record that ends with
	// the run's entry but guts the criteria is not committed.
	repo.Git("-C", wt, "reset", "-q", "--hard", first.BaseSHA)
	if err := os.WriteFile(filepath.Join(wt, filepath.FromSlash(ir10)), []byte(strings.Replace(string(genuine), gwt, "", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	lane := st.Lanes[0]
	_, err = worktreeStep(c, &lane)
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "not the base's record with this run's pick appended") {
		t.Fatalf("an uncommitted crafted record is refused: %+v", r)
	}
	if head := strings.TrimSpace(repo.Git("-C", wt, "rev-parse", "HEAD")); head != first.BaseSHA {
		t.Fatalf("nothing is committed over a crafted record: %s", head)
	}
}

// TestAReceiptRefusesABranchThatDroppedThePick is decision 6's verifier: the
// pick's reason reaches the default branch with the work only while the pick
// commit stays on the lane's branch, so a receipt over a branch whose first
// commit was amended away is refused, naming the remedy.
func TestAReceiptRefusesABranchThatDroppedThePick(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)}})
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	runID := res.Start.RunID
	advanceTo(t, repo, runID, StepImplement)
	if _, err := Advance(repo.Root(), runID, DefaultSteps(), Options{}); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	l := st.Lanes[0]
	if !isFullHex(l.PickSHA) {
		t.Fatalf("the lane records its pick commit: %+v", l)
	}
	repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "-m", "chore: reword the first commit")
	work := laneCommit(t, repo, l, "work.txt")
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), runID, "lane-1")
	rel := writeReceipt(t, dir, goodReceipt(t, runID, l, dir, work))
	_, err = Receipt(repo.Root(), runID, rel, DefaultSteps(), Options{})
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "no longer carries the pick's record-only commit "+shortSHA(l.PickSHA)) ||
		!strings.Contains(r.Remedy, l.PickSHA[:12]) {
		t.Fatalf("a branch that dropped the pick commit is refused, naming the remedy: %+v", r)
	}
}

// TestNextRefusesMoreThanOnePick: continuing under the pace rule (criterion 5)
// is not built, so asking for it is refused by name and writes nothing.
func TestNextRefusesMoreThanOnePick(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", "")}})
	for _, n := range []NextOptions{{Max: 2}, {UntilEmpty: true}} {
		_, err := Next(repo.Root(), Options{}, n)
		if r := mustRefusal(t, err); r.Step != StepPick || !strings.Contains(r.Reason, "criterion 5") {
			t.Fatalf("%+v is refused naming criterion 5: %+v", n, r)
		}
	}
	runTierAbsent(t, repo.Root())
}

// TestAPickedStateRoundTrips: the pick and the lane's pick commit survive the
// strict reader, and a version-2 file carrying a pick is refused.
func TestAPickedStateRoundTrips(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", "")}})
	res, err := Next(repo.Root(), Options{}, NextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(res.Start.RunID)))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw["schema_version"] = 2
	old, _ := json.Marshal(raw)
	if err := os.WriteFile(path, old, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(repo.Root(), res.Start.RunID); err == nil || !strings.Contains(err.Error(), "carries a pick") {
		t.Fatalf("a version-2 file carrying a pick is refused: %v", err)
	}
}

func isFullHex(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
