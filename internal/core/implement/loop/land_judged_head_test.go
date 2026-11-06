package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	secondSpecRel    = ".abcd/development/specs/open/spc-2-second.md"
	shippedIntentRel = ".abcd/development/intents/shipped/itd-10-alpha.md"
)

func landWrite(t *testing.T, wt, rel, body string) {
	t.Helper()
	abs := filepath.Join(wt, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func landRead(t *testing.T, wt, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(wt, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// A landing that closes the spec holds what the close ships, and what holds it back,
// to what the judged head says.
func TestWhatAClosingLaneShipsIsAsTheJudgedHeadHasIt(t *testing.T) {
	const secondBody = "---\nid: spc-2\nslug: second\nintent: itd-10\n---\n# second\n\n## Summary\n\nAnother record.\n"
	for name, c := range map[string]struct {
		atHead func(t *testing.T, wt string)
		change func(t *testing.T, wt string)
		names  string
	}{
		"another open spec of the intent moved to closed": {
			atHead: func(t *testing.T, wt string) { t.Helper(); landWrite(t, wt, secondSpecRel, secondBody) },
			change: func(t *testing.T, wt string) {
				t.Helper()
				landWrite(t, wt, strings.Replace(secondSpecRel, "/open/", "/closed/", 1), secondBody)
				if err := os.Remove(filepath.Join(wt, filepath.FromSlash(secondSpecRel))); err != nil {
					t.Fatal(err)
				}
			},
			names: "spc-2-second.md",
		},
		"another open spec of the intent linked elsewhere": {
			atHead: func(t *testing.T, wt string) { t.Helper(); landWrite(t, wt, secondSpecRel, secondBody) },
			change: func(t *testing.T, wt string) {
				t.Helper()
				landWrite(t, wt, secondSpecRel, strings.Replace(secondBody, "intent: itd-10\n", "intent: itd-11\n", 1))
			},
			names: "spc-2-second.md",
		},
		"another open spec of the intent removed": {
			atHead: func(t *testing.T, wt string) { t.Helper(); landWrite(t, wt, secondSpecRel, secondBody) },
			change: func(t *testing.T, wt string) {
				t.Helper()
				if err := os.Remove(filepath.Join(wt, filepath.FromSlash(secondSpecRel))); err != nil {
					t.Fatal(err)
				}
			},
			names: "spc-2-second.md",
		},
		"a hold on the intent lifted": {
			atHead: func(t *testing.T, wt string) {
				t.Helper()
				landWrite(t, wt, plannedIntentRel, strings.Replace(landRead(t, wt, plannedIntentRel), "\nslug:", "\nheld: \"a reason\"\nslug:", 1))
			},
			change: func(t *testing.T, wt string) {
				t.Helper()
				landWrite(t, wt, plannedIntentRel, strings.Replace(landRead(t, wt, plannedIntentRel), "held: \"a reason\"\n", "", 1))
			},
			names: "itd-10-alpha.md",
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.validatedOnce(t)
			head := f.commitAtTheJudgedHead(t, l, func() { c.atHead(t, l.Worktree) })
			f.step(t)
			if cur := currentLane(t, f.repo, f.runID); cur.Landing == nil || !cur.Landing.Closes {
				t.Fatalf("premise: the landing closes the spec: %+v", cur.Landing)
			}
			c.change(t, l.Worktree)

			_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			if _, statErr := os.Stat(filepath.Join(l.Worktree, filepath.FromSlash(shippedIntentRel))); statErr == nil {
				t.Fatalf("the intent shipped; step: %v", err)
			}
			if err == nil {
				t.Fatal("the records step went through")
			}
			if r := mustRefusal(t, err); !strings.Contains(r.Reason, c.names) || !strings.Contains(r.Remedy, "as the judged head") {
				t.Fatalf("the refusal names the record and the way back: %+v", r)
			}
			if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != head {
				t.Fatalf("no records commit is made: %s, %s", got, head)
			}
		})
	}
}

// What the remedies ask of a closing lane stays open to it: an intent's related_issues,
// its impact, and the text of another record, which a move's relink reaches.
func TestAClosingLaneTakesTheEditsItsRemediesAskFor(t *testing.T) {
	const secondBody = "---\nid: spc-2\nslug: second\nintent: itd-11\n---\n# second\n\n## Summary\n\nAnother record.\n"
	files := map[string]string{".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE")}
	for _, bucket := range []string{"specs/open", "specs/closed", "intents/planned", "intents/shipped", "intents/drafts"} {
		files[".abcd/development/"+bucket+"/README.md"] = "# What this folder holds\n"
	}
	f := newLandFixtureWith(t, files)
	l := f.validated(t)
	f.fileAnotherIssue(t, l)
	f.laneNamesIssue(t, l, "iss-77")
	f.commitAtTheJudgedHead(t, l, func() { landWrite(t, l.Worktree, secondSpecRel, secondBody) })
	r := f.wantsRefusedStepAfterPrepare(t)
	if !strings.Contains(r.Reason, "iss-77") {
		t.Fatalf("the close refuses the link: %+v", r)
	}
	landWrite(t, l.Worktree, plannedIntentRel, strings.Replace(landRead(t, l.Worktree, plannedIntentRel), "related_issues: [iss-77]\n", "", 1))
	landWrite(t, l.Worktree, secondSpecRel, strings.Replace(secondBody, "Another record.", "Another record, relinked.", 1))
	cur := f.wantsRecordsCommit(t)
	if got := f.repo.Git("-C", cur.Worktree, "show", cur.HeadSHA+":"+shippedIntentRel); strings.Contains(got, "iss-77") {
		t.Fatalf("the records commit carries the settled intent:\n%s", got)
	}
}

// wantsRefusedStepAfterPrepare takes the first step and finds the second refuse.
func (f *landFixture) wantsRefusedStepAfterPrepare(t *testing.T) *Refusal {
	t.Helper()
	f.step(t)
	return f.wantsRefusedStep(t)
}

// A record the spec store reads under its own pattern is the lane's own, whatever else
// its name holds.
func TestASpecRecordNamedByHandIsTheLanesOwn(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	named := strings.Replace(specRel, "spc-1-alpha.md", "spc-1-My_Thing.md", 1)
	f.commitAtTheJudgedHead(t, l, func() {
		f.repo.Git("-C", l.Worktree, "mv", specRel, named)
	})
	f.step(t)
	cur := f.wantsRecordsCommit(t)
	files := f.repo.Git("-C", cur.Worktree, "ls-tree", "-r", "--name-only", cur.HeadSHA)
	if !strings.Contains(files, shippedIntentRel) || !strings.Contains(files, "specs/closed/spc-1-My_Thing.md") {
		t.Fatalf("the close ships the intent and closes the record:\n%s", files)
	}
}

// A commit holding more than the mark names the lane's model, so with no model to name
// it is refused, and what the step wrote stays in the lane's worktree for the next call.
func TestACommitHoldingMoreThanTheMarkNeedsAModel(t *testing.T) {
	const steps = "1. The parser\n   - packages: internal/core/spec\n2. The loop\n   - packages: internal/core/implement\n"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps(steps),
	})
	l := f.twoStepLandWithNoResolutions(t)
	head := l.HeadSHA
	setModel(t, f, "", nil, true)
	f.step(t)
	landWrite(t, l.Worktree, "note.md", "a note\n")
	r := f.wantsRefusedStep(t)
	if !strings.Contains(r.Reason, "did not compose") || !strings.Contains(r.Remedy, "model") {
		t.Fatalf("the refusal names the model: %+v", r)
	}
	if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != head {
		t.Fatalf("no records commit is made: %s, %s", got, head)
	}
	if !strings.Contains(landRead(t, l.Worktree, specRel), "- landed: "+shortSHA(head)) {
		t.Fatal("the mark stays in the lane's worktree")
	}
}

