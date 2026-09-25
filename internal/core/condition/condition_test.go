package condition

import (
	"strings"
	"testing"
)

const (
	condA = "cond-2608311949582375"
	condB = "cond-2608311949582376"
)

// verdictBlock is the shape the verdict ingest renders: per-criterion bullets
// above the label, disposition bullets below it.
const verdictBlock = "<!-- abcd-review: INGESTED receipt=rcp-0123456789ab -->\n" +
	"Fidelity review — receipt rcp-0123456789ab (verifier v 1).\n\n" +
	"Per-criterion verdicts:\n" +
	"- ac-1 — MET: holds\n" +
	"  evidence: a.go:1\n\n" +
	"Scope-condition dispositions:\n" +
	"- " + condA + " — survived: the tree still holds it\n" +
	"  evidence: a.go:2\n" +
	"- " + condB + " — narrowed: only half holds\n" +
	"  narrowing: holds for one repository\n" +
	"  evidence: b.go:3\n"

func conditionBlock(id, value, occasion, rationale, narrowing string) string {
	s := "<!-- abcd-condition: " + id + " occasion=" + occasion + " -->\n" +
		"Condition disposition — 2026-09-02, occasioned by " + occasion + ".\n" +
		"- " + id + " — " + value + ": " + rationale + "\n"
	if narrowing != "" {
		s += "  narrowing: " + narrowing + "\n"
	}
	return s
}

func record(blocks ...string) string {
	return "---\nid: itd-1\n---\n\n# An intent\n\n## Audit Notes\n\n" + strings.Join(blocks, "\n") + "\n## Later\n\n- " + condA + " — falsified: outside the section\n"
}

func TestEnumIsTheFourValues(t *testing.T) {
	if got := strings.Join(Enum, ","); got != "survived,narrowed,falsified,untested" {
		t.Fatalf("Enum = %s", got)
	}
	for _, v := range Enum {
		if !Valid(v) {
			t.Errorf("Valid(%q) = false", v)
		}
	}
	for _, v := range []string{"", "MET", "Survived", "narrow"} {
		if Valid(v) {
			t.Errorf("Valid(%q) = true", v)
		}
	}
}

func TestReadDispositionsParsesVerdictBlock(t *testing.T) {
	got := ReadDispositions(record(verdictBlock))
	if len(got) != 2 {
		t.Fatalf("got %d dispositions, want 2 (criteria bullets and out-of-section bullets are not dispositions): %+v", len(got), got)
	}
	if got[0].ConditionID != condA || got[0].Disposition != Survived || got[0].Rationale != "the tree still holds it" ||
		got[0].Source != "verdict rcp-0123456789ab" || got[0].Occasion != "" || got[0].Date != "" {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].Disposition != Narrowed || got[1].Narrowing != "holds for one repository" {
		t.Errorf("second = %+v", got[1])
	}
}

func TestReadDispositionsParsesConditionBlock(t *testing.T) {
	got := ReadDispositions(record(conditionBlock(condA, Narrowed, "rdi-2609011200000001", "the reading named it", "one repository only")))
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
	d := got[0]
	if d.ConditionID != condA || d.Disposition != Narrowed || d.Rationale != "the reading named it" ||
		d.Narrowing != "one repository only" || d.Source != "condition rdi-2609011200000001" ||
		d.Date != "2026-09-02" || d.Occasion != "rdi-2609011200000001" {
		t.Errorf("got %+v", d)
	}
}

func TestReadDispositionsSkipsABulletTheMarkerDoesNotName(t *testing.T) {
	block := "<!-- abcd-condition: " + condA + " occasion=rdi-1 -->\n" +
		"Condition disposition — 2026-09-02, occasioned by rdi-1.\n" +
		"- " + condB + " — falsified: a hand edit keyed to another identity\n"
	if got := ReadDispositions(record(block)); len(got) != 0 {
		t.Errorf("a bullet the marker does not name was read: %+v", got)
	}
}

