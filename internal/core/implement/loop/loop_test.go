package loop

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/gittest"
)

const (
	plannedRel = ".abcd/development/intents/planned/itd-10-alpha.md"
	specRel    = ".abcd/development/specs/open/spc-1-alpha.md"
)

// readyIntent is a planned intent every check passes: criteria, conditions
// declined, no open question, no hold, linked to spc-1.
func readyIntent(extraFrontmatter, body string) string {
	return "---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n" + extraFrontmatter + "---\n# alpha\n\n" +
		"## Mechanism\n\nWe expect it to work because it is small; shown wrong if it is not.\n\n" +
		"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- Given x, when y, then z.\n\n" +
		body +
		"## Grounds\n\n- pursued: we expect the loop to run the lane end to end; shown wrong if a step needs a human\n"
}

const settledQuestions = "## Open Questions\n\n_None open._\n\n"

// specWithSteps is the open spec, listing steps when steps is non-empty.
func specWithSteps(steps string) string {
	s := "---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n\n## Summary\n\nA written design record.\n"
	if steps != "" {
		s += "\n## Steps\n\n" + steps
	}
	return s
}

// loopRepo stands up a committed repository with a READY intent, the local
// tier the run lives in, and a temporary HOME (the peer claims live there).
func loopRepo(t *testing.T, intentBody, spec string) *gittest.Repo {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write(plannedRel, intentBody)
	repo.Write(specRel, spec)
	repo.Commit("init")
	if err := os.MkdirAll(filepath.Join(repo.Root(), ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repo
}

func runTierAbsent(t *testing.T, root string) {
	t.Helper()
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(RunRelDir))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused start must write no state, but %s exists (%v)", RunRelDir, err)
	}
}

func mustRefusal(t *testing.T, err error) *Refusal {
	t.Helper()
	r, ok := AsRefusal(err)
	if !ok {
		t.Fatalf("want a refusal, got %v", err)
	}
	if r.Stage == "" || r.Reason == "" || r.Remedy == "" {
		t.Fatalf("a refusal names the step, the reason and the remedy: %+v", r)
	}
	return r
}

// TestStartRefusesEachFailedCheckAndWritesNoState is criterion 1: an intent not
// in planned/, or READY with an open question, an unanswered claim section or a
// hold, is refused naming the check that failed, and no state is written.
func TestStartRefusesEachFailedCheckAndWritesNoState(t *testing.T) {
	cases := []struct {
		name      string
		key       string
		intentRel string
		intent    string
		spec      string
		check     string
		reason    string
	}{
		{"an issue key without the drain rule", "iss-2609010000001234", plannedRel, readyIntent("", settledQuestions), specWithSteps(""), CheckEligible, "drain eligibility record"},
		{"a key shaped like no issue", "iss-12/x", plannedRel, readyIntent("", settledQuestions), specWithSteps(""), CheckKey, "iss-12/x"},
		{"not an id", "itd-x", plannedRel, readyIntent("", settledQuestions), specWithSteps(""), CheckKey, "itd-x"},
		{"unknown intent", "itd-99", plannedRel, readyIntent("", settledQuestions), specWithSteps(""), CheckReady, "itd-99"},
		{"a draft", "itd-10", ".abcd/development/intents/drafts/itd-10-alpha.md", readyIntent("", settledQuestions), specWithSteps(""), CheckReady, "draft"},
		{"an open question", "itd-10", plannedRel, readyIntent("", "## Open Questions\n\n- Which runner?\n\n"), specWithSteps(""), CheckOpenQuestions, "Which runner?"},
		{"an unanswered mechanism prompt", "itd-10", plannedRel,
			strings.Replace(readyIntent("", settledQuestions), "We expect it to work because it is small; shown wrong if it is not.",
				intent.MechanismPrompt, 1),
			specWithSteps(""), CheckClaimSections, "Mechanism"},
		{"an unrecorded scope condition", "itd-10", plannedRel,
			strings.Replace(readyIntent("", settledQuestions), "## Scope Conditions\n\nNone stated.\n\n", "", 1),
			specWithSteps(""), CheckClaimSections, "Scope Conditions"},
		{"a hold", "itd-10", plannedRel, readyIntent("held: \"awaiting the pacing ruling\"\n", settledQuestions), specWithSteps(""), CheckHold, "awaiting the pacing ruling"},
		{"an unshipped blocker", "itd-10", plannedRel, readyIntent("blocked_by: [itd-99]\n", settledQuestions), specWithSteps(""), CheckBlocked, "itd-99"},
		{"a superseded blocker whose replacement is unshipped", "itd-10", plannedRel, readyIntent("blocked_by: [itd-27]\n", settledQuestions), specWithSteps(""), CheckBlocked, "itd-27 → itd-94"},
		{"every step landed", "itd-10", plannedRel, readyIntent("", settledQuestions),
			specWithSteps("1. The parser\n   - landed: #1\n"), CheckSteps, "landed"},
		{"an unreadable steps section", "itd-10", plannedRel, readyIntent("", settledQuestions),
			specWithSteps("the parser, then the loop\n"), CheckSteps, "Steps"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			if tc.intentRel != plannedRel {
				repo.Remove(plannedRel)
			}
			repo.Write(tc.intentRel, tc.intent)
			repo.Write(specRel, tc.spec)
			repo.Write(supersededBlockerRel, supersededBlocker)
			repo.Write(".abcd/development/intents/planned/itd-94-replacement.md", replacementIntent)
			repo.Commit("fixture")

			_, err := Start(repo.Root(), tc.key, Options{})
			r := mustRefusal(t, err)
			if r.Stage != "check" || r.Check != tc.check {
				t.Fatalf("want the %s check named, got step %q check %q: %v", tc.check, r.Stage, r.Check, r)
			}
			if !strings.Contains(r.Reason, tc.reason) {
				t.Fatalf("the reason must say why (%q): %q", tc.reason, r.Reason)
			}
			if r.Contention {
				t.Fatalf("a failed check is the record's, not a peer's: %+v", r)
			}
			runTierAbsent(t, repo.Root())
		})
	}
}

