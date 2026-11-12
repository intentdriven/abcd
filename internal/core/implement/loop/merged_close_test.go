package loop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/runstop"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// A build of an intent whose lane's pull request has merged closes that lane and
// goes on in a new one (iss-2611101317510356). The fixture is a two-step spec
// whose first lane is armed at its landing, its pull request open.
func mergedLaneFixture(t *testing.T) (*landFixture, Lane) {
	t.Helper()
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		specRel: specWithSteps("1. The parser\n   - packages: internal/core/spec\n" +
			"2. The loop\n   - packages: internal/core/implement\n"),
	})
	l := f.walked(t, "armedFirstOfTwoSteps", func(t *testing.T) Lane {
		f.twoStepLandWithNoResolutions(t)
		f.step(t) // prepare
		f.step(t) // records
		l := currentLane(t, f.repo, f.runID)
		preflighted(t, l, l.HeadSHA)
		f.step(t) // push
		f.step(t) // pull request
		f.step(t) // arm
		return currentLane(t, f.repo, f.runID)
	})
	if l.ID != "lane-1" || l.Stage != StageLand || l.PR != 7 || l.Landing == nil || l.Landing.Merge == "" || !gitutil.IsFullSHA(l.Landing.Pushed) {
		t.Fatalf("premise: the first lane is armed at its landing: %+v %+v", l, l.Landing)
	}
	return f, l
}

// mergeCommit merges head into the remote's default branch with a merge commit,
// as the forge's merge button and this repository's merge queue do, fetches it,
// and tells the stub forge the pull request merged. It returns the merge commit.
func (f *landFixture) mergeCommit(t *testing.T, head string) string {
	t.Helper()
	return f.mergeOnto(t, head, true)
}

// squashMerge lands head's tree on the remote's default branch as one new
// commit, as a squash merge does, and fetches it.
func (f *landFixture) squashMerge(t *testing.T, head string) string {
	t.Helper()
	return f.mergeOnto(t, head, false)
}

