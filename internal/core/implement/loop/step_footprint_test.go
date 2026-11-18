package loop

import (
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// A build reserves the footprint of the steps it still has to build
// (iss-2611181755106054): once some of a spec's steps carry `landed:` lines,
// the reservation is the union of the `packages:` lines of the steps without
// one, so a directory only a landed step touched holds no one off and is
// held off by no one. The `## Footprint` packages line stays the footprint
// when the spec lists no steps, when a step still to build names no
// packages, and when every step has landed.

// fiveStepSpec is spc-1 with five steps, the first four landed, as
// spc-2610240951109420 stood when its build was refused: only step 5 is
// left, and it never touches internal/surface/cli.
var fiveStepSpec = specWithSteps(
	"1. The review set\n   - packages: internal/core/implement/loop, internal/core/layered, internal/surface/cli, commands\n   - landed: a7ab8927bf0b\n"+
		"2. A runner takes a model per request\n   - packages: internal/core/runner\n   - landed: 9163555347f5\n"+
		"3. Rounds keyed by reviewer\n   - packages: internal/core/implement/loop, commands\n   - landed: b22578aee474\n"+
		"4. The per-reviewer findings record\n   - packages: internal/core/implement/loop, internal/surface/cli, commands\n   - landed: 904e2a8f59c9\n"+
		"5. The judge strikes false findings only\n   - packages: internal/core/implement/loop, commands\n") +
	"\n## Footprint\n\n- packages: internal/core/implement/loop, internal/core/runner, internal/core/layered, internal/surface/cli, commands\n- tests: the loop\n"

// TestAStartReservesOnlyTheUnlandedStepsPackages: steps 1 to 4 landed, the
// reservation names step 5's packages alone.
func TestAStartReservesOnlyTheUnlandedStepsPackages(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), fiveStepSpec)
	bare := onOrigin(t, repo)
	if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
		t.Fatal(err)
	}
	want := []string{"internal/core/implement/loop", "commands"}
	if fp := footprintOf(t, bare, "itd-10"); !slices.Equal(fp, want) {
		t.Fatalf("the reservation's footprint = %q, want step 5's packages %q", fp, want)
	}
}

// TestTheFootprintLineStandsWhenNoStepNamesPackages: a spec listing no steps,
// one whose steps name no packages, one whose step still to build names none
// beside one that does, and one whose every step landed each reserve the
// `## Footprint` packages line, as before.
func TestTheFootprintLineStandsWhenNoStepNamesPackages(t *testing.T) {
	const footprint = "\n## Footprint\n\n- packages: internal/widget, commands/\n- tests: the widget\n"
	for name, steps := range map[string]string{
		"no steps":          "",
		"no step packages":  "1. One\n   - landed: #1\n2. Two\n   - tests: the widget\n",
		"a step names none": "1. One\n   - packages: internal/widget\n2. Two\n   - tests: the widget\n",
		"every step landed": "1. One\n   - packages: internal/widget\n   - landed: #1\n2. Two\n   - packages: commands\n   - landed: #2\n",
	} {
		t.Run(name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(steps)+footprint)
			got, err := specPackages(repo.Root(), "spc-1", nil)
			if err != nil {
				t.Fatal(err)
			}
			if want := []string{"internal/widget", "commands"}; !slices.Equal(got, want) {
				t.Fatalf("specPackages = %q, want the Footprint line %q", got, want)
			}
		})
	}
}

