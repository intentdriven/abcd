package loop

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const stepsOfOne = "1. The parser\n   - packages: internal/core/spec\n"

// issueRecordName is the file name of the fixture's issue in open/.
func (f *landFixture) issueRecordName(t *testing.T, l Lane) string {
	t.Helper()
	found, err := filepath.Glob(filepath.Join(l.Worktree, ".abcd", "work", "issues", "open", f.issue+"*"))
	if err != nil || len(found) != 1 {
		t.Fatalf("premise: the issue's record: %v %v", found, err)
	}
	return filepath.Base(found[0])
}

// moveInTree moves a record inside the lane's worktree, making its folder.
func (f *landFixture) moveInTree(t *testing.T, l Lane, from, to string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(l.Worktree, filepath.Dir(filepath.FromSlash(to))), 0o755); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l.Worktree, "mv", from, to)
}

// A record the judged head holds moved already is repointed like one the landing moves,
// so what that repoint writes lands.
func TestALandingCommitsItsRepointOfARecordTheHeadHoldsMoved(t *testing.T) {
	const guide = "docs/x.md"
	closedSpec := strings.Replace(specRel, "/open/", "/closed/", 1)
	for name, moved := range map[string]func(t *testing.T, f *landFixture, l Lane) (string, string){
		"an issue the lane resolves": func(t *testing.T, f *landFixture, l Lane) (string, string) {
			t.Helper()
			name := f.issueRecordName(t, l)
			return ".abcd/work/issues/open/" + name, ".abcd/work/issues/resolved/" + name
		},
		"the intent its close ships": func(*testing.T, *landFixture, Lane) (string, string) { return plannedIntentRel, shippedIntentRel },
		"the spec it closes":         func(*testing.T, *landFixture, Lane) (string, string) { return specRel, closedSpec },
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.validatedOnce(t)
			f.step(t) // prepare
			from, to := moved(t, f, l)
			f.commitAtTheJudgedHead(t, l, func() {
				f.moveInTree(t, l, from, to)
				landWrite(t, l.Worktree, guide, "See [it](../"+from+").\n")
			})
			l = f.wantsRecordsCommit(t)
			if got := committedFiles(f, l, l.HeadSHA); !slices.Contains(got, guide) {
				t.Fatalf("the commit holds the repoint: %v", got)
			}
			if body := landRead(t, l.Worktree, guide); !strings.Contains(body, "(../"+to+")") {
				t.Fatalf("the link follows the move:\n%s", body)
			}
		})
	}
}

// The moves the judged head holds admit what the repoint writes and nothing else: a
// further edit to the file is refused, and so is the same link to a record that is not
// the landing's own.
func TestARecordTheHeadHoldsMovedAdmitsOnlyItsRepoint(t *testing.T) {
	const guide = "docs/x.md"
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	f.step(t) // prepare
	name := f.issueRecordName(t, l)
	from, to := ".abcd/work/issues/open/"+name, ".abcd/work/issues/resolved/"+name
	const other = "iss-77-another.md"
	body := "See [it](../" + from + "), and [the other](../.abcd/work/issues/open/" + other + ").\n"
	f.commitAtTheJudgedHead(t, l, func() {
		f.moveInTree(t, l, from, to)
		landWrite(t, l.Worktree, guide, body)
	})
	landWrite(t, l.Worktree, guide, strings.Replace(body, "open/"+name, "resolved/"+name, 1)+"Another line.\n")
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, guide) || !strings.Contains(r.Reason, strayClaim) {
		t.Fatalf("a further edit is refused: %+v", r)
	}
	landWrite(t, l.Worktree, guide, strings.ReplaceAll(strings.Replace(body, "open/"+name, "resolved/"+name, 1), "open/"+other, "resolved/"+other))
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, guide) || !strings.Contains(r.Reason, strayClaim) {
		t.Fatalf("a link to another record is refused: %+v", r)
	}
}

