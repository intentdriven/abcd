package reading

// candidates_test.go holds itd-2609020625407419's acceptance criteria over the
// comparative channel: the derived run and its two refusals, the projection, the
// ordering guard, the fixed interpretation, and the exclusions the manifest
// asserts (adr-2609021016272867; companion 7.2, 7.4, 7.5, 7.6, R3, R4;
// divergence register 2, 3, 6, 22, 23).

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// assembleComparative runs one comparative assembly over a fixture and returns
// the result and the error, so a case can assert on either.
func assembleComparative(t *testing.T, root string) (AssembleResult, error) {
	t.Helper()
	return Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionComparative, Target: "HEAD", DryRun: true,
	})
}

// removeFixtureRun drops the base fixture's committed widening run, for a case
// about a repository with no candidate set at all.
func removeFixtureRun(t *testing.T, root string) {
	t.Helper()
	for _, rel := range []string{
		".abcd/development/readings/" + fixtureCandidateRun,
		".abcd/work/issues/readings/" + fixtureCandidateRun,
	} {
		if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatal(err)
		}
	}
	gitCommitAll(t, root)
}

// ac-1. TestComparativeDerivesTheOneUndispositionedWideningRun: two committed
// widening runs at the target, one of them dispositioned; the other is selected
// and the manifest names it. No operand chose it — the invocation is a position
// and a target state (adr-2609021016286571; brief invariant 15).
func TestComparativeDerivesTheOneUndispositionedWideningRun(t *testing.T) {
	root := fixtureRepo(t)
	const other = "rdg-2608301200000011"
	items := plantWideningItems(t, root, other, 2)
	for i, item := range items {
		plantDisposition(t, root, item, "dsp-"+string(rune('1'+i)))
	}
	gitCommitAll(t, root)
	commitRunRecord(t, root, other, string(PositionWidening), headOf(t, root))

	res, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("the comparative assembly refused with one qualifying run: %v", err)
	}
	if res.CandidateRun != fixtureCandidateRun {
		t.Fatalf("the assembly derived %q, want %q: the qualifying run is the one whose items "+
			"carry no disposition", res.CandidateRun, fixtureCandidateRun)
	}
	if res.Manifest.CandidateRun != fixtureCandidateRun {
		t.Errorf("the manifest names the candidate run %q, want %q; a reader of the manifest has "+
			"to be able to check the selection against the record",
			res.Manifest.CandidateRun, fixtureCandidateRun)
	}
	if res.Manifest.Candidates != 3 {
		t.Errorf("the manifest records %d candidates, want 3 — the derived run's item count",
			res.Manifest.Candidates)
	}
	// And nothing of the dispositioned run travels.
	for _, m := range res.Manifest.Items {
		if strings.Contains(m.Path, other) {
			t.Errorf("the bundle carries %s, which belongs to the run that did not qualify", m.Path)
		}
	}
}

// ac-1. TestTwoUndispositionedWideningRunsRefuseNamingThem: the ambiguous case
// the design's own next act resolves. The refusal lists both, with counts and
// disposition state, so the operator can see what to disposition.
func TestTwoUndispositionedWideningRunsRefuseNamingThem(t *testing.T) {
	root := fixtureRepo(t)
	const second = "rdg-2608301200000012"
	plantWideningItems(t, root, second, 2)
	gitCommitAll(t, root)
	commitRunRecord(t, root, second, string(PositionWidening), headOf(t, root))

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("two qualifying widening runs assembled; nothing names which, and picking one " +
			"would be a resolution order deciding silently")
	}
	var ambiguous *AmbiguousCandidateRun
	if !errors.As(err, &ambiguous) {
		t.Fatalf("the refusal is not AmbiguousCandidateRun: %v", err)
	}
	if len(ambiguous.Runs) != 2 {
		t.Errorf("the refusal lists %d run(s), want both", len(ambiguous.Runs))
	}
	msg := err.Error()
	for _, want := range []string{
		fixtureCandidateRun, second, "3 item(s)", "2 item(s)", "disposition",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not carry %q: %s", want, msg)
		}
	}
}

// ac-1. TestNoQualifyingWideningRunRefuses: one run planted, every item
// dispositioned. It is LISTED rather than hidden, and the refusal says why it
// did not qualify — which is what the operator needs in order to understand the
// state (adr-2609021016272867, "The operator is told what the assembler selected
// and why").
func TestNoQualifyingWideningRunRefuses(t *testing.T) {
	root := fixtureRepo(t)
	for i, item := range fixtureCandidateItems {
		plantDisposition(t, root, item, "dsp-"+string(rune('1'+i)))
	}
	gitCommitAll(t, root)

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("a fully dispositioned widening run was taken as a candidate set; the candidate " +
			"set is defined as pre-admission")
	}
	var none *NoCandidateRun
	if !errors.As(err, &none) {
		t.Fatalf("the refusal is not NoCandidateRun: %v", err)
	}
	if len(none.Runs) != 1 {
		t.Fatalf("the refusal lists %d run(s); a run that did not qualify is listed rather than "+
			"hidden, because that is what says why", len(none.Runs))
	}
	if !none.Runs[0].Dispositioned {
		t.Error("the listing does not report the run as dispositioned")
	}
	for _, want := range []string{fixtureCandidateRun, "3 item(s)", "dsp-1", "PRE-ADMISSION"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not carry %q: %v", want, err)
		}
	}
}