func (f *landFixture) mergeOnto(t *testing.T, head string, merge bool) string {
	t.Helper()
	f.repo.Git("fetch", "-q", "origin")
	main := strings.TrimSpace(f.repo.Git("rev-parse", "refs/remotes/origin/main"))
	tree := strings.TrimSpace(f.repo.Git("rev-parse", head+"^{tree}"))
	args := []string{"commit-tree", tree, "-p", main}
	if merge {
		args = append(args, "-p", head)
	}
	m := strings.TrimSpace(f.repo.Git(append(args, "-m", "Merge pull request #7")...))
	f.repo.Git("push", "-q", "origin", m+":refs/heads/main")
	f.repo.Git("fetch", "-q", "origin")
	if err := os.WriteFile(filepath.Join(f.gh, "state"), []byte("MERGED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return m
}

// intentShippedOnOrigin moves the fixture's intent to shipped/ on origin's
// default branch, from another copy, unless it is there already, and fetches.
func (f *landFixture) intentShippedOnOrigin(t *testing.T) {
	t.Helper()
	q := otherCopy(t, f.bare, patEmail)
	found, err := filepath.Glob(filepath.Join(q, ".abcd", "development", "intents", "*", "itd-10*"))
	if err != nil || len(found) != 1 {
		t.Fatalf("the fixture's intent: %v, %v", found, err)
	}
	shipped := filepath.Join(q, ".abcd", "development", "intents", "shipped")
	if filepath.Dir(found[0]) != shipped {
		if err := os.MkdirAll(shipped, 0o755); err != nil {
			t.Fatal(err)
		}
		gitIn(t, q, "mv", found[0], filepath.Join(shipped, filepath.Base(found[0])))
		gitIn(t, q, "commit", "-q", "-m", "ship")
		gitIn(t, q, "push", "-q", "origin", "HEAD:main")
	}
	f.repo.Git("fetch", "-q", "origin")
}

// goneHereToo deletes the lane's branch on origin and here, with its worktree,
// as a person tidying up after a merge does.
func (f *landFixture) goneHereToo(t *testing.T, l Lane) {
	t.Helper()
	f.repo.Git("push", "-q", "origin", "--delete", l.Branch)
	f.repo.Git("fetch", "-q", "--prune", "origin")
	f.repo.Git("worktree", "remove", "--force", l.Worktree)
	f.repo.Git("branch", "-D", l.Branch)
}

// leftAsItStands fails unless the lane is still parked at its landing with no
// merge recorded, its worktree and branch where they were.
func (f *landFixture) leftAsItStands(t *testing.T, l Lane) {
	t.Helper()
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(st.Lanes, func(x Lane) bool { return x.ID == l.ID })
	if i < 0 || st.Lanes[i].Stage != StageLand || st.Lanes[i].Landing == nil || st.Lanes[i].Landing.Merged != "" {
		t.Fatalf("the lane is not closed: %+v", st.Lanes)
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("the lane keeps its worktree: %v", err)
	}
	if out := f.repo.Git("branch", "--list", l.Branch); strings.TrimSpace(out) == "" {
		t.Fatal("the lane keeps its branch")
	}
}

// onSeen runs act once the merge is seen and before the closing takes its lock.
func onSeen(t *testing.T, act func(ctx context.Context)) {
	t.Helper()
	was := afterMergedSeen
	t.Cleanup(func() { afterMergedSeen = was })
	afterMergedSeen = act
}

// TestABuildClosesALaneWhosePullRequestMerged: once the lane's pull request has
// merged with a merge commit, building the intent again closes the lane (done,
// its landing naming the merge, its worktree and branch removed) and the run
// goes on in a new lane on a new branch cut at the merged tip, whether origin
// still holds the lane's branch, deleted it after the merge, or a person has
// removed the lane's worktree and branch here too. The lane's own local tier
// (the preflight receipt the fixture leaves there) goes with its worktree.
func TestABuildClosesALaneWhosePullRequestMerged(t *testing.T) {
	for _, tc := range []struct {
		name  string
		after func(t *testing.T, f *landFixture, l Lane)
	}{
		{"its branch still on origin", func(*testing.T, *landFixture, Lane) {}},
		{"its branch gone on origin", func(t *testing.T, f *landFixture, l Lane) {
			f.repo.Git("push", "-q", "origin", "--delete", l.Branch)
			f.repo.Git("fetch", "-q", "--prune", "origin")
		}},
		{"its worktree and branch gone here too", func(t *testing.T, f *landFixture, l Lane) {
			f.goneHereToo(t, l)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, l := mergedLaneFixture(t)
			m := f.mergeCommit(t, l.Landing.Pushed)
			tc.after(t, f, l)

			res, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
			if err != nil {
				t.Fatalf("the build resumes the run: %v", err)
			}
			if !res.Resumed || res.Finished || res.RunID != f.runID || res.Lane.ID == l.ID {
				t.Fatalf("the build resumes the run in another lane than the merged one: resumed %v, finished %v, run %s, lane %s at %s",
					res.Resumed, res.Finished, res.RunID, res.Lane.ID, res.Lane.Stage)
			}
			if len(res.Closed) != 1 || res.Closed[0] != (ClosedLane{Lane: l.ID, PR: 7, Merged: m}) {
				t.Fatalf("the result names the lane it closed: %+v", res.Closed)
			}
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil {
				t.Fatal(err)
			}
			old := st.Lanes[0]
			if old.Stage != StageDone || old.Landing == nil || old.Landing.Merged != m {
				t.Fatalf("the merged lane is closed, its landing naming the merge %s: %+v %+v", m, old, old.Landing)
			}
			if !slices.ContainsFunc(st.Record, func(e Entry) bool {
				return e.Lane == l.ID && e.Stage == StageResume && strings.Contains(e.Note, "pull request #7 merged")
			}) {
				t.Fatalf("the run's record says the build closed the lane: %+v", st.Record)
			}
			if _, err := os.Stat(l.Worktree); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("the merged lane's worktree is removed: %v", err)
			}
			if out := f.repo.Git("branch", "--list", l.Branch); strings.TrimSpace(out) != "" {
				t.Fatalf("the merged lane's branch is deleted: %q", out)
			}

			next := stepTo(t, f.repo, f.runID, f.stages, StageImplement)
			if next.ID == l.ID || next.Branch == "" || next.Branch == l.Branch || next.SpecStep != 2 {
				t.Fatalf("the run goes on in a new lane on a new branch for the next step: %+v", next)
			}
			if next.BaseSHA != m {
				t.Fatalf("the new lane is cut at the merged tip %s, not %s", m, next.BaseSHA)
			}
		})
	}
}

// TestABuildLeavesAMergedLaneWithCommitsPastTheMerge: a lane whose branch holds a
// commit the default branch does not is not read as merged, so the build resumes
// it as before, its worktree and branch where they were.
func TestABuildLeavesAMergedLaneWithCommitsPastTheMerge(t *testing.T) {
	f, l := mergedLaneFixture(t)
	f.mergeCommit(t, l.Landing.Pushed)
	late := laneCommit(t, f.repo, l, "late.txt")

	res, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatalf("the build resumes the run: %v", err)
	}
	if !res.Resumed || res.Lane.ID != l.ID || res.Lane.Stage != StageLand || len(res.Closed) != 0 {
		t.Fatalf("the build resumes the lane at its landing, as before: %+v", res)
	}
	f.leftAsItStands(t, l)
	if tip := strings.TrimSpace(f.repo.Git("rev-parse", "refs/heads/"+l.Branch)); tip != late {
		t.Fatalf("the lane keeps its branch at %s: %s", late, tip)
	}
}

