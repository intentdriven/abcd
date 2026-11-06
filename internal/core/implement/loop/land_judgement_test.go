package loop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
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

// nestedCheckout makes a checkout with a commit of its own inside the lane's worktree.
func (f *landFixture) nestedCheckout(t *testing.T, wt string) {
	t.Helper()
	dir := filepath.Join(wt, "vendored")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", dir, "init", "-q")
	f.repo.Git("-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "--allow-empty", "-m", "nested")
}

// submoduleSettings are the two ways a repository asks git's diff to pass over a gitlink:
// each sets up a fixture so asked.
var submoduleSettings = map[string]func(t *testing.T) *landFixture{
	"its list of submodules": func(t *testing.T) *landFixture {
		t.Helper()
		return newLandFixtureWith(t, map[string]string{
			".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
			".gitmodules": "[submodule \"vendored\"]\n\tpath = vendored\n\turl = ./vendored\n\tignore = all\n",
		})
	},
	"its configuration": func(t *testing.T) *landFixture {
		t.Helper()
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.repo.Git("config", "diff.ignoreSubmodules", "all")
		return f
	},
}

// A nested checkout the landing would stage is judged whatever the repository asks git's
// diff to pass over, at each point the landing reads what it commits.
func TestANestedCheckoutIsJudgedWhateverTheRepositoryIgnores(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T, f *landFixture, l Lane){
		"in the worktree": func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			f.nestedCheckout(t, l.Worktree)
		},
		"staged by the close": func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			beforeLandingStage = func(string) { f.nestedCheckout(t, l.Worktree) }
			t.Cleanup(func() { beforeLandingStage = nil })
		},
		"in a commit found": func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			judged := l.HeadSHA
			f.step(t) // the landing's own commit
			f.nestedCheckout(t, l.Worktree)
			f.repo.Git("-C", l.Worktree, "add", "-A")
			f.repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "--no-edit")
			f.rollBackToJudged(t, judged)
		},
	} {
		for how, fixture := range submoduleSettings {
			t.Run(name+", passed over by "+how, func(t *testing.T) {
				f := fixture(t)
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
}

// standIn makes the object named by the replacement stand in for the one named first, as
// the repository's own replace refs do.
func (f *landFixture) standIn(t *testing.T, wt, object, replacement string) {
	t.Helper()
	f.repo.Git("-C", wt, "replace", object, replacement)
}

// A landing reads what it commits as stored, whatever the repository's replace refs stand
// in for those objects, and leaves those refs where they are.
func TestALandingReadsWhatItCommitsAsStored(t *testing.T) {
	const payload = "Not the landing's.\n"
	for name, standIn := range map[string]func(t *testing.T, f *landFixture, l Lane) string{
		"another blob for the file's": func(t *testing.T, f *landFixture, l Lane) string {
			t.Helper()
			written := f.repo.Git("-C", l.Worktree, "hash-object", "-w", "--", "AGENTS.md")
			f.standIn(t, l.Worktree, written, f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD:AGENTS.md"))
			return written
		},
		"another commit for the judged head": func(t *testing.T, f *landFixture, l Lane) string {
			t.Helper()
			f.repo.Git("-C", l.Worktree, "add", "AGENTS.md")
			tree := f.repo.Git("-C", l.Worktree, "write-tree")
			f.repo.Git("-C", l.Worktree, "reset", "-q")
			other := f.repo.Git("-C", l.Worktree, "commit-tree", tree, "-p", l.HeadSHA+"^", "-m", "the head")
			f.standIn(t, l.Worktree, l.HeadSHA, other)
			return l.HeadSHA
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			f.validatedOnce(t)
			f.step(t) // prepare
			l := currentLane(t, f.repo, f.runID)
			judged := l.HeadSHA
			landWrite(t, l.Worktree, "AGENTS.md", landRead(t, l.Worktree, "AGENTS.md")+payload)
			object := standIn(t, f, l)
			r := f.wantsRefusedStep(t)
			if !strings.Contains(r.Reason, "AGENTS.md") || !strings.Contains(r.Reason, strayClaim) {
				t.Fatalf("the refusal names the file: %+v", r)
			}
			if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != judged {
				t.Fatalf("nothing is committed: %s, %s", got, judged)
			}
			if got := f.repo.Git("-C", l.Worktree, "replace", "-l"); !strings.Contains(got, object) {
				t.Fatalf("the repository's replace ref stays: %q", got)
			}
		})
	}
}

// A commit found on the judged head is read as stored: what stands in for it, its blobs or
// its parents says nothing of what it holds.
func TestACommitFoundIsReadAsStored(t *testing.T) {
	// merged stands a merge of the judged head and a side commit in at the tip, its tree
	// the landing's own.
	merged := func(t *testing.T, f *landFixture, l Lane) string {
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
		return merge
	}
	for name, c := range map[string]struct {
		arrange func(t *testing.T, f *landFixture, l Lane)
		says    string
	}{
		"parents a replace ref gives it": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			merge := merged(t, f, l)
			f.repo.Git("-C", l.Worktree, "replace", "--graft", merge, l.HeadSHA)
		}, "not the landing's records commit"},
		"parents a graft file gives it": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			merge := merged(t, f, l)
			common := f.repo.Git("-C", l.Worktree, "rev-parse", "--path-format=absolute", "--git-common-dir")
			if err := os.MkdirAll(filepath.Join(common, "info"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(common, "info", "grafts"), []byte(merge+" "+l.HeadSHA+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "not the landing's records commit"},
		"the landing's own commit standing in for it": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			own := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD")
			landWrite(t, l.Worktree, "docs/beside.md", "Beside.\n")
			f.repo.Git("-C", l.Worktree, "add", "-A")
			f.repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "--no-edit")
			f.standIn(t, l.Worktree, f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"), own)
		}, "docs/beside.md"},
		"the head's blob standing in for what it holds": {func(t *testing.T, f *landFixture, l Lane) {
			t.Helper()
			landWrite(t, l.Worktree, "AGENTS.md", landRead(t, l.Worktree, "AGENTS.md")+"Not the landing's.\n")
			f.repo.Git("-C", l.Worktree, "add", "-A")
			f.repo.Git("-C", l.Worktree, "commit", "-q", "--amend", "--no-edit")
			f.standIn(t, l.Worktree, f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD:AGENTS.md"), f.repo.Git("-C", l.Worktree, "rev-parse", l.HeadSHA+":AGENTS.md"))
		}, "AGENTS.md"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			f.validatedOnce(t)
			f.step(t) // prepare
			l := currentLane(t, f.repo, f.runID)
			f.step(t) // the landing's own commit
			c.arrange(t, f, l)
			f.rollBackToJudged(t, l.HeadSHA)
			if r := f.wantsRefusedStep(t); !strings.Contains(r.Reason, c.says) {
				t.Fatalf("the refusal says %q: %+v", c.says, r)
			}
			if got := currentLane(t, f.repo, f.runID); got.Landing.RecordsDone {
				t.Fatalf("the commit is not taken as the landing's records: %+v", got.Landing)
			}
		})
	}
}

// A store's records and a mark's blobs are read as stored.
func TestRecordsAndAMarkAreReadAsStored(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	wt, head := l.Worktree, l.HeadSHA
	stored := f.repo.Git("-C", wt, "rev-parse", head+":"+specRel)
	body := landRead(t, wt, specRel)
	const mark = "abc123"
	landWrite(t, wt, ".mark", body+"- landed: "+mark+"\n")
	marked := f.repo.Git("-C", wt, "hash-object", "-w", "--", ".mark")
	landWrite(t, wt, ".mark", body+"- landed: "+mark+"\nNot the mark.\n")
	more := f.repo.Git("-C", wt, "hash-object", "-w", "--", ".mark")
	if err := os.Remove(filepath.Join(wt, ".mark")); err != nil {
		t.Fatal(err)
	}
	f.standIn(t, wt, more, marked)
	if only, err := addsOnlyTheMark(wt, stored, more, mark); err != nil || only {
		t.Errorf("a blob holding more than the mark reads as the mark alone: %v %v", only, err)
	}
	f.standIn(t, wt, stored, more)
	records, err := treeRecords(baseTree{root: wt, sha: head}, specCloseStore)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range records {
		if r.rel == specRel && r.content != body {
			t.Errorf("the record reads as what stands in for it:\n%s", r.content)
		}
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
		"the audit's words changed": {func(t *testing.T, _ *landFixture, l Lane) {
			t.Helper()
			landWrite(t, l.Worktree, shippedIntentRel, strings.Replace(landRead(t, l.Worktree, shippedIntentRel), "Provenance:", "Provenance, as read:", 1))
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

// atRoundHead commits what edit changes and makes it the head of the lane's latest round.
func (f *landFixture) atRoundHead(t *testing.T, l Lane, edit func()) string {
	t.Helper()
	head := f.commitAtTheJudgedHead(t, l, edit)
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		cur := &st.Lanes[st.current()]
		cur.Validation[len(cur.Validation)-1].HeadSHA = head
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	return head
}

// requestOf composes the request a reviewer route is sent for the lane's latest round.
func (f *landFixture) requestOf(t *testing.T) (string, error) {
	t.Helper()
	cur := currentLane(t, f.repo, f.runID)
	v := cur.Validation[len(cur.Validation)-1].Validators[0]
	return reviewRequest(f.repo.Root(), f.runID, cur.ID, Await{Role: v.Role, Receipt: v.Return}, func(string) (string, error) { return "the agent", nil })
}

// reviewOf is the diff section of the request a reviewer route is sent.
func (f *landFixture) reviewOf(t *testing.T) string {
	t.Helper()
	req, err := f.requestOf(t)
	if err != nil {
		t.Fatal(err)
	}
	_, diff, ok := strings.Cut(req, "## The diff\n")
	if !ok {
		t.Fatalf("the request has its diff section:\n%s", req)
	}
	diff, _, _ = strings.Cut(diff, "## What you answer")
	return diff
}

// A reviewer's request shows the text of what changed as the objects store it, whatever
// attributes the lane's worktree holds, replace refs stand in or the repository asks git's
// diff to pass over.
func TestAReviewRequestShowsTheTextWhateverTheAttributesSay(t *testing.T) {
	const line = "A line the reader must see."
	edit := func(t *testing.T, wt string) {
		t.Helper()
		landWrite(t, wt, plannedIntentRel, landRead(t, wt, plannedIntentRel)+line+"\n")
	}
	t.Run("markdown called binary by the attributes", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		l := f.validatedOnce(t)
		f.atRoundHead(t, l, func() {
			landWrite(t, l.Worktree, ".gitattributes", "*.md -diff\n")
			edit(t, l.Worktree)
		})
		if got := f.reviewOf(t); !strings.Contains(got, "+"+line) {
			t.Fatalf("the diff section shows the text:\n%s", got)
		}
	})
	t.Run("a blob standing in for the lane's own", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		l := f.validatedOnce(t)
		was := l.HeadSHA
		head := f.atRoundHead(t, l, func() { edit(t, l.Worktree) })
		f.standIn(t, l.Worktree, f.repo.Git("-C", l.Worktree, "rev-parse", head+":"+plannedIntentRel), f.repo.Git("-C", l.Worktree, "rev-parse", was+":"+plannedIntentRel))
		if got := f.reviewOf(t); !strings.Contains(got, "+"+line) {
			t.Fatalf("the diff section shows the text:\n%s", got)
		}
	})
	for how, fixture := range submoduleSettings {
		t.Run("a nested checkout passed over by "+how, func(t *testing.T) {
			f := fixture(t)
			l := f.validated(t)
			f.atRoundHead(t, l, func() { f.nestedCheckout(t, l.Worktree) })
			if got := f.reviewOf(t); !strings.Contains(got, "Subproject commit") {
				t.Fatalf("the diff section shows the gitlink:\n%s", got)
			}
		})
	}
}

// A file with no NUL byte reaches a reviewer as git's text, whatever other bytes it holds.
func TestAReviewRequestShowsAFileWithNoNulByte(t *testing.T) {
	const payload = "var Payload = 1"
	for name, text := range map[string]string{
		"an escape byte":   "// \x1b[2K\n" + payload + "\n",
		"a bell":           "// a\x07b\n" + payload + "\n",
		"a lone CR":        "// a\rb\n" + payload + "\n",
		"not UTF-8":        "// \xff\xfe\n" + payload + "\n",
		"a line separator": "// a\u2028b\n" + payload + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.validatedOnce(t)
			f.atRoundHead(t, l, func() { landWrite(t, l.Worktree, "cmd/x.go", text) })
			got := f.reviewOf(t)
			if !strings.Contains(got, "+"+payload) {
				t.Fatalf("the diff section shows the file:\n%q", got)
			}
			if strings.Contains(got, "not shown") {
				t.Fatalf("the file is not listed as unread:\n%q", got)
			}
		})
	}
	t.Run("a small change to a file past what is read for its kind", func(t *testing.T) {
		const big = "vendor/deps.lock"
		f := newLandFixtureWith(t, map[string]string{
			".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
			big: strings.Repeat("a locked dependency at a version\n", (maxReaderBlobBytes+1<<20)/33),
		})
		l := f.validated(t)
		f.atRoundHead(t, l, func() { landWrite(t, l.Worktree, big, landRead(t, l.Worktree, big)+payload+"\n") })
		got := f.reviewOf(t)
		if !strings.Contains(got, "+"+payload) || strings.Contains(got, "not shown") {
			t.Fatalf("the hunk is shown:\n%.2000q", got)
		}
	})
}

// A file with a NUL byte is shown with its unprintable bytes escaped under a bound, so text
// beside the NUL stays readable. Past the bound it is one line naming it and it is unread;
// past what is read for its kind it stays in git's text, where a large one overflows the
// request. Nothing raw reaches the reader, and text taking its place is shown.
func TestAReviewRequestBoundsAFileWithANulByte(t *testing.T) {
	const line = "A line the reader must see."
	binary := strings.Repeat("\x00\xff\x01binary\n", 80*1024) // past the request's cap
	nul := func(t *testing.T, got string) {
		t.Helper()
		if strings.Contains(got, "\x00") || strings.Contains(got, "\xff") {
			t.Fatalf("no raw byte of the binary reaches the reader:\n%.2000q", got)
		}
	}
	t.Run("a script under the bound", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		l := f.validatedOnce(t)
		f.atRoundHead(t, l, func() {
			landWrite(t, l.Worktree, "run.sh", "#!/bin/sh\necho "+line+"\n\x00\xff\n")
		})
		got := f.reviewOf(t)
		nul(t, got)
		for _, want := range []string{"+echo " + line, `\x00`, `\xff`, "run.sh"} {
			if !strings.Contains(got, want) {
				t.Errorf("the request shows %q:\n%q", want, got)
			}
		}
		if strings.Contains(got, "not shown") {
			t.Errorf("a file shown is not listed as unread:\n%q", got)
		}
	})
	t.Run("past the bound, beside text", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		l := f.validatedOnce(t)
		head := f.atRoundHead(t, l, func() {
			landWrite(t, l.Worktree, "assets/logo.bin", binary)
			landWrite(t, l.Worktree, plannedIntentRel, landRead(t, l.Worktree, plannedIntentRel)+line+"\n")
		})
		got := f.reviewOf(t)
		nul(t, got)
		if !strings.Contains(got, "+"+line) {
			t.Fatalf("the text beside it is shown:\n%.2000s", got)
		}
		blob := f.repo.Git("-C", l.Worktree, "rev-parse", head+":assets/logo.bin")
		for _, want := range []string{"not shown", "assets/logo.bin", "100644", strconv.Itoa(len(binary)), blob} {
			if !strings.Contains(got, want) {
				t.Errorf("the line names %q:\n%.2000s", want, got)
			}
		}
	})
	t.Run("past what is read for its kind", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		l := f.validatedOnce(t)
		f.atRoundHead(t, l, func() {
			landWrite(t, l.Worktree, "assets/huge.bin", strings.Repeat(binary, 7))
		})
		if _, err := f.requestOf(t); err == nil || !strings.Contains(err.Error(), "cap") {
			t.Fatalf("the request cannot be composed, and the review falls back: %v", err)
		}
	})
	t.Run("text in the place of one under the bound", func(t *testing.T) {
		f := newLandFixtureWith(t, map[string]string{
			".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
			"assets/note.md": "\x00\xff\n",
		})
		l := f.validated(t)
		f.atRoundHead(t, l, func() { landWrite(t, l.Worktree, "assets/note.md", line+"\n") })
		got := f.reviewOf(t)
		nul(t, got)
		if !strings.Contains(got, "+"+line) || !strings.Contains(got, `-\x00\xff`) {
			t.Fatalf("the change is shown, escaped:\n%q", got)
		}
	})
	t.Run("a late NUL past the bound", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		l := f.validatedOnce(t)
		f.atRoundHead(t, l, func() {
			landWrite(t, l.Worktree, "run.sh", "#!/bin/sh\necho "+line+"\n"+strings.Repeat("# a comment\n", 80<<10/12)+"\x00\n")
		})
		got := f.reviewOf(t)
		nul(t, got)
		if !strings.Contains(got, "+echo "+line) || strings.Contains(got, "not shown") {
			t.Fatalf("the file is shown:\n%.2000q", got)
		}
	})
	for name, c := range map[string]struct {
		edit func(t *testing.T, f *landFixture, l Lane)
		want string
	}{
		"text in the place of one past the bound": {func(t *testing.T, _ *landFixture, l Lane) {
			landWrite(t, l.Worktree, "vendored", line+"\n")
		}, "+" + line},
		"a nested checkout in the place of one past the bound": {func(t *testing.T, f *landFixture, l Lane) {
			if err := os.Remove(filepath.Join(l.Worktree, "vendored")); err != nil {
				t.Fatal(err)
			}
			f.nestedCheckout(t, l.Worktree)
		}, "+Subproject commit "},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixtureWith(t, map[string]string{
				".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
				"vendored": binary,
			})
			l := f.validated(t)
			f.atRoundHead(t, l, func() { c.edit(t, f, l) })
			got := f.reviewOf(t)
			nul(t, got)
			if !strings.Contains(got, c.want) || !strings.Contains(got, "not shown") {
				t.Fatalf("the new side is shown beside the line for what it replaced:\n%.2000s", got)
			}
		})
	}
}

