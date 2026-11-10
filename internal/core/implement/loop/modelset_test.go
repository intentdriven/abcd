package loop

import (
	"fmt"
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

// judgedRunAtReview is a run started with a calibrated judge at the top
// class, stepped to its first reviewer: its state carries every stored class
// label, in the review set, the judge and the await.
func judgedRunAtReview(t *testing.T) (repoRoot, runID string, data []byte) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	repoConfig(t, repo, `{"review": {"judge": "top"}}`)
	res, err := Start(repo.Root(), "itd-10", Options{judgeCalibrated: func() (bool, string) { return true, "" }})
	if err != nil {
		t.Fatal(err)
	}
	implemented(t, repo, res.RunID, DefaultStages(), "one.txt")
	stepAwait(t, repo, res.RunID, DefaultStages(), Options{})
	return repo.Root(), res.RunID, stateBytes(t, repo.Root(), res.RunID)
}

// TestAnEarlierStateTakesTheNewClassNames: a state written before the rename
// is migrated by its version: each stored label takes its new name, each
// keeps the model it was resolved to, a stored judge gains its status, and
// the run goes on with the reviewers it started with.
func TestAnEarlierStateTakesTheNewClassNames(t *testing.T) {
	root, id, data := judgedRunAtReview(t)
	current, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	earlier := olderState(t, data, schemaVersionUnrenamed)
	for _, w := range []string{`"class": "top"`, `"class": "middle"`, `"class": "below"`} {
		if !strings.Contains(string(earlier), w) {
			t.Fatalf("the earlier state carries %s:\n%s", w, earlier)
		}
	}
	for _, w := range []string{`"research"`, `"default"`, `"status"`} {
		if strings.Contains(string(earlier), w) {
			t.Fatalf("the earlier state carries no %s:\n%s", w, earlier)
		}
	}
	path := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
	if err := os.WriteFile(path, earlier, 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if st.SchemaVersion != SchemaVersion {
		t.Fatalf("read at the current version: %d", st.SchemaVersion)
	}
	if j := st.Review.Judge; j == nil || j.Class != ClassTop || j.Model != current.Review.Judge.Model || j.Status != JudgeActive {
		t.Fatalf("the judge takes top, keeps its model and is active: %+v", j)
	}
	for i, r := range st.Review.Roles {
		for k, c := range r.Reviewers {
			was := current.Review.Roles[i].Reviewers[k]
			if c.Class != was.Class || c.Model != was.Model {
				t.Fatalf("%s reviewer %d reads as %+v, want %+v", r.Role, k, c, was)
			}
		}
	}
	aw := st.Lanes[0].Awaits
	if len(aw) == 0 || aw[0].Class != current.Lanes[0].Awaits[0].Class || aw[0].Model != current.Lanes[0].Awaits[0].Model {
		t.Fatalf("the await keeps its reviewer: %+v", aw)
	}
	if _, err := advance(root, id, DefaultStages(), Options{}); err != nil {
		t.Fatalf("the migrated run steps: %v", err)
	}
}

// TestAnEarlierStateCarryingWhatItNeverWroteIsRefused: a version-24 file
// carrying a judge's status, or a class label outside the four it named, is
// not one its version wrote, and is refused.
func TestAnEarlierStateCarryingWhatItNeverWroteIsRefused(t *testing.T) {
	root, id, data := judgedRunAtReview(t)
	earlier := string(olderState(t, data, schemaVersionUnrenamed))
	path := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
	for name, edit := range map[string]func(string) string{
		"a judge's status": func(s string) string {
			return strings.Replace(s, `"class": "middle",`, `"class": "middle", "status": "active",`, 1)
		},
		"a new class name": func(s string) string { return strings.Replace(s, `"class": "below"`, `"class": "default"`, 1) },
		"no class at all":  func(s string) string { return strings.Replace(s, `"class": "below"`, `"class": "huge"`, 1) },
	} {
		t.Run(name, func(t *testing.T) {
			edited := edit(earlier)
			if edited == earlier {
				t.Fatalf("the edit changed nothing:\n%s", earlier)
			}
			if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := ReadState(root, id)
			if r := mustRefusal(t, err); r.Stage != "state" {
				t.Fatalf("want the state's refusal: %+v", r)
			}
		})
	}
}

// TestTheRenameMovesTheStateOneVersion: the rename is version 25's, and a
// file of the version after it is refused as a newer abcd's, the refusal an
// abcd before the rename gives a version-25 file.
func TestTheRenameMovesTheStateOneVersion(t *testing.T) {
	if SchemaVersion != 25 || schemaVersionUnrenamed != 24 {
		t.Fatalf("the rename is version 25's: %d %d", SchemaVersion, schemaVersionUnrenamed)
	}
	_, err := decodeState([]byte(fmt.Sprintf(`{"schema_version": %d}`, SchemaVersion+1)))
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "a newer abcd wrote") {
		t.Fatalf("a newer state is refused as a newer abcd's: %+v", r)
	}
}