// An impact the judged head declares stays as it has it; one it does not declare is the
// worktree's to add, as the close's remedy asks.
func TestAnImpactTheJudgedHeadDeclaresStays(t *testing.T) {
	const declared = "impact: additive\n"
	for name, c := range map[string]struct {
		atHead, inWorktree string
		ships              bool
	}{
		"lowered":                         {"impact: breaking\n", "impact: fix\n", false},
		"taken away":                      {"impact: breaking\n", "", false},
		"added to no impact":              {"", "impact: fix\n", true},
		"replacing one the close refuses": {"impact: not-an-impact\n", "impact: fix\n", true},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.validatedOnce(t)
			head := f.commitAtTheJudgedHead(t, l, func() {
				landWrite(t, l.Worktree, plannedIntentRel, strings.Replace(landRead(t, l.Worktree, plannedIntentRel), declared, c.atHead, 1))
			})
			f.step(t)
			body := landRead(t, l.Worktree, plannedIntentRel)
			if c.atHead == "" {
				body = strings.Replace(body, "\nslug:", "\n"+c.inWorktree+"slug:", 1)
			} else {
				body = strings.Replace(body, c.atHead, c.inWorktree, 1)
			}
			landWrite(t, l.Worktree, plannedIntentRel, body)

			_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			_, statErr := os.Stat(filepath.Join(l.Worktree, filepath.FromSlash(shippedIntentRel)))
			if c.ships {
				if err != nil || statErr != nil {
					t.Fatalf("the intent ships with the impact added: %v %v", err, statErr)
				}
				return
			}
			if statErr == nil {
				t.Fatalf("the intent shipped; step: %v", err)
			}
			if err == nil {
				t.Fatal("the records step went through")
			}
			if r := mustRefusal(t, err); !strings.Contains(r.Reason, "itd-10-alpha.md") || !strings.Contains(r.Reason, "impact") {
				t.Fatalf("the refusal names the record and its impact: %+v", r)
			}
			if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != head {
				t.Fatalf("no records commit is made: %s, %s", got, head)
			}
		})
	}
}