// TestStartRefusesAPeerHoldingTheRecord is criterion 2, from both sources a
// peer holds a record through: a branch holding the intent in another bucket
// (the peer listing), and a live claim another session took on it (the run's
// claim store). Each names the peer, is contention, and writes no state.
func TestStartRefusesAPeerHoldingTheRecord(t *testing.T) {
	t.Run("a branch", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		repo.Git("checkout", "-q", "-b", "lane-alpha")
		repo.Remove(plannedRel)
		repo.Write(".abcd/development/intents/shipped/itd-10-alpha.md", readyIntent("", settledQuestions))
		repo.Commit("deliver alpha")
		repo.Git("checkout", "-q", "main")

		_, err := Start(repo.Root(), "itd-10", Options{})
		r := mustRefusal(t, err)
		if r.Check != CheckPeers || !r.Contention {
			t.Fatalf("want the peers check as contention: %+v", r)
		}
		if !strings.Contains(r.Reason, "lane-alpha") || !strings.Contains(r.Reason, "shipped") {
			t.Fatalf("the refusal names the peer and where it holds the record: %q", r.Reason)
		}
		runTierAbsent(t, repo.Root())
	})
	t.Run("a claim", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		sha := repo.Git("rev-list", "--max-parents=0", "HEAD")
		run, err := implement.Open(strings.TrimSpace(sha))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := run.Join("peer-session", implement.RoleFirst, "", "", 0); err != nil {
			t.Fatal(err)
		}
		if _, err := run.Claim(implement.ClaimRequest{Session: "peer-session", Record: "itd-10", Lane: "alpha"}); err != nil {
			t.Fatal(err)
		}

		_, err = Start(repo.Root(), "itd-10", Options{})
		r := mustRefusal(t, err)
		if r.Check != CheckPeers || !r.Contention || !strings.Contains(r.Reason, "peer-session") {
			t.Fatalf("want the peers check naming the claiming session: %+v", r)
		}
		runTierAbsent(t, repo.Root())
	})
}

