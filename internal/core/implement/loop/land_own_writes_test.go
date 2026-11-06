package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// strayClaim is what a refusal says of a change outside the record stores.
const strayClaim = "the links its own moves repointed"

// committedFiles lists what the commit changes, against its first parent.
func committedFiles(f *landFixture, l Lane, sha string) []string {
	return strings.Fields(f.repo.Git("-C", l.Worktree, "diff-tree", "--no-commit-id", "--name-only", "-r", sha))
}

const (
	intentStoreAgents = ".abcd/development/intents/AGENTS.md"
	deepInABucket     = ".abcd/work/issues/open/nested/iss-5-deep.md"
	unreadInABucket   = ".abcd/development/specs/open/CLAUDE.md"
)

// A stray file names itself and how to take it out, on the first attempt and on a
// retry after a refusing hook; the commit made once it is gone does not hold it.
func TestALandingCommitsOnlyWhatItWrote(t *testing.T) {
	strays := map[string]struct {
		rel    string
		remove func(t *testing.T, f *landFixture, l Lane)
	}{
		"a new markdown file": {"docs/stray.md", func(t *testing.T, _ *landFixture, l Lane) {
			t.Helper()
			if err := os.Remove(filepath.Join(l.Worktree, "docs", "stray.md")); err != nil {
				t.Fatal(err)
			}
		}},
		"a tracked markdown file changed": {"AGENTS.md", func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			f.repo.Git("-C", l.Worktree, "checkout", "-q", "--", "AGENTS.md")
		}},
		"instructions directly under a store": {intentStoreAgents, func(t *testing.T, _ *landFixture, l Lane) {
			t.Helper()
			if err := os.Remove(filepath.Join(l.Worktree, filepath.FromSlash(intentStoreAgents))); err != nil {
				t.Fatal(err)
			}
		}},
		"a path deeper than a bucket": {deepInABucket, func(t *testing.T, _ *landFixture, l Lane) {
			t.Helper()
			if err := os.RemoveAll(filepath.Join(l.Worktree, filepath.Dir(filepath.FromSlash(deepInABucket)))); err != nil {
				t.Fatal(err)
			}
		}},
		"a name the store does not read in a bucket": {unreadInABucket, func(t *testing.T, _ *landFixture, l Lane) {
			t.Helper()
			if err := os.Remove(filepath.Join(l.Worktree, filepath.FromSlash(unreadInABucket))); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for name, s := range strays {
		for _, retry := range []bool{false, true} {
			t.Run(name+map[bool]string{false: ", first attempt", true: ", after a refusing hook"}[retry], func(t *testing.T) {
				f := newLandFixture(t, queueRuleset("MERGE"))
				l := f.validatedOnce(t)
				judged := l.HeadSHA
				var hook string
				if retry {
					hook = installHook(t, f, "commit-msg", "#!/bin/sh\necho 'commit-msg: refused by the test' >&2\nexit 1\n")
				}
				f.step(t) // prepare
				if retry {
					if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "refused by the test") {
						t.Fatalf("the hook refuses first: %+v", r)
					}
				}
				landWrite(t, l.Worktree, s.rel, "not the landing's\n")
				r := f.wantsRefusedStep(t)
				if !strings.Contains(r.Reason, s.rel) || !strings.Contains(r.Reason, strayClaim) || !strings.Contains(r.Remedy, "remove it") {
					t.Fatalf("the refusal names the file and how to take it out: %+v", r)
				}
				if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != judged {
					t.Fatalf("nothing is committed: %s, %s", got, judged)
				}
				s.remove(t, f, l)
				if retry {
					if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
						t.Fatal(err)
					}
					acceptHook(t, f)
				}
				l = f.wantsRecordsCommit(t)
				if slices.Contains(committedFiles(f, l, l.HeadSHA), s.rel) {
					t.Fatalf("the commit holds %s: %v", s.rel, committedFiles(f, l, l.HeadSHA))
				}
			})
		}
	}
}

