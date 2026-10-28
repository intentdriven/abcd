package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The review set and the class map (itd-2610240922461281, spc-2610240951109420
// step 1): review.reviewers, review.judge and review.max_reviewers resolved
// once when a run starts, the class map models.classes read from the machine
// or the bundled default alone, and the set kept in the run's state.

func reviewRole(t *testing.T, rs *ReviewSet, role string) RoleReviewers {
	t.Helper()
	if rs == nil {
		t.Fatal("the run carries no review set")
	}
	for _, r := range rs.Roles {
		if r.Role == role {
			return r
		}
	}
	t.Fatalf("the review set names no %s: %+v", role, *rs)
	return RoleReviewers{}
}

func classesOf(r RoleReviewers) []string {
	out := make([]string, 0, len(r.Reviewers))
	for _, c := range r.Reviewers {
		out = append(out, c.Class)
	}
	return out
}

func reviewRecord(t *testing.T, st State) string {
	t.Helper()
	for _, e := range st.Record {
		if e.Stage == StageReview {
			return e.Note
		}
	}
	t.Fatalf("the run record names no review set: %+v", st.Record)
	return ""
}

// TestTheBundledReviewSetIsTwoReviewersPerRole: with no review set in any
// layer, each reviewer role runs twice, at the middle class and the class
// below it, each class named with the model the bundled map holds for it,
// and the payload, the state and the record name the bundled layer.
func TestTheBundledReviewSetIsTwoReviewersPerRole(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, rs := range []*ReviewSet{res.Review} {
		for _, role := range []string{RoleRuthless, RoleSecurity} {
			r := reviewRole(t, rs, role)
			if got := strings.Join(classesOf(r), ","); got != "middle,below" || r.Layer != "bundled" || r.Origin != "bundled" {
				t.Fatalf("%s = %+v, want middle,below from the bundled layer", role, r)
			}
			if r.Reviewers[0].Model != BundledClasses[ClassMiddle] || r.Reviewers[1].Model != BundledClasses[ClassBelow] ||
				r.Reviewers[0].MapLayer != "bundled" {
				t.Fatalf("each class names the model the bundled map holds: %+v", r.Reviewers)
			}
		}
		if rs.Judge != nil || rs.MaxReviewers.Value != 2 || rs.MaxReviewers.Layer != "bundled" {
			t.Fatalf("no judge and a ceiling of two by default: %+v", *rs)
		}
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(classesOf(reviewRole(t, st.Review, RoleSecurity)), ","); got != "middle,below" {
		t.Fatalf("the state keeps the set: %s", got)
	}
	note := reviewRecord(t, st)
	for _, want := range []string{"middle (" + BundledClasses[ClassMiddle] + ")", "below (" + BundledClasses[ClassBelow] + ")", "bundled", "no judge"} {
		if !strings.Contains(note, want) {
			t.Fatalf("the record names %q: %q", want, note)
		}
	}
	if len(res.Diagnostics) != 0 {
		t.Fatalf("the bundled set says nothing on stderr: %q", res.Diagnostics)
	}
}

// TestTheReviewSetPayloadNamesEachValuesLayer: a machine list, a repository
// list for one role and a machine class each name their own layer and file.
func TestTheReviewSetPayloadNamesEachValuesLayer(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	machineConfig(t, `{"review": {"reviewers": ["top"]}, "models": {"classes": {"top": "acme/big-1"}}}`)
	repoConfig(t, repo, `{"review": {"reviewers": {"security-reviewer": ["middle", "smallest"]}}}`)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	ruth := reviewRole(t, res.Review, RoleRuthless)
	if strings.Join(classesOf(ruth), ",") != "top" || ruth.Layer != "machine" || ruth.Origin != abcdhome.Display("config.json") {
		t.Fatalf("the ruthless reviewer falls to the machine's list: %+v", ruth)
	}
	if c := ruth.Reviewers[0]; c.Model != "acme/big-1" || c.MapLayer != "machine" || c.MapOrigin != abcdhome.Display("config.json") {
		t.Fatalf("the machine's class names its model and file: %+v", c)
	}
	sec := reviewRole(t, res.Review, RoleSecurity)
	if strings.Join(classesOf(sec), ",") != "middle,smallest" || sec.Layer != "repo" || sec.Origin != ".abcd/config.json" {
		t.Fatalf("the security reviewer takes the repository's list: %+v", sec)
	}
	if c := sec.Reviewers[1]; c.Model != BundledClasses[ClassSmallest] || c.MapLayer != "bundled" {
		t.Fatalf("a class the machine leaves out is the bundled map's: %+v", c)
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["review"]; !ok {
		t.Fatalf("the payload carries review beside pace: %s", raw)
	}
	if _, ok := payload["diagnostics"]; ok {
		t.Fatalf("the diagnostics go to stderr, not the payload: %s", raw)
	}
}

// TestARepositoryListAboveTheMachinesCeilingIsClamped: the repository's
// count applies only up to the machine's ceiling, and the clamp is one line
// naming both files.
func TestARepositoryListAboveTheMachinesCeilingIsClamped(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	machineConfig(t, `{"review": {"max_reviewers": 1}}`)
	repoConfig(t, repo, `{"review": {"reviewers": ["below", "middle"]}}`)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{RoleRuthless, RoleSecurity} {
		r := reviewRole(t, res.Review, role)
		if strings.Join(classesOf(r), ",") != "below" || r.Clamped != 2 {
			t.Fatalf("%s is clamped to the machine's one: %+v", role, r)
		}
	}
	if res.Review.MaxReviewers.Value != 1 || res.Review.MaxReviewers.Layer != "machine" {
		t.Fatalf("the ceiling names the machine: %+v", res.Review.MaxReviewers)
	}
	if len(res.Diagnostics) != 1 {
		t.Fatalf("one line on stderr, got %q", res.Diagnostics)
	}
	d := res.Diagnostics[0]
	if !strings.Contains(d, ".abcd/config.json") || !strings.Contains(d, abcdhome.Display("config.json")) || !strings.Contains(d, "review.max_reviewers") {
		t.Fatalf("the clamp names both files: %q", d)
	}

	// With no machine ceiling the bundled two applies, and a list of two is
	// not clamped.
	machineConfig(t, `{}`)
	os.RemoveAll(filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir)))
	res, err = Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if r := reviewRole(t, res.Review, RoleRuthless); strings.Join(classesOf(r), ",") != "below,middle" || r.Clamped != 0 || len(res.Diagnostics) != 0 {
		t.Fatalf("two fit the bundled ceiling: %+v %q", r, res.Diagnostics)
	}
}

// TestTheRepositoryCannotSetTheCeilingOrTheClassMap: the ceiling and the class
// map are the machine's; a repository value of either is skipped with a
// warning naming the machine's file, and the run starts on the rest.
func TestTheRepositoryCannotSetTheCeilingOrTheClassMap(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	repoConfig(t, repo, `{"review": {"max_reviewers": 1}, "models": {"classes": {"middle": "acme/cheap-1"}}}`)
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	r := reviewRole(t, res.Review, RoleRuthless)
	if strings.Join(classesOf(r), ",") != "middle,below" || r.Reviewers[0].Model != BundledClasses[ClassMiddle] {
		t.Fatalf("the repository's ceiling and map change nothing: %+v", r)
	}
	if res.Review.MaxReviewers.Layer != "bundled" {
		t.Fatalf("the ceiling stays the bundled one: %+v", res.Review.MaxReviewers)
	}
	joined := strings.Join(res.Diagnostics, "\n")
	if len(res.Diagnostics) != 2 || !strings.Contains(joined, "models.classes") || !strings.Contains(joined, "review.max_reviewers") ||
		strings.Count(joined, abcdhome.Display("config.json")) < 2 || !strings.Contains(joined, "skipped") {
		t.Fatalf("each is skipped with a warning naming the machine's file: %q", res.Diagnostics)
	}
}

// TestAnUnknownClassOrAnUncalibratedJudgeIsRefused: a class the map does not
// carry, and a judge no calibration has cleared, are refused at the review
// stage before anything is written; a list past two, or empty, and a class
// route that is not <provider>/<model> are refused there too.
func TestAnUnknownClassOrAnUncalibratedJudgeIsRefused(t *testing.T) {
	cases := []struct {
		name, repo, machine string
		want                []string
	}{
		{name: "unknown class", repo: `{"review": {"reviewers": ["middle", "huge"]}}`,
			want: []string{`"huge"`, "models.classes", abcdhome.Display("config.json")}},
		{name: "unknown judge class", machine: `{"review": {"judge": "giant"}}`,
			want: []string{`"giant"`, "models.classes"}},
		{name: "uncalibrated judge", repo: `{"review": {"judge": "top"}}`,
			want: []string{"review.judge", "itd-81", "calibration"}},
		{name: "three reviewers", repo: `{"review": {"reviewers": ["top", "middle", "below"]}}`,
			want: []string{"review.reviewers", "one or two"}},
		{name: "no reviewer", repo: `{"review": {"reviewers": []}}`,
			want: []string{"review.reviewers", "one or two"}},
		{name: "unknown role", repo: `{"review": {"reviewers": {"intent-auditor": ["top"]}}}`,
			want: []string{"intent-auditor", RoleRuthless}},
		{name: "misspelt key", repo: `{"review": {"reviewer": ["top"]}}`,
			want: []string{"review.reviewer"}},
		{name: "ceiling past two", machine: `{"review": {"max_reviewers": 3}}`,
			want: []string{"review.max_reviewers", "3"}},
		{name: "a route with no provider", machine: `{"models": {"classes": {"below": "sonnet"}}}`,
			want: []string{"models.classes.below", "<provider>/<model>"}},
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
			if r.Stage != StageReview || r.Contention {
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

// TestACalibratedJudgeJoinsTheSet: once the judge's calibration clears (the
// fixture seam), the judge is kept in the set with its class, model and layer.
func TestACalibratedJudgeJoinsTheSet(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	repoConfig(t, repo, `{"review": {"judge": "top"}}`)
	res, err := Start(repo.Root(), "itd-10", Options{judgeCalibrated: func() (bool, string) { return true, "" }})
	if err != nil {
		t.Fatal(err)
	}
	j := res.Review.Judge
	if j == nil || j.Class != ClassTop || j.Model != BundledClasses[ClassTop] || j.Layer != "repo" || j.Origin != ".abcd/config.json" {
		t.Fatalf("the judge is kept with its class and layer: %+v", j)
	}
}

// TestARunningRunKeepsTheReviewSetItStartedWith: the set is read once, when
// the run starts; a resume reports the stored set whatever the files say by
// then, and writes nothing.
func TestARunningRunKeepsTheReviewSetItStartedWith(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	first, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	before := stateBytes(t, repo.Root(), first.RunID)
	repoConfig(t, repo, `{"review": {"reviewers": ["top"]}}`)
	machineConfig(t, `{"models": {"classes": {"middle": "acme/other-1"}}}`)
	again, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil || !again.Resumed {
		t.Fatalf("the run resumes: %+v %v", again, err)
	}
	r := reviewRole(t, again.Review, RoleRuthless)
	if strings.Join(classesOf(r), ",") != "middle,below" || r.Reviewers[0].Model != BundledClasses[ClassMiddle] {
		t.Fatalf("the resume keeps the set the run started with: %+v", r)
	}
	if string(before) != string(stateBytes(t, repo.Root(), first.RunID)) {
		t.Fatal("a resume writes nothing")
	}
}

// TestAStoredReviewSetTheLoopNeverWritesIsRefused: a review set read back
// from the state is held to what the resolver writes, and an older state
// carrying one is not one its version wrote.
func TestAStoredReviewSetTheLoopNeverWritesIsRefused(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(res.RunID)))
	orig, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	current := fmt.Sprintf(`"schema_version": %d`, SchemaVersion)
	unreviewed := fmt.Sprintf(`"schema_version": %d`, schemaVersionUnreviewed)
	for name, edit := range map[string]func(string) string{
		"a layer it never writes": func(s string) string {
			return strings.Replace(s, `"map_layer": "bundled"`, `"map_layer": "elsewhere"`, 1)
		},
		"a class it never names": func(s string) string { return strings.Replace(s, `"class": "middle"`, `"class": "huge"`, 1) },
		"an older version":       func(s string) string { return strings.Replace(s, current, unreviewed, 1) },
	} {
		t.Run(name, func(t *testing.T) {
			edited := edit(string(orig))
			if edited == string(orig) {
				t.Fatalf("the edit changed nothing; the state is:\n%s", orig)
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
	if err := os.WriteFile(path, orig, 0o600); err != nil {
		t.Fatal(err)
	}

	// A state written before review sets reads as a run with none.
	older := strings.Replace(string(orig), current, unreviewed, 1)
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(older), &m); err != nil {
		t.Fatal(err)
	}
	delete(m, "review")
	raw, _ := json.Marshal(m)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Review != nil || st.SchemaVersion != SchemaVersion {
		t.Fatalf("an older run reads with no review set at the current version: %+v %d", st.Review, st.SchemaVersion)
	}
}

// unreviewOlder drops the review set from a state document whose
// schema_version an edit set below 15: a new run writes one, and no file of
// an earlier version carries it, so a test that reads a run as an older
// version reads it as that version would have written it.
func unreviewOlder(doc map[string]any) {
	var v int
	switch n := doc["schema_version"].(type) {
	case int:
		v = n
	case float64:
		v = int(n)
	default:
		return
	}
	if v <= schemaVersionUnreviewed {
		delete(doc, "review")
	}
}

// olderState is a state file rewritten at version v, without the review set
// no file of a version before 15 carries.
func olderState(t *testing.T, data []byte, v int) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	doc["schema_version"] = v
	unreviewOlder(doc)
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return out
}