// judgeNotes are the run record's lines about the judge.
func judgeNotes(st State) []string {
	var out []string
	for _, e := range st.Record {
		if e.Stage == StageReview && strings.HasPrefix(e.Note, "the judge") {
			out = append(out, e.Note)
		}
	}
	return out
}

// TestTheBundledJudgeWaitsForTheWholeRun: with no calibrated judge agent a
// run starts with the bundled judge waiting, says so in one record line, and
// keeps it waiting whatever is calibrated later; no step hands out or awaits a
// judge, and no estimate counts one.
func TestTheBundledJudgeWaitsForTheWholeRun(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	notes := judgeNotes(st)
	if len(notes) != 1 || !strings.Contains(notes[0], "waits") || !strings.Contains(notes[0], "calibration") ||
		!strings.Contains(notes[0], "stricter verdict") {
		t.Fatalf("one record line says the judge waits on calibration: %q", notes)
	}
	calibrated := Options{judgeCalibrated: func() (bool, string) { return true, "" }}
	again, err := Start(repo.Root(), "itd-10", calibrated)
	if err != nil || !again.Resumed || again.Review.Judge.Status != JudgeWaiting {
		t.Fatalf("a resume keeps the judge it started with: %+v %v", again.Review, err)
	}
	implemented(t, repo, res.RunID, DefaultStages(), "one.txt")
	for i := 0; i < 2; i++ {
		a := stepAwait(t, repo, res.RunID, DefaultStages(), calibrated)
		if a.Role != RoleRuthless && a.Role != RoleSecurity && a.Role != RoleAuditor {
			t.Fatalf("the round hands out only its validators: %+v", a)
		}
	}
	if st, _ = ReadState(repo.Root(), res.RunID); st.Review.Judge.Status != JudgeWaiting {
		t.Fatalf("the judge still waits: %+v", st.Review.Judge)
	}
	runs, err := EstimateRuns(Options{}, repo.Root(), 1)
	if err != nil {
		t.Fatal(err)
	}
	unjudged := *st.Review
	unjudged.Judge = nil
	want := 0
	for _, s := range estimateShares(nil, 1, false, &unjudged) {
		want += s.runs
	}
	if runs != want {
		t.Fatalf("a waiting judge is counted in no estimate: %d runs, %d without it", runs, want)
	}
}

// TestACalibratedBundledJudgeIsActive: once a judge agent has cleared the
// gate (the fixture seam), a new run starts with the bundled judge active,
// and its record says a judge will run on a disagreement.
func TestACalibratedBundledJudgeIsActive(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	res, err := Start(repo.Root(), "itd-10", Options{judgeCalibrated: func() (bool, string) { return true, "" }})
	if err != nil {
		t.Fatal(err)
	}
	if j := res.Review.Judge; j == nil || j.Class != ClassTop || j.Status != JudgeActive || !j.Active() {
		t.Fatalf("the bundled judge is active: %+v", j)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if notes := judgeNotes(st); len(notes) != 1 || !strings.Contains(notes[0], "will run") || !strings.Contains(notes[0], "disagree") {
		t.Fatalf("one record line says a judge will run on a disagreement: %q", notes)
	}
}

// effortsOf is each class of a review set with the effort it carries and the
// layer that set it, the judge last: "research=high@bundled".
func effortsOf(rs *ReviewSet) []string {
	var out []string
	for _, r := range rs.Roles {
		for _, c := range r.Reviewers {
			out = append(out, c.Class+"="+c.Effort+"@"+c.EffortLayer)
		}
	}
	if j := rs.Judge; j != nil {
		out = append(out, "judge "+j.Class+"="+j.Effort+"@"+j.EffortLayer)
	}
	return out
}

// TestEachClassCarriesAnEffort: with nothing configured every class runs at
// high from the bundled layer; one level in the machine's file applies to
// every class; an object applies per class, and a class it leaves out is
// high; the resolved effort is stored with each class and survives a re-read.
func TestEachClassCarriesAnEffort(t *testing.T) {
	machine := abcdhome.Display("config.json")
	cases := []struct {
		name, machine string
		want          string
	}{
		{name: "nothing configured",
			want: "research=high@bundled default=high@bundled research=high@bundled default=high@bundled judge top=high@bundled"},
		{name: "one level", machine: `{"models": {"effort": "max"}}`,
			want: "research=max@machine default=max@machine research=max@machine default=max@machine judge top=max@machine"},
		{name: "a level per class", machine: `{"models": {"effort": {"research": "xhigh", "default": "low"}}}`,
			want: "research=xhigh@machine default=low@machine research=xhigh@machine default=low@machine judge top=high@bundled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			if tc.machine != "" {
				machineConfig(t, tc.machine)
			}
			res, err := Start(repo.Root(), "itd-10", Options{})
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(effortsOf(res.Review), " "); got != tc.want {
				t.Fatalf("efforts = %s, want %s", got, tc.want)
			}
			if c := res.Review.Roles[0].Reviewers[0]; c.EffortLayer == "machine" && c.EffortOrigin != machine {
				t.Fatalf("a machine effort names the machine's file: %+v", c)
			}
			st, err := ReadState(repo.Root(), res.RunID)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(effortsOf(st.Review), " "); got != tc.want {
				t.Fatalf("the state keeps the efforts: %s", got)
			}
		})
	}
}