// TestThePeerRefusalNamesAWorktreeOutsideHomeByItsDirectoryName: the refusal
// named a peer worktree through the home redaction alone, so one outside HOME
// reached the refusal as an absolute local path, in its name and inside its
// not-read reason (iss-2609281329007423). Both name it by its directory name.
func TestThePeerRefusalNamesAWorktreeOutsideHomeByItsDirectoryName(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0 on the shut worktree's planned folder does not deny root, so its could-not-be-read holder never forms")
	}
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	outside := t.TempDir()
	held := filepath.Join(outside, "wt-held")
	repo.Git("worktree", "add", "-q", "-b", "lane-alpha", held)
	shipped := ".abcd/development/intents/shipped/itd-10-alpha.md"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(held, shipped)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(held, plannedRel), filepath.Join(held, shipped)); err != nil {
		t.Fatal(err)
	}
	shut := filepath.Join(outside, "wt-shut")
	repo.Git("worktree", "add", "-q", "-b", "lane-shut", shut)
	if err := os.WriteFile(filepath.Join(shut, "note"), []byte("ahead\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo.Git("-C", shut, "add", "note")
	repo.Git("-C", shut, "commit", "-q", "-m", "ahead of main")
	locked := filepath.Join(shut, ".abcd", "development", "intents", "planned")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	_, err := Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	if r.Check != CheckPeers || !r.Contention {
		t.Fatalf("want the peers check as contention: %+v", r)
	}
	for _, want := range []string{"the worktree at wt-held (branch lane-alpha)", "the worktree at wt-shut (branch lane-shut) could not be read", "wt-shut/"} {
		if !strings.Contains(r.Reason, want) {
			t.Errorf("the refusal lacks %q: %q", want, r.Reason)
		}
	}
	abs := []string{outside}
	if real, err := filepath.EvalSymlinks(outside); err == nil && real != outside {
		abs = append(abs, real)
	}
	for _, a := range abs {
		if strings.Contains(r.Reason, a) {
			t.Errorf("the refusal prints the absolute worktree path under %s: %q", a, r.Reason)
		}
	}
	runTierAbsent(t, repo.Root())
}

// TestStartAgainResumesTheRunItsOwnLaneChanged is criterion 7's resume once
// the run has changed the tree it was judged on: its lane's worktree (in the
// machine-scoped store, piece 6's shape) delivers the intent to shipped/, or
// its lane holds a claim on it. The checks judged the record at the start; a
// live run for the key is found first and resumed, not re-judged into a
// refusal that names the run's own lane as a peer.
func TestStartAgainResumesTheRunItsOwnLaneChanged(t *testing.T) {
	t.Run("its lane worktree ships the intent", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		first, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		wt := filepath.Join(os.Getenv("HOME"), ".abcd", "worktrees", "0123abcd", "lane-1")
		if err := os.MkdirAll(filepath.Dir(wt), 0o755); err != nil {
			t.Fatal(err)
		}
		repo.Git("worktree", "add", "-q", "-b", "build/lane-1", wt)
		if err := os.MkdirAll(filepath.Join(wt, ".abcd", "development", "intents", "shipped"), 0o755); err != nil {
			t.Fatal(err)
		}
		repo.Git("-C", wt, "mv", plannedRel, ".abcd/development/intents/shipped/itd-10-alpha.md")
		repo.Git("-C", wt, "commit", "-q", "-m", "deliver alpha")

		again, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatalf("starting again while the run is in progress resumes it, whatever its lane did: %v", err)
		}
		if !again.Resumed || again.RunID != first.RunID {
			t.Fatalf("want run %s resumed, got %+v", first.RunID, again)
		}
	})
	t.Run("its lane claims the intent", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		first, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		sha := repo.Git("rev-list", "--max-parents=0", "HEAD")
		run, err := implement.Open(strings.TrimSpace(sha))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := run.Join("lane-session", implement.RoleFirst, "", "", 0); err != nil {
			t.Fatal(err)
		}
		if _, err := run.Claim(implement.ClaimRequest{Session: "lane-session", Record: "itd-10", Lane: "lane-1"}); err != nil {
			t.Fatal(err)
		}
		again, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatalf("the run's own claim must not refuse its resume: %v", err)
		}
		if !again.Resumed || again.RunID != first.RunID {
			t.Fatalf("want run %s resumed, got %+v", first.RunID, again)
		}
	})
	t.Run("a key that is not an intent is refused before any lookup", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
			t.Fatal(err)
		}
		_, err := Start(repo.Root(), "../itd-10", Options{})
		if r := mustRefusal(t, err); r.Check != CheckKey {
			t.Fatalf("want the key check named: %+v", r)
		}
	})
}

