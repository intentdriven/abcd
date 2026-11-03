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
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gittest"
)

// Rounds keyed by reviewer, and the stricter verdict across them
// (itd-2610240922461281, spc-2610240951109420 step 3).

// oneReviewerPerRole sets the test's machine layer to a review set of one
// reviewer per role, at the middle class, for a test of something else that
// counts one ruthless and one security reviewer a round; it is written over
// the test's HOME, so call it after the repository is made and before the
// run starts.
func oneReviewerPerRole(t *testing.T) {
	t.Helper()
	machineConfig(t, `{"review":{"reviewers":["middle"]}}`)
}

// bundledClassProvider configures the provider the bundled class map's
// models name, listing them, so a runner handed a reviewer's class model
// admits it (the runner's admission, spc-2610240951109420 step 2).
const bundledClassProvider = `"oracle":{"api":{"anthropic":{"base_url":"https://api.anthropic.com/v1","key":"anthropic","models":["claude-opus-5-5","claude-sonnet-5-5"]}}}`

// stepAwait takes the run's next step and returns what it handed out.
func stepAwait(t *testing.T, repo *gittest.Repo, runID string, stages Stages, o Options) Await {
	t.Helper()
	res, err := advance(repo.Root(), runID, stages, o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil || !res.handed {
		t.Fatalf("the step hands an agent out: %+v", res)
	}
	return *res.Awaiting
}

// returnAt writes body as the return a and hands it back.
func returnAt(t *testing.T, repo *gittest.Repo, runID string, stages Stages, a Await, body string) StepResult {
	t.Helper()
	path := filepath.Join(repo.Root(), filepath.FromSlash(a.Receipt))
	if a.Role == RoleAuditor {
		body = auditorVerdict(t, filepath.Join(filepath.Dir(path), AuditRequestFileName), body)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Receipt(repo.Root(), runID, path, stages, Options{})
	if err != nil {
		t.Fatalf("the %s's return at %s: %v", a.Role, a.Receipt, err)
	}
	return res
}

// wantReviewer checks an await names a review set's reviewer.
func wantReviewer(t *testing.T, a Await, role string, index int, class string) {
	t.Helper()
	if a.Role != role || a.Reviewer == nil || *a.Reviewer != index || a.Class != class || a.Model != BundledClasses[class] {
		t.Fatalf("want %s reviewer %d at %s (%s): %+v (reviewer %v)", role, index, class, BundledClasses[class], a, a.Reviewer)
	}
}

// TestTwoReviewersOfOneRoleAreKeyedByIndex: under the bundled pair, each
// reviewer role's two reviewers are handed out in consecutive steps on the
// round's one head, each awaiting naming its index, its class and the model
// the class resolves to; each has its own round path, and their briefs differ
// only in their return path; a return is matched to its reviewer by that path,
// so neither overwrites the other, whichever comes back first.
func TestTwoReviewersOfOneRoleAreKeyedByIndex(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	l := implemented(t, repo, id, stages, "one.txt")

	first := stepAwait(t, repo, id, stages, Options{})
	second := stepAwait(t, repo, id, stages, Options{})
	wantReviewer(t, first, RoleRuthless, 0, ClassMiddle)
	wantReviewer(t, second, RoleRuthless, 1, ClassBelow)

	dir0, dir1 := filepath.Dir(first.Receipt), filepath.Dir(second.Receipt)
	if dir0 == dir1 || !strings.HasSuffix(dir0, "/round-1/"+RoleRuthless) || !strings.HasSuffix(dir1, "/round-1/"+RoleRuthless+".1") {
		t.Fatalf("each reviewer has its own round path: %s and %s", dir0, dir1)
	}
	if filepath.Dir(first.Brief) != dir0 || filepath.Dir(second.Brief) != dir1 {
		t.Fatalf("each brief sits in its reviewer's directory: %+v %+v", first, second)
	}
	b0, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(first.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	b1, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(second.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	r0, r1 := abs(repo.Root(), first.Receipt), abs(repo.Root(), second.Receipt)
	if !strings.Contains(string(b1), r1) || strings.Contains(string(b1), r0) {
		t.Fatalf("the second brief names its own return alone:\n%s", b1)
	}
	if string(b0) == string(b1) || strings.ReplaceAll(string(b1), r1, r0) != string(b0) {
		t.Fatalf("the two briefs differ only in their return path:\n--- 0\n%s\n--- 1\n%s", b0, b1)
	}

	// The payload names the index, the class and its model; index 0 too.
	for i, a := range []Await{first, second} {
		raw, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{fmt.Sprintf(`"reviewer":%d`, i), `"class":"` + a.Class + `"`, `"model":"` + a.Model + `"`} {
			if !strings.Contains(string(raw), want) {
				t.Fatalf("the awaiting payload names %s: %s", want, raw)
			}
		}
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	cur := st.Lanes[0].Validation[0]
	if cur.HeadSHA != l.HeadSHA || len(st.Lanes[0].Awaits) != 2 {
		t.Fatalf("both reviewers are out on the round's one head: %+v", st.Lanes[0])
	}
	var got []string
	for _, v := range cur.Validators {
		got = append(got, fmt.Sprintf("%s/%d/%s", v.Role, v.Reviewer, v.Class))
	}
	want := []string{RoleRuthless + "/0/middle", RoleRuthless + "/1/below", RoleSecurity + "/0/middle", RoleSecurity + "/1/below", RoleAuditor + "/0/"}
	if !slices.Equal(got, want) {
		t.Fatalf("the round's validators, in reviewer order: %v, want %v", got, want)
	}

	// The second reviewer returns first: its verdict is its own.
	res := returnAt(t, repo, id, stages, second, reviewerReturn("FIX FIRST"))
	st, _ = ReadState(repo.Root(), id)
	vs := st.Lanes[0].Validation[0].Validators
	if vs[1].Verdict != "FIX FIRST" || vs[1].Pass || vs[0].Verdict != "" || len(st.Lanes[0].Awaits) != 1 || st.Lanes[0].Awaits[0].Receipt != first.Receipt {
		t.Fatalf("the return is matched to reviewer 1 alone: %+v awaits %+v (%+v)", vs[:2], st.Lanes[0].Awaits, res)
	}
	if vs[1].ModelStatus != ModelAsked || vs[1].Route != nil {
		t.Fatalf("a host-run reviewer's model is recorded as asked, not confirmed: %+v", vs[1])
	}
	if len(st.Lanes[0].Validation[0].Comparisons) != 0 {
		t.Fatalf("no comparison while a reviewer of the role is out: %+v", st.Lanes[0].Validation[0].Comparisons)
	}
	returnAt(t, repo, id, stages, first, reviewerReturn("SHIP"))
	st, _ = ReadState(repo.Root(), id)
	cmps := st.Lanes[0].Validation[0].Comparisons
	if len(cmps) != 1 || cmps[0].Role != RoleRuthless || cmps[0].Outcome != ComparisonDisagree || cmps[0].Reviewers != 2 ||
		!slices.Equal(cmps[0].Verdicts, []string{"SHIP", "FIX FIRST"}) {
		t.Fatalf("SHIP against FIX FIRST is a disagreement: %+v", cmps)
	}
	if note := st.Record[len(st.Record)-1].Note; !strings.Contains(note, "disagree") || !strings.Contains(note, "stricter verdict") {
		t.Fatalf("the record says the reviewers disagree: %s", note)
	}
	rec, err := ReadRecord(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	rv := rec.Lanes[0].Verdicts
	if len(rv) != 2 || rv[1].Reviewer != 1 || rv[1].Class != ClassBelow || rv[1].Model != BundledClasses[ClassBelow] || rv[1].ModelStatus != ModelAsked ||
		rv[0].Reviewer != 0 || rv[0].Class != ClassMiddle || rv[0].Verdict != "SHIP" {
		t.Fatalf("the run record names each reviewer's index, class, model and its status: %+v", rv)
	}
	if c := rec.Lanes[0].Comparisons; len(c) != 1 || c[0].Round != 1 || c[0].Outcome != ComparisonDisagree {
		t.Fatalf("the run record carries the round's comparison: %+v", c)
	}

	// The security reviewers both fail, BLOCK and NEEDS-INPUT: that agrees.
	sec0 := stepAwait(t, repo, id, stages, Options{})
	sec1 := stepAwait(t, repo, id, stages, Options{})
	wantReviewer(t, sec0, RoleSecurity, 0, ClassMiddle)
	wantReviewer(t, sec1, RoleSecurity, 1, ClassBelow)
	returnAt(t, repo, id, stages, sec0, reviewerReturn("BLOCK"))
	returnAt(t, repo, id, stages, sec1, reviewerReturn("NEEDS-INPUT"))
	aud := stepAwait(t, repo, id, stages, Options{})
	if aud.Role != RoleAuditor || aud.Reviewer != nil || aud.Class != "" {
		t.Fatalf("the auditor keeps its single run and names no reviewer: %+v", aud)
	}
	returnAt(t, repo, id, stages, aud, "MET")
	st, _ = ReadState(repo.Root(), id)
	cmps = st.Lanes[0].Validation[0].Comparisons
	if len(cmps) != 2 || cmps[1].Role != RoleSecurity || cmps[1].Outcome != ComparisonAgree {
		t.Fatalf("two failing verdicts are agreement: %+v", cmps)
	}

	// One reviewer failing fails the round: the stricter verdict decides.
	fix := stepAwait(t, repo, id, stages, Options{})
	if fix.Role != RoleImplementer {
		t.Fatalf("a round a reviewer failed goes to a fresh implementer: %+v", fix)
	}
	fb, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(fix.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"ruthless-reviewer (reviewer 1 at below (" + BundledClasses[ClassBelow] + ")): FIX FIRST", r1,
		"security-reviewer (reviewer 0 at middle", "security-reviewer (reviewer 1 at below"} {
		if !strings.Contains(string(fb), w) {
			t.Fatalf("the fix brief names %q:\n%s", w, fb)
		}
	}
	if strings.Contains(string(fb), r0) {
		t.Fatalf("the fix brief names only the reviewers that failed:\n%s", fb)
	}
}

// TestCompareRoleIsThePassBitOfEachVerdict: two reviewers disagree exactly
// when one verdict passes and another fails; two passing and two failing
// verdicts agree; a role that ran one reviewer records that no comparison was
// made; and a round no review set opened records none.
func TestCompareRoleIsThePassBitOfEachVerdict(t *testing.T) {
	v := func(role string, i int, word string) ValidatorRun {
		var pass bool
		for _, w := range reviewerVerdicts[role] {
			if w.word == word {
				pass = w.pass
			}
		}
		return ValidatorRun{Role: role, Reviewer: i, Class: Classes[i], Model: BundledClasses[Classes[i]], Verdict: word, Pass: pass}
	}
	for _, tc := range []struct {
		vs   []ValidatorRun
		want string
	}{
		{[]ValidatorRun{v(RoleRuthless, 0, "SHIP"), v(RoleRuthless, 1, "SHIP")}, ComparisonAgree},
		{[]ValidatorRun{v(RoleRuthless, 0, "SHIP"), v(RoleRuthless, 1, "FIX FIRST")}, ComparisonDisagree},
		{[]ValidatorRun{v(RoleRuthless, 0, "FIX FIRST"), v(RoleRuthless, 1, "SHIP")}, ComparisonDisagree},
		{[]ValidatorRun{v(RoleSecurity, 0, "BLOCK"), v(RoleSecurity, 1, "NEEDS-INPUT")}, ComparisonAgree},
		{[]ValidatorRun{v(RoleSecurity, 0, "APPROVE"), v(RoleSecurity, 1, "NEEDS-INPUT")}, ComparisonDisagree},
		{[]ValidatorRun{v(RoleSecurity, 0, "BLOCK")}, ComparisonNone},
	} {
		r := ValidationRound{Validators: tc.vs}
		cmp, ok := compareRole(&r, tc.vs[0].Role)
		if !ok || cmp.Outcome != tc.want || len(r.Comparisons) != 1 || cmp.Reviewers != len(tc.vs) {
			t.Errorf("%s: %+v (recorded %v), want %s", verdictsLine(r), cmp, ok, tc.want)
		}
	}
	r := ValidationRound{Validators: []ValidatorRun{v(RoleRuthless, 0, "SHIP"), {Role: RoleRuthless, Reviewer: 1, Class: ClassBelow, Model: BundledClasses[ClassBelow]}}}
	if _, ok := compareRole(&r, RoleRuthless); ok || len(r.Comparisons) != 0 {
		t.Fatalf("no comparison while a reviewer is out: %+v", r.Comparisons)
	}
	r = ValidationRound{Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "SHIP", Pass: true}}}
	if _, ok := compareRole(&r, RoleRuthless); ok {
		t.Fatal("a round no review set opened records no comparison")
	}
	if c := comparisonNote(Comparison{Role: RoleSecurity, Reviewers: 1, Outcome: ComparisonNone, Verdicts: []string{"BLOCK"}}); !strings.Contains(c, "no comparison was made") {
		t.Fatalf("a one-reviewer role says no comparison was made: %s", c)
	}
}

// TestBothReviewersAreHandedOutWithinTheSubAgentCeiling: at a ceiling of one,
// the role's second reviewer waits for a slot rather than being handed out
// beside the first, and takes it, on the same head, once the first returns;
// a review set of one reviewer runs each role once and records that no
// comparison was made.
func TestBothReviewersAreHandedOutWithinTheSubAgentCeiling(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{SubAgents: strp("1")})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	l := implemented(t, repo, id, stages, "one.txt")
	first := stepAwait(t, repo, id, stages, Options{})
	wantReviewer(t, first, RoleRuthless, 0, ClassMiddle)
	res, err := advance(repo.Root(), id, stages, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.CeilingReached || res.handed || res.Slots != 1 || res.Ceiling != 1 {
		t.Fatalf("the second reviewer waits at the ceiling: %+v", res)
	}
	returnAt(t, repo, id, stages, first, reviewerReturn("SHIP"))
	second := stepAwait(t, repo, id, stages, Options{})
	wantReviewer(t, second, RoleRuthless, 1, ClassBelow)
	st, _ := ReadState(repo.Root(), id)
	if st.Lanes[0].Validation[0].HeadSHA != l.HeadSHA || len(st.Lanes[0].Validation) != 1 {
		t.Fatalf("the second reviewer reads the round's one head: %+v", st.Lanes[0].Validation)
	}

	// One reviewer per role: each role runs once, and no comparison is made.
	repo = briefRepo(t, agentsMarked)
	machineConfig(t, `{"review":{"reviewers":["below"]}}`)
	start, err = Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id = start.RunID
	implemented(t, repo, id, stages, "one.txt")
	ruth := stepAwait(t, repo, id, stages, Options{})
	wantReviewer(t, ruth, RoleRuthless, 0, ClassBelow)
	sec := stepAwait(t, repo, id, stages, Options{})
	wantReviewer(t, sec, RoleSecurity, 0, ClassBelow)
	returnAt(t, repo, id, stages, ruth, reviewerReturn("SHIP"))
	st, _ = ReadState(repo.Root(), id)
	cmps := st.Lanes[0].Validation[0].Comparisons
	if len(st.Lanes[0].Validation[0].Validators) != 3 || len(cmps) != 1 || cmps[0].Outcome != ComparisonNone {
		t.Fatalf("one reviewer per role, and the record says no comparison was made: %+v", st.Lanes[0].Validation[0])
	}
	if note := st.Record[len(st.Record)-1].Note; !strings.Contains(note, "no comparison was made") {
		t.Fatalf("the record says no comparison was made: %s", note)
	}
}

// TestAnOlderStateMigratesItsReviewerToIndexZero: a state written before
// rounds were keyed by reviewer reads at the current version with its round's
// one reviewer per role as index 0, at the path index 0 still takes, asked
// for no class, and its return is recorded there; one of that version
// carrying a reviewer's index, class or model, an await's reviewer or a
// round's comparisons is not one its version wrote, and is refused naming it.
func TestAnOlderStateMigratesItsReviewerToIndexZero(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implemented(t, repo, id, stages, "one.txt")
	first := stepAwait(t, repo, id, stages, Options{})
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	keyed := stateBytes(t, repo.Root(), id)

	// As version 19 wrote it: one reviewer per role, no index, class or model.
	var doc map[string]any
	if err := json.Unmarshal(keyed, &doc); err != nil {
		t.Fatal(err)
	}
	doc["schema_version"] = schemaVersionUnkeyed
	lane := doc["lanes"].([]any)[0].(map[string]any)
	for _, a := range lane["awaits"].([]any) {
		for _, k := range []string{"reviewer", "class", "model"} {
			delete(a.(map[string]any), k)
		}
	}
	round := lane["validation"].([]any)[0].(map[string]any)
	var kept []any
	for _, v := range round["validators"].([]any) {
		m := v.(map[string]any)
		if _, later := m["reviewer"]; later {
			continue
		}
		for _, k := range []string{"class", "model"} {
			delete(m, k)
		}
		kept = append(kept, m)
	}
	round["validators"] = kept
	older, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	for name, edit := range map[string]func(string) string{
		"a reviewer's class": func(s string) string {
			return strings.Replace(s, `"pass": false`, `"pass": false, "class": "middle", "model": "`+BundledClasses[ClassMiddle]+`"`, 1)
		},
		"a reviewer's index": func(s string) string { return strings.Replace(s, `"pass": false`, `"pass": false, "reviewer": 1`, 1) },
		"an await's reviewer": func(s string) string {
			return strings.Replace(s, `"role": "ruthless-reviewer",`, `"role": "ruthless-reviewer", "reviewer": 0,`, 1)
		},
		"a round's comparison": func(s string) string {
			return strings.Replace(s, `"validators": [`, `"comparisons": [], "validators": [`, 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			edited := edit(string(older))
			if edited == string(older) {
				t.Fatalf("the edit changed nothing:\n%s", older)
			}
			if name == "a round's comparison" {
				// An empty list is dropped on read; one entry is carried.
				edited = strings.Replace(edited, `"comparisons": []`, `"comparisons": [{"role":"ruthless-reviewer","reviewers":1,"outcome":"none","verdicts":["SHIP"]}]`, 1)
			}
			if err := os.WriteFile(path, []byte(edited), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := ReadState(repo.Root(), id)
			r := mustRefusal(t, err)
			if r.Stage != "state" || !strings.Contains(r.Reason, fmt.Sprintf("schema version %d", schemaVersionUnkeyed)) {
				t.Fatalf("a version-%d file carrying %s is refused: %+v", schemaVersionUnkeyed, name, r)
			}
		})
	}

	if err := os.WriteFile(path, older, 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatalf("a version-%d file reads: %v", schemaVersionUnkeyed, err)
	}
	vs := st.Lanes[0].Validation[0].Validators
	if st.SchemaVersion != SchemaVersion || len(vs) != 3 || vs[0].Reviewer != 0 || vs[0].Class != "" || vs[0].Return != first.Receipt {
		t.Fatalf("its reviewer is index 0 at the path index 0 takes: %d %+v", st.SchemaVersion, vs)
	}
	returnAt(t, repo, id, stages, Await{Role: RoleRuthless, Receipt: first.Receipt}, reviewerReturn("SHIP"))
	st, _ = ReadState(repo.Root(), id)
	if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "SHIP" || v.ModelStatus != "" || len(st.Lanes[0].Validation[0].Comparisons) != 0 {
		t.Fatalf("the migrated reviewer's verdict is recorded, with no class to compare: %+v", st.Lanes[0].Validation[0])
	}
	if !strings.Contains(string(stateBytes(t, repo.Root(), id)), fmt.Sprintf(`"schema_version": %d`, SchemaVersion)) {
		t.Fatal("the next write carries the current version")
	}
}

// TestAStoredReviewerTheLoopNeverWritesIsRefused: a reviewer's class, model,
// index and model status read back from the state are held to what the loop
// writes, so a hand edit never reaches a runner's request or a payload.
func TestAStoredReviewerTheLoopNeverWritesIsRefused(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implemented(t, repo, id, stages, "one.txt")
	stepAwait(t, repo, id, stages, Options{})
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(id)))
	orig := stateBytes(t, repo.Root(), id)
	model := BundledClasses[ClassMiddle]
	for name, edit := range map[string]func(string) string{
		"a class the map does not name": func(s string) string { return strings.Replace(s, `"class": "below"`, `"class": "huge"`, 1) },
		"a model that is no route":      func(s string) string { return strings.Replace(s, `"model": "`+model+`"`, `"model": "no-slash"`, 1) },
		"an index past the ceiling":     func(s string) string { return strings.Replace(s, `"reviewer": 1`, `"reviewer": 7`, 1) },
		"a model status it never writes": func(s string) string {
			return strings.Replace(s, `"class": "below"`, `"class": "below", "model_status": "confirmed"`, 1)
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
			if r := mustRefusal(t, func() error { _, err := ReadState(repo.Root(), id); return err }()); r.Stage != "state" {
				t.Fatalf("want the state's refusal: %+v", r)
			}
		})
	}
}

// TestARunnerRunReviewerIsHandedItsClassModel: a review set's reviewer routed
// to a runner is started at its class's model, over the runner's own, and its
// record says the model the runner reported stands beside the one asked for;
// the same reviewer on the host records its model as asked, not confirmed.
func TestARunnerRunReviewerIsHandedItsClassModel(t *testing.T) {
	const model = "local/qwen3-small"
	review := func(t *testing.T, routed bool) (ValidatorRun, driveEnv) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		start, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		steps := driveSteps(t.TempDir())
		const briefRel, returnRel = "review/brief.md", "review/return.md"
		if err := os.MkdirAll(filepath.Join(repo.Root(), "review"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo.Root(), "review", "brief.md"), []byte("# review\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		v := ValidatorRun{Role: RoleRuthless, Brief: briefRel, Return: returnRel, Reviewer: 1, Class: ClassBelow, Model: model}
		steps[2] = StageDef{Name: StageImplement, Piece: 7,
			Run: func(c Context, l *Lane) (Outcome, error) {
				if len(l.Validation) == 0 {
					l.Validation = []ValidationRound{{Round: 1, HeadSHA: strings.Repeat("a", 40), Validators: []ValidatorRun{v}}}
				}
				aw := v.await()
				return Outcome{Await: &aw}, nil
			},
			Verify: func(c Context, l *Lane, receipt string) error {
				return verifyValidation(c, l, receipt)
			}}
		o := Options{Now: func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) }}
		for range 2 {
			if _, err := advance(repo.Root(), start.RunID, steps, o); err != nil {
				t.Fatal(err)
			}
		}
		var env driveEnv
		if routed {
			env = newDriveEnv(t, "ok", "claude")
			cfg := runnerConfig(t, `{"roles":{"ruthless-reviewer":{"runner":"claude"}},`+
				`"oracle":{"api":{"local":{"base_url":"http://localhost:11434/v1","models":["qwen3-coder","qwen3-small"]}}},`+
				`"runner":{"claude":{"model":"local/qwen3-coder"}}}`, "")
			res, err := Drive(context.Background(), repo.Root(), start.RunID, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
			if err != nil || res.PerformedStage != StageImplement {
				t.Fatalf("the runner's review is verified: %+v %v", res, err)
			}
		} else {
			res, err := advance(repo.Root(), start.RunID, steps, o)
			if err != nil || res.Awaiting == nil || res.Awaiting.Model != model || !strings.Contains(res.Next, model) {
				t.Fatalf("the host is handed the review with its class's model: %+v %v", res, err)
			}
			if err := os.WriteFile(filepath.Join(repo.Root(), returnRel), []byte(reviewReturn), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Receipt(repo.Root(), start.RunID, returnRel, steps, o); err != nil {
				t.Fatal(err)
			}
		}
		st, err := ReadState(repo.Root(), start.RunID)
		if err != nil {
			t.Fatal(err)
		}
		return st.Lanes[0].Validation[0].Validators[0], env
	}
	host, _ := review(t, false)
	if host.Route != nil || host.ModelStatus != ModelAsked || host.Model != model {
		t.Fatalf("a host-run reviewer's model is recorded as asked, not confirmed: %+v", host)
	}
	routed, env := review(t, true)
	if !slices.Contains(env.argv(t, "claude"), "--model=qwen3-small") {
		t.Fatalf("the runner is handed the class's model, not its own: %v", env.argv(t, "claude"))
	}
	if routed.ModelStatus != ModelReported || routed.Route == nil || routed.Route.Model == "" || routed.Route.Asked != runner.Claude {
		t.Fatalf("a runner-run reviewer records the model its runner reported beside the one asked: %+v route %+v", routed, routed.Route)
	}
	if routed.Route.ModelAsked != "qwen3-small" {
		t.Fatalf("the route records the class's model as the one asked: %+v", routed.Route)
	}
}