// ac-1. TestNoWideningRunAtTheTargetRefusesSayingSo: a target with no committed
// widening run at all says so in the first refusal, rather than listing nothing
// and leaving the operator to guess which of the reasons applied.
func TestNoWideningRunAtTheTargetRefusesSayingSo(t *testing.T) {
	root := fixtureRepo(t)
	removeFixtureRun(t, root)

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("a comparative assembly ran over a repository with no widening run")
	}
	var none *NoCandidateRun
	if !errors.As(err, &none) {
		t.Fatalf("the refusal is not NoCandidateRun: %v", err)
	}
	if len(none.Runs) != 0 {
		t.Errorf("the refusal lists %d run(s) where none is committed", len(none.Runs))
	}
	if !strings.Contains(err.Error(), "no widening run is committed at this target") {
		t.Errorf("the refusal does not say there is no widening run at all: %v", err)
	}
}

// TestAWideningRunAtAnotherTargetIsNotDerived: the derivation is by TARGET, and
// a run that read a different commit read a different object. It is not listed
// either: the listing is the widening runs at THIS target.
func TestAWideningRunAtAnotherTargetIsNotDerived(t *testing.T) {
	root := fixtureRepo(t)
	removeFixtureRun(t, root)
	const elsewhere = "rdg-2608301200000013"
	plantWideningItems(t, root, elsewhere, 3)
	gitCommitAll(t, root)
	commitRunRecord(t, root, elsewhere, string(PositionWidening),
		"0123456789abcdef0123456789abcdef01234567")

	_, err := assembleComparative(t, root)
	var none *NoCandidateRun
	if !errors.As(err, &none) {
		t.Fatalf("a run at another target was derived: %v", err)
	}
	if len(none.Runs) != 0 {
		t.Errorf("the listing carries a run at another target: %+v", none.Runs)
	}
}

// TestARunAtAnotherPositionIsNotACandidateSet: the object is the WIDENING
// reading's returned configurations, and a run at any other position is not a
// candidate set (companion 7.2, R4).
func TestARunAtAnotherPositionIsNotACandidateSet(t *testing.T) {
	root := fixtureRepo(t)
	removeFixtureRun(t, root)
	const detection = "rdg-2608301200000014"
	plantWideningItems(t, root, detection, 3)
	gitCommitAll(t, root)
	commitRunRecord(t, root, detection, string(PositionDetection), headOf(t, root))

	_, err := assembleComparative(t, root)
	var none *NoCandidateRun
	if !errors.As(err, &none) {
		t.Fatalf("a detection run was derived as a candidate set: %v", err)
	}
	if len(none.Runs) != 0 {
		t.Errorf("the listing carries a run at another position: %+v", none.Runs)
	}
}

// TestComparativeRefusesAnUncommittedRun: a run without its commit marker never
// happened and its records are the next ingest sweep's to roll back, so it is
// listed and never selected.
func TestComparativeRefusesAnUncommittedRun(t *testing.T) {
	root := fixtureRepo(t)
	removeFixtureRun(t, root)
	const parked = "rdg-2608301200000015"
	plantWideningItems(t, root, parked, 3)
	gitCommitAll(t, root)
	parkRunManifest(t, root, parked, string(PositionWidening), headOf(t, root))

	_, err := assembleComparative(t, root)
	var none *NoCandidateRun
	if !errors.As(err, &none) {
		t.Fatalf("a run with no commit marker was derived: %v", err)
	}
	if len(none.Runs) != 1 || none.Runs[0].Committed {
		t.Fatalf("the listing does not report the run as uncommitted: %+v", none.Runs)
	}
	for _, want := range []string{parked, "NOT committed", "never happened"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not carry %q: %v", want, err)
		}
	}
}

