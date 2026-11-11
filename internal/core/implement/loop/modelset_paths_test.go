package loop

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestClassRoutesRefusesWhatTheReviewSetRefuses: a drive reads the class map
// through ClassRoutes before anything else, so it runs the review set's own
// rename checks: an old class name is refused with the whole mapping, from
// either file, and a machine map that sets top without research is refused
// as the review stage refuses it, never as a key nothing reads.
func TestClassRoutesRefusesWhatTheReviewSetRefuses(t *testing.T) {
	machine := abcdhome.Display("config.json")
	cases := []struct {
		name, repo, machine string
		want                []string
	}{
		{name: "an old name in the machine's map", machine: `{"models": {"classes": {"research": "acme/big-1", "middle": "acme/mid-1"}}}`,
			want: append([]string{"models.classes.middle", machine}, renameMapping...)},
		{name: "an old name in the repository's map", repo: `{"models": {"classes": {"below": "acme/mid-1"}}}`,
			want: append([]string{"models.classes.below", ".abcd/config.json"}, renameMapping...)},
		{name: "top without research", machine: `{"models": {"classes": {"top": "acme/big-1"}}}`,
			want: []string{"models.classes.top", "models.classes.research", machine, "strongest", "second"}},
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
			_, err := ClassRoutes(Options{}.roots(repo.Root()))
			r := mustRefusal(t, err)
			if r.Stage != StageReview {
				t.Fatalf("want the review stage's refusal: %+v", r)
			}
			for _, w := range tc.want {
				if !strings.Contains(r.Reason+" "+r.Remedy, w) {
					t.Fatalf("the refusal names %q: %+v", w, r)
				}
			}
		})
	}
}

// TestATopEffortSaysNothingOfTheRename: the lone-top line is for a review
// list or a judge naming top, which cannot show which names it was written
// for. A top key under models.effort can only be in a file written for the
// new names, so it never prompts the line, and it counts as a file naming the
// new vocabulary.
func TestATopEffortSaysNothingOfTheRename(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	machineConfig(t, `{"models": {"effort": {"top": "max"}}}`)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagnostics) != 0 {
		t.Fatalf("a top effort alone says nothing on stderr: %q", res.Diagnostics)
	}

	runsGone(t, repo.Root())
	repoConfig(t, repo, `{"review": {"reviewers": ["top"]}}`)
	if res, err = Start(repo.Root(), "itd-10", Options{}); err != nil {
		t.Fatal(err)
	}
	if len(res.Diagnostics) != 0 {
		t.Fatalf("an effort object keyed by the new names shows the files use them: %q", res.Diagnostics)
	}
}

// TestALongRoleKeyCannotLengthenTheRefusal: a per-role review list keyed by
// a role the set does not govern is refused naming the role bounded, however
// long the key the file holds, and an old class name under it is not echoed
// with the key first.
func TestALongRoleKeyCannotLengthenTheRefusal(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	role := strings.Repeat("r", 20000)
	repoConfig(t, repo, `{"review": {"reviewers": {"`+role+`": ["middle"]}}}`)
	_, err := Start(repo.Root(), "itd-10", Options{})
	r := mustRefusal(t, err)
	if strings.Contains(r.Reason+r.Remedy, role) {
		t.Fatal("the refusal echoes the whole role key")
	}
	if n := len(r.Reason) + len(r.Remedy); n > 2000 {
		t.Fatalf("the refusal is %d bytes long", n)
	}
	runTierAbsent(t, repo.Root())
}
