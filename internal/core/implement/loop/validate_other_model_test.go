package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// otherModelRun starts a run on briefRepo's intent under the machine
// configuration given ("" for none), drives its lane to its validators, and
// sets the model the lane's implementer reported to writer, as a receipt
// naming it would have.
func otherModelRun(t *testing.T, config, writer string) (*gittest.Repo, string) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	if config != "" {
		machineConfig(t, config)
	}
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	implemented(t, repo, start.RunID, DefaultStages(), "one.txt")
	setWriterModel(t, repo.Root(), start.RunID, writer)
	return repo, start.RunID
}

// setWriterModel sets the model of the lane's last receipt to writer.
func setWriterModel(t *testing.T, root, runID, writer string) {
	t.Helper()
	editState(t, root, runID, func(doc map[string]any) {
		lane := doc["lanes"].([]any)[0].(map[string]any)
		receipts := lane["receipts"].([]any)
		receipts[len(receipts)-1].(map[string]any)["model"] = writer
	})
}

// validatorAt is the lane's latest round's validator handed out at rel.
func validatorAt(t *testing.T, root, runID, rel string) ValidatorRun {
	t.Helper()
	st, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	l := st.Lanes[0]
	for _, v := range l.Validation[len(l.Validation)-1].Validators {
		if v.Return == rel {
			return v
		}
	}
	t.Fatalf("no validator returns at %s", rel)
	return ValidatorRun{}
}