// ac-2. TestComparativeBundleCarriesTwoFieldsPerCandidate: a three-item run with
// no fate; the bundle carries each item's identifier, its configuration and what
// admits it, and no other field of those items. The pattern stays behind:
// provenance is the envelope's, not the candidate's (divergence register 23).
func TestComparativeBundleCarriesTwoFieldsPerCandidate(t *testing.T) {
	root := fixtureRepo(t)
	res, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	seen := map[string][]string{}
	for _, it := range res.Bundle.Items {
		if it.Kind != KindCandidate {
			continue
		}
		if it.Candidate == "" {
			t.Errorf("a candidate item carries no candidate id; the body cites one, so the reading "+
				"has to be told which rdi-N each text belongs to: %+v", it)
		}
		seen[it.Candidate] = append(seen[it.Candidate], it.Field)
	}
	if len(seen) != 3 {
		t.Fatalf("the bundle carries candidates from %d item(s), want 3", len(seen))
	}
	for id, fields := range seen {
		if strings.Join(fields, "|") != strings.Join(CandidateFields, "|") {
			t.Errorf("candidate %s travels as %v, want %v", id, fields, CandidateFields)
		}
	}
	// The carrier arrives and the envelope does not.
	text := bundleText(res.Bundle)
	if !strings.Contains(text, sentinelCandidate) {
		t.Error("the configuration text did not reach the bundle; the candidate channel carries " +
			"the returned text or it carries nothing")
	}
	if strings.Contains(text, sentinelEnvelope) {
		t.Error("the item's pattern reached the bundle; provenance is the envelope's and not the " +
			"candidate's, and the projection is two fields")
	}
	// The manifest states both halves of the projection.
	if strings.Join(res.Manifest.CandidateFields, "|") != strings.Join(CandidateFields, "|") {
		t.Errorf("the manifest states the projected fields as %v", res.Manifest.CandidateFields)
	}
	if res.Manifest.Exercised == nil || !*res.Manifest.Exercised {
		t.Errorf("the manifest does not state the position as exercised: %v", res.Manifest.Exercised)
	}
}

// ac-2. TestComparativeBundleCarriesTheCriteriaDiscipline: the declared criteria
// travel beside the candidates, read off the record and never supplied at
// invocation (itd-191's gate).
func TestComparativeBundleCarriesTheCriteriaDiscipline(t *testing.T) {
	root := fixtureRepo(t)
	res, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	found := false
	for _, m := range res.Manifest.Items {
		if m.Kind == KindDiscipline && strings.Contains(m.Path, CriteriaDiscipline) {
			found = true
		}
	}
	if !found {
		t.Error("the comparative bundle carries no criteria discipline; a reading with no criteria " +
			"characterises against nothing")
	}
	if strings.Join(res.Manifest.Criteria, "|") != strings.Join(fixtureCriteria, "|") {
		t.Errorf("the manifest states the criteria %v, want %v — parsed off the record's own "+
			"`## The rule` bullets", res.Manifest.Criteria, fixtureCriteria)
	}
}

// ac-2. TestComparativeAdmitsNoOtherRow: at this position the include table is
// the whole account, and no source but the candidates and the criteria is
// admitted (companion 7.2, R3; divergence register 22).
func TestComparativeAdmitsNoOtherRow(t *testing.T) {
	root := fixtureRepo(t)
	res, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	for _, m := range res.Manifest.Items {
		switch m.Kind {
		case KindCandidate:
		case KindDiscipline:
			if !strings.Contains(m.Path, CriteriaDiscipline) {
				t.Errorf("the comparative bundle carries the discipline %s; the row is narrowed to "+
					"%s and no entry can widen it", m.Path, CriteriaDiscipline)
			}
		default:
			t.Errorf("the comparative bundle carries %s of kind %s; every other row withdraws from "+
				"this position", m.Path, m.Kind)
		}
	}
}

// TestComparativeRefusesWithoutTheCriteriaDiscipline: the criteria are never
// supplied at invocation, so an assembly that selected none refuses rather than
// handing a reading nothing to characterise against.
func TestComparativeRefusesWithoutTheCriteriaDiscipline(t *testing.T) {
	root := fixtureRepo(t)
	if err := os.Remove(filepath.Join(root, ".abcd", "development", "intents", "disciplines",
		fixtureCriteriaFile)); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root)

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("a comparative assembly ran with no criteria discipline; the criteria come from " +
			"the record and never from the invocation")
	}
	for _, want := range []string{CriteriaDiscipline, "criteria"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
}

// ac-3. TestComparativeRefusesADispositionedCandidate: one item of the run
// carries a standing disposition, and the assembly refuses NAMING THAT ITEM.
// The order is fixed — characterise first, admit second (companion 8.3) — and
// this guard stands against a hand-written record, because the writer's own gate
// makes it unreachable through any verb.
func TestComparativeRefusesADispositionedCandidate(t *testing.T) {
	root := fixtureRepo(t)
	plantDisposition(t, root, fixtureCandidateItems[1], "dsp-7")
	gitCommitAll(t, root)

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("a run with a dispositioned candidate was taken as a candidate set")
	}
	for _, want := range []string{fixtureCandidateItems[1], "dsp-7", "standing disposition"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	// And the two untouched items are not what the refusal blames.
	if strings.Contains(err.Error(), "rdi-1 carries") || strings.Contains(err.Error(), "rdi-3 carries") {
		t.Errorf("the refusal blames an item that carries nothing: %v", err)
	}
}

// ac-4. TestComparativeRefusesAnAdmittedCandidate: the same, for an admission,
// keyed on the (run, proposal) pair.
func TestComparativeRefusesAnAdmittedCandidate(t *testing.T) {
	root := fixtureRepo(t)
	plantAdmission(t, root, fixtureCandidateRun, fixtureCandidateItems[2], "adm-4")
	gitCommitAll(t, root)

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("a run with an admitted candidate was taken as a candidate set; the candidate set " +
			"is defined as pre-admission")
	}
	for _, want := range []string{fixtureCandidateItems[2], "adm-4", "admission"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
}