// TestABuildRefusesAMergedLaneThatWouldLoseFiles: a merged lane whose worktree
// holds an uncommitted file, or an ignored one outside its own local tier, is
// refused by name, naming its worktree and the file, and nothing is removed or
// written.
func TestABuildRefusesAMergedLaneThatWouldLoseFiles(t *testing.T) {
	for _, tc := range []struct {
		name, file, kind string
		ignore           bool
	}{
		{"an uncommitted file", "unsaved.txt", "1 uncommitted or untracked file(s) and 0 ignored", false},
		{"an ignored file", "notes.env", "0 uncommitted or untracked file(s) and 1 ignored", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, l := mergedLaneFixture(t)
			f.mergeCommit(t, l.Landing.Pushed)
			if tc.ignore {
				exclude := filepath.Join(f.repo.Root(), ".git", "info", "exclude")
				if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
					t.Fatal(err)
				}
				fh, err := os.OpenFile(exclude, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := fh.WriteString("\n*.env\n"); err != nil {
					t.Fatal(err)
				}
				fh.Close()
			}
			stray := filepath.Join(l.Worktree, tc.file)
			if err := os.WriteFile(stray, []byte("not committed\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			before := f.stateBytes(t)

			_, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
			rf, ok := AsRefusal(err)
			if !ok || rf.Stage != StageResume || rf.Lane != l.ID || !strings.Contains(rf.Reason, "#7") ||
				!strings.Contains(rf.Reason, tc.kind) || !strings.Contains(rf.Reason, tc.file) ||
				!strings.Contains(rf.Reason, fsutil.RedactHome(l.Worktree)) || !strings.Contains(rf.Remedy, "abcd build itd-10") {
				t.Fatalf("the build refuses the lane by name, naming its worktree and the file: %v", err)
			}
			if _, err := os.Stat(stray); err != nil {
				t.Fatalf("the file stays: %v", err)
			}
			f.leftAsItStands(t, l)
			if string(f.stateBytes(t)) != string(before) {
				t.Fatalf("the refusal writes nothing to the run's state")
			}
		})
	}
}

// TestABuildLeavesARunAnotherDriverOrAStopHasTheSayOver: a merged lane is left
// as it stands, its worktree and branch kept, where the run is not this
// build's to settle, each read inside the run's lock immediately before
// anything is removed: its state file states another version than this
// binary's and this session is not its driver (mayWholeSave), a drive stopped on it, its control record holds a
// stop, a stop is set, or the stop read for the folder the run started under
// cannot be trusted. Each but the version and the hold is set after the merge
// was seen, so only the read under the lock can find it.
func TestABuildLeavesARunAnotherDriverOrAStopHasTheSayOver(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(t *testing.T, f *landFixture, o *Options)
	}{
		{"another version's state", func(t *testing.T, f *landFixture, _ *Options) {
			stateAtVersion(t, f.repo.Root(), f.runID, schemaVersionPrevious)
		}},
		{"a drive stopped on it", func(t *testing.T, f *landFixture, _ *Options) {
			p, err := driveStopPath(f.runID)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Remove(p) })
			onSeen(t, func(context.Context) {
				if err := recordDriveStop(f.runID); err != nil {
					t.Error(err)
				}
			})
		}},
		{"a stop in its control record", func(t *testing.T, f *landFixture, _ *Options) {
			onSeen(t, func(context.Context) {
				root, err := os.OpenRoot(f.repo.Root())
				if err != nil {
					t.Error(err)
					return
				}
				defer root.Close()
				c, err := readControl(root, f.runID)
				if err != nil {
					t.Error(err)
					return
				}
				now := time.Now()
				c.Stop = &RunStop{SetBy: "a person", SetAt: now, NotedAt: now}
				if err := writeControl(root, f.runID, c); err != nil {
					t.Error(err)
				}
			})
		}},
		{"a stop set", func(t *testing.T, f *landFixture, o *Options) {
			home, _ := stopWorld(t, false)
			o.stopHome = home
			onSeen(t, func(context.Context) {
				if _, err := runstop.Set(home, time.Now()); err != nil {
					t.Error(err)
				}
			})
		}},
		{"the folder it started under not to be trusted", func(t *testing.T, f *landFixture, o *Options) {
			home, _ := stopWorld(t, true)
			o.stopHome = home
			pin := runstop.PinOf(home)
			if pin == nil {
				t.Fatal("premise: the setting pins a folder")
			}
			root, err := os.OpenRoot(f.repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			c, err := readControl(root, f.runID)
			if err != nil {
				t.Fatal(err)
			}
			c.Shared = pin
			if err := writeControl(root, f.runID, c); err != nil {
				t.Fatal(err)
			}
			other := filepath.Join(t.TempDir(), "elsewhere")
			if err := os.Mkdir(other, 0o755); err != nil {
				t.Fatal(err)
			}
			if _, err := runstop.WriteSetting(home, other, ""); err != nil {
				t.Fatal(err)
			}
			if r := o.stopRead(pin); r.Held == "" || r.Stop != nil {
				t.Fatalf("premise: the read for the run's folder is held, with no stop: %+v", r)
			}
			if r := o.stopRead(nil); r.Held != "" || r.Stop != nil {
				t.Fatalf("premise: the start's own read finds nothing: %+v", r)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, l := mergedLaneFixture(t)
			f.mergeCommit(t, l.Landing.Pushed)
			o := Options{noReserve: true}
			tc.set(t, f, &o)

			res, err := Start(f.repo.Root(), "itd-10", o)
			if err != nil {
				if _, ok := AsRefusal(err); !ok {
					t.Fatalf("the build: %v", err)
				}
			} else if len(res.Closed) != 0 || !slices.ContainsFunc(res.Diagnostics, func(d string) bool { return strings.Contains(d, "left as it stands") }) {
				t.Fatalf("the build closes nothing and says why: %+v", res)
			}
			f.leftAsItStands(t, l)
		})
	}
}

// TestADrainOrATrialStartClosesNothing: a drain's start and a trial's start
// resume the run as they did, and leave its merged lane for the drain's own move.
func TestADrainOrATrialStartClosesNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts func(t *testing.T, f *landFixture) Options
	}{
		{"a drain's start", func(*testing.T, *landFixture) Options {
			return Options{noReserve: true, drain: true}
		}},
		{"a trial's start", func(t *testing.T, f *landFixture) Options {
			editState(t, f.repo.Root(), f.runID, func(doc map[string]any) { doc["trial"] = true })
			head := strings.TrimSpace(f.repo.Git("rev-parse", "HEAD"))
			return Options{noReserve: true, Trial: true, Frozen: head}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, l := mergedLaneFixture(t)
			f.mergeCommit(t, l.Landing.Pushed)
			o := tc.opts(t, f)
			res, err := Start(f.repo.Root(), "itd-10", o)
			if err != nil {
				t.Fatalf("the start resumes the run: %v", err)
			}
			if !res.Resumed || res.RunID != f.runID || len(res.Closed) != 0 {
				t.Fatalf("the start resumes the run and closes nothing: %+v", res)
			}
			f.leftAsItStands(t, l)
		})
	}
}

