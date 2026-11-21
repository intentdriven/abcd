package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gittest"
)

// otherModelRun starts a run on briefRepo's intent under the machine
// configuration given ("" for none), drives its lane to its validators, and
// sets the model the lane's implementer named to writer, as a receipt naming
// it would have.
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

// setWriterRoute records on the lane's last receipt that a runner ran it and
// reported model.
func setWriterRoute(t *testing.T, root, runID, model string) {
	t.Helper()
	editState(t, root, runID, func(doc map[string]any) {
		lane := doc["lanes"].([]any)[0].(map[string]any)
		receipts := lane["receipts"].([]any)
		receipts[len(receipts)-1].(map[string]any)["route"] = map[string]any{"asked": "claude", "ran": "claude", "model": model}
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

// stateRefusedOnRead writes edited over the run's state file and checks the read
// refuses it, then puts orig back.
func stateRefusedOnRead(t *testing.T, root, runID string, orig []byte, edited string) {
	t.Helper()
	if edited == string(orig) {
		t.Fatalf("the edit changed nothing:\n%s", orig)
	}
	path := filepath.Join(root, filepath.FromSlash(StateRelPath(runID)))
	if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ReadState(root, runID)
	if r := mustRefusal(t, err); r.Stage != "state" {
		t.Fatalf("want the state's refusal: %+v", r)
	}
	if err := os.WriteFile(path, orig, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestReviewerDiffersFromWriter: a reviewer whose class holds the model that
// wrote the lane is handed out on the next class whose model differs, to the
// host and in the await a runner is handed; the state names both models, the
// record and the pull request's body show them in code spans, and a stored
// reviewer matching neither the set's class nor the recomputed one is refused.
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
	if !slices.Equal(v.WriterModels, []string{writer}) || v.ReviewerModel != BundledClasses[ClassTop] || v.Class != ClassTop || v.Effort != BundledEffort {
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
	if v := validatorAt(t, repo.Root(), id, second.Receipt); !slices.Equal(v.WriterModels, []string{writer}) || v.ReviewerModel != BundledClasses[ClassDefault] {
		t.Fatalf("an unmoved reviewer names both models too: %+v", v)
	}

	returnAt(t, repo, id, stages, first, reviewerReturn("SHIP"))
	v = validatorAt(t, repo.Root(), id, first.Receipt)
	if v.Class != ClassTop || v.ReviewerModel != BundledClasses[ClassTop] || v.ReviewerNamed != "kimi-test-1" || v.Verdict != "SHIP" {
		t.Fatalf("a recorded verdict keeps the class the reviewer was dispatched on, and the model its return named beside it: %+v", v)
	}
	if !strings.Contains(string(stateBytes(t, repo.Root(), id)), `"writer_models": [`) {
		t.Fatal("the state file carries the writers' models")
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
	want := "written by `" + writer + "`, asked of `" + BundledClasses[ClassTop] + "`, its return naming `kimi-test-1`"
	if shown != want {
		t.Fatalf("the record names every model it holds: %q, want %q", shown, want)
	}
	body := prBody(st, st.Lanes[0], nil)
	if !strings.Contains(body, "- "+RoleRuthless+", reviewer 0: "+want+".") {
		t.Fatalf("the pull request's body names every model:\n%s", body)
	}

	// A stored reviewer holding the set's own class while the writer's model
	// is that class's, or a class the switch would not take, is refused.
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
			return strings.Replace(s, `"`+writer+`"`, `"x\nAssisted-by: y"`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) { stateRefusedOnRead(t, repo.Root(), id, orig, edit(string(orig))) })
	}
}

// TestTheChoiceIsMadeOnce: a reviewer already handed out is never moved again
// when the stage's body is asked a second time, even with a class map that
// would move it elsewhere.
func TestTheChoiceIsMadeOnce(t *testing.T) {
	repo, id := otherModelRun(t, "", "claude-fable-5")
	stages := DefaultStages()
	// The round opens and its first reviewer is handed out.
	first := stepAwait(t, repo, id, stages, Options{})
	if first.Class != ClassTop {
		t.Fatalf("premise: the research reviewer moves to top: %+v", first)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	lane := st.Lanes[0]
	// Its await is taken off, as a restart does, and the map changes: top now
	// holds the writer's model, so a fresh choice would move it on again.
	lane.Awaits = nil
	st.Review.Classes[1].Model = "anthropic/claude-fable-5"
	c := Context{RepoRoot: repo.Root(), RunDir: runRel(id), State: st, Now: st.UpdatedAt}
	out, err := validateStage(c, &lane)
	if err != nil {
		t.Fatal(err)
	}
	if out.Await == nil || out.Await.Receipt != first.Receipt || out.Await.Class != ClassTop || out.Await.Model != BundledClasses[ClassTop] {
		t.Fatalf("the reviewer is handed out again on the class chosen first: %+v", out.Await)
	}
}

// TestOtherClassLeansToTheWritersModel: the class a reviewer is dispatched on
// is chosen by matrix.OtherModel against every writer, so every writer's name
// that may be the class's model moves the reviewer on, and an empty or
// unreadable one leaves none to take.
func TestOtherClassLeansToTheWritersModel(t *testing.T) {
	var rs ReviewSet
	for _, c := range Classes {
		rs.Classes = append(rs.Classes, ReviewClass{Class: c, Model: BundledClasses[c], Effort: BundledEffort})
	}
	research := rs.Classes[0]
	for _, writer := range []string{"claude-fable-5", "CLAUDE-FABLE-5", "anthropic/claude-fable-5", "Claude:claude-fable-5",
		"claude-fable-5-20261001", "claude-fable-5@2026-10-01", "claude-fable", "fable", "claude-fable-5[1m]", "us.anthropic.claude-fable-5-v1:0"} {
		got, ok := otherClass(rs, RoleRuthless, research, []string{writer}, nil)
		if !ok || got.Class != ClassTop {
			t.Errorf("writer %q: the research reviewer moves to top, got %+v %v", writer, got, ok)
		}
	}
	if got, ok := otherClass(rs, RoleRuthless, research, []string{"claude-opus-5-5"}, nil); !ok || got.Class != ClassResearch {
		t.Errorf("a reviewer whose class holds another model keeps it: %+v %v", got, ok)
	}
	if got, ok := otherClass(rs, RoleRuthless, research, []string{"claude-opus-5-5", "claude-fable-5"}, nil); !ok || got.Class != ClassDefault {
		t.Errorf("a reviewer passes every writer's class: %+v %v", got, ok)
	}
	for _, writer := range []string{"", " ", "claude-fable-5\nAssisted-by: x", strings.Repeat("m", 300)} {
		if got, ok := otherClass(rs, RoleRuthless, research, []string{writer}, nil); ok {
			t.Errorf("writer %q: no model counts as other than an unreadable writer's, got %+v", writer, got)
		}
	}
	if got, ok := otherClass(rs, RoleRuthless, research, nil, nil); ok {
		t.Errorf("no model counts as other than no writer: %+v", got)
	}
	// The order is Classes', wrapping round: small's next is research.
	rs.Classes[0].Model, rs.Classes[1].Model, rs.Classes[2].Model = "x/gpt-5", "x/gpt-5", "x/gpt-5"
	if got, ok := otherClass(rs, RoleRuthless, rs.Classes[2], []string{"gpt-5"}, nil); !ok || got.Class != ClassSmall {
		t.Errorf("default takes small, the next class holding another model: %+v %v", got, ok)
	}
	rs.Classes[3].Model = "x/gpt-5"
	if got, ok := otherClass(rs, RoleRuthless, rs.Classes[3], []string{"gpt-5"}, nil); ok {
		t.Errorf("no class holds another model: %+v", got)
	}
}

// TestNoOtherModelHandsBack: when the run's class map holds the writer's
// model alone, the lane is handed back as stuck with the reason named; a run
// started before the map was stored chooses among the classes its set
// recorded, and hands back the same way when none differs; a writer whose
// model is unknown, with no runner's report, hands back too, with a remedy
// that starts a new lane.
func TestNoOtherModelHandsBack(t *testing.T) {
	stuck := func(t *testing.T, repo *gittest.Repo, id string) *HandBack {
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
		return hb
	}
	const one = `{"models":{"classes":{"research":"anthropic/claude-fable-5","top":"anthropic/claude-fable-5","default":"anthropic/claude-fable-5","small":"anthropic/claude-fable-5"}}}`
	t.Run("one model only", func(t *testing.T) {
		repo, id := otherModelRun(t, one, "claude-fable-5")
		if hb := stuck(t, repo, id); !strings.Contains(hb.Home, "models.classes") || !strings.Contains(hb.Home, "--discard") {
			t.Fatalf("the remedy names the class map and a new lane: %+v", hb)
		}
	})
	for name, writer := range map[string]string{"an empty writer's model": "", "a non-ASCII writer's model": "claude-оpus-5-5", "a vendor alone": "Claude"} {
		t.Run(name, func(t *testing.T) {
			repo, id := otherModelRun(t, "", writer)
			hb := stuck(t, repo, id)
			if !strings.Contains(hb.Reason, "no runner reported") || !strings.Contains(hb.Home, "--discard") || strings.Contains(hb.Home, "models.classes") {
				t.Fatalf("an unknown writer is named, and the remedy is a new lane: %+v", hb)
			}
		})
	}
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

// TestTheWritersAreEveryModelThatWroteTheLane: the reviewer differs from the
// model the implementer's runner reported and the one its receipt named; an
// unknown name on a receipt is passed over where its runner reported the
// model; and a fix round's implementer joins the writers, so the next round's
// reviewers differ from the first writer's model and the fixer's, and the
// pull request's body names both.
func TestTheWritersAreEveryModelThatWroteTheLane(t *testing.T) {
	t.Run("the runner's report and the receipt's name", func(t *testing.T) {
		repo, id := otherModelRun(t, "", "claude-sonnet-5-5")
		setWriterRoute(t, repo.Root(), id, "claude-fable-5")
		stages := DefaultStages()
		first := stepAwait(t, repo, id, stages, Options{})
		if first.Class != ClassTop {
			t.Fatalf("the research reviewer passes the runner's reported model: %+v", first)
		}
		if v := validatorAt(t, repo.Root(), id, first.Receipt); !slices.Equal(v.WriterModels, []string{"claude-fable-5", "claude-sonnet-5-5"}) {
			t.Fatalf("both writers are recorded, the runner's first: %+v", v.WriterModels)
		}
		second := stepAwait(t, repo, id, stages, Options{})
		if second.Class != ClassSmall {
			t.Fatalf("the default reviewer passes the receipt's named model for small: %+v", second)
		}
	})
	t.Run("an unknown name beside a runner's report", func(t *testing.T) {
		repo, id := otherModelRun(t, "", "claude-оpus-5-5")
		setWriterRoute(t, repo.Root(), id, "claude-fable-5")
		first := stepAwait(t, repo, id, DefaultStages(), Options{})
		if v := validatorAt(t, repo.Root(), id, first.Receipt); first.Class != ClassTop || !slices.Equal(v.WriterModels, []string{"claude-fable-5"}) {
			t.Fatalf("the runner's report is the writer: %+v %+v", first, v.WriterModels)
		}
	})
	t.Run("a second round after a fix by another model", func(t *testing.T) {
		repo, id := otherModelRun(t, "", "claude-sonnet-5-5")
		stages := DefaultStages()
		handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("FIX FIRST"))
		handBack(t, repo, id, stages, RoleSecurity, reviewerReturn("APPROVE"))
		handBack(t, repo, id, stages, RoleAuditor, "MET")
		fix := laneCommit(t, repo, currentLane(t, repo, id), "fix.txt")
		fixed(t, repo, id, stages, "applied the finding in "+fix+"\n", fix)
		setWriterModel(t, repo.Root(), id, "claude-fable-5")
		first := stepAwait(t, repo, id, stages, Options{})
		second := stepAwait(t, repo, id, stages, Options{})
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		l := st.Lanes[0]
		if len(l.Validation) != 2 {
			t.Fatalf("premise: the second round is open: %d rounds", len(l.Validation))
		}
		if first.Class != ClassTop || second.Class != ClassSmall {
			t.Fatalf("neither the first writer's model nor the fixer's reviews the second round: %+v %+v", first, second)
		}
		if v := validatorAt(t, repo.Root(), id, second.Receipt); !slices.Equal(v.WriterModels, []string{"claude-sonnet-5-5", "claude-fable-5"}) {
			t.Fatalf("the writers are recorded in the order they wrote: %+v", v.WriterModels)
		}
		if ws := l.Validation[0].Validators[0].WriterModels; !slices.Equal(ws, []string{"claude-sonnet-5-5"}) {
			t.Fatalf("the first round keeps the writers it was handed out under: %+v", ws)
		}
		returnAt(t, repo, id, stages, first, reviewerReturn("SHIP"))
		st, _ = ReadState(repo.Root(), id)
		if body := prBody(st, st.Lanes[0], nil); !strings.Contains(body, "written by `claude-sonnet-5-5` and `claude-fable-5`") {
			t.Fatalf("the pull request's body names every writer:\n%s", body)
		}
		orig := stateBytes(t, repo.Root(), id)
		// The fixer's model dropped from the second round's writers is refused.
		stateRefusedOnRead(t, repo.Root(), id, orig, editedBytes(t, orig, func(doc map[string]any) {
			firstValidator(doc, 1)["writer_models"] = []any{"claude-sonnet-5-5"}
		}))
	})
}

// TestAReviewerReturnIsCheckedOnEveryModelItHolds: at its return a reviewer is
// held to the model its runner reported and the one its return names as well
// as the one it was asked for; the record keeps the runner's report, or the
// model asked for, and beside it a named model that is another; a reviewer no
// class was asked of keeps the model its return named.
func TestAReviewerReturnIsCheckedOnEveryModelItHolds(t *testing.T) {
	const writer = "claude-opus-5-5"
	reported := func(model string) *runner.RouteRecord {
		return &runner.RouteRecord{Asked: "claude", Ran: "claude", Model: model}
	}
	hand := func(t *testing.T, repo *gittest.Repo, id string, a Await, body string, route *runner.RouteRecord) error {
		t.Helper()
		path := filepath.Join(repo.Root(), filepath.FromSlash(a.Receipt))
		if err := os.WriteFile(path, []byte(statingHeadAt(t, repo.Root(), id, a.Receipt, body)), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := handIn(repo.Root(), id, path, DefaultStages(), Options{}, route, false)
		return err
	}
	t.Run("a runner reporting the writer's model", func(t *testing.T) {
		repo, id := otherModelRun(t, "", writer)
		a := stepAwait(t, repo, id, DefaultStages(), Options{})
		err := hand(t, repo, id, a, reviewerReturn("SHIP"), reported("anthropic/claude-opus-5-5"))
		if r := mustRefusal(t, err); !strings.Contains(r.Reason, "its runner reported") {
			t.Fatalf("the return is refused naming the runner's report: %+v", r)
		}
		if v := validatorAt(t, repo.Root(), id, a.Receipt); v.Verdict != "" {
			t.Fatalf("no verdict is recorded: %+v", v)
		}
	})
	t.Run("a runner reporting another model, its return naming a third", func(t *testing.T) {
		repo, id := otherModelRun(t, "", writer)
		a := stepAwait(t, repo, id, DefaultStages(), Options{})
		if err := hand(t, repo, id, a, reviewerReturn("SHIP"), reported("claude-haiku-4-5")); err != nil {
			t.Fatal(err)
		}
		v := validatorAt(t, repo.Root(), id, a.Receipt)
		if v.Verdict != "SHIP" || v.ModelStatus != ModelReported || v.ReviewerModel != "claude-haiku-4-5" || v.ReviewerNamed != "kimi-test-1" {
			t.Fatalf("the record keeps the runner's report and the named model beside it: %+v", v)
		}
		if got, want := v.models(), "written by `"+writer+"`, run on, as its runner reported, `claude-haiku-4-5`, its return naming `kimi-test-1`"; got != want {
			t.Fatalf("the line says both: %q, want %q", got, want)
		}
	})
	t.Run("a return naming the model it was asked for", func(t *testing.T) {
		repo, id := otherModelRun(t, "", writer)
		a := stepAwait(t, repo, id, DefaultStages(), Options{})
		body := strings.TrimPrefix(reviewerReturn("SHIP"), testReviewerModelLine)
		returnAt(t, repo, id, DefaultStages(), a, ReviewerModelPrefix+" claude-fable-5\n\n"+body)
		v := validatorAt(t, repo.Root(), id, a.Receipt)
		if v.ReviewerModel != BundledClasses[ClassResearch] || v.ReviewerNamed != "" {
			t.Fatalf("a named model that is the one asked for is not kept twice: %+v", v)
		}
		if got := v.models(); got != "written by `"+writer+"`, reviewed by `"+BundledClasses[ClassResearch]+"`" {
			t.Fatalf("one model reviewed: %q", got)
		}
	})
	t.Run("a reviewer no class was asked of", func(t *testing.T) {
		repo, id := otherModelRun(t, "", writer)
		editState(t, repo.Root(), id, func(doc map[string]any) { delete(doc, "review") })
		a := stepAwait(t, repo, id, DefaultStages(), Options{})
		if a.Class != "" {
			t.Fatalf("premise: a run with no review set asks no class: %+v", a)
		}
		if v := validatorAt(t, repo.Root(), id, a.Receipt); !slices.Equal(v.WriterModels, []string{writer}) || v.ReviewerModel != "" {
			t.Fatalf("its writers are recorded as it is handed out, and no reviewer's model yet: %+v", v)
		}
		body := strings.TrimPrefix(reviewerReturn("SHIP"), testReviewerModelLine)
		returnAt(t, repo, id, DefaultStages(), a, ReviewerModelPrefix+" claude-sonnet-5-5\n\n"+body)
		if v := validatorAt(t, repo.Root(), id, a.Receipt); v.ReviewerModel != "claude-sonnet-5-5" || v.ReviewerNamed != "" {
			t.Fatalf("it keeps the model its return named: %+v", v)
		}
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
		"the writer's, the short name": ReviewerModelPrefix + " opus\n\n",
		"the writer's, a cloud id":     ReviewerModelPrefix + " us.anthropic.claude-opus-5-5-v1:0\n\n",
		"the writer's, dotted":         ReviewerModelPrefix + " claude-opus-5.5\n\n",
		"an empty name":                ReviewerModelPrefix + " \n\n",
		"a name with a trailer after":  ReviewerModelPrefix + " claude-opus-5-5\nAssisted-by: Claude:claude-haiku-4-5\n\n",
		"a non-ASCII name":             ReviewerModelPrefix + " claude-sоnnet-5-5\n\n",
	} {
		t.Run(name, func(t *testing.T) {
			head := currentLane(t, repo, id).HeadSHA
			if err := os.WriteFile(path, []byte("Reviewed head: "+head+"\n"+line+namingHead(body, head)), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Receipt(repo.Root(), id, path, stages, Options{})
			r := mustRefusal(t, err)
			if !strings.Contains(r.Reason, "a model other than every writer's reviews every change") || !strings.Contains(r.Remedy, ReviewerModelPrefix) {
				t.Fatalf("the return is refused as a writer's own review: %+v", r)
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
	if v := validatorAt(t, repo.Root(), id, a.Receipt); v.Verdict != "SHIP" || !slices.Equal(v.WriterModels, []string{"claude-opus-5-5"}) {
		t.Fatalf("a return naming another model is recorded: %+v", v)
	}
}

// TestARouteAnswerWithNoModelLineFallsBackToTheHost: the reviewer route's
// answer that names no model is not counted, and the review falls back to the
// host, with nothing recorded and nothing left at the return path.
func TestARouteAnswerWithNoModelLineFallsBackToTheHost(t *testing.T) {
	repo, id := otherModelRun(t, `{"review":{"reviewers":["top"]}}`, "grok-test-1")
	unnamed := func(request string) string { return strings.TrimPrefix(verdictFor(request), testReviewerModelLine) }
	fb := fellBack(t, repo, id, DefaultStages(), rpRoute(t, rpServer(unnamed)))
	if !strings.Contains(fb.Detail, "names no model it ran on") {
		t.Fatalf("the fallback says why: %+v", fb)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "" {
		t.Fatalf("no verdict is recorded from the route: %+v", v)
	}
}

// TestAMovedReviewerTakesALaunchableClassNoSiblingHolds: a moved reviewer
// passes over a class its sibling holds while a further class differs, and
// over a class the drive's runners could not launch when the run started.
func TestAMovedReviewerTakesALaunchableClassNoSiblingHolds(t *testing.T) {
	const reviewers = `"review":{"reviewers":["small","research"]}`
	t.Run("a sibling's class is passed over", func(t *testing.T) {
		repo, id := otherModelRun(t, `{`+reviewers+`}`, "claude-haiku-4-5")
		a := stepAwait(t, repo, id, DefaultStages(), Options{})
		if *a.Reviewer != 0 || a.Class != ClassTop {
			t.Fatalf("the small reviewer passes research, which its sibling holds, for top: %+v", a)
		}
	})
	t.Run("a class the drive cannot launch is passed over", func(t *testing.T) {
		repo := briefRepo(t, agentsMarked)
		machineConfig(t, `{`+reviewers+`,"oracle":{"api":{"anthropic":{"base_url":"https://api.example.invalid/v1","key":"anthropic",`+
			`"models":["claude-fable-5","claude-sonnet-5-5","claude-haiku-4-5"]}}},"runner":{"claude":{}}}`)
		roots, _ := layered.RootsFor(repo.Root())
		cfg := walledFor(t, roots)
		start, err := Start(repo.Root(), "itd-10", Options{noReserve: true, Profile: "drive", Runners: cfg,
			Quota: func(context.Context, string) (runner.Quota, bool, error) { return runner.Quota{}, false, nil }})
		if err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), start.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if got := st.Review; !slices.Equal(got.unlaunchableFor(RoleRuthless), []string{ClassTop}) || !slices.Equal(got.unlaunchableFor(RoleSecurity), []string{ClassTop}) {
			t.Fatalf("the run records the class its runners cannot launch: %+v", got.Unlaunchable)
		}
		implemented(t, repo, start.RunID, DefaultStages(), "one.txt")
		setWriterModel(t, repo.Root(), start.RunID, "claude-haiku-4-5")
		a := stepAwait(t, repo, start.RunID, DefaultStages(), Options{})
		if a.Class != ClassDefault {
			t.Fatalf("the small reviewer passes research (its sibling's) and top (unlaunchable) for default: %+v", a)
		}
	})
}

// editedBytes is orig, a state file, with edit applied to its document.
func editedBytes(t *testing.T, orig []byte, edit func(doc map[string]any)) string {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(orig, &doc); err != nil {
		t.Fatal(err)
	}
	edit(doc)
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// firstValidator is the first validator of the first lane's round ri in doc.
func firstValidator(doc map[string]any, ri int) map[string]any {
	lane := doc["lanes"].([]any)[0].(map[string]any)
	round := lane["validation"].([]any)[ri].(map[string]any)
	return round["validators"].([]any)[0].(map[string]any)
}

// TestTheStoredModelsAreHeldToTheRestOfTheFile: an edited file is refused on
// its read where the writers are not the lane's receipts', a class map entry
// disagrees with the set's reviewer of that class, or a reviewer handed out
// records no model in a run started at version 27; a reviewer an earlier
// version handed out reads, and keeps reading after the file is written back.
func TestTheStoredModelsAreHeldToTheRestOfTheFile(t *testing.T) {
	repo, id := otherModelRun(t, "", "claude-fable-5")
	stepAwait(t, repo, id, DefaultStages(), Options{})
	orig := stateBytes(t, repo.Root(), id)
	// An edit that changes nothing reads, so each refusal below is its edit's.
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	if err := os.WriteFile(path, []byte(editedBytes(t, orig, func(map[string]any) {})), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(repo.Root(), id); err != nil {
		t.Fatalf("premise: the state file written back unedited reads: %v", err)
	}
	if err := os.WriteFile(path, orig, 0o600); err != nil {
		t.Fatal(err)
	}
	for name, edit := range map[string]func(doc map[string]any){
		"writers other than the receipts'": func(doc map[string]any) {
			firstValidator(doc, 0)["writer_models"] = []any{"claude-haiku-4-5"}
		},
		"a writer dropped": func(doc map[string]any) {
			lane := doc["lanes"].([]any)[0].(map[string]any)
			lane["receipts"].([]any)[0].(map[string]any)["route"] = map[string]any{"asked": "claude", "ran": "claude", "model": "claude-opus-5-5"}
		},
		"the judge's class map entry relabelled": func(doc map[string]any) {
			doc["review"].(map[string]any)["classes"].([]any)[1].(map[string]any)["model"] = "anthropic/claude-other-1"
		},
		"a reviewer's class relabelled with its map entry": func(doc map[string]any) {
			rs := doc["review"].(map[string]any)
			rs["classes"].([]any)[0].(map[string]any)["model"] = "anthropic/claude-other-1"
		},
		"a handed-out reviewer naming no model": func(doc map[string]any) {
			v := firstValidator(doc, 0)
			delete(v, "writer_models")
			delete(v, "reviewer_model")
		},
		"a handed-out reviewer marked an earlier version's in a run started at 27": func(doc map[string]any) {
			v := firstValidator(doc, 0)
			delete(v, "writer_models")
			delete(v, "reviewer_model")
			v["models_unrecorded"] = true
			v["class"], v["model"] = "research", BundledClasses[ClassResearch]
			lane := doc["lanes"].([]any)[0].(map[string]any)
			aw := lane["awaits"].([]any)[0].(map[string]any)
			aw["class"], aw["model"] = "research", BundledClasses[ClassResearch]
		},
		"a named model that is the reviewer's own": func(doc map[string]any) {
			firstValidator(doc, 0)["reviewer_named"] = BundledClasses[ClassTop]
		},
	} {
		t.Run(name, func(t *testing.T) { stateRefusedOnRead(t, repo.Root(), id, orig, editedBytes(t, orig, edit)) })
	}
	t.Run("an earlier version's reviewer", func(t *testing.T) {
		repo, id := otherModelRun(t, "", "grok-test-1")
		first := stepAwait(t, repo, id, DefaultStages(), Options{})
		claimAsDriver(t, repo.Root(), id, Options{})
		path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
		if err := os.WriteFile(path, downgraded(t, stateBytes(t, repo.Root(), id), schemaVersionUnmodelled), 0o600); err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		if v := st.Lanes[0].Validation[0].Validators[0]; !v.ModelsUnrecorded || len(v.WriterModels) != 0 {
			t.Fatalf("the reviewer it handed out is marked as it is read: %+v", v)
		}
		// The next step writes the file back at version 27, and it reads.
		stepAwait(t, repo, id, DefaultStages(), Options{})
		if !strings.Contains(string(stateBytes(t, repo.Root(), id)), `"models_unrecorded": true`) {
			t.Fatal("the mark is written back")
		}
		returnAt(t, repo, id, DefaultStages(), first, reviewerReturn("SHIP"))
		if v := validatorAt(t, repo.Root(), id, first.Receipt); v.Verdict != "SHIP" || v.ReviewerModel != "" {
			t.Fatalf("its return is checked and recorded, keeping no model: %+v", v)
		}
	})
}

// TestTheModelsAreVersion27s: a version-26 file carrying a reviewer's models or
// the class map is refused naming them, and one without them reads as the
// current version.
func TestTheModelsAreVersion27s(t *testing.T) {
	// A writer whose model no class holds, so the reviewer keeps the set's
	// class, which a version-26 file can carry.
	repo, id := otherModelRun(t, "", "grok-test-1")
	stepAwait(t, repo, id, DefaultStages(), Options{})
	data := stateBytes(t, repo.Root(), id)
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	at := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	if SchemaVersion != schemaVersionUnmodelled+4 || !strings.Contains(string(data), at) || !strings.Contains(string(data), `"writer_models"`) {
		t.Fatalf("premise: the file is the current version, four after version 26 (version 28 added the judge, 29 its kept strikes, 30 a run's members), and carries the fields: %s", data)
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

// TestAModelIsPrintedOnlyAsAPlainToken: a name that could mention a person,
// make a link or break out of a code span is described, never printed, and a
// printed name stands in a code span.
func TestAModelIsPrintedOnlyAsAPlainToken(t *testing.T) {
	for _, name := range []string{"@name", "https://host/x", "claude`opus", "claude-opus-5-5[1m]", "a b", "/lead", "trail:", "x//y", "a\nb"} {
		if got := shownModel(name); got == name || strings.Contains(got, name) {
			t.Errorf("%q is printed: %q", name, got)
		}
		if got := modelsPhrase([]string{name}, "anthropic/claude-sonnet-5-5", "", "", false, false); strings.Contains(got, name) {
			t.Errorf("%q reaches the record: %q", name, got)
		}
	}
	for _, name := range []string{"anthropic/claude-opus-5-5", "Claude:claude-opus-5-5", "claude_opus-5.5"} {
		if got := shownModel(name); got != name {
			t.Errorf("%q is a plain token, shown %q", name, got)
		}
	}
	if got := modelsPhrase([]string{"grok-test-1"}, "anthropic/claude-sonnet-5-5", "", "", false, false); got != "written by `grok-test-1`, reviewed by `anthropic/claude-sonnet-5-5`" {
		t.Errorf("a shown name stands in a code span: %q", got)
	}
}