// TestAnAdmissionUnderAnotherRunDoesNotRefuse is the ordering guard's own
// non-vacuity: the fate is keyed on the (run, proposal) PAIR, so an admission
// filed under a different run does not silence — or here, refuse — this one.
func TestAnAdmissionUnderAnotherRunDoesNotRefuse(t *testing.T) {
	root := fixtureRepo(t)
	plantAdmission(t, root, "rdg-2608301200000019", fixtureCandidateItems[0], "adm-9")
	gitCommitAll(t, root)

	if _, err := assembleComparative(t, root); err != nil {
		t.Fatalf("an admission filed under another run refused this one: %v", err)
	}
}

// TestComparativeRefusesANonWideningRun: every record the row enumerates is
// validated, and one returned at another position refuses the assembly by name.
// The row's path cannot establish it — every position's records live in one
// family.
func TestComparativeRefusesANonWideningRun(t *testing.T) {
	root := fixtureRepo(t)
	// The run record still says widening; the RECORD says otherwise. That is the
	// only way the two can disagree, and it is what the per-record check exists
	// for.
	item := fixtureCandidateItems[1]
	writeFile(t, root, ".abcd/work/issues/readings/"+fixtureCandidateRun+"/"+item+".md",
		"---\nschema_version: 1\nid: \""+item+"\"\nrun: \""+fixtureCandidateRun+"\"\n"+
			"manifest: \""+strings.Repeat("a", 64)+"\"\nposition: \"detection\"\n"+
			"regime: \"registrative\"\npattern: \"a stated constraint\"\n"+
			"tension: \"t\"\nconstraint_in_play: \"c\"\nwhy_a_tension: \"w\"\n---\n")
	gitCommitAll(t, root)

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("a record returned at the detection position was admitted as a candidate")
	}
	for _, want := range []string{fixtureCandidateItems[1], "detection", "candidate set"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
}

// ac-5. TestOneCandidateStagesAnEmptyComparativeRun is the interpretation fixed
// in advance, before any reading ran: fewer than two configurations and the
// comparative reading has nothing to compare, so the position is not exercised
// (the 2026-09-02 interpretations entry; companion 7.6; divergence register 3).
//
// The assembly REFUSES and still stages, because the outcome of a widening run
// is one shape either way: a committed comparative run naming it.
func TestOneCandidateStagesAnEmptyComparativeRun(t *testing.T) {
	root := fixtureRepo(t)
	for _, id := range fixtureCandidateItems[1:] {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(
			".abcd/work/issues/readings/"+fixtureCandidateRun+"/"+id+".md"))); err != nil {
			t.Fatal(err)
		}
	}
	gitCommitAll(t, root)

	out := filepath.Join(t.TempDir(), "staged")
	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionComparative, Target: "HEAD", OutDir: out,
	})
	var notExercised *PositionNotExercised
	if !errors.As(err, &notExercised) {
		t.Fatalf("a one-candidate run did not report the fixed interpretation: %v", err)
	}
	if notExercised.CandidateRun != fixtureCandidateRun || notExercised.Candidates != 1 {
		t.Errorf("the refusal reports run %q with %d candidate(s)",
			notExercised.CandidateRun, notExercised.Candidates)
	}
	if !strings.Contains(err.Error(), "NOT EXERCISED") {
		t.Errorf("the refusal does not name the interpretation: %v", err)
	}
	if !res.NotExercised {
		t.Error("the result does not report the position as not exercised")
	}
	if !res.Written {
		t.Fatal("the run was not staged; the outcome of a widening run is a committed comparative " +
			"run naming it, and the ingest needs a staged run to commit")
	}
	// The staged bundle carries no candidate item, and the manifest states the
	// run, its own item count, and that the position was not exercised.
	for _, it := range res.Bundle.Items {
		if it.Kind == KindCandidate {
			t.Errorf("the staged bundle carries the candidate item %s; a not-exercised run stages "+
				"an EMPTY candidate set", it.Candidate)
		}
	}
	if res.Manifest.CandidateRun != fixtureCandidateRun {
		t.Errorf("the staged manifest names %q, want %q", res.Manifest.CandidateRun, fixtureCandidateRun)
	}
	if res.Manifest.Candidates != 1 {
		t.Errorf("the staged manifest records %d candidates, want 1: the count is the derived run's "+
			"item count and is never written as zero for a run that holds an item",
			res.Manifest.Candidates)
	}
	if res.Manifest.Exercised == nil || *res.Manifest.Exercised {
		t.Errorf("the staged manifest does not state exercised:false — a false value is a "+
			"statement and not an absence: %v", res.Manifest.Exercised)
	}
	for _, name := range []string{BundleFileName, ManifestFileName} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("the staged run is missing %s: %v", name, err)
		}
	}
}