// A link the close repoints in a file outside the record stores is committed, on the
// attempt after a hook refused it too, and a further edit to that file is not.
func TestALandingCommitsTheLinksItsCloseRepointed(t *testing.T) {
	const guide = "docs/guide.md"
	link := "See [the intent](../" + plannedIntentRel + "), and `planned/itd-10-alpha.md` by name.\nMore.\n"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		guide: link,
	})
	l := f.validated(t)
	hook := installHook(t, f, "commit-msg", "#!/bin/sh\necho 'commit-msg: refused by the test' >&2\nexit 1\n")
	f.step(t) // prepare
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "refused by the test") {
		t.Fatalf("the hook refuses what the landing wrote, in the one step: %+v", r)
	}
	repointed := landRead(t, l.Worktree, guide)
	if !strings.Contains(repointed, "/shipped/") {
		t.Fatalf("the link follows the move:\n%s", repointed)
	}
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "refused by the test") {
		t.Fatalf("the repointed file passes the retry's judgement: %+v", r)
	}
	for name, edit := range map[string]string{
		"a word on the link's line": strings.Replace(repointed, "See ", "Do see ", 1),
		"a line added":              repointed + "Another.\n",
		"another line changed":      strings.Replace(repointed, "More.", "Less.", 1),
	} {
		landWrite(t, l.Worktree, guide, edit)
		if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, guide) || !strings.Contains(r.Reason, strayClaim) {
			t.Fatalf("%s: the further edit is refused: %+v", name, r)
		}
	}
	landWrite(t, l.Worktree, guide, repointed)
	landWrite(t, l.Worktree, "docs/stray.md", "not the landing's\n")
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "docs/stray.md") {
		t.Fatalf("a stray file beside the repointed one is refused: %+v", r)
	}
	if err := os.Remove(filepath.Join(l.Worktree, "docs", "stray.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	acceptHook(t, f)
	l = f.wantsRecordsCommit(t)
	got := committedFiles(f, l, l.HeadSHA)
	if !slices.Contains(got, guide) || slices.Contains(got, "docs/stray.md") {
		t.Fatalf("the commit holds the repointed link and nothing stray: %v", got)
	}
}

// A link the repoint rewrites in a form the judgement cannot read is a fact of the judged
// head, since the repoint writes it again on every call: the lane is handed back, and the
// way out holds.
func TestALinkTheJudgementCannotReadHandsTheLaneBack(t *testing.T) {
	for name, c := range map[string]struct{ rel, body string }{
		"written from beside the record": {".abcd/development/intents/planned/notes.md", "See [it](itd-10-alpha.md).\n"},
		"not in its shortest form":       {"docs/long.md", "See [it](../docs/../" + plannedIntentRel + ").\n"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixtureWith(t, map[string]string{
				".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
				c.rel: c.body,
			})
			l := f.validated(t)
			f.step(t) // prepare
			res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			if err != nil {
				t.Fatalf("the step hands the lane back rather than refuse: %v", err)
			}
			if res.HandBack == nil || res.HandBack.Kind != HandBackStuck || !strings.Contains(res.HandBack.Reason, c.rel) {
				t.Fatalf("the hand-back names the file: %+v", res.HandBack)
			}
			f.repo.Git("-C", l.Worktree, "reset", "-q", "--hard", "HEAD")
			f.repo.Git("-C", l.Worktree, "clean", "-qfd")
			if _, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err != nil {
				t.Fatalf("the lane is discarded as the hand-back says: %v", err)
			}
		})
	}
}

// A file a person restores after a refusal is repointed again by the next call, and a
// store's README a move reaches is judged by what it holds, like any other file.
func TestALandingRepointsARestoredFileAgain(t *testing.T) {
	const readme = ".abcd/development/intents/README.md"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		readme: "The [intent](planned/itd-10-alpha.md).\n",
	})
	l := f.validated(t)
	hook := installHook(t, f, "commit-msg", "#!/bin/sh\nexit 1\n")
	f.step(t) // prepare
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "could not commit") {
		t.Fatalf("the hook refuses: %+v", r)
	}
	landWrite(t, l.Worktree, readme, landRead(t, l.Worktree, readme)+"A hand's line.\n")
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, readme) {
		t.Fatalf("a hand's edit to a store's README is refused: %+v", r)
	}
	f.repo.Git("-C", l.Worktree, "checkout", "-q", "HEAD", "--", readme)
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	acceptHook(t, f)
	l = f.wantsRecordsCommit(t)
	if body := landRead(t, l.Worktree, readme); !strings.Contains(body, "(shipped/itd-10-alpha.md)") {
		t.Fatalf("the link is repointed again:\n%s", body)
	}
}