// A nested checkout the landing would stage is judged whatever the repository's list of
// submodules says of it, at each point the landing reads what it commits.
func TestANestedCheckoutIsJudgedWhateverTheRepositoryIgnores(t *testing.T) {
	nested := func(t *testing.T, f *landFixture, wt string) {
		t.Helper()
		dir := filepath.Join(wt, "vendored")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		f.repo.Git("-C", dir, "init", "-q")
		f.repo.Git("-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "--allow-empty", "-m", "nested")
	}
	for name, setup := range map[string]func(t *testing.T, f *landFixture, l Lane){
		"in the worktree": func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			nested(t, f, l.Worktree)
		},
		"staged by the close": func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			beforeLandingStage = func(string) { nested(t, f, l.Worktree) }
			t.Cleanup(func() { beforeLandingStage = nil })
		},
		"in a commit found": func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			judged := l.HeadSHA
			f.step(t) // the landing's own commit
			nested(t, f, l.Worktree)
			f.repo.Git("-C", l.Worktree, "add", "-A")
			f.repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "--no-edit")
			f.rollBackToJudged(t, judged)
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixtureWith(t, map[string]string{
				".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
				".gitmodules": "[submodule \"vendored\"]\n\tpath = vendored\n\turl = ./vendored\n\tignore = all\n",
			})
			f.validated(t)
			f.step(t) // prepare
			l := currentLane(t, f.repo, f.runID)
			setup(t, f, l)
			if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "vendored") {
				t.Fatalf("the refusal names it: %+v", r)
			}
			if got := currentLane(t, f.repo, f.runID); got.Landing.RecordsDone {
				t.Fatalf("nothing holding it is taken as the landing's records: %+v", got.Landing)
			}
		})
	}
}

// A commit found for a bundle's close is asked for the moves that close makes: a member
// superseded, or one that left the bundle, is passed over as the close passes it over.
func TestACommitFoundForABundleIsAskedForWhatItsCloseMoves(t *testing.T) {
	beta := func(extra string) string {
		return strings.Replace(strings.Replace(readyIntent("impact: additive\n"+extra, settledQuestions), "id: itd-10\n", "id: itd-11\n", 1), "slug: alpha\n", "slug: beta\n", 1)
	}
	for name, c := range map[string]struct{ rel, body string }{
		"a superseded member":   {".abcd/development/intents/superseded/itd-11-beta.md", beta("bundle: pair\n")},
		"a member that left it": {".abcd/development/intents/planned/itd-11-beta.md", beta("bundle: another\n")},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixtureWith(t, map[string]string{
				".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
				c.rel: c.body,
			})
			l := f.validated(t)
			judged := f.commitAtTheJudgedHead(t, l, func() {
				landWrite(t, l.Worktree, specRel, strings.Replace(landRead(t, l.Worktree, specRel), "intent: itd-10\n", "intent: itd-10\nbundle: pair\nintents: [itd-10, itd-11]\n", 1))
				landWrite(t, l.Worktree, plannedIntentRel, strings.Replace(landRead(t, l.Worktree, plannedIntentRel), "\nslug:", "\nbundle: pair\nslug:", 1))
			})
			f.step(t) // prepare
			tip := f.wantsRecordsCommit(t).HeadSHA
			f.rollBackToJudged(t, judged)
			if got := f.wantsRecordsCommit(t); got.HeadSHA != tip {
				t.Fatalf("the landing's own commit is found: %s, %s", got.HeadSHA, tip)
			}
		})
	}
}