// TestALandedStepsPackageHeldByAPeerDoesNotRefuseTheStart is the issue's
// case: a file under internal/surface/cli is in another issue's live
// reservation, and only steps that landed name that directory, so the build
// of step 5 is reserved rather than refused as contention.
func TestALandedStepsPackageHeldByAPeerDoesNotRefuseTheStart(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), fiveStepSpec)
	bare := onOrigin(t, repo)
	p2 := otherCopy(t, bare, patEmail)
	if _, err := claim.Reserve(p2, claim.Request{Record: "iss-77", Footprint: []string{"internal/surface/cli/root.go"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
		if r, ok := AsRefusal(err); ok && r.Contention && strings.Contains(r.Reason, "internal/surface/cli") {
			t.Fatalf("the build was refused over a directory only landed steps name: %s", r.Reason)
		}
		t.Fatal(err)
	}
	if fp := footprintOf(t, bare, "itd-10"); slices.Contains(fp, "internal/surface/cli") {
		t.Fatalf("the reservation names internal/surface/cli, which only landed steps touch: %q", fp)
	}
}

// TestAResumeWithEveryStepLandedReservesTheStepsItBuilt: a run that started
// at step 5 and whose step 5 has since landed reserves step 5's packages on a
// resume, never the Footprint line, so the resume does not widen its start's
// hold over a package only steps landed before it touched.
func TestAResumeWithEveryStepLandedReservesTheStepsItBuilt(t *testing.T) {
	allLanded := strings.Replace(fiveStepSpec, "   - packages: internal/core/implement/loop, commands\n\n",
		"   - packages: internal/core/implement/loop, commands\n   - landed: 0123456789ab\n\n", 1)
	if allLanded == fiveStepSpec {
		t.Fatal("the fixture's step 5 was not marked landed")
	}
	repo := loopRepo(t, readyIntent("", settledQuestions), allLanded)
	got, err := specPackages(repo.Root(), "spc-1", []int{5})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"internal/core/implement/loop", "commands"}; !slices.Equal(got, want) {
		t.Fatalf("specPackages over the steps the run built = %q, want %q", got, want)
	}
	got, err = specPackages(repo.Root(), "spc-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(got, "internal/surface/cli") {
		t.Fatalf("with no step built by the run, the Footprint line stands: %q", got)
	}
}

// TestAProseStepPackagesLineReservesTheFootprintLine: a step still to build
// whose packages line holds an entry that names no path cannot say what the
// build touches, so the build reserves the `## Footprint` packages line, as
// the other fallbacks do, rather than refusing over a line the Footprint
// section never held.
func TestAProseStepPackagesLineReservesTheFootprintLine(t *testing.T) {
	steps := "1. One\n   - packages: internal/widget\n   - landed: #1\n" +
		"2. Two\n   - packages: the loop and commands, internal/widget\n"
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(steps)+
		"\n## Footprint\n\n- packages: internal/widget, commands/\n- tests: the widget\n")
	got, err := specPackages(repo.Root(), "spc-1", nil)
	if err != nil {
		t.Fatalf("a prose step line falls back to the Footprint line, never a refusal: %v", err)
	}
	if want := []string{"internal/widget", "commands"}; !slices.Equal(got, want) {
		t.Fatalf("specPackages = %q, want the Footprint line %q", got, want)
	}
}

// memoryRecallSpec is spc-1 in the shape of spc-2610111627213138: seven
// steps landed, step 8's packages line prose ("this spec (...)"), and a
// Footprint line of real paths, each line as that spec writes it.
var memoryRecallSpec = specWithSteps(
	"1. The notes store\n   - packages: internal/core/memnote (new: schema.go)\n   - landed: c8d1ab0df\n"+
		"2. One recall\n   - packages: internal/core/recall (new)\n   - landed: e6d2a7df7\n"+
		"3. Pages join the same recall\n   - packages: internal/core/memory\n   - landed: 57b55ed1f\n"+
		"4. The relevance ranking\n   - packages: internal/core/recall (ranking.go)\n   - landed: f752318ff\n"+
		"5. How full the session is\n   - packages: internal/core/statusline\n   - landed: a5b89898725f\n"+
		"6. Context brackets\n   - packages: internal/core/rules\n   - landed: acb18d1836fc\n"+
		"7. The docs and the brief\n   - packages: docs/explanation\n   - landed: b0ea88c41\n"+
		"8. The live session, the doc-fidelity review and the close\n"+
		"   - packages: this spec (the live captures under a `## Live captures` heading after Steps), .abcd/development/intents (both intents move to shipped by the close)\n") +
	"\n## Footprint\n\n- packages: internal/core/memnote, internal/core/recall, internal/core/rules, internal/core/memory, internal/core/recordid, internal/core/statusline, internal/core/history, internal/core/surface, internal/surface/cli, commands/memory.md, commands/ahoy.md, .abcd/rules.json, .abcd/development/release/surface.json, docs/reference, docs/explanation, docs/how-to, mkdocs.yml, README.md, AGENTS.md, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals, .abcd/development/brief/glossary\n- tests: the recall\n"