// TestStartRefusesAPeerItCannotRead: a peer the listing names and cannot read
// holds what nobody can say, so the peers check fails closed on it, as it does
// on an unreadable claim, naming the peer and why; a peer of the shape that
// holds nothing at the committed layout is not a holding.
func TestStartRefusesAPeerItCannotRead(t *testing.T) {
	t.Run("a branch whose ledger holds one id twice", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		repo.Git("checkout", "-q", "-b", "lane-beta")
		beta := "---\nid: itd-20\nslug: beta\n---\n# beta\n"
		repo.Write(".abcd/development/intents/drafts/itd-20-beta.md", beta)
		repo.Write(".abcd/development/intents/planned/itd-20-beta.md", beta)
		repo.Commit("split beta")
		repo.Git("checkout", "-q", "main")

		_, err := Start(repo.Root(), "itd-10", Options{})
		r := mustRefusal(t, err)
		if r.Check != CheckPeers || !r.Contention {
			t.Fatalf("want the peers check as contention: %+v", r)
		}
		if !strings.Contains(r.Reason, "lane-beta") || !strings.Contains(r.Reason, "could not be read") {
			t.Fatalf("the refusal names the unread peer: %q", r.Reason)
		}
		runTierAbsent(t, repo.Root())
	})
	t.Run("a branch from before the record layout", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		repo.Git("checkout", "-q", "--orphan", "old-layout")
		repo.Git("rm", "-rq", "--cached", ".")
		repo.Write("README.md", "an old tree\n")
		repo.Git("add", "README.md")
		repo.Git("commit", "-q", "-m", "old")
		repo.Git("checkout", "-q", "-f", "main")

		if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
			t.Fatalf("a peer holding no records at the layout holds nothing: %v", err)
		}
	})
}

// TestStartRefusesWithoutTheLocalTier: the tier is never created, so a
// repository abcd does not manage has no run.
func TestStartRefusesWithoutTheLocalTier(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	if err := os.RemoveAll(filepath.Join(repo.Root(), ".abcd", ".work.local")); err != nil {
		t.Fatal(err)
	}
	_, err := Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	if r.Stage != "state" || !strings.Contains(r.Reason, TierRelDir) {
		t.Fatalf("want the missing tier named: %+v", r)
	}
	if _, err := os.Lstat(filepath.Join(repo.Root(), ".abcd", ".work.local")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the tier must never be created: %v", err)
	}
}

// TestStartCreatesOneLaneAndAStartAgainResumesIt is criterion 3's state half
// and criterion 7's entry: the state file exists with one lane for the first
// unlanded spec step, the rest pending, and a second start resumes the same run
// rather than opening another.
func TestStartCreatesOneLaneAndAStartAgainResumesIt(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions),
		specWithSteps("1. The parser\n   - landed: #1\n2. The loop\n   - packages: internal/core/implement/loop\n3. The verb\n"))
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Resumed || !ValidRunID(res.RunID) || res.State != StateRelPath(res.RunID) {
		t.Fatalf("StartResult = %+v", res)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Key != "itd-10" || st.Intent != "itd-10" || st.Spec != "spc-1" || st.Driver != DriverHost {
		t.Fatalf("state = %+v", st)
	}
	if len(st.Lanes) != 1 {
		t.Fatalf("a run starts with one lane, got %d", len(st.Lanes))
	}
	l := st.Lanes[0]
	if l.ID != "lane-1" || l.Key != "itd-10" || l.SpecStep != 2 || l.StepTitle != "The loop" || l.Stage != StageWorktree {
		t.Fatalf("lane = %+v", l)
	}
	if len(st.Pending) != 1 || st.Pending[0].Number != 3 {
		t.Fatalf("the unlanded steps after the first wait as pending: %+v", st.Pending)
	}
	if len(st.Record) != 2 || st.Record[0].Stage != "start" || st.Record[1].Stage != StagePace {
		t.Fatalf("the record opens with the start, then names the pace: %+v", st.Record)
	}
	fi, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(res.State)))
	if err != nil || fi.Mode().Perm() != filePerm {
		t.Fatalf("the state file is the caller's own: %v %v", fi, err)
	}

	again, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !again.Resumed || again.RunID != res.RunID {
		t.Fatalf("a second start resumes the run: %+v", again)
	}
	runs, err := Runs(repo.Root())
	if err != nil || len(runs) != 1 {
		t.Fatalf("one run, not two: %d %v", len(runs), err)
	}
}

// The blocked check's supersession fixture: itd-27 is superseded by itd-94.
const (
	supersededBlockerRel = ".abcd/development/intents/superseded/itd-27-replaced.md"
	supersededBlocker    = "---\nid: itd-27\nslug: replaced\nkind: standalone\nsuperseded_by: itd-94\nkind_at_supersession: standalone\n---\n# replaced\n"
	replacementIntent    = "---\nid: itd-94\nslug: replacement\nkind: standalone\n---\n# replacement\n"
)