// TestAnEffortOutsideTheLevelsIsRefused: an effort that is not one of the
// five levels, or an object keyed by anything but the four classes, is
// refused before anything is written, naming the key, the value and what is
// accepted; an old class name there is refused with the mapping.
func TestAnEffortOutsideTheLevelsIsRefused(t *testing.T) {
	levels := "low, medium, high, xhigh and max"
	cases := []struct {
		name, machine string
		want          []string
	}{
		{name: "an unknown level", machine: `{"models": {"effort": "extreme"}}`, want: []string{"models.effort", `"extreme"`, levels}},
		{name: "an unknown level for a class", machine: `{"models": {"effort": {"default": "turbo"}}}`,
			want: []string{"models.effort.default", `"turbo"`, levels}},
		{name: "an unknown class", machine: `{"models": {"effort": {"huge": "high"}}}`,
			want: []string{"models.effort", `"huge"`, "research, top, default, small"}},
		{name: "an old class", machine: `{"models": {"effort": {"middle": "high"}}}`,
			want: append([]string{"models.effort.middle"}, renameMapping...)},
		{name: "not a level or an object", machine: `{"models": {"effort": 3}}`, want: []string{"models.effort", "3", levels}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			machineConfig(t, tc.machine)
			_, err := Start(repo.Root(), "itd-10", Options{})
			r := mustRefusal(t, err)
			if r.Stage != StageReview {
				t.Fatalf("want the review stage's refusal: %+v", r)
			}
			for _, w := range tc.want {
				if !strings.Contains(r.Reason+" "+r.Remedy, w) {
					t.Fatalf("the refusal names %q: %+v", w, r)
				}
			}
			runTierAbsent(t, repo.Root())
		})
	}
}

// TestARepositoryEffortIsSkipped: a repository must not raise what a review
// spends, so its models.effort is skipped with one line naming the machine's
// file, and every class runs at the bundled high.
func TestARepositoryEffortIsSkipped(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	repoConfig(t, repo, `{"models": {"effort": "max"}}`)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(effortsOf(res.Review), " "); strings.Contains(got, "max") {
		t.Fatalf("the repository's effort changes nothing: %s", got)
	}
	if len(res.Diagnostics) != 1 || !strings.Contains(res.Diagnostics[0], "models.effort") ||
		!strings.Contains(res.Diagnostics[0], abcdhome.Display("config.json")) || !strings.Contains(res.Diagnostics[0], "skipped") {
		t.Fatalf("one line names the machine's file: %q", res.Diagnostics)
	}
}

// TestAStoredEffortTheLoopNeverWritesIsRefused: an effort read back from the
// state is one of the five levels from the machine or the bundled layer, and
// an earlier version's file carrying one is not one its version wrote.
func TestAStoredEffortTheLoopNeverWritesIsRefused(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(res.RunID)))
	orig := stateBytes(t, repo.Root(), res.RunID)
	earlier := string(olderState(t, orig, schemaVersionUnrenamed))
	for name, edit := range map[string]func() string{
		"a level it never writes": func() string { return strings.Replace(string(orig), `"effort": "high"`, `"effort": "turbo"`, 1) },
		"a layer it never writes": func() string {
			return strings.Replace(string(orig), `"effort_layer": "bundled"`, `"effort_layer": "repo"`, 1)
		},
		"an effort in an earlier version": func() string {
			return strings.Replace(earlier, `"map_layer": "bundled",`, `"map_layer": "bundled", "effort": "high",`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			edited := edit()
			if edited == string(orig) || edited == earlier {
				t.Fatalf("the edit changed nothing:\n%s", orig)
			}
			if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := ReadState(repo.Root(), res.RunID)
			if r := mustRefusal(t, err); r.Stage != "state" {
				t.Fatalf("want the state's refusal: %+v", r)
			}
		})
	}
}
