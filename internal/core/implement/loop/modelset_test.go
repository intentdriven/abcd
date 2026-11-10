package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The model set is the person's, in classes named for their use
// (itd-2611101419006280, spc-2611101423523725): the four classes take the
// names research, top, default and small, and a configuration naming an old
// class is refused with the whole mapping.

// renameMapping is what every refusal of an old class name says.
var renameMapping = []string{"`top` is now `research`", "`middle` is `top`", "`below` is `default`", "`smallest` is `small`"}

// TestTheBundledMapCarriesTheFourNewClasses: the classes are research, top,
// default and small, strongest first, and the bundled map holds a route for
// each.
func TestTheBundledMapCarriesTheFourNewClasses(t *testing.T) {
	if want := []string{"research", "top", "default", "small"}; !slices.Equal(Classes, want) {
		t.Fatalf("the classes are %v, want %v", Classes, want)
	}
	for _, c := range Classes {
		if !validRoute(BundledClasses[c]) {
			t.Fatalf("the bundled map holds no route for %s: %q", c, BundledClasses[c])
		}
	}
	if BundledClasses[ClassResearch] != "anthropic/claude-fable-5" || BundledClasses[ClassSmall] != "anthropic/claude-haiku-4-5" {
		t.Fatalf("the strongest and the smallest keep their models: %v", BundledClasses)
	}
}

// TestAnOldClassNameIsRefusedWithTheMapping: middle, below or smallest in a
// class map, a review list or a judge is refused at the review stage naming
// the key, the file and the new name of each old class, and nothing is
// written.
func TestAnOldClassNameIsRefusedWithTheMapping(t *testing.T) {
	machine := abcdhome.Display("config.json")
	cases := []struct {
		name, repo, machine string
		want                []string
	}{
		{name: "a machine class map", machine: `{"models": {"classes": {"research": "acme/big-1", "middle": "acme/mid-1"}}}`,
			want: []string{"models.classes.middle", machine}},
		{name: "a repository class map", repo: `{"models": {"classes": {"smallest": "acme/tiny-1"}}}`,
			want: []string{"models.classes.smallest", ".abcd/config.json"}},
		{name: "a review list", repo: `{"review": {"reviewers": ["default", "below"]}}`,
			want: []string{"review.reviewers", `"below"`, ".abcd/config.json"}},
		{name: "a per-role list", machine: `{"review": {"reviewers": {"security-reviewer": ["smallest"]}}}`,
			want: []string{"review.reviewers.security-reviewer", `"smallest"`, machine}},
		{name: "a judge", repo: `{"review": {"judge": "middle"}}`,
			want: []string{"review.judge", `"middle"`, ".abcd/config.json"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			if tc.repo != "" {
				repoConfig(t, repo, tc.repo)
			}
			if tc.machine != "" {
				machineConfig(t, tc.machine)
			}
			_, err := Start(repo.Root(), "itd-10", Options{})
			r := mustRefusal(t, err)
			if r.Stage != StageReview {
				t.Fatalf("want the review stage's refusal: %+v", r)
			}
			for _, w := range append(tc.want, renameMapping...) {
				if !strings.Contains(r.Reason+" "+r.Remedy, w) {
					t.Fatalf("the refusal names %q: %+v", w, r)
				}
			}
			runTierAbsent(t, repo.Root())
		})
	}
}

// TestAMachineMapSettingTopWithoutResearchIsRefused: an old file can never
// hold research, so a map that sets top alone is refused, saying top once
// named the strongest class; a map that sets both resolves.
func TestAMachineMapSettingTopWithoutResearchIsRefused(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	machineConfig(t, `{"models": {"classes": {"top": "acme/big-1"}}}`)
	_, err := Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	for _, w := range []string{"models.classes.top", "models.classes.research", abcdhome.Display("config.json"), "strongest", "second"} {
		if !strings.Contains(r.Reason+" "+r.Remedy, w) {
			t.Fatalf("the refusal names %q: %+v", w, r)
		}
	}
	runTierAbsent(t, repo.Root())

	machineConfig(t, `{"models": {"classes": {"top": "acme/big-1", "research": "acme/huge-1"}}}`)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagnostics) != 0 {
		t.Fatalf("a map naming both says nothing on stderr: %q", res.Diagnostics)
	}
}

// TestATopInAReviewListIsAnnouncedUntilANewNameIsUsed: a review list naming
// top, while no configuration file names research, default or small, resolves
// with one line saying which model top now stands for and that the strongest
// class is research; once any file names a new class it says nothing.
func TestATopInAReviewListIsAnnouncedUntilANewNameIsUsed(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	repoConfig(t, repo, `{"review": {"reviewers": ["top"]}}`)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r := reviewRole(t, res.Review, RoleRuthless); strings.Join(classesOf(r), ",") != ClassTop {
		t.Fatalf("top reads as the new top: %+v", r)
	}
	if len(res.Diagnostics) != 1 {
		t.Fatalf("one line on stderr, got %q", res.Diagnostics)
	}
	d := res.Diagnostics[0]
	for _, w := range []string{"review.reviewers", ".abcd/config.json", BundledClasses[ClassTop], "research", BundledClasses[ClassResearch]} {
		if !strings.Contains(d, w) {
			t.Fatalf("the line names %q: %q", w, d)
		}
	}

	runsGone(t, repo.Root())
	machineConfig(t, `{"models": {"classes": {"small": "acme/tiny-1"}}}`)
	res, err = Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagnostics) != 0 {
		t.Fatalf("with a new name in a file, nothing is said: %q", res.Diagnostics)
	}
}

// runsGone removes every run of the checkout, so the next start is a fresh
// run that reads the configuration again.
func runsGone(t *testing.T, root string) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(RunRelDir))); err != nil {
		t.Fatal(err)
	}
}