// TestStartFollowsASupersededBlockerToItsShippedReplacement is ruling BZ2 of
// 2026-09-29 at the build: an intent whose blocker was superseded waits on the
// intent that replaced it, so once that replacement has shipped the blocked
// check passes and the run starts.
func TestStartFollowsASupersededBlockerToItsShippedReplacement(t *testing.T) {
	repo := loopRepo(t, readyIntent("blocked_by: [itd-27]\n", settledQuestions), specWithSteps(""))
	repo.Write(supersededBlockerRel, supersededBlocker)
	repo.Write(".abcd/development/intents/shipped/itd-94-replacement.md", replacementIntent)
	repo.Commit("the blocker's replacement shipped")

	if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
		t.Fatalf("a blocker whose replacement shipped no longer blocks: %v", err)
	}
}

// TestAnUnsteppedSpecIsOneLane: a spec listing no steps is built as one step.
func TestAnUnsteppedSpecIsOneLane(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	st, _ := ReadState(repo.Root(), res.RunID)
	if len(st.Lanes) != 1 || st.Lanes[0].SpecStep != 1 || st.Lanes[0].StepTitle != "the whole spec" || len(st.Pending) != 0 {
		t.Fatalf("state = %+v", st)
	}
}

// fakeSteps is a lane sequence whose bodies a test controls: each counts its
// calls; the implement stage hands work to an agent and verifies the receipt.
type fakeSteps struct {
	calls   map[Stage]int
	failing Stage
}

func (f *fakeSteps) steps() Stages {
	body := func(name Stage, fill func(*Lane)) Handler {
		return func(c Context, lane *Lane) (Outcome, error) {
			f.calls[name]++
			if name == f.failing {
				return Outcome{}, errors.New("killed mid-step")
			}
			if fill != nil {
				fill(lane)
			}
			return Outcome{Note: string(name) + " done"}, nil
		}
	}
	return Stages{
		{Name: StageWorktree, Piece: 6, Run: body(StageWorktree, func(l *Lane) { l.Branch = "build/" + l.ID })},
		{Name: StageBrief, Piece: 5, Run: body(StageBrief, func(l *Lane) { l.Brief = RunRelDir + "/brief.md" })},
		{Name: StageImplement, Piece: 7,
			Run: func(c Context, lane *Lane) (Outcome, error) {
				f.calls[StageImplement]++
				return Outcome{Await: &Await{Role: "implementer", Brief: lane.Brief,
					Receipt: filepath.Join(c.RepoRoot, "receipt-"+lane.ID+".json")}}, nil
			},
			Verify: func(c Context, lane *Lane, receipt string) error {
				if _, err := os.Stat(receipt); err != nil {
					return refuse("receipt", "", lane.ID, "the receipt names no report", "write the report, then hand the receipt back")
				}
				return nil
			}},
		{Name: StageValidate, Piece: 8, Run: body(StageValidate, nil)},
		{Name: StageLand, Piece: 9, Run: body(StageLand, nil)},
	}
}

