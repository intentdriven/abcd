package reading

// ingest_tables_test.go pins the supply-regime gate's data tables by CONTENT
// (iss-2608311518109233).
//
// A guard whose behaviour is driven by a table is only as bound as the table is
// pinned. Every other test over these tables iterates the table under test —
// "for each reserved name, it refuses" — so an EMPTY table satisfies all of them
// at once, and the package stayed green with any of the three reserved-name
// rows deleted. The criteria those tests cite (itd-185 ac-4, ac-6 and ac-8)
// enumerate the names as literals, and nothing repeated the literals.
//
// So each test here names what the table must hold, in the criterion's own
// words, and holds the table to it in both directions: every enumerated name is
// present, no name outside the enumeration is present, and the count is asserted
// first so an emptied table fails on a message that says so. The names are then
// pushed through the gate one by one, so the pin cannot be satisfied by a table
// the gate has stopped consulting.
//
// The same shape is owed to the other tables the gate reads from, which the
// same audit found unpinned by the same reasoning: the closed body vocabulary,
// the per-position body field sets, and the licence sentences.

import (
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// reservedByCriterion is the acceptance criteria's OWN enumeration of the
// reserved names, transcribed rather than read from the table. If the table and
// a criterion ever disagree, the test that fails is the finding; the table is
// not silently aligned.
var reservedByCriterion = map[string]struct {
	criterion string
	names     []string
}{
	RegimeRegistrative: {"itd-185 ac-4", []string{"resolution", "fix", "remedy"}},
	RegimeEvaluative:   {"itd-185 ac-6", []string{"rank", "score", "order", "recommended"}},
	RegimeExplicative:  {"itd-185 ac-8", []string{"disposition", "status"}},
}

// TestRegistrativeReservedTableIsPinnedToItsCriterion is the content pin for
// ac-4's table: `resolution`, `fix`, `remedy`.
func TestRegistrativeReservedTableIsPinnedToItsCriterion(t *testing.T) {
	assertReservedTablePinned(t, RegimeRegistrative)
}

// TestEvaluativeReservedTableIsPinnedToItsCriterion is the content pin for
// ac-6's table: `rank`, `score`, `order`, `recommended`.
func TestEvaluativeReservedTableIsPinnedToItsCriterion(t *testing.T) {
	assertReservedTablePinned(t, RegimeEvaluative)
}

// TestExplicativeReservedTableIsPinnedToItsCriterion is the content pin for
// ac-8's table: `disposition`, `status`. ac-8 also refuses "any field outside
// the explicative body schema"; that half is the closed key set, held by
// TestWrongPositionBodyIsUndecodable, and is not a reserved name.
func TestExplicativeReservedTableIsPinnedToItsCriterion(t *testing.T) {
	assertReservedTablePinned(t, RegimeExplicative)
}

// TestReservedNamesHasExactlyTheThreeCriteriaRows pins the table's SHAPE: one
// row per criterion that enumerates reserved names, and none for generative,
// whose licence is the widest and whose constraint falls at admission. A fourth
// row would be a reserved-name rule no criterion states.
func TestReservedNamesHasExactlyTheThreeCriteriaRows(t *testing.T) {
	if _, has := ReservedNames[RegimeGenerative]; has {
		t.Errorf("the generative regime carries a reserved-name row; no criterion enumerates one, and " +
			"the generative constraint falls at admission")
	}
	for regime := range ReservedNames {
		if _, ok := reservedByCriterion[regime]; !ok {
			t.Errorf("the table carries a row for regime %q, which no criterion enumerates", regime)
		}
	}
	if got, want := len(ReservedNames), len(reservedByCriterion); got != want {
		t.Errorf("the table carries %d row(s); the criteria enumerate %d", got, want)
	}
}

// assertReservedTablePinned holds one regime's row to its criterion.
//
// The floor comes FIRST and is fatal, so the failure an emptied table produces
// says "0 name(s)" rather than a list of absences. Then presence, then the
// absence of extras, then the exact count, and finally each literal is pushed
// through the gate at a position of that regime and must refuse under the
// reserved-name rule, naming itself.
func assertReservedTablePinned(t *testing.T, regime string) {
	t.Helper()
	spec := reservedByCriterion[regime]
	got := ReservedNames[regime]

	if len(got) < len(spec.names) {
		t.Fatalf("the %s reserved-name table holds %d name(s); %s enumerates %d: %v",
			regime, len(got), spec.criterion, len(spec.names), spec.names)
	}
	have := map[string]bool{}
	for _, n := range got {
		if have[n] {
			t.Errorf("the %s reserved-name table carries %q twice", regime, n)
		}
		have[n] = true
	}
	for _, want := range spec.names {
		if !have[want] {
			t.Errorf("the %s reserved-name table does not carry %q, which %s enumerates",
				regime, want, spec.criterion)
		}
	}
	for _, n := range got {
		if !containsToken(spec.names, n) {
			t.Errorf("the %s reserved-name table carries %q, which %s does not enumerate; a name reserved "+
				"beyond the criterion refuses a field no ruling reserved", regime, n, spec.criterion)
		}
	}
	if len(got) != len(spec.names) {
		t.Errorf("the %s reserved-name table holds %d name(s), want exactly %d", regime, len(got), len(spec.names))
	}

	// The literal, not the table entry, goes through the gate: a table the gate
	// no longer reads would pass every assertion above.
	pos := positionOfRegime(t, regime)
	for _, name := range spec.names {
		name := name
		t.Run(name, func(t *testing.T) {
			f := newIngestFixture(t, pos)
			doc := f.payload(2)
			doc["items"].([]any)[1].(map[string]any)[name] = "a value the licence does not admit"

			r := f.refusedItem(doc, 2, 2)
			if r.Rule != "reserved-name" {
				t.Errorf("%q refused under rule %q, not the reserved-name table", name, r.Rule)
			}
			if !strings.Contains(r.Field, name) {
				t.Errorf("the refusal names field %q, want %q", r.Field, name)
			}
		})
	}
}

// positionOfRegime resolves the one shipped position read under a regime.
func positionOfRegime(t *testing.T, regime string) Position {
	t.Helper()
	for _, p := range Positions() {
		if issueschema.ReadingRegime(string(p)) == regime {
			return p
		}
	}
	t.Fatalf("no shipped position is read under the %s regime", regime)
	return ""
}

// TestClosedVocabularyIsPinnedToSpc63 pins the body vocabulary spc-63 tables:
// `claim_type` is `criterion` / `causal` / `context`, and no other body field
// is closed. The existing enforcement test iterates the set, so a set reduced
// to one token would still satisfy it.
func TestClosedVocabularyIsPinnedToSpc63(t *testing.T) {
	want := map[string][]string{"claim_type": {"criterion", "causal", "context"}}
	if len(ClosedVocabularies) < len(want) {
		t.Fatalf("ClosedVocabularies declares %d field(s); spc-63 tables %d", len(ClosedVocabularies), len(want))
	}
	for field := range ClosedVocabularies {
		if _, ok := want[field]; !ok {
			t.Errorf("ClosedVocabularies closes %q, which spc-63 does not table as closed", field)
		}
	}
	for field, tokens := range want {
		got := ClosedVocabularies[field]
		if len(got) < len(tokens) {
			t.Fatalf("the %s vocabulary holds %d token(s); spc-63 tables %d: %v", field, len(got), len(tokens), tokens)
		}
		if !sameTokens(got, tokens) {
			t.Errorf("the %s vocabulary is %v; spc-63 tables %v", field, got, tokens)
		}
	}
}

// TestBodyFieldSetsArePinnedToSpc63 pins the per-position body field sets the
// gate's closed key set derives from, in spc-63's own table. Order is pinned
// too: the table's order is the order a refusal quotes and a record writes.
func TestBodyFieldSetsArePinnedToSpc63(t *testing.T) {
	want := map[string][]string{
		"widening":    {"configuration", "what_admits_it"},
		"entailment":  {"claim_surfaced", "claim_type", "what_implies_it"},
		"comparative": {"candidate_id", "criterion", "characterisation"},
		"detection":   {"tension", "constraint_in_play", "why_a_tension"},
	}
	if len(issueschema.ReadingBodyFields) < len(want) {
		t.Fatalf("ReadingBodyFields declares %d position(s); spc-63 tables %d", len(issueschema.ReadingBodyFields), len(want))
	}
	for pos := range issueschema.ReadingBodyFields {
		if _, ok := want[pos]; !ok {
			t.Errorf("ReadingBodyFields declares a body for %q, which spc-63 does not table", pos)
		}
	}
	for pos, fields := range want {
		got := issueschema.ReadingBodyFields[pos]
		if len(got) < len(fields) {
			t.Fatalf("the %s body holds %d field(s); spc-63 tables %d: %v", pos, len(got), len(fields), fields)
		}
		if strings.Join(got, "|") != strings.Join(fields, "|") {
			t.Errorf("the %s body is %v; spc-63 tables %v", pos, got, fields)
		}
	}
}

// TestEveryRegimeStatesALicence pins the licence table: each of the four regimes
// states a sentence, and no fifth regime does. A refusal quotes the sentence, so
// a regime with none would refuse without saying what was breached.
func TestEveryRegimeStatesALicence(t *testing.T) {
	regimes := []string{RegimeGenerative, RegimeExplicative, RegimeEvaluative, RegimeRegistrative}
	if len(regimeLicence) < len(regimes) {
		t.Fatalf("regimeLicence states %d licence(s); there are %d regimes", len(regimeLicence), len(regimes))
	}
	for _, r := range regimes {
		if strings.TrimSpace(regimeLicence[r]) == "" {
			t.Errorf("the %s regime states no licence", r)
		}
	}
	for r := range regimeLicence {
		if !containsToken(regimes, r) {
			t.Errorf("regimeLicence states a licence for %q, which is not one of the four regimes", r)
		}
	}
}

// sameTokens is set equality over two token lists.
func sameTokens(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := append([]string{}, a...), append([]string{}, b...)
	sort.Strings(x)
	sort.Strings(y)
	return strings.Join(x, "|") == strings.Join(y, "|")
}