// TestTheMemoryRecallSpecsShapeIsReservedWithItsFootprintLine: the build of
// spc-2610111627213138's intent, whose last step's packages line is prose,
// starts and reserves the Footprint line, as it did before steps were read.
func TestTheMemoryRecallSpecsShapeIsReservedWithItsFootprintLine(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), memoryRecallSpec)
	bare := onOrigin(t, repo)
	if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
		t.Fatalf("the build starts: %v", err)
	}
	fp := footprintOf(t, bare, "itd-10")
	for _, want := range []string{"internal/core/memnote", "internal/surface/cli", ".abcd/development/brief/glossary"} {
		if !slices.Contains(fp, want) {
			t.Fatalf("the reservation is the Footprint line, %q missing: %q", want, fp)
		}
	}
	for _, p := range fp {
		if strings.ContainsFunc(p, func(r rune) bool { return r == ' ' }) {
			t.Fatalf("the reservation names a prose entry %q: %q", p, fp)
		}
	}
}

// TestAnUnreadableStepsSectionReservesTheFootprintLine: a steps section that
// cannot be read (here a parked HTML comment in it) says nothing of what is
// left, so the build reserves the Footprint line; the steps check is where
// the section itself is refused.
func TestAnUnreadableStepsSectionReservesTheFootprintLine(t *testing.T) {
	content := specWithSteps("1. One\n   - packages: internal/other\n<!-- 2. Parked\n   - packages: elsewhere -->\n") +
		"\n## Footprint\n\n- packages: internal/widget, commands/\n- tests: the widget\n"
	repo := loopRepo(t, readyIntent("", settledQuestions), content)
	got, err := specPackages(repo.Root(), "spc-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"internal/widget", "commands"}; !slices.Equal(got, want) {
		t.Fatalf("specPackages = %q, want the Footprint line %q", got, want)
	}
}

// TestAResumeReadsTheStepsItsLanesBuilt drives builtSteps through the resume
// path: a run started at step 5 with nothing reserved, whose step 5 has since
// landed, reserves step 5's packages when it is resumed, never the Footprint
// line, since its lane is the step it built.
func TestAResumeReadsTheStepsItsLanesBuilt(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), fiveStepSpec)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.ReservedAt != nil {
		t.Fatalf("with no origin nothing is reserved: %+v", st.ReservedAt)
	}
	if !slices.ContainsFunc(st.Lanes, func(l Lane) bool { return l.SpecStep == 5 }) {
		t.Fatalf("the start opens a lane for step 5: %+v", st.Lanes)
	}
	allLanded := strings.Replace(fiveStepSpec, "   - packages: internal/core/implement/loop, commands\n\n",
		"   - packages: internal/core/implement/loop, commands\n   - landed: 0123456789ab\n\n", 1)
	if allLanded == fiveStepSpec {
		t.Fatal("the fixture's step 5 was not marked landed")
	}
	repo.Write(specRel, allLanded)
	repo.Commit("step 5 lands")
	bare := onOrigin(t, repo)

	again, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil || !again.Resumed || again.RunID != res.RunID {
		t.Fatalf("starting again resumes the run: %+v %v", again, err)
	}
	if fp, want := footprintOf(t, bare, "itd-10"), []string{"internal/core/implement/loop", "commands"}; !slices.Equal(fp, want) {
		t.Fatalf("the resume reserves the step its lane built, %q, not the Footprint line: %q", want, fp)
	}
}

// TestAWalledStageIsDeclaredTheFootprintLineAndTheSteps: the paths declared
// to a walled stage are never narrower than the Footprint line, though the
// reservation narrows to the steps still to build; a step's own path outside
// the line joins them.
func TestAWalledStageIsDeclaredTheFootprintLineAndTheSteps(t *testing.T) {
	content := strings.Replace(fiveStepSpec, "5. The judge strikes false findings only\n   - packages: internal/core/implement/loop, commands\n",
		"5. The judge strikes false findings only\n   - packages: internal/core/implement/loop, commands, internal/core/judge\n", 1)
	if content == fiveStepSpec {
		t.Fatal("the fixture's step 5 was not widened")
	}
	repo := loopRepo(t, readyIntent("", settledQuestions), content)
	st := State{Key: "itd-10", Intent: "itd-10", Spec: "spc-1"}
	got := declaredPaths(repo.Root(), st, "nothing")
	want := []string{"internal/core/implement/loop", "internal/core/runner", "internal/core/layered", "internal/surface/cli", "commands", "internal/core/judge"}
	if !slices.Equal(got, want) {
		t.Fatalf("declared = %q, want the Footprint line and the steps' packages %q", got, want)
	}
}