// TestABuildDoesNotTakeALaneOffTheDefaultBranchAsMerged: what the lane pushed
// must be on origin's default branch as fetched now. A squash merge leaves it
// off; a branch merely gone says nothing of where its head went; a
// remote-tracking ref moved here onto the lane's head is put back by the fetch;
// and a merge origin's default branch has since been reset past is not on it.
// In each, the lane resumes as it was.
func TestABuildDoesNotTakeALaneOffTheDefaultBranchAsMerged(t *testing.T) {
	for _, tc := range []struct {
		name  string
		after func(t *testing.T, f *landFixture, l Lane)
		gone  bool
	}{
		{"a squash merge, the branch gone here too", func(t *testing.T, f *landFixture, l Lane) {
			f.squashMerge(t, l.Landing.Pushed)
			f.goneHereToo(t, l)
		}, true},
		{"a branch merely gone", func(t *testing.T, f *landFixture, l Lane) {
			f.goneHereToo(t, l)
		}, true},
		{"a tracking ref moved here, origin unchanged", func(t *testing.T, f *landFixture, l Lane) {
			f.repo.Git("update-ref", "refs/remotes/origin/main", l.Landing.Pushed)
		}, false},
		{"origin's default branch reset after the merge", func(t *testing.T, f *landFixture, l Lane) {
			before := strings.TrimSpace(f.repo.Git("rev-parse", "refs/remotes/origin/main"))
			m := f.mergeCommit(t, l.Landing.Pushed)
			f.repo.Git("push", "-q", "--force", "origin", before+":refs/heads/main")
			// The push moved the tracking ref back too; this checkout last
			// fetched the merge.
			f.repo.Git("update-ref", "refs/remotes/origin/main", m)
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, l := mergedLaneFixture(t)
			tc.after(t, f, l)
			origin := strings.TrimSpace(f.repo.Git("--git-dir", f.bare, "rev-parse", "refs/heads/main"))

			res, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
			if err != nil {
				t.Fatalf("the build resumes the run: %v", err)
			}
			if !res.Resumed || res.Lane.ID != l.ID || len(res.Closed) != 0 {
				t.Fatalf("the build resumes the lane as it was: %+v", res)
			}
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil {
				t.Fatal(err)
			}
			if st.Lanes[0].Stage != StageLand || st.Lanes[0].Landing.Merged != "" {
				t.Fatalf("the lane is not closed: %+v", st.Lanes[0])
			}
			if !tc.gone {
				f.leftAsItStands(t, l)
			}
			if got := strings.TrimSpace(f.repo.Git("rev-parse", "refs/remotes/origin/main")); got != origin {
				t.Fatalf("the build fetched origin's default branch: the tracking ref is at %s, origin at %s", got, origin)
			}
		})
	}
}

// TestABuildThatCannotFetchClosesNothing: a fetch of the default branch that
// fails, or does not finish within the closing's bound, is a merge not seen:
// the lane resumes as it was, and a line says why.
func TestABuildThatCannotFetchClosesNothing(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		set        func(t *testing.T, f *landFixture)
	}{
		{"origin gone", "failed", func(t *testing.T, f *landFixture) {
			if err := os.Rename(f.bare, f.bare+".gone"); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Rename(f.bare+".gone", f.bare) })
		}},
		{"out of time", "did not finish within", func(t *testing.T, f *landFixture) {
			was := closeMergedTimeout
			t.Cleanup(func() { closeMergedTimeout = was })
			closeMergedTimeout = time.Nanosecond
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, l := mergedLaneFixture(t)
			f.mergeCommit(t, l.Landing.Pushed)
			tc.set(t, f)

			res, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
			if err != nil {
				t.Fatalf("the build resumes the run: %v", err)
			}
			if len(res.Closed) != 0 || !slices.ContainsFunc(res.Diagnostics, func(d string) bool {
				return strings.Contains(d, "no merge was seen and nothing was closed") && strings.Contains(d, tc.want)
			}) {
				t.Fatalf("the build closes nothing and says why: %+v %q", res.Closed, res.Diagnostics)
			}
			f.leftAsItStands(t, l)
		})
	}
}

// TestABuildOutOfTimeRemovesNothing: a closing whose bound runs out between the
// fetch that saw the merge and the removal is refused at the resume stage,
// saying so, and nothing is removed or written.
func TestABuildOutOfTimeRemovesNothing(t *testing.T) {
	f, l := mergedLaneFixture(t)
	f.mergeCommit(t, l.Landing.Pushed)
	was := closeMergedTimeout
	t.Cleanup(func() { closeMergedTimeout = was })
	closeMergedTimeout = 5 * time.Second
	onSeen(t, func(ctx context.Context) { <-ctx.Done() })
	before := f.stateBytes(t)

	_, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
	rf, ok := AsRefusal(err)
	if !ok || rf.Stage != StageResume || rf.Lane != l.ID || !strings.Contains(rf.Reason, "did not finish within 5s") ||
		!strings.Contains(rf.Reason, "nothing was removed") || !strings.Contains(rf.Remedy, "abcd build itd-10") {
		t.Fatalf("the closing out of time is refused, removing nothing: %v", err)
	}
	f.leftAsItStands(t, l)
	if string(f.stateBytes(t)) != string(before) {
		t.Fatal("the refusal writes nothing to the run's state")
	}
}