func TestStandingIsTheLatestBlock(t *testing.T) {
	content := record(verdictBlock, conditionBlock(condA, Falsified, "rdi-7", "the detection named it", ""))
	all := ReadDispositions(content)
	if len(all) != 3 {
		t.Fatalf("both blocks must stand in the history: %+v", all)
	}
	s := Standing(content)
	if s[condA].Disposition != Falsified || s[condA].Source != "condition rdi-7" {
		t.Errorf("standing for %s = %+v, want the later condition block", condA, s[condA])
	}
	if s[condB].Disposition != Narrowed || s[condB].Source != "verdict rcp-0123456789ab" {
		t.Errorf("standing for %s = %+v, want the verdict", condB, s[condB])
	}
	// Two condition blocks: the later one stands.
	content = record(conditionBlock(condA, Falsified, "rdi-7", "first", ""), conditionBlock(condA, Survived, "itd-9", "second", ""))
	if got := Standing(content)[condA]; got.Occasion != "itd-9" {
		t.Errorf("standing = %+v, want the later condition block", got)
	}
}

func TestVerdictDoesNotOverrideAReadingOccasionedBlock(t *testing.T) {
	content := record(conditionBlock(condA, Falsified, "rdi-7", "the detection named it", ""), verdictBlock)
	if got := Standing(content)[condA]; got.Occasion != "rdi-7" || got.Disposition != Falsified {
		t.Errorf("a later verdict naming no occasion overrode the condition block: %+v", got)
	}
	// A rationale citing a longer id is not a citation of this one.
	near := strings.Replace(verdictBlock, "the tree still holds it", "weighed rdi-70 and it holds", 1)
	content = record(conditionBlock(condA, Falsified, "rdi-7", "the detection named it", ""), near)
	if got := Standing(content)[condA]; got.Occasion != "rdi-7" {
		t.Errorf("rdi-70 was read as naming rdi-7: %+v", got)
	}
}

func TestVerdictNamingTheOccasionOverrides(t *testing.T) {
	named := strings.Replace(verdictBlock, "the tree still holds it", "weighed rdi-7 and the tree holds it", 1)
	content := record(conditionBlock(condA, Falsified, "rdi-7", "the detection named it", ""), named)
	got := Standing(content)[condA]
	if got.Disposition != Survived || got.Occasion != "" {
		t.Errorf("a verdict naming the occasion did not override: %+v", got)
	}
}

func TestIsBlockMarkerAcceptsBothGrammars(t *testing.T) {
	for _, ln := range []string{
		"<!-- abcd-review: OWED receipt=rcp-0123456789ab -->",
		"<!-- abcd-review: INGESTED receipt=rcp-0123456789ab -->\r",
		"<!-- abcd-condition: " + condA + " occasion=rdi-1 -->",
	} {
		if !IsBlockMarker(ln) {
			t.Errorf("IsBlockMarker(%q) = false", ln)
		}
	}
	for _, ln := range []string{
		"",
		"text <!-- abcd-condition: " + condA + " occasion=rdi-1 -->",
		"<!-- abcd-condition: cond-1 occasion=rdi-1 -->",
		"<!-- cond: " + condA + " -->",
	} {
		if IsBlockMarker(ln) {
			t.Errorf("IsBlockMarker(%q) = true", ln)
		}
	}
}

func TestBlockMarkerAdmitsBothOccasionForms(t *testing.T) {
	for _, occ := range []string{"rdi-2609011200000001", "itd-199"} {
		ln := "<!-- abcd-condition: " + condA + " occasion=" + occ + " -->"
		if m := BlockMarkerRe.FindStringSubmatch(ln); m == nil || m[2] != occ {
			t.Errorf("occasion %s not admitted: %v", occ, m)
		}
	}
	for _, occ := range []string{"iss-1", "dsp-1", "spc-1", "rdi-", "rdi-1 x"} {
		if BlockMarkerRe.MatchString("<!-- abcd-condition: " + condA + " occasion=" + occ + " -->") {
			t.Errorf("occasion %q admitted", occ)
		}
	}
}