// A repoint that fails stops the call before its commit, and the next call repoints.
func TestAFailedRepointStopsTheCallBeforeTheCommit(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes through a read-only directory")
	}
	const guide = "docs/locked/guide.md"
	for name, c := range map[string]struct {
		link func(t *testing.T, f *landFixture, l Lane) string
		to   string
	}{
		"on a close": {func(*testing.T, *landFixture, Lane) string { return plannedIntentRel }, "/shipped/"},
		"on a resolution": {func(t *testing.T, f *landFixture, l Lane) string {
			t.Helper()
			return ".abcd/work/issues/open/" + f.issueRecordName(t, l)
		}, "/resolved/"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.validatedOnce(t)
			judged := f.commitAtTheJudgedHead(t, l, func() {
				if err := os.MkdirAll(filepath.Join(l.Worktree, "docs", "locked"), 0o755); err != nil {
					t.Fatal(err)
				}
				landWrite(t, l.Worktree, guide, "See [it](../../"+c.link(t, f, l)+").\n")
			})
			f.step(t) // prepare
			dir := filepath.Join(l.Worktree, "docs", "locked")
			if err := os.Chmod(dir, 0o555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
			r := f.wantsRefusedStep(t)
			if !strings.Contains(r.Reason, "repointed") || !strings.Contains(r.Remedy, "leave it uncommitted") {
				t.Fatalf("the refusal says the links could not be repointed: %+v", r)
			}
			if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != judged {
				t.Fatalf("nothing is committed: %s, %s", got, judged)
			}
			if err := os.Chmod(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			l = f.wantsRecordsCommit(t)
			if body := landRead(t, l.Worktree, guide); !strings.Contains(body, c.to) {
				t.Fatalf("the next call repoints it:\n%s", body)
			}
		})
	}
}

// A commit found on the judged head counts only if it reads as the landing's own: one
// parent, the trailers the landing writes, and the audit on the intent's record.
func TestACommitFoundMustReadAsTheLandingsOwn(t *testing.T) {
	closed := strings.Replace(specRel, "/open/", "/closed/", 1)
	reword := func(t *testing.T, f *landFixture, l Lane, change func(msg string) string) {
		t.Helper()
		msg := strings.TrimSpace(f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B"))
		changed := change(msg)
		if changed == msg {
			t.Fatalf("premise: the message did not change:\n%s", msg)
		}
		f.repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "-m", changed)
	}
	model := regexp.MustCompile(`(?m)^Assisted-by: .*$`)
	for name, c := range map[string]struct {
		undo func(t *testing.T, f *landFixture, l Lane)
		says string
	}{
		"a merge whose second parent carries a new file": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			own := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD")
			msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B")
			f.repo.Git("-C", l.Worktree, "reset", "-q", "--hard", l.HeadSHA)
			landWrite(t, l.Worktree, "docs/beside.md", "Beside.\n")
			f.repo.Git("-C", l.Worktree, "add", "-A")
			f.repo.Git("-C", l.Worktree, "commit", "-q", "-m", "beside")
			side := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD")
			merge := f.repo.Git("-C", l.Worktree, "commit-tree", own+"^{tree}", "-p", l.HeadSHA, "-p", side, "-m", msg)
			f.repo.Git("-C", l.Worktree, "reset", "-q", "--hard", merge)
		}, "not the landing's records commit"},
		"another model's disclosure": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			reword(t, f, l, func(m string) string { return model.ReplaceAllString(m, "Assisted-by: Claude:claude-other-1") })
		}, "lacks the landing's trailers"},
		"no disclosure": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			reword(t, f, l, func(m string) string { return model.ReplaceAllString(m, "") })
		}, "lacks the landing's trailers"},
		"a resolution dropped": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			reword(t, f, l, func(m string) string { return strings.ReplaceAll(m, "Resolves: "+f.issue+"\n", "") })
		}, "lacks the landing's trailers"},
		"a resolution added": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			reword(t, f, l, func(m string) string { return m + "\nResolves: iss-77" })
		}, "lacks the landing's trailers"},
		"a delivery dropped": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			reword(t, f, l, func(m string) string { return strings.ReplaceAll(m, "Delivers: itd-10\n", "") })
		}, "lacks the landing's trailers"},
		"a delivery in the body": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			reword(t, f, l, func(m string) string { return strings.Replace(m, "\n\n", "\n\nDelivers: itd-77\n\n", 1) })
		}, "lacks the landing's trailers"},
		"a disclosure folded over two lines": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			reword(t, f, l, func(m string) string {
				return model.ReplaceAllStringFunc(m, func(s string) string { return s + "\n  and more" })
			})
		}, "lacks the landing's trailers"},
		"the audit left off the record": {func(t *testing.T, _ *landFixture, l Lane) {
			t.Helper()
			landWrite(t, l.Worktree, shippedIntentRel, strings.ReplaceAll(landRead(t, l.Worktree, shippedIntentRel), "abcd-review: INGESTED", "abcd-review: OWED"))
		}, "lacks the audit"},
		"a record made executable": {func(t *testing.T, _ *landFixture, l Lane) {
			t.Helper()
			if err := os.Chmod(filepath.Join(l.Worktree, filepath.FromSlash(closed)), 0o755); err != nil {
				t.Fatal(err)
			}
		}, closed},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixtureWith(t, map[string]string{
				".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
				specRel: specWithSteps(stepsOfOne),
			})
			f.validatedOnce(t)
			f.step(t) // prepare
			l := currentLane(t, f.repo, f.runID)
			judged := l.HeadSHA
			f.step(t) // the landing's own commit
			c.undo(t, f, l)
			f.repo.Git("-C", l.Worktree, "add", "-A")
			f.repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "--no-edit")
			f.rollBackToJudged(t, judged)
			r := f.wantsRefusedStep(t)
			if !strings.Contains(r.Reason, c.says) || !strings.Contains(r.Remedy, "restore the branch") {
				t.Fatalf("the refusal says %q and the way back: %+v", c.says, r)
			}
			if got := currentLane(t, f.repo, f.runID); got.Landing.RecordsDone {
				t.Fatalf("the commit is not taken as the landing's records: %+v", got.Landing)
			}
		})
	}
}