// ac-5. TestAnEmptyComparativeRunIsNotStagedOnADryRun: staging follows the same
// rules as any other assembly, so a dry run with no output directory writes
// nothing.
func TestAnEmptyComparativeRunIsNotStagedOnADryRun(t *testing.T) {
	root := fixtureRepo(t)
	for _, id := range fixtureCandidateItems[1:] {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(
			".abcd/work/issues/readings/"+fixtureCandidateRun+"/"+id+".md"))); err != nil {
			t.Fatal(err)
		}
	}
	gitCommitAll(t, root)

	res, err := assembleComparative(t, root)
	var notExercised *PositionNotExercised
	if !errors.As(err, &notExercised) {
		t.Fatalf("a one-candidate dry run did not report the fixed interpretation: %v", err)
	}
	if res.Written {
		t.Error("a dry run staged a comparative run; the not-exercised branch stages on the same " +
			"rules as any other assembly, and a dry run with no --out writes nothing")
	}
}

// ac-6. TestComparativeManifestAssertsTheDerivedExclusions: with a second,
// dispositioned run, dispositions and admissions all present, none of it appears
// in the bundle and the manifest asserts their exclusion by name.
func TestComparativeManifestAssertsTheDerivedExclusions(t *testing.T) {
	root := fixtureRepo(t)
	const other = "rdg-2608301200000021"
	items := plantWideningItems(t, root, other, 2)
	for i, item := range items {
		plantDisposition(t, root, item, "dsp-"+string(rune('1'+i)))
		plantAdmission(t, root, other, item, "adm-"+string(rune('1'+i)))
	}
	writeFile(t, root, ".abcd/work/issues/surprises/srp-1.md",
		"---\nschema_version: 1\nid: \"srp-1\"\noccasioned_by: \""+items[0]+"\"\n---\n\n"+sentinelFate+"\n")
	gitCommitAll(t, root)
	commitRunRecord(t, root, other, string(PositionWidening), headOf(t, root))

	res, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if text := bundleText(res.Bundle); strings.Contains(text, sentinelFate) {
		t.Error("a disposition, admission or surprise reached the comparative bundle; the reading " +
			"receives candidates and never their fate")
	}
	asserted := map[string]bool{}
	for _, e := range res.Manifest.Exclusions {
		asserted[e.Detail] = true
	}
	for _, want := range []string{
		".abcd/work/issues/dispositions",
		".abcd/work/issues/admissions",
		".abcd/work/issues/surprises",
		".abcd/work/issues/open",
		".abcd/work/issues/resolved",
		".abcd/work/issues/wontfix",
	} {
		if !asserted[want] {
			t.Errorf("the manifest does not assert the exclusion of %s", want)
		}
	}
	signal := false
	for _, e := range res.Manifest.Exclusions {
		if e.Signal == "readings store" {
			signal = true
		}
	}
	if !signal {
		t.Error("the manifest does not assert what the readings store did NOT supply")
	}
}

// ac-6. TestCandidateProjectionRefusesAForeignRun is the fail-closed half of the
// readings-store signal row: an assembler that emitted a candidate from outside
// the derived run, or under a third field, refuses rather than disclosing
// (adr-56; brief invariant 16).
func TestCandidateProjectionRefusesAForeignRun(t *testing.T) {
	base := CandidateSource + "/" + fixtureCandidateRun + "/"
	cands := []candidate{
		{path: base + "rdi-1.md", field: CandidateFields[0], kind: KindCandidate},
		{path: base + "rdi-1.md", field: CandidateFields[1], kind: KindCandidate},
	}
	if err := assertCandidateProjection(cands, fixtureCandidateRun); err != nil {
		t.Fatalf("a legal projection was refused: %v", err)
	}

	foreign := append(append([]candidate{}, cands...), candidate{
		path:  CandidateSource + "/rdg-2608301200000099/rdi-9.md",
		field: CandidateFields[0], kind: KindCandidate,
	})
	err := assertCandidateProjection(foreign, fixtureCandidateRun)
	if err == nil {
		t.Fatal("a candidate from another run was emitted; every run other than the candidate run " +
			"is what the manifest asserts stays behind")
	}
	if !strings.Contains(err.Error(), "rdg-2608301200000099") {
		t.Errorf("the refusal does not name the foreign run: %v", err)
	}

	thirdField := append(append([]candidate{}, cands...), candidate{
		path: base + "rdi-1.md", field: "pattern", kind: KindCandidate,
	})
	err = assertCandidateProjection(thirdField, fixtureCandidateRun)
	if err == nil {
		t.Fatal("a third field was emitted; the projection is two fields and the manifest says so")
	}
	if !strings.Contains(err.Error(), "pattern") {
		t.Errorf("the refusal does not name the field: %v", err)
	}

	// And nothing else may come out of the readings store at all.
	wrongKind := append(append([]candidate{}, cands...), candidate{
		path: base + "rdi-1.md", field: "", kind: KindDoc,
	})
	if err := assertCandidateProjection(wrongKind, fixtureCandidateRun); err == nil {
		t.Fatal("a non-candidate item was emitted from the readings store")
	}
}