func stateBytes(t *testing.T, root, runID string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(StateRelPath(runID))))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestTheHostDrivesTheLoopEndToEnd plays the host (the spec's Approach): step
// performs one binary-owned step per call, returns the agent, brief and
// receipt path at an agent step, advances only on the receipt, and opens the
// next spec step's lane when a lane is done, until the run is complete.
func TestTheHostDrivesTheLoopEndToEnd(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps("1. The parser\n2. The loop\n"))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	steps := f.steps()
	id := start.RunID

	for lane := 1; lane <= 2; lane++ {
		for _, want := range []Stage{StageWorktree, StageBrief} {
			res, err := advance(repo.Root(), id, steps, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if res.PerformedStage != want {
				t.Fatalf("lane %d: performed %q, want %q (%+v)", lane, res.PerformedStage, want, res)
			}
		}
		res, err := advance(repo.Root(), id, steps, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Awaiting == nil || res.Awaiting.Role != "implementer" || res.Awaiting.Receipt == "" || res.Awaiting.Brief == "" {
			t.Fatalf("an agent step tells the host which agent, which brief and where the receipt goes: %+v", res)
		}
		if !strings.Contains(res.Next, "implement receipt") {
			t.Fatalf("the next move names the receipt verb: %q", res.Next)
		}
		receipt := res.Awaiting.Receipt

		// Asking again tells the same thing and moves nothing.
		before := stateBytes(t, repo.Root(), id)
		again, err := advance(repo.Root(), id, steps, Options{})
		if err != nil || again.Awaiting == nil || again.Awaiting.Receipt != receipt || again.PerformedStage != "" {
			t.Fatalf("a step while awaiting re-tells the await: %+v %v", again, err)
		}
		if !bytes.Equal(before, stateBytes(t, repo.Root(), id)) {
			t.Fatal("a step while awaiting must not write the state")
		}

		// The loop advances only on the receipt it named, and only once it verifies.
		if _, err := Receipt(repo.Root(), id, filepath.Join(repo.Root(), "elsewhere.json"), steps, Options{}); err == nil {
			t.Fatal("a receipt at another path must be refused")
		}
		_, err = Receipt(repo.Root(), id, receipt, steps, Options{})
		if r := mustRefusal(t, err); r.Stage != "receipt" {
			t.Fatalf("an unverified receipt is refused at the receipt: %+v", r)
		}
		if !bytes.Equal(before, stateBytes(t, repo.Root(), id)) {
			t.Fatal("a refused receipt must not move the lane")
		}
		if err := os.WriteFile(receipt, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := Receipt(repo.Root(), id, receipt, steps, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if got.PerformedStage != StageImplement || got.Stage != StageValidate {
			t.Fatalf("a verified receipt completes the agent step: %+v", got)
		}
		for _, want := range []Stage{StageValidate, StageLand} {
			res, err := advance(repo.Root(), id, steps, Options{})
			if err != nil || res.PerformedStage != want {
				t.Fatalf("lane %d: performed %q, want %q (%v)", lane, res.PerformedStage, want, err)
			}
		}
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Complete() || len(st.Lanes) != 2 || st.Lanes[1].SpecStep != 2 || st.Lanes[1].Branch != "build/lane-2" {
		t.Fatalf("both spec steps landed through their own lanes: %+v", st)
	}
	fin, err := advance(repo.Root(), id, steps, Options{})
	if err != nil || !fin.Complete {
		t.Fatalf("a complete run says so: %+v %v", fin, err)
	}
	for name, n := range f.calls {
		if n != 2 {
			t.Fatalf("step %s ran %d times over two lanes, want 2", name, n)
		}
	}
}

// TestAKilledStepRepeatsAndACompletedStepDoesNot is criterion 7: a step that
// did not complete leaves the state as it was, so the next invocation performs
// it again, and a step the state records as done is never performed twice.
func TestAKilledStepRepeatsAndACompletedStepDoesNot(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}, failing: StageBrief}
	if _, err := advance(repo.Root(), start.RunID, f.steps(), Options{}); err != nil {
		t.Fatal(err)
	}
	before := stateBytes(t, repo.Root(), start.RunID)
	if _, err := advance(repo.Root(), start.RunID, f.steps(), Options{}); err == nil {
		t.Fatal("a step that fails must report it")
	}
	if !bytes.Equal(before, stateBytes(t, repo.Root(), start.RunID)) {
		t.Fatal("a step that did not complete must leave the state as it was")
	}
	f.failing = ""
	res, err := advance(repo.Root(), start.RunID, f.steps(), Options{})
	if err != nil || res.PerformedStage != StageBrief {
		t.Fatalf("the next invocation performs the step that did not complete: %+v %v", res, err)
	}
	if f.calls[StageWorktree] != 1 || f.calls[StageBrief] != 2 {
		t.Fatalf("completed steps are not repeated: %v", f.calls)
	}
}

// TestAStepThisBuildDoesNotCarryIsRefusedByName: the production sequence names
// every step; one whose body is not built (here, every body stripped) is
// refused with the piece that delivers it, and the run is unchanged.
func TestAStepThisBuildDoesNotCarryIsRefusedByName(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	before := stateBytes(t, repo.Root(), start.RunID)
	bare := DefaultStages()
	for i := range bare {
		bare[i].Run, bare[i].Verify = nil, nil
	}
	_, err = advance(repo.Root(), start.RunID, bare, Options{})
	r := mustRefusal(t, err)
	if r.Stage != string(StageWorktree) || r.Lane != "lane-1" || !strings.Contains(r.Reason, "piece 6") {
		t.Fatalf("want the unbuilt step and its piece named: %+v", r)
	}
	if !bytes.Equal(before, stateBytes(t, repo.Root(), start.RunID)) {
		t.Fatal("a refused step must leave the run unchanged")
	}
	if names := DefaultStages(); len(names) != len(Sequence) {
		t.Fatalf("the production sequence names every step: %d of %d", len(names), len(Sequence))
	}
}

// TestAPauseRefusesUntilNextEligibleAt is decision 2's pause: before the
// window clock's next_eligible_at a step is refused as contention, naming the
// time, and nothing moves.
func TestAPauseRefusesUntilNextEligibleAt(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	start, err := Start(repo.Root(), "itd-10", Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	st, _ := ReadState(repo.Root(), start.RunID)
	later := now.Add(time.Hour)
	st.NextEligibleAt = &later
	root, _ := os.OpenRoot(repo.Root())
	defer root.Close()
	if err := writeState(root, st); err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	_, err = advance(repo.Root(), start.RunID, f.steps(), Options{Now: func() time.Time { return now }})
	r := mustRefusal(t, err)
	if r.Stage != "pause" || !r.Contention || !strings.Contains(r.Reason, "2026-09-25T13:00:00Z") {
		t.Fatalf("want the pause named: %+v", r)
	}
	if f.calls[StageWorktree] != 0 {
		t.Fatal("a paused run performs nothing")
	}
	res, err := advance(repo.Root(), start.RunID, f.steps(), Options{Now: func() time.Time { return later }})
	if err != nil || res.PerformedStage != StageWorktree {
		t.Fatalf("at next_eligible_at the loop moves again: %+v %v", res, err)
	}
}

// TestReadStateFailsClosed: a run id of the wrong shape, an unknown field and
// another schema version are each refused rather than read.
// A repeated key (exact or a case twin encoding/json binds to the same field)
// and a second document are refused too, not read last-wins or first-only
// (iss-2609281204381700).
func TestReadStateFailsClosed(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(repo.Root(), "../../etc"); err == nil {
		t.Fatal("a traversal is not a run id")
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(start.RunID)))
	good := stateBytes(t, repo.Root(), start.RunID)
	cur := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	for name, bad := range map[string]string{
		"unknown field":  strings.Replace(string(good), cur, cur+` "verdict": "SHIP",`, 1),
		"schema version": strings.Replace(string(good), cur, fmt.Sprintf(`"schema_version": %d,`, SchemaVersion+1), 1),
	} {
		if err := os.WriteFile(path, []byte(bad), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadState(repo.Root(), start.RunID); err == nil {
			t.Fatalf("%s: the reader must refuse", name)
		}
	}
	// Each of these reads as the current version under a last-wins or
	// first-document decode, so only a strict reader refuses them; the reason
	// names why.
	for name, tc := range map[string]struct{ bad, want string }{
		"a repeated key": {
			strings.Replace(string(good), cur, `"schema_version": 9, `+cur, 1),
			`duplicate key "schema_version"`,
		},
		"a repeated key spelt as a case twin": {
			strings.Replace(string(good), cur, `"schema_version": 9, "SCHEMA_VERSION"`+strings.TrimPrefix(cur, `"schema_version"`), 1),
			`duplicate key "SCHEMA_VERSION"`,
		},
		"a second document": {string(good) + "\n{}\n", "content after the one JSON document"},
	} {
		if err := os.WriteFile(path, []byte(tc.bad), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := ReadState(repo.Root(), start.RunID)
		if err == nil {
			t.Fatalf("%s: the reader must refuse", name)
		}
		if r := mustRefusal(t, err); r.Stage != "state" || !strings.Contains(r.Reason, tc.want) {
			t.Fatalf("%s: want a state refusal naming %q, got %+v", name, tc.want, r)
		}
	}
}

// TestASymlinkedRunStateIsRefusedInTheRefusalShape: a state file or a run
// directory that is a symlink out of the checkout fails closed as the loop's
// refusal — the step, the reason naming the file, the remedy — not as a
// generic error, from the direct read and from the listing alike.
func TestASymlinkedRunStateIsRefusedInTheRefusalShape(t *testing.T) {
	t.Run("the state file", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		start, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(start.RunID)))
		outside := filepath.Join(t.TempDir(), "state.json")
		if err := os.WriteFile(outside, stateBytes(t, repo.Root(), start.RunID), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, path); err != nil {
			t.Fatal(err)
		}
		_, err = ReadState(repo.Root(), start.RunID)
		if r := mustRefusal(t, err); r.Stage != "state" || !strings.Contains(r.Reason, StateRelPath(start.RunID)) {
			t.Fatalf("want the state file named at the state step: %+v", r)
		}
		_, err = Runs(repo.Root())
		mustRefusal(t, err)
	})
	t.Run("the run directory", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		start, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), start.RunID)
		outside := filepath.Join(t.TempDir(), start.RunID)
		if err := os.Rename(dir, outside); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, dir); err != nil {
			t.Fatal(err)
		}
		_, err = Runs(repo.Root())
		if r := mustRefusal(t, err); r.Stage != "state" {
			t.Fatalf("want the state step: %+v", r)
		}
		_, err = Start(repo.Root(), "itd-10", Options{})
		mustRefusal(t, err)
	})
}

