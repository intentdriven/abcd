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