// TestARefusedResumeClosesNothing: a resume its own checks refuse (another pace
// named, a pick that chose an intent already in progress, the reservation
// another person holds) removes nothing, though the lane's pull request merged.
func TestARefusedResumeClosesNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(t *testing.T, f *landFixture) error
	}{
		{"another pace", func(t *testing.T, f *landFixture) error {
			pace := "1/1"
			_, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true, Pace: &pace})
			return err
		}},
		{"a pick", func(t *testing.T, f *landFixture) error {
			_, err := start(f.repo.Root(), "itd-10", Options{noReserve: true}, &RunPick{})
			return err
		}},
		{"a reservation another holds", func(t *testing.T, f *landFixture) error {
			now := time.Now().UTC().Truncate(time.Second)
			reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-130*time.Minute), "internal/widget")
			q := otherCopy(t, f.bare, quinnEmail)
			if res, err := claim.Reserve(q, claim.Request{Record: "itd-10", Footprint: []string{"internal/gadget"}}, as(t, q, clockAt(now.Add(-time.Minute)))); err != nil || res.Act != claim.ActTakenOver {
				t.Fatalf("premise: Q takes the lapsed reservation over: %+v %v", res, err)
			}
			if _, err := claim.Fetch(f.repo.Root(), clockAt(now)); err != nil {
				t.Fatal(err)
			}
			_, err := Start(f.repo.Root(), "itd-10", Options{})
			return err
		}},
		{"a record finished that the closing would not finish", func(t *testing.T, f *landFixture) error {
			specHasFootprint(t, f.repo)
			reserveRun(t, f.repo, f.runID, "itd-10", time.Now().UTC().Truncate(time.Second), "commands/x.md")
			f.intentShippedOnOrigin(t)
			if !claim.Finished(f.repo.Root(), "itd-10") {
				t.Fatal("premise: the record is finished on origin's default branch")
			}
			_, err := Start(f.repo.Root(), "itd-10", Options{})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, l := mergedLaneFixture(t)
			f.mergeCommit(t, l.Landing.Pushed)
			if _, ok := AsRefusal(tc.run(t, f)); !ok {
				t.Fatal("premise: the resume is refused")
			}
			f.leftAsItStands(t, l)
		})
	}
}

// TestABuildThatClosesTheLastLaneFinishesTheRun: closing a run's last lane
// finishes the run. The state is written before the reservation is released,
// the result says the lane was closed and the run finished, and the invocation
// starts nothing more: no other run is made.
func TestABuildThatClosesTheLastLaneFinishesTheRun(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	if l.Stage != StageLand || l.PR != 7 || l.Landing == nil || !gitutil.IsFullSHA(l.Landing.Pushed) {
		t.Fatalf("premise: the run's one lane is armed at its landing: %+v", l)
	}
	specHasFootprint(t, f.repo)
	reserveRun(t, f.repo, f.runID, "itd-10", time.Now().UTC().Truncate(time.Second), "commands/x.md")
	m := f.mergeCommit(t, l.Landing.Pushed)
	was := claimReserve
	t.Cleanup(func() { claimReserve = was })
	released := 0
	claimReserve = func(dir string, req claim.Request, o claim.Options) (claim.Result, error) {
		if req.Release {
			released++
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil || !st.Complete() {
				t.Errorf("the reservation is released before the state shows the run finished: %v %+v", err, st.Lanes)
			}
		}
		return was(dir, req, o)
	}
	runs := func() []string {
		entries, err := os.ReadDir(filepath.Join(f.repo.Root(), filepath.FromSlash(RunRelDir)))
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, e := range entries {
			ids = append(ids, e.Name())
		}
		return ids
	}
	before := runs()

	res, err := Start(f.repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatalf("the build closes the lane: %v", err)
	}
	if !res.Resumed || !res.Finished || res.RunID != f.runID || len(res.Closed) != 1 || res.Closed[0].Merged != m {
		t.Fatalf("the result says the lane was closed and the run finished: %+v", res)
	}
	if !strings.Contains(res.Next, "closed lane-1 (pull request #7)") || !strings.Contains(res.Next, "is finished") || !strings.Contains(res.Next, "nothing more was started") {
		t.Fatalf("the next line says so: %q", res.Next)
	}
	if released != 1 || !strings.Contains(res.Reservation, "released the reservation on itd-10") {
		t.Fatalf("the reservation is released once, after the state: %d %q", released, res.Reservation)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Complete() || st.Lanes[0].Landing.Merged != m {
		t.Fatalf("the run is complete: %+v", st.Lanes)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageReservationReleased && strings.Contains(e.Note, "released the reservation")
	}) {
		t.Fatalf("the run's record logs the release: %+v", st.Record)
	}
	if after := runs(); !slices.Equal(after, before) {
		t.Fatalf("no other run is made: %v, was %v", after, before)
	}
}