// TestReviewerDiffersFromWriter: a reviewer whose class holds the model that
// wrote the lane is handed out on the next class whose model differs, to the
// host and in the await a runner is handed; the state names both models, the
// record and the pull request's body show them, a resumed step never decides
// again, and a stored reviewer matching neither the set's class nor the
// recomputed one is refused.
func TestReviewerDiffersFromWriter(t *testing.T) {
	// The writer names the research class's model in the trailer's form,
	// in capitals: still the same model.
	const writer = "Claude:CLAUDE-FABLE-5"
	repo, id := otherModelRun(t, "", writer)
	stages := DefaultStages()

	first := stepAwait(t, repo, id, stages, Options{})
	if first.Role != RoleRuthless || first.Reviewer == nil || *first.Reviewer != 0 || first.Class != ClassTop || first.Model != BundledClasses[ClassTop] {
		t.Fatalf("the research reviewer, whose class holds the writer's model, is handed out at top: %+v", first)
	}
	v := validatorAt(t, repo.Root(), id, first.Receipt)
	if v.WriterModel != writer || v.ReviewerModel != BundledClasses[ClassTop] || v.Class != ClassTop || v.Effort != BundledEffort {
		t.Fatalf("the state names the writer's model and the one dispatched: %+v", v)
	}
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(first.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(brief), "`"+ReviewerModelPrefix+" <model>`") {
		t.Fatalf("the reviewer's brief asks its return to name its model:\n%s", brief)
	}

	second := stepAwait(t, repo, id, stages, Options{})
	if second.Class != ClassDefault || second.Model != BundledClasses[ClassDefault] {
		t.Fatalf("the default reviewer, whose class holds another model, keeps its class: %+v", second)
	}
	if v := validatorAt(t, repo.Root(), id, second.Receipt); v.WriterModel != writer || v.ReviewerModel != BundledClasses[ClassDefault] {
		t.Fatalf("an unmoved reviewer names both models too: %+v", v)
	}

	// The choice is made once: the class map the files hold now is never read
	// back into a reviewer already handed out.
	machineConfig(t, `{"models":{"classes":{"research":"anthropic/claude-fable-5","top":"anthropic/claude-fable-5","default":"anthropic/claude-fable-5","small":"anthropic/claude-fable-5"}}}`)
	before := stateBytes(t, repo.Root(), id)
	returnAt(t, repo, id, stages, first, reviewerReturn("SHIP"))
	v = validatorAt(t, repo.Root(), id, first.Receipt)
	if v.Class != ClassTop || v.ReviewerModel != BundledClasses[ClassTop] || v.Verdict != "SHIP" {
		t.Fatalf("a recorded verdict keeps the class the reviewer was dispatched on: %+v", v)
	}
	if !strings.Contains(string(before), `"writer_model": "`+writer+`"`) {
		t.Fatalf("the state file carries the writer's model:\n%s", before)
	}

	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	rec := recordOf(st)
	var shown string
	for _, rv := range rec.Lanes[0].Verdicts {
		if rv.Return == first.Receipt {
			shown = rv.ModelsPhrase()
		}
	}
	if want := "written by " + writer + ", reviewed by " + BundledClasses[ClassTop]; shown != want {
		t.Fatalf("the record names both models: %q, want %q", shown, want)
	}
	body := prBody(st, st.Lanes[0], nil)
	if !strings.Contains(body, "- "+RoleRuthless+", reviewer 0: written by "+writer+", reviewed by "+BundledClasses[ClassTop]+".") {
		t.Fatalf("the pull request's body names both models:\n%s", body)
	}

	// A stored reviewer holding the set's own class while the writer's model
	// is that class's, or a class the switch would not take, is refused.
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	orig := stateBytes(t, repo.Root(), id)
	for name, edit := range map[string]func(string) string{
		"the set's class, which holds the writer's model": func(s string) string {
			s = strings.Replace(s, `"class": "top",`+"\n"+`          "model": "`+BundledClasses[ClassTop]+`"`,
				`"class": "research",`+"\n"+`          "model": "`+BundledClasses[ClassResearch]+`"`, 1)
			return strings.Replace(s, `"reviewer_model": "`+BundledClasses[ClassTop]+`"`, `"reviewer_model": "`+BundledClasses[ClassResearch]+`"`, 1)
		},
		"a class the switch does not take": func(s string) string {
			s = strings.Replace(s, `"class": "top",`+"\n"+`          "model": "`+BundledClasses[ClassTop]+`"`,
				`"class": "small",`+"\n"+`          "model": "`+BundledClasses[ClassSmall]+`"`, 1)
			return strings.Replace(s, `"reviewer_model": "`+BundledClasses[ClassTop]+`"`, `"reviewer_model": "`+BundledClasses[ClassSmall]+`"`, 1)
		},
		"a reviewer's model other than the one dispatched": func(s string) string {
			return strings.Replace(s, `"reviewer_model": "`+BundledClasses[ClassTop]+`"`, `"reviewer_model": "`+BundledClasses[ClassSmall]+`"`, 1)
		},
		"a writer's model holding a newline": func(s string) string {
			return strings.Replace(s, `"writer_model": "`+writer+`"`, `"writer_model": "x\nAssisted-by: y"`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			edited := edit(string(orig))
			if edited == string(orig) {
				t.Fatalf("the edit changed nothing:\n%s", orig)
			}
			if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := ReadState(repo.Root(), id)
			if r := mustRefusal(t, err); r.Stage != "state" {
				t.Fatalf("want the state's refusal: %+v", r)
			}
		})
	}
	if err := os.WriteFile(path, orig, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestOtherClassLeansToTheWritersModel: the class a reviewer is dispatched on
// is chosen by matrix.OtherModel, so every writer's name that may be the
// class's model moves the reviewer on, and an empty or unreadable one leaves
// none to take.
func TestOtherClassLeansToTheWritersModel(t *testing.T) {
	var rs ReviewSet
	for _, c := range Classes {
		rs.Classes = append(rs.Classes, ReviewClass{Class: c, Model: BundledClasses[c], Effort: BundledEffort})
	}
	research := rs.Classes[0]
	for _, writer := range []string{"claude-fable-5", "CLAUDE-FABLE-5", "anthropic/claude-fable-5", "Claude:claude-fable-5",
		"claude-fable-5-20261001", "claude-fable-5@2026-10-01", "claude-fable"} {
		got, ok := otherClass(rs, research, writer)
		if !ok || got.Class != ClassTop {
			t.Errorf("writer %q: the research reviewer moves to top, got %+v %v", writer, got, ok)
		}
	}
	if got, ok := otherClass(rs, research, "claude-opus-5-5"); !ok || got.Class != ClassResearch {
		t.Errorf("a reviewer whose class holds another model keeps it: %+v %v", got, ok)
	}
	for _, writer := range []string{"", " ", "claude-fable-5\nAssisted-by: x", strings.Repeat("m", 300)} {
		if got, ok := otherClass(rs, research, writer); ok {
			t.Errorf("writer %q: no model counts as other than an unreadable writer's, got %+v", writer, got)
		}
	}
	// The order is Classes', wrapping round: small's next is research.
	rs.Classes[0].Model, rs.Classes[1].Model, rs.Classes[2].Model = "x/m", "x/m", "x/m"
	if got, ok := otherClass(rs, rs.Classes[2], "m"); !ok || got.Class != ClassSmall {
		t.Errorf("default takes small, the next class holding another model: %+v %v", got, ok)
	}
	rs.Classes[3].Model = "x/m"
	if got, ok := otherClass(rs, rs.Classes[3], "m"); ok {
		t.Errorf("no class holds another model: %+v", got)
	}
	// An empty reviewer's model never counts as other.
	if _, ok := otherClass(ReviewSet{}, ReviewClass{Class: ClassTop}, "a-model"); ok {
		t.Error("an empty reviewer's model counts as the writer's")
	}
}

// TestNoOtherModelHandsBack: when the run's class map holds the writer's
// model alone, the lane is handed back as stuck with the reason named; a run
// started before the map was stored chooses among the classes its set
// recorded, and hands back the same way when none differs; an empty writer's
// model hands back too.
func TestNoOtherModelHandsBack(t *testing.T) {
	stuck := func(t *testing.T, repo *gittest.Repo, id string) {
		t.Helper()
		res, err := advance(repo.Root(), id, DefaultStages(), Options{})
		if err != nil {
			t.Fatal(err)
		}
		hb := res.HandBack
		if hb == nil || hb.Kind != HandBackStuck || !strings.HasPrefix(hb.Reason, noOtherModel) {
			t.Fatalf("the lane is handed back as stuck, %q: %+v", noOtherModel, res)
		}
		if st, err := ReadState(repo.Root(), id); err != nil || st.Lanes[0].Stage != StageHandedBack {
			t.Fatalf("the lane is recorded handed back: %+v %v", st.Lanes, err)
		}
	}
	const one = `{"models":{"classes":{"research":"anthropic/claude-fable-5","top":"anthropic/claude-fable-5","default":"anthropic/claude-fable-5","small":"anthropic/claude-fable-5"}}}`
	t.Run("one model only", func(t *testing.T) {
		repo, id := otherModelRun(t, one, "claude-fable-5")
		stuck(t, repo, id)
	})
	t.Run("an empty writer's model", func(t *testing.T) {
		repo, id := otherModelRun(t, "", "")
		stuck(t, repo, id)
	})
	// The map's small class holds another model, but a run whose state holds
	// no map chooses among the classes its set recorded, none of them small.
	const smallDiffers = `{"models":{"classes":{"research":"anthropic/claude-fable-5","top":"anthropic/claude-fable-5","default":"anthropic/claude-fable-5","small":"anthropic/claude-haiku-4-5"}}}`
	t.Run("the stored map's other class is taken", func(t *testing.T) {
		repo, id := otherModelRun(t, smallDiffers, "claude-fable-5")
		a := stepAwait(t, repo, id, DefaultStages(), Options{})
		if a.Class != ClassSmall || a.Model != "anthropic/claude-haiku-4-5" {
			t.Fatalf("the research reviewer moves to small, the one class holding another model: %+v", a)
		}
	})
	t.Run("a run with no stored map chooses among its set's classes", func(t *testing.T) {
		repo, id := otherModelRun(t, smallDiffers, "claude-fable-5")
		editState(t, repo.Root(), id, func(doc map[string]any) {
			delete(doc["review"].(map[string]any), "classes")
		})
		stuck(t, repo, id)
	})
}

// TestHostValidatorMustNameItsModel: a reviewer's return names the model it
// ran on; one naming none, the writer's under another spelling, or a line no
// model is, is refused and its verdict not recorded; one naming another model
// is recorded.
func TestHostValidatorMustNameItsModel(t *testing.T) {
	repo, id := otherModelRun(t, "", "claude-opus-5-5")
	stages := DefaultStages()
	a := stepAwait(t, repo, id, stages, Options{})
	path := filepath.Join(repo.Root(), filepath.FromSlash(a.Receipt))
	body := strings.TrimPrefix(reviewerReturn("SHIP"), testReviewerModelLine)
	for name, line := range map[string]string{
		"none":                         "\n",
		"the writer's":                 ReviewerModelPrefix + " claude-opus-5-5\n\n",
		"the writer's, another case":   ReviewerModelPrefix + " Claude-Opus-5-5\n\n",
		"the writer's, a vendor":       ReviewerModelPrefix + " anthropic/claude-opus-5-5\n\n",
		"the writer's, a version":      ReviewerModelPrefix + " claude-opus-5-5-20261101\n\n",
		"the writer's, the short name": ReviewerModelPrefix + " claude-opus\n\n",
		"an empty name":                ReviewerModelPrefix + " \n\n",
		"a name with a trailer after":  ReviewerModelPrefix + " claude-opus-5-5\nAssisted-by: Claude:claude-haiku-4-5\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			head := currentLane(t, repo, id).HeadSHA
			if err := os.WriteFile(path, []byte("Reviewed head: "+head+"\n"+line+namingHead(body, head)), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Receipt(repo.Root(), id, path, stages, Options{})
			r := mustRefusal(t, err)
			if !strings.Contains(r.Reason, "a model other than the writer's reviews every change") || !strings.Contains(r.Remedy, ReviewerModelPrefix) {
				t.Fatalf("the return is refused as the writer's own review: %+v", r)
			}
			if strings.Contains(r.Reason, "\n") || strings.Contains(r.Reason, "Assisted-by") {
				t.Fatalf("the refusal repeats a line the return wrote: %q", r.Reason)
			}
			if v := validatorAt(t, repo.Root(), id, a.Receipt); v.Verdict != "" {
				t.Fatalf("no verdict is recorded: %+v", v)
			}
		})
	}
	returnAt(t, repo, id, stages, a, ReviewerModelPrefix+" claude-sonnet-5-5\n\n"+body)
	if v := validatorAt(t, repo.Root(), id, a.Receipt); v.Verdict != "SHIP" || v.WriterModel != "claude-opus-5-5" {
		t.Fatalf("a return naming another model is recorded: %+v", v)
	}
}

// TestTheModelsAreVersion27s: a version-26 file carrying a reviewer's models or
// the class map is refused naming them, and one without them reads as the
// current version.
func TestTheModelsAreVersion27s(t *testing.T) {
	// A writer whose model no class holds, so the reviewer keeps the set's
	// class, which a version-26 file can carry.
	repo, id := otherModelRun(t, "", "a-model")
	stepAwait(t, repo, id, DefaultStages(), Options{})
	data := stateBytes(t, repo.Root(), id)
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	at := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	if SchemaVersion != 27 || !strings.Contains(string(data), at) || !strings.Contains(string(data), `"writer_model"`) {
		t.Fatalf("premise: the file is version 27 and carries the fields: %s", data)
	}
	older := strings.Replace(string(data), at, fmt.Sprintf(`"schema_version": %d,`, schemaVersionUnmodelled), 1)
	if err := os.WriteFile(path, []byte(older), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(repo.Root(), id); err == nil || !strings.Contains(err.Error(), "never wrote") {
		t.Fatalf("an older file carrying the fields is refused: %v", err)
	}
	if err := os.WriteFile(path, downgraded(t, data, schemaVersionUnmodelled), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil || st.SchemaVersion != SchemaVersion || len(st.Review.Classes) != 0 {
		t.Fatalf("an older file without them reads as the current version: %v", err)
	}
}