// TestResolveNamesTheOnlyLiveRun: a call without a run id addresses the one
// live run, and is refused naming them when there are several or none.
func TestResolveNamesTheOnlyLiveRun(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	if _, err := Resolve(repo.Root(), ""); err == nil {
		t.Fatal("no run: refused")
	}
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, err := Resolve(repo.Root(), "")
	if err != nil || id != start.RunID {
		t.Fatalf("Resolve = %q %v, want %q", id, err, start.RunID)
	}
}

// TestRefusalRendersStepReasonAndRemedy is criterion 13's text half.
func TestRefusalRendersStepReasonAndRemedy(t *testing.T) {
	r := &Refusal{Stage: "check", Check: CheckHold, Reason: "itd-10 is held", Remedy: "run `abcd intent unhold itd-10`"}
	if got := r.Error(); got != "refused at check (hold): itd-10 is held; remedy: run `abcd intent unhold itd-10`" {
		t.Fatalf("Error() = %q", got)
	}
}

// TestAReceiptNamedThroughASymlinkedPathIsTheReceiptAwaited: the host may
// reach the checkout through a symlinked spelling of its path (macOS's /var
// and /tmp are symlinks) while the loop's root is git's resolved toplevel; the
// receipt the lane awaits is the same file under either spelling, and is
// taken rather than refused as "not at the path given" (iss-2609261534097255).
func TestAReceiptNamedThroughASymlinkedPathIsTheReceiptAwaited(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	var res StepResult
	for range 3 {
		if res, err = advance(repo.Root(), start.RunID, f.steps(), Options{}); err != nil {
			t.Fatal(err)
		}
	}
	if res.Awaiting == nil {
		t.Fatalf("want the lane awaiting: %+v", res)
	}
	if err := os.WriteFile(res.Awaiting.Receipt, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "checkout")
	if err := os.Symlink(repo.Root(), link); err != nil {
		t.Fatal(err)
	}
	via := filepath.Join(link, filepath.Base(res.Awaiting.Receipt))
	got, err := Receipt(repo.Root(), start.RunID, via, f.steps(), Options{})
	if err != nil {
		t.Fatalf("the awaited receipt named through a symlinked path is the same receipt: %v", err)
	}
	if got.PerformedStage != StageImplement {
		t.Fatalf("the receipt completes the step: %+v", got)
	}
}