// TestCandidateFieldsAreEmptyAtEveryOtherPosition: `candidate` and `field` are a
// candidate item's alone, so no other position's bundle carries either.
func TestCandidateFieldsAreEmptyAtEveryOtherPosition(t *testing.T) {
	root := fixtureRepo(t)
	for _, p := range AssemblingPositions() {
		if p == PositionComparative {
			continue
		}
		res := assembleFixture(t, root, p)
		for _, it := range res.Bundle.Items {
			if it.Candidate != "" || it.Field != "" {
				t.Errorf("the %s bundle carries candidate=%q field=%q on item %s; both are a "+
					"candidate item's alone", p, it.Candidate, it.Field, it.ItemKey)
			}
		}
		if res.Manifest.CandidateRun != "" || res.Manifest.Candidates != 0 ||
			res.Manifest.Exercised != nil || len(res.Manifest.Criteria) != 0 {
			t.Errorf("the %s manifest carries comparative fields: run=%q candidates=%d "+
				"exercised=%v criteria=%v", p, res.Manifest.CandidateRun, res.Manifest.Candidates,
				res.Manifest.Exercised, res.Manifest.Criteria)
		}
	}
}

// TestTwoComparativeAssembliesAreByteIdentical: candidate items are ordered by
// id then by field, so two assemblies of one run over one repository state
// produce the same bundle — the property the amnesia eval falsifies for the
// other positions, held here for the one whose items come from a directory
// listing.
func TestTwoComparativeAssembliesAreByteIdentical(t *testing.T) {
	root := fixtureRepo(t)
	first, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("first assembly: %v", err)
	}
	second, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("second assembly: %v", err)
	}
	a, err := EncodeBundle(first.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncodeBundle(second.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Error("two comparative assemblies of one repository state produced different bundles")
	}
}

// TestDeclaredCriteriaReadsTheCommittedDiscipline pins the COMMITTED itd-191 to
// its six names, so an amendment to the slate is a recorded change here too
// (itd-191, "amendable only by ordinary discipline amendment").
func TestDeclaredCriteriaReadsTheCommittedDiscipline(t *testing.T) {
	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, ".abcd", "development", "intents",
		"disciplines", CriteriaDiscipline+"-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("the committed %s is not a single file: %v (%v)", CriteriaDiscipline, matches, err)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	got, err := declaredCriteria(string(raw))
	if err != nil {
		t.Fatalf("declaredCriteria over the committed discipline: %v", err)
	}
	want := []string{"Plausibility", "Generativity", "Cost", "Risk", "Learning value",
		"Practical importance"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("the committed discipline declares %v, want %v", got, want)
	}
}

// TestDeclaredCriteriaRefusesAnEmptySlate: a discipline with no section, or with
// a section declaring nothing, is refused rather than read as declaring an empty
// slate — against which every criterion would be undeclared and nothing could be
// characterised.
func TestDeclaredCriteriaRefusesAnEmptySlate(t *testing.T) {
	for name, doc := range map[string]string{
		"no section":    "---\nid: itd-191\n---\n\n# A discipline\n\n## The gate\n\nNothing here.\n",
		"no bullets":    "---\nid: itd-191\n---\n\n## The rule\n\nProse and no bullets.\n\n## The gate\n\n",
		"empty bullets": "---\nid: itd-191\n---\n\n## The rule\n\n- — a gloss with no name.\n",
	} {
		if _, err := declaredCriteria(doc); err == nil {
			t.Errorf("%s: an empty slate was accepted", name)
		}
	}
}

// ---------------------------------------------------------------------------
// "At the target": a run whose own records were committed since it read
// ---------------------------------------------------------------------------
//
// The loop the design sequences is: assemble at widening, dispatch, ingest,
// commit what the ingest wrote, assemble at comparative over the run just
// ingested. The middle step is what these cases are about. `reading ingest`
// leaves a run's reading records in the committed ledger UNCOMMITTED, and the
// comparative position's candidate row reaches that store — so the assembly that
// follows refuses on the dirty gate until they are committed, and committing
// them moves HEAD off the commit the run's own record names.
//
// So "the one committed widening run AT THE TARGET" (adr-2609021016272867) is
// read as reaching an ancestor whose object set has not moved: the run's target
// equals the target, or is an ancestor of it with every path changed between the
// two inside the readings store and the issue ledger's own families. Divergence
// register 27 is the ground — "the object set, not the commit, names what the
// readings are about" — and companion 7.2 fixes the comparative object as the
// widening run's returned items and the criteria discipline, neither of which
// the tree supplies. The ruling is owed (iss-2609021857343626).

// ingestedWideningRun is the run these cases plant: one widening run whose
// records are written as an ingest leaves them and committed afterwards.
const ingestedWideningRun = "rdg-2608301200000021"