// A file git hides from the worktree's status either stays out of the index or is
// staged and judged: a flag that stands leaves it out of the commit, one cleared between
// calls meets the next call's judgement, and one cleared during the close meets the
// judgement of what was staged.
func TestAFlaggedFileIsJudgedWhenItIsStaged(t *testing.T) {
	const guide = "docs/guide.md"
	body := "See [the intent](../" + plannedIntentRel + ").\n"
	payload := "Not the landing's.\n"
	for _, when := range []string{"stands", "cleared between calls", "cleared during the close"} {
		t.Run(when, func(t *testing.T) {
			f := newLandFixtureWith(t, map[string]string{
				".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
				guide: body,
			})
			l := f.validated(t)
			hide := func() { f.repo.Git("-C", l.Worktree, "update-index", "--skip-worktree", guide) }
			show := func() { f.repo.Git("-C", l.Worktree, "update-index", "--no-skip-worktree", guide) }
			hide()
			landWrite(t, l.Worktree, guide, body+payload)
			f.step(t) // prepare
			switch when {
			case "stands":
				l = f.wantsRecordsCommit(t)
				if got := committedFiles(f, l, l.HeadSHA); slices.Contains(got, guide) {
					t.Fatalf("the flagged file stays out of the commit: %v", got)
				}
				return
			case "cleared between calls":
				hook := installHook(t, f, "commit-msg", "#!/bin/sh\nexit 1\n")
				if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "could not commit") {
					t.Fatalf("the hook refuses: %+v", r)
				}
				if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				acceptHook(t, f)
				show()
			default:
				beforeLandingStage = func(string) { show() }
				t.Cleanup(func() { beforeLandingStage = nil })
			}
			r := f.wantsRefusedStep(t)
			if !strings.Contains(r.Reason, guide) || !strings.Contains(r.Reason, strayClaim) {
				t.Fatalf("the payload is refused: %+v", r)
			}
			beforeLandingStage = nil
			f.repo.Git("-C", l.Worktree, "checkout", "-q", "HEAD", "--", guide)
			l = f.wantsRecordsCommit(t)
			if got := landRead(t, l.Worktree, guide); strings.Contains(got, "Not the landing's") || !strings.Contains(got, "/shipped/") {
				t.Fatalf("the commit holds the link and no payload:\n%s", got)
			}
		})
	}
}

// A commit found on the judged head under the landing's subject meets the judgement the
// landing's own commit meets.
func TestACommitFoundOnTheJudgedHeadIsJudgedLikeTheLandingsOwn(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	f.step(t) // prepare
	cur := currentLane(t, f.repo, f.runID)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	subject := landSubject(st, cur)
	landWrite(t, l.Worktree, "docs/found.md", "not the landing's\n")
	f.repo.Git("-C", l.Worktree, "add", "-A")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", subject)
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "docs/found.md") || !strings.Contains(r.Remedy, "restore the branch") {
		t.Fatalf("the refusal names the file and how to restore the branch: %+v", r)
	}
	if got := currentLane(t, f.repo, f.runID); got.Landing.RecordsDone {
		t.Fatalf("the commit is not taken for the landing's records: %+v", got.Landing)
	}
}