// A file the request lists says both its modes, so a change of mode or of kind reads as one.
func TestAReviewRequestNamesBothModesOfAFileNotShown(t *testing.T) {
	binary := strings.Repeat("\x00\xff\x01binary\n", 80*1024)
	for name, c := range map[string]struct {
		edit func(t *testing.T, wt string)
		now  string
	}{
		"a link in its place": {func(t *testing.T, wt string) {
			t.Helper()
			if err := os.Remove(filepath.Join(wt, "vendored")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("elsewhere", filepath.Join(wt, "vendored")); err != nil {
				t.Fatal(err)
			}
		}, "120000"},
		"its mode changed alone": {func(t *testing.T, wt string) {
			t.Helper()
			if err := os.Chmod(filepath.Join(wt, "vendored"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, "100755"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newLandFixtureWith(t, map[string]string{
				".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
				"vendored": binary,
			})
			l := f.validated(t)
			f.atRoundHead(t, l, func() { c.edit(t, l.Worktree) })
			got := f.reviewOf(t)
			i := strings.Index(got, "# binary, not shown")
			if i < 0 {
				t.Fatalf("the file is listed:\n%.2000s", got)
			}
			listed, _, _ := strings.Cut(got[i:], "\n")
			if !strings.Contains(listed, "100644") || !strings.Contains(listed, c.now) {
				t.Fatalf("the line names both modes: %s", listed)
			}
		})
	}
}

// Every changed file reaches the request or the request is not composed: a path the list of
// shown files lacks, however git's pathspecs came to leave it out, sends the review to the
// host reader.
func TestAReviewRequestShowsEveryFileBesideOneItLists(t *testing.T) {
	const line = "A line the reader must see."
	binary := strings.Repeat("\x00\xff\x01binary\n", 80*1024)
	wants := func(t *testing.T, f *landFixture) {
		t.Helper()
		got, err := f.requestOf(t)
		if err != nil {
			if !strings.Contains(err.Error(), "the lane's diff") {
				t.Fatalf("the request cannot be composed, for its diff: %v", err)
			}
			return
		}
		if !strings.Contains(got, line) {
			t.Fatalf("the line is shown, or the review falls back:\n%.4000s", got)
		}
	}
	t.Run("a directory of the same name", func(t *testing.T) {
		f := newLandFixtureWith(t, map[string]string{
			".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
			"assets/big.bin": binary,
		})
		l := f.validated(t)
		f.atRoundHead(t, l, func() {
			if err := os.Remove(filepath.Join(l.Worktree, "assets/big.bin")); err != nil {
				t.Fatal(err)
			}
			landWrite(t, l.Worktree, "assets/big.bin/run.sh", line+"\n")
		})
		wants(t, f)
	})
	t.Run("a name that composes to another's", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		l := f.validatedOnce(t)
		object := func(text string) string {
			t.Helper()
			p := filepath.Join(t.TempDir(), "blob")
			if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
				t.Fatal(err)
			}
			return f.repo.Git("-C", l.Worktree, "hash-object", "-w", "--", p)
		}
		add := func(name, text string) {
			t.Helper()
			f.repo.Git("-C", l.Worktree, "-c", "core.precomposeunicode=false", "update-index", "--add", "--cacheinfo", "100644,"+object(text)+","+name)
		}
		add("e\u0301.bin", binary[:70<<10])
		add("\u00e9.bin", line+"\n")
		tree := f.repo.Git("-C", l.Worktree, "write-tree")
		head := f.repo.Git("-C", l.Worktree, "commit-tree", tree, "-p", l.HeadSHA, "-m", "two names")
		if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
			cur := &st.Lanes[st.current()]
			cur.HeadSHA, cur.Validation[len(cur.Validation)-1].HeadSHA = head, head
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
		f.repo.Git("config", "core.precomposeunicode", "true")
		wants(t, f)
	})
}

// A run of files with a NUL byte stops at the request's cap, not after the last of them.
func TestAReviewRequestStopsAtItsCap(t *testing.T) {
	const files = 12
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	f.atRoundHead(t, l, func() {
		for i := range files {
			landWrite(t, l.Worktree, fmt.Sprintf("assets/%02d.bin", i), strings.Repeat("\x00\xff\x01\x02", 60<<10/4))
		}
	})
	calls := gittest.SlowPipeGit(t, "diff", 0)
	if _, err := f.requestOf(t); err == nil || !strings.Contains(err.Error(), "cap") {
		t.Fatalf("the request cannot be composed: %v", err)
	}
	if n := calls(); n >= files {
		t.Fatalf("git diff ran %d times for a request full after a few", n)
	}
}

// What the request holds is what the review reads, so it says what a file it lists as not
// shown means for a verdict.
func TestAReviewRequestSaysAFileNotShownIsUnread(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validatedOnce(t)
	got, err := f.requestOf(t)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"listed as not shown is unread", "naming it", "its kind fits its path and the lane", "finding"} {
		if !strings.Contains(strings.Join(strings.Fields(got), " "), want) {
			t.Errorf("the request says %q:\n%s", want, got)
		}
	}
}