// TestTheRecordNamesEachLaneAsItOpens is itd-2609212103565953's fourth
// criterion, the record half: the run record lists the spec's steps as it
// lists the lanes, one line for each lane opened, naming the spec step it
// builds, the first at the start and each later one when the lane before it is
// done.
func TestTheRecordNamesEachLaneAsItOpens(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps("1. The parser\n2. The loop\n3. The page\n"))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	steps := f.steps()
	id := start.RunID
	for range 64 {
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		i := st.current()
		if i < 0 {
			break
		}
		if a := st.Lanes[i].awaiting(); a != nil {
			if err := os.WriteFile(a.Receipt, []byte("{}\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Receipt(repo.Root(), id, a.Receipt, steps, Options{}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if _, err := advance(repo.Root(), id, steps, Options{}); err != nil {
			t.Fatal(err)
		}
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Complete() {
		t.Fatalf("the run did not complete: %+v", st)
	}
	var opened []string
	for _, e := range st.Record {
		if e.Stage == "start" || e.Stage == "open" {
			opened = append(opened, e.Lane+": "+e.Note)
		}
	}
	want := []string{
		"lane-1: checks passed; lane-1 opened for step 1 of spc-1 (The parser)",
		"lane-2: lane-2 opened for step 2 of spc-1 (The loop)",
		"lane-3: lane-3 opened for step 3 of spc-1 (The page)",
	}
	if strings.Join(opened, "\n") != strings.Join(want, "\n") {
		t.Fatalf("the record names each lane's step as it opens:\n got %q\nwant %q", opened, want)
	}
}