// plantIngestedRunAtHead rehearses the ingest inside the unit fixture. The
// fixture's own planted run is removed first, so the run planted here is the
// only widening run and the derivation is unambiguous; its records are left
// UNCOMMITTED, exactly as an ingest leaves them, and its commit marker names the
// commit its reading actually read. It returns that commit.
func plantIngestedRunAtHead(t *testing.T, root string) string {
	t.Helper()
	removeFixtureRun(t, root)
	plantWideningItems(t, root, ingestedWideningRun, 2)
	runTarget := headOf(t, root)
	commitRunRecord(t, root, ingestedWideningRun, string(PositionWidening), runTarget)
	return runTarget
}

// commitWithoutRestamping commits the whole tree and leaves every run record
// naming the commit it already named. gitCommitAll re-points the fixture's run
// at the new HEAD, which is what makes most cases readable; these cases are
// about the run that does NOT move, so they commit for themselves.
func commitWithoutRestamping(t *testing.T, root string) {
	t.Helper()
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-q", "-m", "the run's own records, committed")
}

// TestAWideningRunAtTheTargetItselfStillQualifies: the unmoved case, unchanged.
// The fixture's run names HEAD, and the manifest records that commit as both the
// candidate run's target and the assembly's own (adr-2609021016272867).
func TestAWideningRunAtTheTargetItselfStillQualifies(t *testing.T) {
	root := fixtureRepo(t)
	head := headOf(t, root)

	res, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("the comparative assembly refused over a run at the target itself: %v", err)
	}
	if res.CandidateRun != fixtureCandidateRun {
		t.Fatalf("the assembly derived %q, want %q", res.CandidateRun, fixtureCandidateRun)
	}
	if res.Manifest.CandidateRunTarget != head || res.Manifest.TargetCommit != head {
		t.Errorf("the manifest records candidate_run_target %q and target_commit %q; at the "+
			"unmoved case both are HEAD (%s)",
			res.Manifest.CandidateRunTarget, res.Manifest.TargetCommit, head)
	}
}

// TestAWideningRunWhoseRecordsWereCommittedSinceItsTargetQualifies: the
// deadlock's release. The run read at C, its records were committed at C+1, and
// nothing else moved — so the object set the run is about is the object set at
// the assembly's target, and the run is still the one this target's comparative
// reading characterises (adr-2609021016272867 "at the target", as read here;
// divergence register 27; companion 7.2).
func TestAWideningRunWhoseRecordsWereCommittedSinceItsTargetQualifies(t *testing.T) {
	root := fixtureRepo(t)
	runTarget := plantIngestedRunAtHead(t, root)
	commitWithoutRestamping(t, root)
	head := headOf(t, root)
	if head == runTarget {
		t.Fatal("committing the run's records did not move HEAD, so this case rehearses nothing")
	}

	res, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("the comparative assembly refused after the widening run's own records were "+
			"committed, which is the act the design sequences between the ingest and this "+
			"assembly: %v", err)
	}
	if res.CandidateRun != ingestedWideningRun {
		t.Fatalf("the assembly derived %q, want the run whose records were just committed (%q)",
			res.CandidateRun, ingestedWideningRun)
	}
	if res.Manifest.CandidateRun != ingestedWideningRun {
		t.Errorf("the manifest names the candidate run %q, want %q",
			res.Manifest.CandidateRun, ingestedWideningRun)
	}
	if res.Manifest.CandidateRunTarget != runTarget {
		t.Errorf("the manifest records candidate_run_target %q, want the commit the widening run "+
			"actually read (%s); a reader has to be able to diff the two",
			res.Manifest.CandidateRunTarget, runTarget)
	}
	if res.Manifest.TargetCommit != head {
		t.Errorf("the manifest records target_commit %q, want this assembly's own target (%s)",
			res.Manifest.TargetCommit, head)
	}
}

// TestAWideningRunWhoseObjectSetMovedSinceItsTargetDoesNotQualify: a source file
// changed between the run's target and this one, so the object set moved and the
// run is not about this target's state. It is LISTED rather than hidden, and the
// refusal names the first path that moved (divergence register 27).
func TestAWideningRunWhoseObjectSetMovedSinceItsTargetDoesNotQualify(t *testing.T) {
	root := fixtureRepo(t)
	plantIngestedRunAtHead(t, root)
	writeFile(t, root, "main.go", "package main\n\nfunc main() { _ = 1 }\n")
	commitWithoutRestamping(t, root)

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("the comparative assembly selected a widening run read over a different object " +
			"set; the run's items characterise a state this target no longer holds")
	}
	var none *NoCandidateRun
	if !errors.As(err, &none) {
		t.Fatalf("the refusal is not NoCandidateRun: %v", err)
	}
	if len(none.Runs) != 1 {
		t.Fatalf("the refusal lists %d run(s), want the one that did not qualify: a run hidden "+
			"from the listing leaves the operator with no way to see why", len(none.Runs))
	}
	msg := err.Error()
	for _, want := range []string{
		ingestedWideningRun, "main.go", "the object set changed since the run",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not carry %q: %s", want, msg)
		}
	}
}