// What a person pastes from a hand-back is built from the ids the loop validates, never
// from the branch a state file spells.
func TestAHandBackRendersItsCommandsFromValidatedIDs(t *testing.T) {
	const runID = "run-2611061512102308"
	lane := Lane{ID: "lane-1", Branch: "build/x`; touch y"}
	st := State{RunID: runID, Intent: "itd-10", Lanes: []Lane{lane}}
	home := closeHandBack(Context{State: st}, lane, "a reason").HandBack.Home
	if strings.Contains(home, "touch y") {
		t.Fatalf("the state's spelling reaches the command: %s", home)
	}
	if want := "`git tag saved-" + runID + "-lane-1 " + BranchPrefix + runID + "-lane-1`"; !strings.Contains(home, want) {
		t.Fatalf("the command is the derived one, %s: %s", want, home)
	}
	st.RunID, lane.ID = "run-`x`", "lane-`y`"
	home = closeHandBack(Context{State: st}, lane, "a reason").HandBack.Home
	if strings.Contains(home, "`x`") || strings.Contains(home, "`y`") || strings.Contains(home, "git tag") {
		t.Fatalf("ids the loop does not mint are in no command: %s", home)
	}
	if !strings.Contains(home, "state file") {
		t.Fatalf("the text says what to restore: %s", home)
	}
}

// A resolution found in the worktree still repoints the links naming the record where
// it stood: a lane whose second resolution is refused does not end with the first's link
// dangling.
func TestAResolutionFoundDoneRepointsItsLinks(t *testing.T) {
	const guide = "docs/guide.md"
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	found, err := filepath.Glob(filepath.Join(l.Worktree, ".abcd", "work", "issues", "open", f.issue+"*"))
	if err != nil || len(found) != 1 {
		t.Fatalf("premise: the issue's record: %v %v", found, err)
	}
	f.commitAtTheJudgedHead(t, l, func() {
		landWrite(t, l.Worktree, guide, "See [the issue](../.abcd/work/issues/open/"+filepath.Base(found[0])+").\n")
	})
	f.step(t) // prepare
	cur := currentLane(t, f.repo, f.runID)
	first := cur.Resolves[0]
	second := Resolution{Issue: "iss-77", Commit: first.Commit, Note: first.Note, Impact: "not an impact", Grounds: first.Grounds}
	setResolves := func(rs ...Resolution) {
		t.Helper()
		if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
			st.Lanes[st.current()].Resolves = rs
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	f.fileAnotherIssue(t, cur)
	f.repo.Git("-C", l.Worktree, "add", "-A")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "another issue at the judged head")
	head := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD")
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[st.current()].HeadSHA = head
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	setResolves(first, second)
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "iss-77") {
		t.Fatalf("the second resolution is refused: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(l.Worktree, ".abcd", "work", "issues", "resolved", filepath.Base(found[0]))); err != nil {
		t.Fatalf("premise: the first resolution stands: %v", err)
	}
	// A call killed between the move and the repoint leaves the link as the head has it.
	f.repo.Git("-C", l.Worktree, "checkout", "-q", "HEAD", "--", guide)
	second.Impact = "fix"
	setResolves(first, second)
	l = f.wantsRecordsCommit(t)
	if body := landRead(t, l.Worktree, guide); !strings.Contains(body, "/resolved/") {
		t.Fatalf("the first's link follows its move:\n%s", body)
	}
}