// Each command or range a validator's brief hands it reads git's own text diff of the
// objects as stored, and the command works as written.
func TestAValidatorBriefNamesGitsOwnTextDiff(t *testing.T) {
	const flags = "--no-replace-objects diff --text --ignore-submodules=none --no-ext-diff --no-textconv "
	const line = "A line the reader must see."
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validatedOnce(t)
	was := l.HeadSHA
	head := f.atRoundHead(t, l, func() {
		landWrite(t, l.Worktree, ".gitattributes", "*.md -diff\n")
		landWrite(t, l.Worktree, "cmd/nul.bin", "\x00\n")
		landWrite(t, l.Worktree, plannedIntentRel, landRead(t, l.Worktree, plannedIntentRel)+line+"\n")
	})
	f.standIn(t, l.Worktree, f.repo.Git("-C", l.Worktree, "rev-parse", head+":"+plannedIntentRel), f.repo.Git("-C", l.Worktree, "rev-parse", was+":"+plannedIntentRel))
	cur := currentLane(t, f.repo, f.runID)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	r := cur.Validation[len(cur.Validation)-1]
	v := r.Validators[0]
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
	for _, want := range []string{"is unread", "naming it", "its kind fits its path and the lane", "finding"} {
		if !strings.Contains(strings.Join(strings.Fields(string(brief)), " "), want) {
			t.Errorf("the brief says %q of a file left out:\n%s", want, brief)
		}
	}
	if strings.Contains(string(brief), "ls-tree") {
		t.Errorf("the brief asks of a file what its kind is, not its size:\n%s", brief)
	}
	if m := regexp.MustCompile("`(git -C \\S+ --no-replace-objects cat-file blob \\S+:<path> [^`]*)`").FindStringSubmatch(string(brief)); m == nil {
		t.Errorf("the brief names the test of a file's kind:\n%s", brief)
	} else {
		for path, want := range map[string]string{"cmd/nul.bin": "1", plannedIntentRel: "0"} {
			out, err := exec.Command("sh", "-c", strings.ReplaceAll(m[1], "<path>", path)).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != want {
				t.Errorf("the test of %s counts %s NUL bytes: %q %v", path, want, out, err)
			}
		}
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