// The landing's own commit is found after a kill whichever label it declares: abcd's on a
// commit of the mark alone, the model's on one carrying more.
func TestACommitFoundOfTheMarkAloneDeclaresAbcd(t *testing.T) {
	const steps = "1. The parser\n   - packages: internal/core/spec\n2. The loop\n   - packages: internal/core/implement\n"
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps(steps),
	})
	l := f.twoStepLandWithNoResolutions(t)
	judged := currentLane(t, f.repo, f.runID).HeadSHA
	f.step(t) // prepare
	tip := f.wantsRecordsCommit(t).HeadSHA
	f.rollBackToJudged(t, judged)
	if got := f.wantsRecordsCommit(t); got.HeadSHA != tip {
		t.Fatalf("the landing's own commit is found: %s, %s", got.HeadSHA, tip)
	}
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "-m", strings.ReplaceAll(msg, composedAssistedBy(), "Assisted-by: Claude:claude-test-5"))
	f.rollBackToJudged(t, judged)
	if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, "lacks the landing's trailers") {
		t.Fatalf("a model's label on a commit of the mark alone is not the landing's: %+v", r)
	}
}

// A record a store reads stays an ordinary file: a mode no write of the landing's gives
// it stops the call, and a mode the judged head already holds stands.
func TestARecordKeepsTheModeItsWritesGive(t *testing.T) {
	symlink := func(t *testing.T, abs string) func() {
		t.Helper()
		body, err := os.ReadFile(abs)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(abs); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../../../../AGENTS.md", abs); err != nil {
			t.Fatal(err)
		}
		return func() {
			if err := os.Remove(abs); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(abs, body, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	executable := func(t *testing.T, abs string) func() {
		t.Helper()
		if err := os.Chmod(abs, 0o755); err != nil {
			t.Fatal(err)
		}
		return func() {
			if err := os.Chmod(abs, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	for name, c := range map[string]struct {
		rel    func(t *testing.T, f *landFixture, l Lane) string
		change func(t *testing.T, abs string) func()
	}{
		"a spec made executable":    {func(*testing.T, *landFixture, Lane) string { return specRel }, executable},
		"an intent made executable": {func(*testing.T, *landFixture, Lane) string { return plannedIntentRel }, executable},
		"an issue made executable": {func(t *testing.T, f *landFixture, l Lane) string {
			t.Helper()
			return ".abcd/work/issues/open/" + f.issueRecordName(t, l)
		}, executable},
		"an issue made a symlink": {func(t *testing.T, f *landFixture, l Lane) string {
			t.Helper()
			return ".abcd/work/issues/open/" + f.issueRecordName(t, l)
		}, symlink},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.validatedOnce(t)
			judged := l.HeadSHA
			f.step(t) // prepare
			rel := c.rel(t, f, l)
			restore := c.change(t, filepath.Join(l.Worktree, filepath.FromSlash(rel)))
			if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, rel) {
				t.Fatalf("the refusal names the record: %+v", r)
			}
			if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != judged {
				t.Fatalf("nothing is committed: %s, %s", got, judged)
			}
			restore()
			f.wantsRecordsCommit(t)
		})
	}
	t.Run("a mode the judged head holds", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		l := f.validatedOnce(t)
		f.step(t) // prepare
		f.commitAtTheJudgedHead(t, l, func() {
			if err := os.Chmod(filepath.Join(l.Worktree, filepath.FromSlash(specRel)), 0o755); err != nil {
				t.Fatal(err)
			}
		})
		f.wantsRecordsCommit(t)
	})
}

// laneWithMarkdownBinary commits a markdown edit under attributes that call markdown
// binary, and makes it the head of the lane's latest round.
func (f *landFixture) laneWithMarkdownBinary(t *testing.T, l Lane) string {
	t.Helper()
	const line = "A line the reader must see."
	head := f.commitAtTheJudgedHead(t, l, func() {
		landWrite(t, l.Worktree, ".gitattributes", "*.md -diff\n")
		landWrite(t, l.Worktree, plannedIntentRel, landRead(t, l.Worktree, plannedIntentRel)+line+"\n")
	})
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		cur := &st.Lanes[st.current()]
		cur.Validation[len(cur.Validation)-1].HeadSHA = head
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	return line
}

// A reviewer's request shows the text of what changed, whatever attributes the lane's
// worktree holds.
func TestAReviewRequestShowsTheTextWhateverTheAttributesSay(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	line := f.laneWithMarkdownBinary(t, l)
	cur := currentLane(t, f.repo, f.runID)
	v := cur.Validation[len(cur.Validation)-1].Validators[0]
	req, err := reviewRequest(f.repo.Root(), f.runID, cur.ID, Await{Role: v.Role, Receipt: v.Return}, func(string) (string, error) { return "the agent", nil })
	if err != nil {
		t.Fatal(err)
	}
	_, diff, ok := strings.Cut(req, "## The diff\n")
	if !ok {
		t.Fatalf("the request has its diff section:\n%s", req)
	}
	if !strings.Contains(diff, "+"+line) {
		t.Fatalf("the diff section shows the text:\n%s", diff)
	}
}

// Each command or range a validator's brief hands it reads git's own text diff, whatever
// attributes or drivers the lane's worktree holds, and the command works as written.
func TestAValidatorBriefNamesGitsOwnTextDiff(t *testing.T) {
	const flags = "diff --text --no-ext-diff --no-textconv "
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	line := f.laneWithMarkdownBinary(t, l)
	cur := currentLane(t, f.repo, f.runID)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	var v ValidatorRun
	r := cur.Validation[len(cur.Validation)-1]
	v = r.Validators[0]
	if err := writeValidatorBrief(Context{RepoRoot: f.repo.Root(), State: st}, cur, r, &v, nil); err != nil {
		t.Fatal(err)
	}
	brief, err := os.ReadFile(abs(f.repo.Root(), v.Brief))
	if err != nil {
		t.Fatal(err)
	}
	if m := regexp.MustCompile("`git -C (\\S+) (" + flags + "\\S+)`").FindStringSubmatch(string(brief)); m == nil {
		t.Errorf("the brief names the command:\n%s", brief)
	} else if got := f.repo.Git(append([]string{"-C", m[1]}, strings.Fields(m[2])...)...); !strings.Contains(got, "+"+line) {
		t.Errorf("the command shows the text:\n%s", got)
	}
	if got := recordsRoundScope("abc", "def"); !strings.Contains(got, "git "+flags+"abc..def") {
		t.Errorf("the records round names the command:\n%s", got)
	}
	_, delivered, err := composeAudit(Context{RepoRoot: f.repo.Root(), State: st}, cur)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(delivered, "git "+flags) {
		t.Errorf("the audit's delivery names the command:\n%s", delivered)
	}
}