// moveToShipped moves the lane's own intent as a close moves it, by hand.
func moveToShipped(t *testing.T, wt string) {
	t.Helper()
	landWrite(t, wt, shippedIntentRel, landRead(t, wt, plannedIntentRel))
	if err := os.Remove(filepath.Join(wt, filepath.FromSlash(plannedIntentRel))); err != nil {
		t.Fatal(err)
	}
}

// A closing lane's intent stands in shipped/ only as the close would have put it there:
// the judged head holds nothing back, and the record carries its impact.
func TestAnIntentInShippedIsOneTheCloseWouldHaveMoved(t *testing.T) {
	const secondBody = "---\nid: spc-2\nslug: second\nintent: itd-10\n---\n# second\n\n## Summary\n\nAnother record.\n"
	for name, atHead := range map[string]func(t *testing.T, wt string){
		"a hold at the head": func(t *testing.T, wt string) {
			t.Helper()
			landWrite(t, wt, plannedIntentRel, strings.Replace(landRead(t, wt, plannedIntentRel), "\nslug:", "\nheld: \"a reason\"\nslug:", 1))
		},
		"another open spec at the head": func(t *testing.T, wt string) {
			t.Helper()
			landWrite(t, wt, secondSpecRel, secondBody)
		},
		"no impact at the head or in the record": func(t *testing.T, wt string) {
			t.Helper()
			landWrite(t, wt, plannedIntentRel, strings.Replace(landRead(t, wt, plannedIntentRel), "impact: additive\n", "", 1))
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.validatedOnce(t)
			head := f.commitAtTheJudgedHead(t, l, func() { atHead(t, l.Worktree) })
			f.step(t)
			moveToShipped(t, l.Worktree)

			_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			if err == nil {
				t.Fatal("the records step went through")
			}
			if r := mustRefusal(t, err); !strings.Contains(r.Reason, "itd-10-alpha.md") {
				t.Fatalf("the refusal names the record: %+v", r)
			}
			if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != head {
				t.Fatalf("no records commit is made: %s, %s", got, head)
			}
			if files := f.repo.Git("-C", l.Worktree, "ls-tree", "-r", "--name-only", "HEAD"); strings.Contains(files, shippedIntentRel) {
				t.Fatalf("no shipped record reaches the branch:\n%s", files)
			}
		})
	}
}