// TestARefusalAfterAnEarlierLaneClosedKeepsThatClosing: with two merged lanes,
// the second refused at its removal (its path holds a worktree on another
// branch), the first lane's closing is written, and the refusal names the
// resume stage and the build to run again.
func TestARefusalAfterAnEarlierLaneClosedKeepsThatClosing(t *testing.T) {
	f, l := mergedLaneFixture(t)
	f.mergeCommit(t, l.Landing.Pushed)
	lw, err := laneWorktree(f.repo.Root(), f.runID, "lane-2")
	if err != nil {
		t.Fatal(err)
	}
	f.repo.Git("branch", lw.Branch, l.Landing.Pushed)
	f.repo.Git("worktree", "add", "-q", "-b", "elsewhere", lw.Path, l.Landing.Pushed)
	editState(t, f.repo.Root(), f.runID, func(doc map[string]any) {
		lanes := doc["lanes"].([]any)
		second := map[string]any{}
		for k, v := range lanes[0].(map[string]any) {
			second[k] = v
		}
		second["id"], second["spec_step"], second["pr"] = "lane-2", 2, 8
		second["branch"], second["worktree"] = lw.Branch, lw.Path
		doc["lanes"] = append(lanes, second)
		doc["pending"] = []any{}
	})

	_, err = Start(f.repo.Root(), "itd-10", Options{noReserve: true})
	rf, ok := AsRefusal(err)
	if !ok || rf.Lane != "lane-2" || rf.Stage != StageResume || !strings.Contains(rf.Reason, "on another branch") || !strings.Contains(rf.Remedy, "abcd build itd-10") {
		t.Fatalf("the second lane is refused at the resume stage, naming the build: %v", err)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Stage != StageDone || st.Lanes[0].Landing.Merged == "" || st.Lanes[1].Stage != StageLand {
		t.Fatalf("the first lane's closing is kept and the second is as it was: %+v", st.Lanes)
	}
	if _, err := os.Stat(l.Worktree); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the first lane's worktree is removed: %v", err)
	}
	if _, err := os.Stat(lw.Path); err != nil {
		t.Fatalf("the second lane's path keeps what stands there: %v", err)
	}
}

// TestABuildRefusesAMergedLaneWhoseBranchIsCheckedOutElsewhere: a merged lane
// whose branch a person has checked out in another worktree is refused at the
// resume stage, naming that checkout and the build to run again, and its
// branch and that checkout are kept.
func TestABuildRefusesAMergedLaneWhoseBranchIsCheckedOutElsewhere(t *testing.T) {
	f, l := mergedLaneFixture(t)
	f.mergeCommit(t, l.Landing.Pushed)
	f.repo.Git("worktree", "remove", "--force", l.Worktree)
	other := filepath.Join(t.TempDir(), "other")
	f.repo.Git("worktree", "add", "-q", other, l.Branch)
	before := f.stateBytes(t)

	_, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
	rf, ok := AsRefusal(err)
	if !ok || rf.Stage != StageResume || rf.Lane != l.ID || !strings.Contains(rf.Reason, "is checked out in") || !strings.Contains(rf.Reason, "other") || !strings.Contains(rf.Remedy, "abcd build itd-10") {
		t.Fatalf("the build refuses the lane, naming the other checkout: %v", err)
	}
	if out := f.repo.Git("branch", "--list", l.Branch); strings.TrimSpace(out) == "" {
		t.Fatal("the lane's branch is kept")
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("the other checkout is kept: %v", err)
	}
	if string(f.stateBytes(t)) != string(before) {
		t.Fatal("the refusal writes nothing to the run's state")
	}
}

// runIDs lists the runs the checkout holds.
func (f *landFixture) runIDs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(f.repo.Root(), filepath.FromSlash(RunRelDir)))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, e := range entries {
		ids = append(ids, e.Name())
	}
	return ids
}

// TestABuildOfARecordFinishedOnOriginClosesItsLastLane: a one-lane run whose
// pull request merged and whose intent sits shipped on origin's default branch
// has its reservation refused for that reason alone (claim.ErrFinished), and the
// build goes on to close the lane, since closing it finishes the run: the result
// is finished, one lane closed, no other run made, and a reservation this copy
// held is released once, after the state. A run so finished records no late
// admission.
func TestABuildOfARecordFinishedOnOriginClosesItsLastLane(t *testing.T) {
	for _, tc := range []struct {
		name string
		held bool
	}{
		{"this copy's live reservation", true},
		{"no reservation held", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.landedToArmedOnce(t)
			specHasFootprint(t, f.repo)
			if tc.held {
				reserveRun(t, f.repo, f.runID, "itd-10", time.Now().UTC().Truncate(time.Second), "commands/x.md")
			} else {
				asPat(t, f.repo)
			}
			m := f.mergeCommit(t, l.Landing.Pushed)
			f.intentShippedOnOrigin(t)
			// A state that names no admission, as a run begun before runs
			// recorded theirs: a resume would record one late.
			editState(t, f.repo.Root(), f.runID, func(doc map[string]any) { delete(doc, "admitted") })
			was := claimReserve
			t.Cleanup(func() { claimReserve = was })
			released := 0
			claimReserve = func(dir string, req claim.Request, o claim.Options) (claim.Result, error) {
				if req.Release {
					released++
				}
				return was(dir, req, o)
			}
			before := f.runIDs(t)

			res, err := Start(f.repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatalf("the build closes the lane of a record finished on origin: %v", err)
			}
			if !res.Resumed || !res.Finished || len(res.Closed) != 1 || res.Closed[0].Lane != l.ID || res.Closed[0].Merged != m {
				t.Fatalf("the result says the lane was closed and the run finished: %+v", res)
			}
			if _, err := os.Stat(l.Worktree); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("the lane's worktree is removed: %v", err)
			}
			if tc.held && (released != 1 || !strings.Contains(res.Reservation, "released the reservation on itd-10")) {
				t.Fatalf("the reservation is released once: %d %q", released, res.Reservation)
			}
			if after := f.runIDs(t); !slices.Equal(after, before) {
				t.Fatalf("no other run is made: %v, was %v", after, before)
			}
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil {
				t.Fatal(err)
			}
			if !st.Complete() {
				t.Fatalf("the run is complete: %+v", st.Lanes)
			}
			if st.Admitted != nil || slices.ContainsFunc(st.Record, func(e Entry) bool {
				return e.Stage == StageAdmitted && strings.Contains(e.Note, "recorded as the run resumed")
			}) {
				t.Fatalf("a run the closing finished admits nothing late: %+v %+v", st.Admitted, st.Record)
			}
		})
	}
}

