package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// steppedBriefRepo is briefRepo with an open spec listing steps.
func steppedBriefRepo(t *testing.T, steps string) *gittest.Repo {
	t.Helper()
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(steps))
	repo.Write("AGENTS.md", agentsMarked)
	repo.Commit("the conventions")
	return repo
}

// laneBrief reads the brief of the run's lane i.
func laneBrief(t *testing.T, repo *gittest.Repo, runID string, i int) string {
	t.Helper()
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(st.Lanes[i].Brief)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// section is the brief's text under heading, up to the next `## ` heading.
func section(t *testing.T, brief, heading string) string {
	t.Helper()
	at := strings.Index(brief, heading+"\n")
	if at < 0 {
		t.Fatalf("the brief has no %q section:\n%s", heading, brief)
	}
	rest := brief[at+len(heading)+1:]
	if end := strings.Index(rest, "\n## "); end >= 0 {
		rest = rest[:end]
	}
	return rest
}

const stepsHeading = "## The spec's steps before yours"

// TestTheBriefNamesTheStepsBeforeItsOwn is itd-2609212103565953's fourth
// criterion, the brief half: a lane's brief names the step it builds and the
// steps before it, each with what landed it as the spec at the lane's base
// records, and names no step after it. The section sits before the outbound
// policy's own section, and before the record the brief carries.
func TestTheBriefNamesTheStepsBeforeItsOwn(t *testing.T) {
	repo := steppedBriefRepo(t,
		"1. The parser\n   - packages: internal/core/spec\n   - landed: #12\n"+
			"2. The remainder\n   - landed: 0123abc\n"+
			"3. The loop\n   - packages: internal/core/implement\n"+
			"4. The page\n")
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StageImplement)
	brief := laneBrief(t, repo, start.RunID, 0)

	if !strings.Contains(brief, "- Build spec step 3, \"The loop\", and nothing else of the spec.") {
		t.Fatalf("the brief names the lane's own step:\n%s", brief)
	}
	got := section(t, brief, stepsHeading)
	for _, want := range []string{
		"step 3 of the 4 spc-1 lists",
		"1. \"The parser\" — landed: #12 (the spec at the lane's base)",
		"2. \"The remainder\" — landed: 0123abc (the spec at the lane's base)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("the steps section must carry %q:\n%s", want, got)
		}
	}
	for _, absent := range []string{"The page", "\"The loop\" —"} {
		if strings.Contains(got, absent) {
			t.Fatalf("the steps section names only the steps before the lane's, not %q:\n%s", absent, got)
		}
	}
	at := strings.Index(brief, stepsHeading)
	if policy, record := strings.Index(brief, "## Outward-facing text\n"), strings.Index(brief, "<!-- begin "); policy < 0 || record < 0 || at > record {
		t.Fatalf("the steps are the brief's own instruction, before the record it carries (steps at %d, record at %d)", at, record)
	}
}

// TestTheBriefSaysWhenNoStepComesBefore: the first step of a stepped spec, and
// a spec that lists none (one implicit step), each say so rather than render an
// empty list.
func TestTheBriefSaysWhenNoStepComesBefore(t *testing.T) {
	for _, tc := range []struct {
		name, steps, want string
	}{
		{"the first step", "1. The parser\n2. The loop\n", "step 1 of the 2 spc-1 lists: no step comes before it."},
		{"an unstepped spec", "", "spc-1 lists no steps, so this lane builds the whole spec: no step comes before it."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := steppedBriefRepo(t, tc.steps)
			start, err := Start(repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatal(err)
			}
			advanceTo(t, repo, start.RunID, StageImplement)
			if got := section(t, laneBrief(t, repo, start.RunID, 0), stepsHeading); !strings.Contains(got, tc.want) {
				t.Fatalf("want %q:\n%s", tc.want, got)
			}
		})
	}
}

// TestTheBriefNamesWhatAnEarlierLaneOfTheRunBuilt: a step an earlier lane of
// this run built is named with that lane, its branch, its head and its pull
// request, from the run's own state: the spec at the next lane's base need not
// mark it landed yet.
func TestTheBriefNamesWhatAnEarlierLaneOfTheRunBuilt(t *testing.T) {
	repo := steppedBriefRepo(t, "1. The parser\n2. The loop\n")
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	steps := f.steps()
	steps[0].Run, steps[1].Run = worktreeStage, briefStage
	steps[4].Run = func(c Context, lane *Lane) (Outcome, error) {
		lane.PR = 7
		return Outcome{Note: "landed"}, nil
	}
	id := start.RunID
	for {
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		i := st.current()
		if i == 1 && st.Lanes[1].Stage == StageImplement {
			break
		}
		if i < 0 {
			t.Fatal("the run completed before the second lane was briefed")
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
		if _, err := Advance(repo.Root(), id, steps, Options{}); err != nil {
			t.Fatal(err)
		}
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	first := st.Lanes[0]
	got := section(t, laneBrief(t, repo, id, 1), stepsHeading)
	want := "1. \"The parser\" — built by lane-1 of this run: branch `" + first.Branch + "` at " + first.HeadSHA + ", pull request #7"
	if !strings.Contains(got, want) {
		t.Fatalf("want %q:\n%s", want, got)
	}
}

// TestABriefWhoseStepTheBaseListsOtherwiseIsRefused: the lane was opened for a
// step as the spec listed it when the run started. A base whose spec lists that
// step under another title (the steps were reordered or rewritten since), or
// whose steps cannot be read, is refused at the brief rather than briefed with
// the wrong predecessors, and the lane stays at its brief.
func TestABriefWhoseStepTheBaseListsOtherwiseIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name, steps, want string
	}{
		{"reordered", "1. The loop\n2. The parser\n", "lists step 1 as \"The loop\""},
		{"unreadable", "1. The parser\nnot a step\n", "not a numbered step"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := steppedBriefRepo(t, "1. The parser\n2. The loop\n")
			start, err := Start(repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatal(err)
			}
			repo.Write(specRel, specWithSteps(tc.steps))
			repo.Commit("the steps change on the default branch")
			advanceTo(t, repo, start.RunID, StageBrief)
			_, err = Advance(repo.Root(), start.RunID, DefaultStages(), Options{})
			if r := mustRefusal(t, err); r.Stage != string(StageBrief) || !strings.Contains(r.Reason, tc.want) {
				t.Fatalf("want the brief refused naming %q: %+v", tc.want, r)
			}
			if st, _ := ReadState(repo.Root(), start.RunID); st.Lanes[0].Stage != StageBrief || st.Lanes[0].Brief != "" {
				t.Fatalf("the lane stays at its brief: %+v", st.Lanes[0])
			}
		})
	}
}
