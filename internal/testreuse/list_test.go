package testreuse

import (
	"context"
	"slices"
	"strings"
	"testing"
)

// TestTheListRefusesALineItCannotRead: a declaration that cannot be read must
// not become a test that is reused, so one bad line refuses the whole list.
func TestTheListRefusesALineItCannotRead(t *testing.T) {
	good := "# a comment\n\ninternal/core/x # reads a file a subprocess opens\ninternal/core/y/... TestA TestB # builds the binary\n"
	list, err := ParseList(good)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Pattern != "internal/core/x" || list[0].Reason != "reads a file a subprocess opens" ||
		!slices.Equal(list[1].Tests, []string{"TestA", "TestB"}) {
		t.Fatalf("parsed %+v", list)
	}
	for _, bad := range []string{
		"internal/core/x\n",                // no reason
		"internal/core/x #\n",              // an empty reason
		"# fine\n../outside # up\n",        // a path out of the module
		"internal/core/x notATest # why\n", // not a test name
		"# why alone\n #why\n bad path$ # why\n",
	} {
		if _, err := ParseList(bad); err == nil {
			t.Errorf("the list took %q", bad)
		}
	}
}

// TestTheListIsReadFromTheDefaultBranchNeverTheWorkingTree: a change that takes
// its own test off the list in its own diff changes nothing about this check,
// and a repository with no list on its default branch has none.
func TestTheListIsReadFromTheDefaultBranchNeverTheWorkingTree(t *testing.T) {
	r := newGitRepo(t)
	r.write("a.txt", "a\n")
	r.commit("one")
	if _, _, err := ReadList(context.Background(), r.root); err == nil {
		t.Fatal("a repository with no origin read as having a list")
	}
	r.git("update-ref", "refs/remotes/origin/main", "HEAD")
	if _, _, err := ReadList(context.Background(), r.root); err == nil || !strings.Contains(err.Error(), ListPath) {
		t.Fatalf("a default branch holding no list: %v", err)
	}

	r.write(ListPath, "internal/core/x # a subprocess reads it\n")
	r.commit("the list")
	r.git("update-ref", "refs/remotes/origin/main", "HEAD")
	r.git("checkout", "-q", "-b", "change")
	r.write(ListPath, "# nothing is declared here\n")
	r.commit("take the test off the list")

	list, ref, err := ReadList(context.Background(), r.root)
	if err != nil {
		t.Fatal(err)
	}
	if ref != "origin/main" || len(list) != 1 || list[0].Pattern != "internal/core/x" {
		t.Errorf("read %+v from %s; want the default branch's one entry", list, ref)
	}

	r.git("checkout", "-q", "main")
	r.write(ListPath, "this line does not read\n")
	r.commit("a broken list")
	r.git("update-ref", "refs/remotes/origin/main", "HEAD")
	if _, _, err := ReadList(context.Background(), r.root); err == nil {
		t.Error("a list that does not read was taken")
	}
}

// TestThePlanRoutesEveryPackageOnce: declared and detected packages rerun whole,
// declared and detected tests split their package, everything else is reused,
// and no package is dropped or named twice.
func TestThePlanRoutesEveryPackageOnce(t *testing.T) {
	pkgs := []string{"m/pure", "m/listed", "m/listedtest", "m/tree/one", "m/detected", "m/whole", "m/notests"}
	rels := map[string]string{}
	for _, p := range pkgs {
		rels[p] = strings.TrimPrefix(p, "m/")
	}
	verdicts := map[string]Verdict{
		"m/pure":       {Tests: map[string]string{}},
		"m/listed":     {Tests: map[string]string{"TestFound": whyTree}},
		"m/listedtest": {Tests: map[string]string{}},
		"m/tree/one":   {Tests: map[string]string{}},
		"m/detected":   {Tests: map[string]string{"TestFound": whyProgram}},
		"m/whole":      {Whole: "TestMain " + whyProgram},
	}
	list := List{
		{Pattern: "listed", Reason: "a subprocess reads it"},
		{Pattern: "listedtest", Tests: []string{"TestDeclared"}, Reason: "the network"},
		{Pattern: "tree/...", Reason: "the clock"},
	}
	plan := MakePlan(pkgs, rels, verdicts, list, false)

	if !slices.Equal(plan.Cached, []string{"m/pure", "m/notests"}) {
		t.Errorf("reused: %v", plan.Cached)
	}
	var rerun, split []string
	for _, r := range plan.Rerun {
		rerun = append(rerun, r.Path)
	}
	for _, r := range plan.Split {
		split = append(split, r.Path)
	}
	if !slices.Equal(rerun, []string{"m/listed", "m/tree/one", "m/whole"}) {
		t.Fatalf("rerun whole: %v", rerun)
	}
	if !slices.Equal(split, []string{"m/listedtest", "m/detected"}) {
		t.Fatalf("split: %v", split)
	}
	if got := plan.Rerun[0].Rerun; !strings.Contains(got, "always-rerun list") {
		t.Errorf("a listed package's reason: %q", got)
	}
	if got := plan.Rerun[2].Rerun; !strings.Contains(got, "undeclared") {
		t.Errorf("an undeclared package's reason should say so: %q", got)
	}
	if got := plan.Split[1].Tests["TestFound"]; !strings.Contains(got, "undeclared") {
		t.Errorf("an undeclared test's reason should say so: %q", got)
	}

	narrowed := MakePlan(pkgs, rels, verdicts, list, true)
	if len(narrowed.Split) != 0 || len(narrowed.Rerun) != 5 {
		t.Errorf("under the stage's own -run, a package with any test to rerun reruns whole: %+v", narrowed)
	}
}

// TestAPatternMatchesAPackageNotItsNamesake: a line names one package, or a
// tree of them with /..., and never one whose path only starts the same.
func TestAPatternMatchesAPackageNotItsNamesake(t *testing.T) {
	one := Declared{Pattern: "internal/core"}
	if !one.matches("internal/core") || one.matches("internal/core/x") || one.matches("internal/corex") {
		t.Error("a plain pattern matched wrongly")
	}
	tree := Declared{Pattern: "internal/core/..."}
	if !tree.matches("internal/core") || !tree.matches("internal/core/x/y") || tree.matches("internal/corex") {
		t.Error("a /... pattern matched wrongly")
	}
}

// TestExactlyMatchesOnlyTheNames: the pattern a split package's lines share
// matches the whole name, so TestA is not TestAB.
func TestExactlyMatchesOnlyTheNames(t *testing.T) {
	if got := exactly([]string{"TestA", "TestB"}); got != "^(TestA|TestB)$" {
		t.Errorf("got %q", got)
	}
}

// TestALineThatNamesNothingIsNamed: a line whose package or test is gone holds
// nothing to rerun, which is a test leaving the list by a rename.
func TestALineThatNamesNothingIsNamed(t *testing.T) {
	known := map[string][]string{"internal/a": {"TestOne"}, "internal/b/c": {"TestTwo"}}
	for _, ok := range []List{
		{{Pattern: "internal/a"}},
		{{Pattern: "internal/a", Tests: []string{"TestOne"}}},
		{{Pattern: "internal/b/..."}},
	} {
		if bad := unresolved(ok, known); len(bad) != 0 {
			t.Errorf("%+v read as unresolved: %v", ok, bad)
		}
	}
	for _, gone := range []List{
		{{Pattern: "internal/gone"}},
		{{Pattern: "internal/gone/..."}},
		{{Pattern: "internal/a", Tests: []string{"TestGone"}}},
	} {
		if bad := unresolved(gone, known); len(bad) != 1 {
			t.Errorf("%+v read as %v", gone, bad)
		}
	}
}