// TestARecordFinishedOnOriginWithAnotherRefusalClosesNothing: a resume whose
// reservation is refused for any reason but the record's being finished, here
// another person's hold, is refused, though the record sits finished on origin
// and closing the lane would finish the run: nothing is removed.
func TestARecordFinishedOnOriginWithAnotherRefusalClosesNothing(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	specHasFootprint(t, f.repo)
	reserveRun(t, f.repo, f.runID, "itd-10", time.Now().UTC().Truncate(time.Second), "commands/x.md")
	f.mergeCommit(t, l.Landing.Pushed)
	f.intentShippedOnOrigin(t)
	if !claim.Finished(f.repo.Root(), "itd-10") {
		t.Fatal("premise: the record is finished on origin's default branch")
	}
	was := reserveFresh
	t.Cleanup(func() { reserveFresh = was })
	asked := 0
	reserveFresh = func(string, claim.Request, claim.Options) (claim.Result, error) {
		asked++
		return claim.Result{}, fmt.Errorf("itd-10 is reserved by quinn; back off and take other work: %w", claim.ErrBackOff)
	}

	_, err := Start(f.repo.Root(), "itd-10", Options{})
	if rf, ok := AsRefusal(err); !ok || !rf.Contention || asked != 1 {
		t.Fatalf("premise: the resume's reservation is refused by another's hold: %v (asked %d)", err, asked)
	}
	f.leftAsItStands(t, l)
}

// TestABuildClosesALaneWhoseLandingLeftAbcdsOwnFiles: a lane whose landing
// resolved an issue leaves the issue ledger's allocator lock in its worktree,
// ignored by the repository's .gitignore as this repository ignores it; the
// lock is abcd's own, so the lane is closed with its worktree.
func TestABuildClosesALaneWhoseLandingLeftAbcdsOwnFiles(t *testing.T) {
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": queueRuleset("MERGE"),
		".gitignore": ".abcd/.work.local/\n.abcd/work/issues/.iss-alloc.lock\n",
	})
	l := f.landedToArmedOnce(t)
	lock := filepath.Join(l.Worktree, ".abcd", "work", "issues", ".iss-alloc.lock")
	if _, err := os.Stat(lock); err != nil || len(l.Resolves) == 0 {
		t.Fatalf("premise: the landing resolved an issue and left the allocator lock: %v %+v", err, l.Resolves)
	}
	if out := gitIn(t, l.Worktree, "check-ignore", ".abcd/work/issues/.iss-alloc.lock"); strings.TrimSpace(out) == "" {
		t.Fatal("premise: the lock is ignored")
	}
	m := f.mergeCommit(t, l.Landing.Pushed)

	res, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatalf("the build closes the lane: %v", err)
	}
	if len(res.Closed) != 1 || res.Closed[0].Merged != m {
		t.Fatalf("the lane is closed: %+v", res.Closed)
	}
	if _, err := os.Stat(l.Worktree); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the worktree goes, with abcd's lock in it: %v", err)
	}
}

// TestAbcdsOwnArtefactsAreMatchedByExactPath: the files a lane's worktree may
// lose with it are its local tier and abcd's own artefacts, matched by their
// exact paths from the root and the bootstrap's one pattern, nothing else.
func TestAbcdsOwnArtefactsAreMatchedByExactPath(t *testing.T) {
	for p, want := range map[string]bool{
		".abcd/.work.local/":                    true,
		".abcd/.work.local/receipts/x.json":     true,
		".abcd/work/issues/.iss-alloc.lock":     true,
		"abcd":                                  true,
		".binary-meta":                          true,
		".bootstrap.lock/":                      true,
		".bootstrap.attempt":                    true,
		".bootstrap.tmp.4242/":                  true,
		".bootstrap.tmp.4242/artefact":          true,
		".abcd/work/issues/other.lock":          false,
		".abcd/work/issues/.iss-alloc.lock.bak": false,
		"sub/.binary-meta":                      false,
		"abcd/":                                 false,
		".bootstrap.tmp./":                      false,
		".bootstrap.tmp.x1/":                    false,
		"notes.env":                             false,
		".abcd/.work.localx":                    false,
	} {
		if got := abcdArtefact(p); got != want {
			t.Errorf("abcdArtefact(%q) = %v, want %v", p, got, want)
		}
	}
}