// walkIssueLaneToLand takes a drain's issue lane to its first landing step.
func (f *landFixture) walkIssueLaneToLand(t *testing.T) Lane {
	t.Helper()
	first := f.firstMove(t)
	f.runID, f.issue = first.Start.RunID, secondIssue
	repo := f.repo
	stepTo(t, repo, f.runID, f.stages, StageImplement)
	if res, err := advance(repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, f.runID)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	sha := laneCommit(t, repo, l, "fix.txt")
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Model = "claude-test-5"
	rc.Resolves = []Resolution{{Issue: f.issue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	if _, err := Receipt(repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	passRound(t, repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	stepTo(t, repo, f.runID, f.stages, StageLand)
	return currentLane(t, repo, f.runID)
}

// An issue lane moves no intent: a move in its worktree is put back before its records
// commit, which carries its resolution alone.
func TestAnIssueLanesRecordsCarryNoIntentMove(t *testing.T) {
	f := drainLandFixture(t)
	l := f.walkIssueLaneToLand(t)
	planned := readyIntent("impact: additive\n", settledQuestions)
	f.commitAtTheJudgedHead(t, l, func() { landWrite(t, l.Worktree, plannedIntentRel, planned) })
	f.step(t) // prepare
	if l = currentLane(t, f.repo, f.runID); l.Landing == nil {
		t.Fatalf("premise: the landing is prepared: %+v", l)
	}
	landWrite(t, l.Worktree, shippedIntentRel, planned)
	if err := os.Remove(filepath.Join(l.Worktree, filepath.FromSlash(plannedIntentRel))); err != nil {
		t.Fatal(err)
	}
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "itd-10") || !strings.Contains(r.Remedy, "put it back") {
		t.Fatalf("the refusal names the record and how to put it back: %+v", r)
	}
	if err := os.Remove(filepath.Join(l.Worktree, filepath.FromSlash(shippedIntentRel))); err != nil {
		t.Fatal(err)
	}
	landWrite(t, l.Worktree, plannedIntentRel, planned)
	l = f.wantsRecordsCommit(t)
	for _, p := range committedFiles(f, l, l.HeadSHA) {
		if strings.Contains(p, "/intents/") {
			t.Fatalf("the records commit carries %s", p)
		}
	}
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	if !strings.Contains(msg, "Resolves: "+f.issue) || strings.Contains(msg, "Delivers:") {
		t.Fatalf("the commit resolves the issue and delivers nothing:\n%s", msg)
	}
}

// A link a resolution repoints outside the record stores is the landing's own write on
// an issue lane too.
func TestAnIssueLaneCommitsTheLinksItsResolutionRepointed(t *testing.T) {
	const guide = "docs/guide.md"
	f := drainLandFixture(t)
	l := f.walkIssueLaneToLand(t)
	found, err := filepath.Glob(filepath.Join(l.Worktree, ".abcd", "work", "issues", "open", f.issue+"*"))
	if err != nil || len(found) != 1 {
		t.Fatalf("premise: the issue's record: %v %v", found, err)
	}
	f.commitAtTheJudgedHead(t, l, func() {
		landWrite(t, l.Worktree, guide, "See [the issue](../.abcd/work/issues/open/"+filepath.Base(found[0])+").\n")
	})
	f.step(t) // prepare
	l = f.wantsRecordsCommit(t)
	if got := committedFiles(f, l, l.HeadSHA); !slices.Contains(got, guide) {
		t.Fatalf("the commit holds the repointed link: %v", got)
	}
	if body := landRead(t, l.Worktree, guide); !strings.Contains(body, "/resolved/") {
		t.Fatalf("the link follows the move:\n%s", body)
	}
}

// A changed line follows the moves when it is the head's with moved paths substituted,
// occurrence by occurrence; nothing else does.
func TestAChangeFollowsTheMovesOnlyAsTheyWouldWriteIt(t *testing.T) {
	moves := []recordMove{{"planned/itd-10-a.md", "shipped/itd-10-a.md"}, {"open/iss-1-b.md", "resolved/iss-1-b.md"}}
	const head = "A [link](../i/planned/itd-10-a.md) and [one](x/open/iss-1-b.md#f).\nOther.\n"
	for name, c := range map[string]struct {
		now  string
		want bool
	}{
		"every link moved":   {"A [link](../i/shipped/itd-10-a.md) and [one](x/resolved/iss-1-b.md#f).\nOther.\n", true},
		"one link moved":     {"A [link](../i/shipped/itd-10-a.md) and [one](x/open/iss-1-b.md#f).\nOther.\n", true},
		"no line differs":    {head, false},
		"a word beside":      {"A [link](../i/shipped/itd-10-a.md) and [one](x/open/iss-1-b.md#f)!\nOther.\n", false},
		"another path":       {"A [link](../i/shipped/itd-11-a.md) and [one](x/open/iss-1-b.md#f).\nOther.\n", false},
		"a move backwards":   {"A [link](../i/planned/itd-10-a.md) and [one](x/open/iss-1-b.md#f).\nOther?\n", false},
		"a line added":       {"A [link](../i/shipped/itd-10-a.md) and [one](x/open/iss-1-b.md#f).\nOther.\nMore.\n", false},
		"a line removed":     {"A [link](../i/shipped/itd-10-a.md) and [one](x/open/iss-1-b.md#f).\n", false},
		"nothing was linked": {"Nothing.\nOther.\n", false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := followsMoves(head, c.now, moves); got != c.want {
				t.Fatalf("followsMoves = %v, want %v", got, c.want)
			}
		})
	}
	if followsMoves(head, strings.Replace(head, "planned", "shipped", 1), nil) {
		t.Fatal("with no move made, no change follows")
	}
}

// A hand-back while siblings run does not send the person to delete the directory they
// use, and the discard it names works now.
func TestAHandBackMidRunLeavesTheRunsDirectoryToLater(t *testing.T) {
	f := siblingAtArm(t, false, queueRuleset("MERGE"), func(ld *Landing) bool { return !ld.RecordsDone })
	l := f.lane(t, "lane-2")
	if !strings.Contains(strings.Join(busyLanes(f.state(t)), " "), "lane-1") {
		t.Fatalf("premise: lane 1 is at work: %v", busyLanes(f.state(t)))
	}
	landWrite(t, l.Worktree, ".abcd/development/specs/open/spc-0-twin.md", landRead(t, l.Worktree, specRel))
	f.repo.Git("-C", l.Worktree, "add", "-A")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "a change at the judged head")
	head := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD")
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		for i := range st.Lanes {
			if st.Lanes[i].ID == l.ID {
				st.Lanes[i].HeadSHA = head
			}
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	res := f.step(t)
	if res.HandBack == nil || res.HandBack.Kind != HandBackStuck {
		t.Fatalf("the lane is handed back: %+v", res)
	}
	home := res.HandBack.Home
	discard, wait, remove := strings.Index(home, "--discard "+l.ID), strings.Index(home, "once"), strings.Index(home, runRel(f.runID))
	if discard < 0 || wait < discard || remove < wait || !strings.Contains(home, "lane-1") {
		t.Fatalf("the directory waits for the other lanes, which the text names: %s", home)
	}
	if _, err := Discard(f.repo.Root(), f.runID, l.ID, f.opts()); err != nil {
		t.Fatalf("the lane is discarded as the hand-back says, while its sibling works: %v", err)
	}
	if got := f.lane(t, "lane-1"); len(got.Awaits) != 1 {
		t.Fatalf("the sibling stays at work: %+v", got)
	}
}

// An uncommitted change left at a hand-back stops the discard, which says so and says
// to run itself again; putting the change back lets it through.
func TestAHandBackWithAChangeInTheWorktreeHasAWayOut(t *testing.T) {
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps("1. The parser\n   - packages: internal/core/spec\n"),
	})
	l := f.validatedOnce(t)
	f.commitAtTheJudgedHead(t, l, func() {
		landWrite(t, l.Worktree, plannedIntentRel, strings.Replace(landRead(t, l.Worktree, plannedIntentRel), "\nslug:", "\nheld: \"a reason\"\nslug:", 1))
	})
	landWrite(t, l.Worktree, specRel, strings.Replace(landRead(t, l.Worktree, specRel), "A written design record.", "A written design record, settled.", 1))
	f.step(t) // prepare
	if got := currentLane(t, f.repo, f.runID); got.Landing == nil || !got.Landing.SpecSettled {
		t.Fatalf("premise: the landing carries a settled spec: %+v", got.Landing)
	}
	res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if err != nil || res.HandBack == nil {
		t.Fatalf("the lane is handed back: %+v %v", res, err)
	}
	if !strings.Contains(res.HandBack.Home, "uncommitted") {
		t.Fatalf("the way out says what to do with what is uncommitted: %s", res.HandBack.Home)
	}
	_, err = Discard(f.repo.Root(), f.runID, l.ID, Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Remedy, "--discard "+l.ID) {
		t.Fatalf("the refusal says to discard again, not to step: %+v", r)
	}
	f.repo.Git("-C", l.Worktree, "checkout", "-q", "--", specRel)
	if _, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err != nil {
		t.Fatalf("the discard goes through once the change is put back: %v", err)
	}
}