// What the judged head itself holds against a close no worktree edit may settle, so the
// lane goes back to a person, naming it, with a way out that works.
func TestAFactTheJudgedHeadHoldsHandsTheLaneBack(t *testing.T) {
	const betaRel = ".abcd/development/intents/planned/itd-11-beta.md"
	beta := strings.Replace(strings.Replace(readyIntent("impact: additive\nbundle: pair\n", settledQuestions), "id: itd-10\n", "id: itd-11\n", 1), "slug: alpha\n", "slug: beta\n", 1)
	for name, c := range map[string]struct {
		atHead func(t *testing.T, wt string)
		names  string
	}{
		"a hold": {func(t *testing.T, wt string) {
			t.Helper()
			landWrite(t, wt, plannedIntentRel, strings.Replace(landRead(t, wt, plannedIntentRel), "\nslug:", "\nheld: \"a reason\"\nslug:", 1))
		}, "is held"},
		"a drifted back-link": {func(t *testing.T, wt string) {
			t.Helper()
			landWrite(t, wt, plannedIntentRel, strings.Replace(landRead(t, wt, plannedIntentRel), "spec_id: spc-1\n", "spec_id: spc-9\n", 1))
		}, "spec_id"},
		"a bundle member another open spec keeps": {func(t *testing.T, wt string) {
			t.Helper()
			landWrite(t, wt, specRel, strings.Replace(landRead(t, wt, specRel), "intent: itd-10\n", "intent: itd-10\nbundle: pair\nintents: [itd-10, itd-11]\n", 1))
			landWrite(t, wt, plannedIntentRel, strings.Replace(landRead(t, wt, plannedIntentRel), "\nslug:", "\nbundle: pair\nslug:", 1))
			landWrite(t, wt, betaRel, beta)
			landWrite(t, wt, secondSpecRel, "---\nid: spc-2\nslug: second\nintent: itd-11\n---\n# second\n\n## Summary\n\nAnother record.\n")
		}, "still realised by"},
		"the lane's spec id twice at the head": {func(t *testing.T, wt string) {
			t.Helper()
			landWrite(t, wt, ".abcd/development/specs/open/spc-0-twin.md", landRead(t, wt, specRel))
		}, "spc-0-twin.md"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.validatedOnce(t)
			f.commitAtTheJudgedHead(t, l, func() { c.atHead(t, l.Worktree) })
			f.step(t)

			res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			if err != nil {
				t.Fatalf("the step hands the lane back rather than refuse: %v", err)
			}
			if res.HandBack == nil || res.HandBack.Kind != HandBackStuck || !strings.Contains(res.HandBack.Reason, c.names) {
				t.Fatalf("the hand-back names what holds it: %+v", res.HandBack)
			}
			if !strings.Contains(res.HandBack.Home, "--discard "+l.ID) || !strings.Contains(res.HandBack.Home, "default branch") {
				t.Fatalf("the hand-back names the way out: %+v", res.HandBack)
			}
			if _, err := os.Stat(filepath.Join(l.Worktree, filepath.FromSlash(shippedIntentRel))); err == nil {
				t.Fatal("the intent shipped")
			}
			if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err == nil {
				t.Fatal("every later step refuses")
			}
			// The way the hand-back names: discard the lane, remove the run, build afresh.
			if _, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err != nil {
				t.Fatalf("the lane is discarded as the hand-back says: %v", err)
			}
			if err := os.RemoveAll(filepath.Join(f.repo.Root(), filepath.FromSlash(runRel(f.runID)))); err != nil {
				t.Fatal(err)
			}
			if _, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true}); err != nil {
				t.Fatalf("the intent is built afresh: %v", err)
			}
		})
	}
}

// A commit found on the judged head under the landing's subject meets the rule the
// landing's own records meet.
func TestACommitFoundOnTheJudgedHeadMeetsTheSameRule(t *testing.T) {
	const secondBody = "---\nid: spc-2\nslug: second\nintent: itd-10\n---\n# second\n\n## Summary\n\nAnother record.\n"
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	head := f.commitAtTheJudgedHead(t, l, func() { landWrite(t, l.Worktree, secondSpecRel, secondBody) })
	f.step(t)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	cur := currentLane(t, f.repo, f.runID)
	if err := os.Remove(filepath.Join(l.Worktree, filepath.FromSlash(secondSpecRel))); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l.Worktree, "add", "-A")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", landSubject(st, cur))
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{})
	if err == nil {
		t.Fatal("the commit is taken as the landing's own")
	}
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "spc-2-second.md") || !strings.Contains(r.Remedy, "restore the branch") {
		t.Fatalf("the refusal names the record and the way back: %+v", r)
	}
	if got := currentLane(t, f.repo, f.runID); got.HeadSHA != head || got.Landing.RecordsDone {
		t.Fatalf("the lane stays at the judged head: %+v", got.Landing)
	}
}
