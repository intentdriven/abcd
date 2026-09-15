package reading

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// TestCandidateRowIsComparativeOnly is the positional half of
// adr-2609021016272867: the exception is one row, at one position, projecting
// two fields.
func TestCandidateRowIsComparativeOnly(t *testing.T) {
	var rows []Row
	for _, row := range Table {
		if row.Kind == KindCandidate {
			rows = append(rows, row)
		}
	}
	if len(rows) != 1 {
		t.Fatalf("the table carries %d candidate row(s), want exactly 1: the exception to the "+
			"prior-run exhaust is ONE row", len(rows))
	}
	row := rows[0]
	if len(row.Positions) != 1 || row.Positions[0] != PositionComparative {
		t.Errorf("the candidate row is admitted at %v; the exception is the comparative "+
			"position's alone", row.Positions)
	}
	if row.Source != CandidateSource {
		t.Errorf("the candidate row's source is %q, want %q — the leaf bucket the readings "+
			"family keys on, which is what assembler rule 1 permits a row to name",
			row.Source, CandidateSource)
	}
	if strings.Join(row.Fields, "|") != strings.Join(CandidateFields, "|") {
		t.Errorf("the candidate row projects %v; the widening body's two fields are %v, and the "+
			"item's pattern and envelope stay behind", row.Fields, CandidateFields)
	}
	if row.Store != issueschema.ReadingItemFamily {
		t.Errorf("the candidate row's store is %q, want %q: the narrowing to one run is the "+
			"row's own bucket selector", row.Store, issueschema.ReadingItemFamily)
	}
	if !strings.Contains(row.Rule, "adr-2609021016272867") {
		t.Errorf("the candidate row's rule does not cite the decision that admits it: %q", row.Rule)
	}
	// The two projected fields ARE the widening body, read off the schema rather
	// than restated: a body that gained a field and a projection that did not
	// would leave the candidate set describing something narrower than the run.
	want := issueschema.ReadingBodyFields[string(PositionWidening)]
	if strings.Join(CandidateFields, "|") != strings.Join(want, "|") {
		t.Errorf("CandidateFields = %v and the widening body is %v", CandidateFields, want)
	}
}

// TestEveryOtherRowWithdrawsFromComparative holds the readings companion's
// section 7.2 and its ratified position R3: at the comparative position the
// include table is the whole account, and no source is admitted but the
// candidates and the declared criteria (divergence register 22).
func TestEveryOtherRowWithdrawsFromComparative(t *testing.T) {
	for _, row := range Table {
		if !row.AdmittedAt(PositionComparative) {
			continue
		}
		switch row.Kind {
		case KindCandidate, KindDiscipline:
			continue
		default:
			t.Errorf("the row %q (%s) is still admitted at the comparative position; every row "+
				"but the candidates and the criteria discipline withdraws", row.Source, row.Kind)
		}
	}
	// And the narrowing that makes the disciplines row mean the criteria alone.
	// It is applied BEFORE the committed entry, so no entry can widen it.
	var disciplines Row
	for _, row := range Table {
		if row.Kind == KindDiscipline {
			disciplines = row
		}
	}
	narrowed := narrowPaths([]string{
		".abcd/development/intents/disciplines/" + CriteriaDiscipline + "-the-criteria.md",
		".abcd/development/intents/disciplines/itd-4-something-else.md",
	}, disciplines, PositionComparative)
	if len(narrowed) != 1 || !strings.Contains(narrowed[0], CriteriaDiscipline) {
		t.Errorf("the comparative narrowing of the disciplines row selects %v; it selects %s and "+
			"nothing else", narrowed, CriteriaDiscipline)
	}
	// At every other position the same row is untouched.
	if wide := narrowPaths([]string{"a.md", "b.md"}, disciplines, PositionDetection); len(wide) != 2 {
		t.Errorf("the disciplines row is narrowed at the detection position too (%v); the "+
			"narrowing is the comparative position's alone", wide)
	}
}

// TestComparativeExclusionRowsAreDerivedFromLedgerDirs holds the derivation the
// manifest's assertion rests on (spc-2609020626039834, "The manifest and the
// exclusions it asserts"; brief invariant 16): a ledger family added later is
// excluded at this position the day its constant is declared, because the rows
// are derived from issueschema.LedgerDirs rather than listed.
func TestComparativeExclusionRowsAreDerivedFromLedgerDirs(t *testing.T) {
	got := map[string]Exclusion{}
	for _, e := range ExclusionsFor(PositionComparative) {
		got[e.Detail] = e
	}
	for _, dir := range issueschema.LedgerDirs() {
		if dir == issueschema.ReadingsDir {
			continue // a SIGNAL row, checked below: one run's items do travel
		}
		detail := capture.LedgerRelPath + "/" + dir
		e, ok := got[detail]
		if !ok {
			t.Errorf("the comparative exclusion floor names no entry for %s; the rows are derived "+
				"from the ledger's own directory list, so a family it declares is covered", detail)
			continue
		}
		if e.Signal != "directory" {
			t.Errorf("the exclusion for %s carries the signal %q, want \"directory\": a directory "+
				"row is what assertExclusions enforces by path", detail, e.Signal)
		}
	}

	// The readings row is present, is a signal row, and states the limit.
	var signal *Exclusion
	for _, e := range ExclusionsFor(PositionComparative) {
		if e.Signal == "readings store" {
			ex := e
			signal = &ex
		}
	}
	if signal == nil {
		t.Fatal("the comparative exclusion floor carries no readings-store row; the manifest has " +
			"to say what the candidate channel does NOT carry")
	}
	for _, want := range []string{
		"every run other than the candidate run", CandidateFields[0], CandidateFields[1],
	} {
		if !strings.Contains(signal.Detail, want) {
			t.Errorf("the readings-store exclusion does not state %q: %q", want, signal.Detail)
		}
	}

	// The container row withdraws, and only from this position: a row binding at
	// `.abcd/work/issues` here would contradict the row that admits the
	// candidates.
	if _, present := got[capture.LedgerRelPath]; present {
		t.Errorf("the comparative floor still excludes the container %s, which the candidate "+
			"row reaches into", capture.LedgerRelPath)
	}
	for _, p := range []Position{PositionWidening, PositionEntailment, PositionDetection} {
		found := false
		for _, e := range ExclusionsFor(p) {
			if e.Detail == capture.LedgerRelPath {
				found = true
			}
		}
		if !found {
			t.Errorf("the %s floor no longer excludes %s; the withdrawal is the comparative "+
				"position's alone", p, capture.LedgerRelPath)
		}
	}
}

// TestRenderCarriesTheCandidateRow: the charter names the channel, because the
// rendering IS the contract the assembler version digests. A channel the
// rendered table did not carry would be one no reader of the charter could check
// and no version could move for.
func TestRenderCarriesTheCandidateRow(t *testing.T) {
	rendered := Render()
	for _, want := range []string{
		"`" + string(KindCandidate) + "`",
		"`" + CandidateSource + "`",
		"`" + CandidateFields[0] + "`",
		"`" + CandidateFields[1] + "`",
		"adr-2609021016272867",
		"readings store",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("the rendered include table does not carry %q", want)
		}
	}
}