// TestAWideningRunOnADifferentHistoryDoesNotQualify: the run's target is not an
// ancestor of this target at all, so no diff between the two says anything about
// what moved. It is not a run at this target and is not selected.
func TestAWideningRunOnADifferentHistoryDoesNotQualify(t *testing.T) {
	root := fixtureRepo(t)
	removeFixtureRun(t, root)
	plantWideningItems(t, root, ingestedWideningRun, 2)
	commitWithoutRestamping(t, root)

	// A commit on a branch this target never saw.
	gitRun(t, root, "checkout", "-q", "-b", "aside")
	writeFile(t, root, "docs/reference/aside.md", "# Aside\n\nProse on another history.\n")
	commitWithoutRestamping(t, root)
	aside := headOf(t, root)
	gitRun(t, root, "checkout", "-q", "main")
	commitRunRecord(t, root, ingestedWideningRun, string(PositionWidening), aside)

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("the comparative assembly selected a widening run read on a history this target " +
			"is not descended from")
	}
	var none *NoCandidateRun
	if !errors.As(err, &none) {
		t.Fatalf("the refusal is not NoCandidateRun: %v", err)
	}
	if len(none.Runs) != 0 {
		t.Errorf("the refusal lists %d run(s); a run on another history is not a run at this "+
			"target and the listing is of the runs at this target", len(none.Runs))
	}
}

// ---------------------------------------------------------------------------
// The dirty gate over the fate families
// ---------------------------------------------------------------------------
//
// The comparative derivation reads the fate families from the FILESYSTEM:
// capture.ItemFate walks the ledger's dispositions and admissions directories to
// decide whether a widening run is still pre-admission (adr-2609021016272867;
// companion 8.3, the sequence that places dispositioning after the comparative
// reading). Neither family is admitted by an include row at any position — a
// fate is the researcher's judgement and never a reading's input — so the
// include table cannot put them in the dirty gate's set, and the gate has to
// name them the way it names the two configuration files.

// TestComparativeRefusesAnUncommittedDisposition: the disposition is committed
// and then deleted in the working tree alone. The derivation reads the working
// tree, so it would take the run as pre-admission — while the commit the
// manifest names still carries the fate, and a re-run at that commit would
// derive a different candidate set. The gate refuses and names the path.
func TestComparativeRefusesAnUncommittedDisposition(t *testing.T) {
	root := fixtureRepo(t)
	item := fixtureCandidateItems[1]
	plantDisposition(t, root, item, "dsp-7")
	gitCommitAll(t, root)

	rel := ".abcd/work/issues/dispositions/" + item + "/dsp-7.md"
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		t.Fatal(err)
	}

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("the comparative assembly ran over a working tree whose fate records differ " +
			"from the commit it names; the manifest would promise a re-run that derived a " +
			"different candidate set")
	}
	if !strings.Contains(err.Error(), rel) {
		t.Errorf("the refusal does not name the uncommitted fate record %s: %v", rel, err)
	}
}

// TestComparativeRefusesAnUncommittedAdmission: the same gate from the other
// side — an admission ADDED and not committed, over the SECOND of two runs the
// commit leaves ambiguous. The working tree makes the selection unambiguous and
// the commit the manifest names does not, so a re-run at that commit would
// refuse where this one selected.
func TestComparativeRefusesAnUncommittedAdmission(t *testing.T) {
	root := fixtureRepo(t)
	const other = "rdg-2608301200000014"
	items := plantWideningItems(t, root, other, 2)
	gitCommitAll(t, root)
	commitRunRecord(t, root, other, string(PositionWidening), headOf(t, root))

	rel := ".abcd/work/issues/admissions/" + other + "/adm-3.md"
	plantAdmission(t, root, other, items[0], "adm-3")

	_, err := assembleComparative(t, root)
	if err == nil {
		t.Fatal("the comparative assembly selected a run because an UNCOMMITTED admission " +
			"disqualified the other; the commit the manifest names holds two qualifying runs " +
			"and a re-run over it would refuse")
	}
	if !strings.Contains(err.Error(), rel) {
		t.Errorf("the refusal does not name the uncommitted fate record %s: %v", rel, err)
	}
}

// TestComparativeStillDerivesOverACleanFateStore: the gate is about the DIFF
// between the tree and the commit, not about the families being present. A
// committed disposition over a second run still leaves the first run derivable.
func TestComparativeStillDerivesOverACleanFateStore(t *testing.T) {
	root := fixtureRepo(t)
	const other = "rdg-2608301200000013"
	items := plantWideningItems(t, root, other, 2)
	for i, item := range items {
		plantDisposition(t, root, item, "dsp-"+string(rune('1'+i)))
	}
	gitCommitAll(t, root)
	commitRunRecord(t, root, other, string(PositionWidening), headOf(t, root))

	res, err := assembleComparative(t, root)
	if err != nil {
		t.Fatalf("the comparative assembly refused over a clean tree carrying committed fate "+
			"records: %v", err)
	}
	if res.CandidateRun != fixtureCandidateRun {
		t.Errorf("the assembly derived %q, want %q", res.CandidateRun, fixtureCandidateRun)
	}
}