// TestTheClosingHoldsToTheCommitItFetched: the proof of a merge is the commit
// the fetch brought, resolved straight after it, never the remote-tracking ref
// read again under the lock. A ref moved back after the fetch still closes the
// lane on the fetched commit; a ref moved forward to take in a commit the lane's
// branch gained after its push closes nothing.
func TestTheClosingHoldsToTheCommitItFetched(t *testing.T) {
	t.Run("the ref moved back", func(t *testing.T) {
		f, l := mergedLaneFixture(t)
		base := strings.TrimSpace(f.repo.Git("rev-parse", "refs/remotes/origin/main"))
		m := f.mergeCommit(t, l.Landing.Pushed)
		onSeen(t, func(context.Context) { f.repo.Git("update-ref", "refs/remotes/origin/main", base) })

		res, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
		if err != nil {
			t.Fatalf("the build closes the lane on the fetched commit: %v", err)
		}
		if len(res.Closed) != 1 || res.Closed[0].Merged != m {
			t.Fatalf("the lane is closed on the fetched commit %s: %+v", m, res.Closed)
		}
		if out := f.repo.Git("branch", "--list", l.Branch); strings.TrimSpace(out) != "" {
			t.Fatalf("the lane's branch is deleted: %q", out)
		}
	})
	t.Run("the ref moved forward over a later commit", func(t *testing.T) {
		f, l := mergedLaneFixture(t)
		f.mergeCommit(t, l.Landing.Pushed)
		var late string
		onSeen(t, func(context.Context) {
			late = laneCommit(t, f.repo, l, "late.txt")
			f.repo.Git("update-ref", "refs/remotes/origin/main", late)
		})

		_, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
		rf, ok := AsRefusal(err)
		if !ok || rf.Stage != StageResume || rf.Lane != l.ID || !strings.Contains(rf.Reason, "past what landed") {
			t.Fatalf("the lane's branch past the fetched commit is refused: %v", err)
		}
		f.leftAsItStands(t, l)
		if tip := strings.TrimSpace(f.repo.Git("rev-parse", "refs/heads/"+l.Branch)); tip != late {
			t.Fatalf("the branch keeps its later commit %s: %s", late, tip)
		}
	})
}

// endClosingAfterRemoval makes the closing's context end once a lane's worktree
// has been removed, before its branch is deleted.
func endClosingAfterRemoval(t *testing.T) {
	t.Helper()
	wasCtx, wasHook := closingContext, afterLaneRemoved
	t.Cleanup(func() { closingContext, afterLaneRemoved = wasCtx, wasHook })
	var end context.CancelFunc
	closingContext = func() (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(context.Background())
		end = cancel
		return ctx, cancel
	}
	afterLaneRemoved = func(context.Context) {
		if end != nil {
			end()
		}
	}
}

// TestAClosingOutOfTimeAfterARemovalFinishesThatLane: once a lane's worktree
// removal has begun, the closing's deadline does not cut its branch's deletion
// off, so the lane is closed whole; a later lane the deadline then reaches is
// refused as out of time, never as a branch past what landed, nothing of it
// removed, and the earlier lane stays closed.
func TestAClosingOutOfTimeAfterARemovalFinishesThatLane(t *testing.T) {
	t.Run("one lane", func(t *testing.T) {
		f, l := mergedLaneFixture(t)
		f.mergeCommit(t, l.Landing.Pushed)
		endClosingAfterRemoval(t)

		res, err := Start(f.repo.Root(), "itd-10", Options{noReserve: true})
		if err != nil {
			t.Fatalf("the lane whose removal began is closed whole: %v", err)
		}
		if len(res.Closed) != 1 {
			t.Fatalf("the lane is closed: %+v", res.Closed)
		}
		if out := f.repo.Git("branch", "--list", l.Branch); strings.TrimSpace(out) != "" {
			t.Fatalf("the branch is deleted though the deadline ended after the removal: %q", out)
		}
	})
	t.Run("a later lane", func(t *testing.T) {
		f, l := mergedLaneFixture(t)
		f.mergeCommit(t, l.Landing.Pushed)
		lw, err := laneWorktree(f.repo.Root(), f.runID, "lane-2")
		if err != nil {
			t.Fatal(err)
		}
		f.repo.Git("worktree", "add", "-q", "-b", lw.Branch, lw.Path, l.Landing.Pushed)
		editState(t, f.repo.Root(), f.runID, func(doc map[string]any) {
			lanes := doc["lanes"].([]any)
			second := map[string]any{}
			for k, v := range lanes[0].(map[string]any) {
				second[k] = v
			}
			second["id"], second["spec_step"], second["pr"] = "lane-2", 2, 8
			second["branch"], second["worktree"] = lw.Branch, lw.Path
			doc["lanes"] = append(lanes, second)
			doc["pending"] = []any{}
		})
		endClosingAfterRemoval(t)

		_, err = Start(f.repo.Root(), "itd-10", Options{noReserve: true})
		rf, ok := AsRefusal(err)
		if !ok || rf.Stage != StageResume || rf.Lane != "lane-2" || !strings.Contains(rf.Reason, "did not finish within") ||
			strings.Contains(rf.Reason, "past what landed") || !strings.Contains(rf.Remedy, "abcd build itd-10") {
			t.Fatalf("the later lane is refused as out of time: %v", err)
		}
		st, err := ReadState(f.repo.Root(), f.runID)
		if err != nil {
			t.Fatal(err)
		}
		if st.Lanes[0].Stage != StageDone || st.Lanes[1].Stage != StageLand {
			t.Fatalf("the first lane stays closed and the second is as it was: %+v", st.Lanes)
		}
		if _, err := os.Stat(lw.Path); err != nil {
			t.Fatalf("the second lane keeps its worktree: %v", err)
		}
		if out := f.repo.Git("branch", "--list", lw.Branch); strings.TrimSpace(out) == "" {
			t.Fatal("the second lane keeps its branch")
		}
	})
}
