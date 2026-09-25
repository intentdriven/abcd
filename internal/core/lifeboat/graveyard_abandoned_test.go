package lifeboat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// abandonedWriter builds an empty temp repo and returns its root plus a closure
// that writes a repo-relative file (creating parent dirs). It is the layer-2
// analogue of nativeTierFixture: each test writes exactly the record material it
// asserts on, so a fixture is never coupled to another test's expectations.
func abandonedWriter(t *testing.T) (string, func(rel, content string)) {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir, write
}

func abandonedCtx(t *testing.T, dir string) *SourceContext {
	t.Helper()
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ctx.Close() })
	return ctx
}

func gvFindingByID(fs []Finding, id string) (Finding, bool) {
	for _, f := range fs {
		if f.ID == id {
			return f, true
		}
	}
	return Finding{}, false
}

func gvCountSignal(fs []Finding, sig Signal) int {
	n := 0
	for _, f := range fs {
		if f.Signal == sig {
			n++
		}
	}
	return n
}

func gvEvidenceContains(f Finding, sub string) bool {
	for _, e := range f.Evidence {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}

// gvNoticesContain is gvEvidenceContains for the binary's own notices channel.
func gvNoticesContain(f Finding, sub string) bool {
	for _, e := range f.Notices {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}

// --- 1. superseded intents ---------------------------------------------------

func TestAbandonedSupersededIntent(t *testing.T) {
	dir, write := abandonedWriter(t)
	// A superseded intent (its bucket IS the lifecycle state).
	write(".abcd/development/intents/superseded/itd-31-cross-document-fidelity-reviewer.md",
		"---\nid: itd-31\nslug: cross-document-fidelity-reviewer\nsuperseded_by: itd-48\n---\n\n# Superseded intent\n")
	// A live intent in drafts/ must NOT be reported as superseded.
	write(".abcd/development/intents/drafts/itd-9-live.md",
		"---\nid: itd-9\nslug: live\n---\n\n# A live intent\n")

	fs := gvSupersededIntents(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalSupersededIntent) != 1 {
		t.Fatalf("want exactly one superseded-intent finding, got %d (%v)", len(fs), fs)
	}
	f, ok := gvFindingByID(fs, "itd-31")
	if !ok {
		t.Fatalf("want a finding keyed itd-31, got %v", fs)
	}
	if f.Signal != SignalSupersededIntent {
		t.Errorf("signal = %s, want %s", f.Signal, SignalSupersededIntent)
	}
	if !gvEvidenceContains(f, ".abcd/development/intents/superseded/itd-31-cross-document-fidelity-reviewer.md") {
		t.Errorf("evidence = %v, want the superseded path cited", f.Evidence)
	}
	if _, ok := gvFindingByID(fs, "itd-9"); ok {
		t.Errorf("a live intent (itd-9) must not appear as superseded")
	}
}

func TestAbandonedSupersededIntentsSortedNumerically(t *testing.T) {
	dir, write := abandonedWriter(t)
	write(".abcd/development/intents/superseded/itd-10-ten.md", "---\nid: itd-10\n---\n")
	write(".abcd/development/intents/superseded/itd-2-two.md", "---\nid: itd-2\n---\n")
	fs := gvSupersededIntents(abandonedCtx(t, dir))
	if len(fs) != 2 {
		t.Fatalf("want 2 findings, got %d", len(fs))
	}
	if fs[0].ID != "itd-2" || fs[1].ID != "itd-10" {
		t.Errorf("order = [%s %s], want numeric [itd-2 itd-10]", fs[0].ID, fs[1].ID)
	}
}

// --- 2. superseded ADRs ------------------------------------------------------

func TestAbandonedSupersededADRByStatus(t *testing.T) {
	dir, write := abandonedWriter(t)
	write(".abcd/development/decisions/adrs/0007-old.md",
		"---\nid: adr-7\nstatus: superseded\nsuperseded_by: null\n---\n\n# Old decision\n")
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	f, ok := gvFindingByID(fs, "adr-7")
	if !ok {
		t.Fatalf("status: superseded ADR should yield adr-7, got %v", fs)
	}
	if f.Signal != SignalSupersededADR {
		t.Errorf("signal = %s, want %s", f.Signal, SignalSupersededADR)
	}
}

func TestAbandonedSupersededADRBySupersededBy(t *testing.T) {
	dir, write := abandonedWriter(t)
	write(".abcd/development/decisions/adrs/0012-thing.md",
		"---\nid: adr-12\nstatus: accepted\nsuperseded_by: adr-31\n---\n\n# Thing\n")
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	f, ok := gvFindingByID(fs, "adr-12")
	if !ok {
		t.Fatalf("superseded_by ADR should yield adr-12, got %v", fs)
	}
	if !gvEvidenceContains(f, "adr-31") {
		t.Errorf("evidence = %v, want the superseding target adr-31 named", f.Evidence)
	}
}

func TestAbandonedAcceptedADRIsNotReported(t *testing.T) {
	dir, write := abandonedWriter(t)
	// Mirrors the real adr-35 frontmatter: accepted, superseded_by null.
	write(".abcd/development/decisions/adrs/0035-live.md",
		"---\nid: adr-35\nstatus: accepted\nsuperseded_by: null\n---\n\n# Live decision\n")
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	if len(fs) != 0 {
		t.Fatalf("an accepted ADR (superseded_by null) must not be reported, got %v", fs)
	}
}

func TestAbandonedAcceptedADRWithUppercaseNullIsNotReported(t *testing.T) {
	// Regression (iss #290): superseded_by carrying an uppercase YAML null —
	// NULL or Null — must read as "not superseded", exactly as lowercase null
	// and ~ do. frontmatter.IsNull previously missed the uppercase spellings, so
	// a live (status: accepted) ADR packed from a foreign repo via
	// `disembark pack` was silently emitted as a superseded-adr finding, quoting
	// `superseded_by: NULL` as its evidence. The status is not-superseded and
	// case-folded, so only the null literal decides the finding.
	//
	// This walks the bare-spelling matrix on the lifeboat path: only the four
	// UNQUOTED YAML nulls decide the finding here. A *quoted* null (`"NULL"`,
	// `'Null'`) is deliberately NOT in this table: per YAML scalar semantics a
	// quoted value is a string, and frontmatter.IsNull — which sees what Fields
	// captured, quotes intact — must keep reading it as non-null (asserted by
	// TestIsNull's negative controls). gvSupersededADRs currently calls
	// gvUnquote BEFORE IsNull, so in the lifeboat path alone a quoted null
	// happens to read as absent today; that is quote-insensitive sentinel
	// behaviour of gvUnquote, not YAML null semantics, and it is an open
	// heuristic decision for lifeboat supersession handling tracked separately —
	// this regression pins only the unquoted spellings. A real record handle
	// (`adr-9`) is the positive control: widening the null set must not
	// suppress a genuine superseding pointer.
	nulls := []string{"NULL", "Null", "null", "~"}
	for _, nul := range nulls {
		dir, write := abandonedWriter(t)
		write(".abcd/development/decisions/adrs/0035-live.md",
			"---\nid: adr-35\nstatus: accepted\nsuperseded_by: "+nul+"\n---\n\n# Live decision\n")
		fs := gvSupersededADRs(abandonedCtx(t, dir))
		if len(fs) != 0 {
			t.Fatalf("superseded_by: %s is a YAML null and must not be reported, got %v", nul, fs)
		}
	}
	// Positive control: a real handle still yields a superseded finding.
	dir, write := abandonedWriter(t)
	write(".abcd/development/decisions/adrs/0035-live.md",
		"---\nid: adr-35\nstatus: accepted\nsuperseded_by: adr-9\n---\n\n# Live decision\n")
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	if _, ok := gvFindingByID(fs, "adr-35"); !ok {
		t.Fatalf("superseded_by: adr-9 is a real handle and must still be reported, got %v", fs)
	}
}

func TestAbandonedSupersededADRAcrossBothHomesDedupes(t *testing.T) {
	dir, write := abandonedWriter(t)
	// Same ADR id present in the native home AND a conventional home.
	write(".abcd/development/decisions/adrs/0012-thing.md",
		"---\nid: adr-12\nstatus: superseded\nsuperseded_by: adr-31\n---\n\n# Native copy\n")
	write("docs/adr/0012-thing.md",
		"---\nid: adr-12\nstatus: superseded\nsuperseded_by: adr-31\n---\n\n# Conventional copy\n")
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalSupersededADR) != 1 {
		t.Fatalf("same ADR in both homes must dedupe to one finding, got %d (%v)", len(fs), fs)
	}
	f, _ := gvFindingByID(fs, "adr-12")
	if !gvEvidenceContains(f, ".abcd/development/decisions/adrs/0012-thing.md") {
		t.Errorf("first-wins should cite the native home, evidence = %v", f.Evidence)
	}
}

func TestAbandonedSupersededADRIDFromFilenameFallback(t *testing.T) {
	dir, write := abandonedWriter(t)
	// No usable frontmatter id — must derive adr-12 from the NNNN- filename.
	write("docs/adrs/0012-thing.md",
		"---\nstatus: superseded\n---\n\n# No id in frontmatter\n")
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	if _, ok := gvFindingByID(fs, "adr-12"); !ok {
		t.Fatalf("want adr-12 derived from filename, got %v", fs)
	}
}

// --- 3. alternatives considered ---------------------------------------------

func TestAbandonedAlternativesConsidered(t *testing.T) {
	dir, write := abandonedWriter(t)
	write(".abcd/development/decisions/adrs/0004-voyage.md",
		"---\nid: adr-4\nstatus: accepted\n---\n\n# 4. Voyage\n\n## Context\n\nWe need to pack.\n\n"+
			"## Alternatives Considered\n\n- Voyage inside the source repo — rejected because it mutates the tree.\n"+
			"- A second clone — rejected because it doubles disk.\n\n## Decision\n\nOut-of-tree.\n")
	// An ADR with no such section must not produce an alternatives finding.
	write(".abcd/development/decisions/adrs/0005-plain.md",
		"---\nid: adr-5\nstatus: accepted\n---\n\n# 5. Plain\n\n## Context\n\nNo alternatives here.\n")

	fs := gvAlternativesConsidered(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalAlternativesConsidered) != 1 {
		t.Fatalf("want exactly one alternatives finding, got %d (%v)", len(fs), fs)
	}
	f, ok := gvFindingByID(fs, "adr-4-alt")
	if !ok {
		t.Fatalf("want adr-4-alt, got %v", fs)
	}
	if !gvEvidenceContains(f, "Voyage inside the source repo") {
		t.Errorf("evidence = %v, want the first bullet quoted", f.Evidence)
	}
	if len(f.Evidence) != 2 {
		t.Errorf("want the two top-level bullets, got %d (%v)", len(f.Evidence), f.Evidence)
	}
	if _, ok := gvFindingByID(fs, "adr-5-alt"); ok {
		t.Errorf("adr-5 has no Alternatives section and must not appear")
	}
}

// --- 4. wontfix issues -------------------------------------------------------

func TestAbandonedWontfixIssue(t *testing.T) {
	dir, write := abandonedWriter(t)
	// Mirrors the real issue frontmatter shape: quoted id/slug.
	write(".abcd/work/issues/wontfix/iss-30-atomic-write.md",
		"---\nschema_version: 1\nid: \"iss-30\"\nslug: \"atomic-write\"\nseverity: \"minor\"\n"+
			"wontfix_reason: \"superseded by the atomic-write consolidation\"\n---\n\nbody.\n")
	fs := gvWontfixIssues(abandonedCtx(t, dir))
	f, ok := gvFindingByID(fs, "iss-30")
	if !ok {
		t.Fatalf("want iss-30 finding, got %v", fs)
	}
	if f.Signal != SignalWontfixIssue {
		t.Errorf("signal = %s, want %s", f.Signal, SignalWontfixIssue)
	}
	if !gvEvidenceContains(f, "superseded by the atomic-write consolidation") {
		t.Errorf("evidence = %v, want the wontfix reason quoted (unquoted)", f.Evidence)
	}
}

func TestAbandonedEmptyWontfixDirIsNone(t *testing.T) {
	dir, write := abandonedWriter(t)
	// A wontfix dir that exists but holds no iss-*.md.
	write(".abcd/work/issues/wontfix/README.md", "# wontfix ledger\n")
	fs := gvWontfixIssues(abandonedCtx(t, dir))
	if len(fs) != 0 {
		t.Fatalf("an empty wontfix ledger must yield no findings, got %v", fs)
	}
}

// --- 5. rejected options in DECISIONS.md ------------------------------------

func TestAbandonedRejectedOptions(t *testing.T) {
	dir, write := abandonedWriter(t)
	write(".abcd/work/DECISIONS.md",
		"# DECISIONS\n\n"+ // line 1, 2
			"- 2026-07-06 — Adopt Cobra as the CLI framework.\n"+ // line 3 (neutral)
			"- 2026-07-08 — RAG rejected at this scale; grep corpus instead.\n"+ // line 4 (rejected)
			"- 2026-07-09 — flow-next dropped in favour of native.\n"+ // line 5 (dropped)
			"- 2026-07-10 — Private companion repo deferred (trigger: shared transcripts).\n") // line 6 (deferred)

	fs := gvRejectedOptions(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalRejectedOption) != 3 {
		t.Fatalf("want 3 rejected-option findings (rejected/dropped/deferred), got %d (%v)", len(fs), fs)
	}
	// dec-L<line> keyed by 1-based line number, file order preserved.
	if fs[0].ID != "dec-L4" || fs[1].ID != "dec-L5" || fs[2].ID != "dec-L6" {
		t.Errorf("ids = [%s %s %s], want [dec-L4 dec-L5 dec-L6]", fs[0].ID, fs[1].ID, fs[2].ID)
	}
	if !gvEvidenceContains(fs[0], "RAG rejected at this scale") {
		t.Errorf("evidence = %v, want the line quoted verbatim", fs[0].Evidence)
	}
	if _, ok := gvFindingByID(fs, "dec-L3"); ok {
		t.Errorf("the neutral bullet on line 3 must not be reported")
	}
}

func TestAbandonedRejectedOptionsConservativeMatcher(t *testing.T) {
	dir, write := abandonedWriter(t)
	// "rejection" is a different word from the verb "rejected": the substring
	// matcher must NOT fire on it (documented conservative-verbs-only edge).
	write(".abcd/work/DECISIONS.md",
		"- 2026-07-06 — Handling of the rejection path is documented in the spec.\n")
	fs := gvRejectedOptions(abandonedCtx(t, dir))
	if len(fs) != 0 {
		t.Fatalf("a bullet containing \"rejection\" (not the verb) must not fire, got %v", fs)
	}
}

// --- 6. empty record ---------------------------------------------------------

func TestAbandonedEmptyRecordIsEmptySlice(t *testing.T) {
	dir := t.TempDir() // no .abcd at all
	ab := buildAbandoned(abandonedCtx(t, dir))
	if ab.SchemaVersion != GraveyardSchemaVersion {
		t.Errorf("schema_version = %d, want %d", ab.SchemaVersion, GraveyardSchemaVersion)
	}
	if ab.Findings == nil {
		t.Fatal("Findings must be a non-nil empty slice, not nil")
	}
	if len(ab.Findings) != 0 {
		t.Fatalf("want no findings for an empty record, got %v", ab.Findings)
	}
	j, err := json.MarshalIndent(ab, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(j), "\"findings\": []") {
		t.Errorf("empty abandoned must marshal findings as [], got:\n%s", j)
	}
}

// --- 7. integration + id stability ------------------------------------------

func TestAbandonedBuildGroupsAllSignalsInRankOrder(t *testing.T) {
	dir := abandonedFullFixture(t)
	ab := buildAbandoned(abandonedCtx(t, dir))
	// Signals must appear grouped in signalRank order, never re-sorted globally.
	lastRank := -1
	for _, f := range ab.Findings {
		r, ok := signalRank[f.Signal]
		if !ok {
			t.Fatalf("finding %s carries an unranked signal %s", f.ID, f.Signal)
		}
		if r < lastRank {
			t.Fatalf("signal %s (rank %d) appears after rank %d — not grouped by signalRank", f.Signal, r, lastRank)
		}
		lastRank = r
	}
	// Sanity: at least one finding from each of the five layer-2 signals.
	for _, sig := range []Signal{
		SignalSupersededIntent, SignalSupersededADR, SignalAlternativesConsidered,
		SignalWontfixIssue, SignalRejectedOption,
	} {
		if gvCountSignal(ab.Findings, sig) == 0 {
			t.Errorf("full fixture produced no %s finding", sig)
		}
	}
}

func TestAbandonedIDStabilityAcrossCalls(t *testing.T) {
	dir := abandonedFullFixture(t)
	a := buildAbandoned(abandonedCtx(t, dir))
	b := buildAbandoned(abandonedCtx(t, dir))
	ja, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	jb, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(ja) != string(jb) {
		t.Errorf("buildAbandoned is not byte-stable across calls:\n--- a ---\n%s\n--- b ---\n%s", ja, jb)
	}
}

// --- 8. sanitisation ---------------------------------------------------------

func TestAbandonedSanitisesControlChars(t *testing.T) {
	dir, write := abandonedWriter(t)
	// A DECISIONS line and a wontfix reason each carrying an ANSI escape (0x1b)
	// and a NUL (0x00) plus an HTML-comment marker. sanitize() maps C0/DEL to '?'
	// and tab to space; markers are inert in JSON so they pass through unchanged.
	write(".abcd/work/DECISIONS.md",
		"- 2026-07-06 — option \x1bdropped\x00 here <!-- hide -->\n")
	write(".abcd/work/issues/wontfix/iss-9-x.md",
		"---\nid: \"iss-9\"\nwontfix_reason: \"bad\x1breason\x00 <!-- x -->\"\n---\n")

	dec := gvRejectedOptions(abandonedCtx(t, dir))
	if len(dec) != 1 {
		t.Fatalf("want one decision finding, got %v", dec)
	}
	for _, e := range dec[0].Evidence {
		if strings.ContainsRune(e, 0x1b) || strings.ContainsRune(e, 0x00) {
			t.Errorf("evidence retains a control char: %q", e)
		}
		if !strings.Contains(e, "?dropped?") {
			t.Errorf("control chars should map to '?', got %q", e)
		}
	}

	won := gvWontfixIssues(abandonedCtx(t, dir))
	f, ok := gvFindingByID(won, "iss-9")
	if !ok {
		t.Fatalf("want iss-9, got %v", won)
	}
	for _, e := range f.Evidence {
		if strings.ContainsRune(e, 0x1b) || strings.ContainsRune(e, 0x00) {
			t.Errorf("wontfix evidence retains a control char: %q", e)
		}
	}
}

// abandonedFullFixture writes a record exercising all five layer-2 signals at
// once. It is the integration/stability fixture; the per-signal tests keep their
// own minimal fixtures.
func abandonedFullFixture(t *testing.T) string {
	t.Helper()
	dir, write := abandonedWriter(t)
	write(".abcd/development/intents/superseded/itd-47-oracle-gates.md",
		"---\nid: itd-47\nslug: oracle-gates\nsuperseded_by: itd-48\n---\n\n# Superseded\n")
	write(".abcd/development/decisions/adrs/0012-old.md",
		"---\nid: adr-12\nstatus: superseded\nsuperseded_by: adr-31\n---\n\n# Old\n")
	write(".abcd/development/decisions/adrs/0004-voyage.md",
		"---\nid: adr-4\nstatus: accepted\n---\n\n# Voyage\n\n"+
			"## Alternatives Considered\n\n- Voyage inside the source repo — rejected.\n- A second clone — rejected.\n")
	write(".abcd/work/issues/wontfix/iss-30-atomic.md",
		"---\nid: \"iss-30\"\nslug: \"atomic\"\nwontfix_reason: \"superseded by consolidation\"\n---\n\nbody.\n")
	write(".abcd/work/DECISIONS.md",
		"# DECISIONS\n\n- 2026-07-08 — RAG rejected at this scale.\n")
	return dir
}

// --- 9. the cap notice -------------------------------------------------------

// TestAbandonedSupersededADRCapNotesTruncation pins the contract stated on
// maxGraveyardFindingsPerSignal — "the last retained finding for a truncated
// signal notes the cap" — for a layer-2 signal. Layer 1's capSignalFindings
// already honours it; a layer-2 signal that truncated silently would present a
// truncated abandoned.json as complete.
func TestAbandonedSupersededADRCapNotesTruncation(t *testing.T) {
	dir, write := abandonedWriter(t)
	for i := 1; i <= maxGraveyardFindingsPerSignal+2; i++ {
		write(fmt.Sprintf("docs/adr/%04d-thing.md", i),
			fmt.Sprintf("---\nid: adr-%d\nstatus: superseded\n---\n\n# %d\n", i, i))
	}
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	if len(fs) != maxGraveyardFindingsPerSignal {
		t.Fatalf("findings = %d, want the cap %d", len(fs), maxGraveyardFindingsPerSignal)
	}
	last := fs[len(fs)-1]
	want := fmt.Sprintf("+2 further findings omitted; capped at %d", maxGraveyardFindingsPerSignal)
	if !gvNoticesContain(last, want) {
		t.Errorf("truncated signal did not note the cap: last finding %s notices = %v, want a line containing %q",
			last.ID, last.Notices, want)
	}
}

// --- 10. one ADR identity, every claimant announced --------------------------

// TestAbandonedSupersededADRDuplicateNumberAnnouncesShadow: an ADR number is not
// a unique name — adr-tools branches collide on one, and log4brains' YYYYMMDD-
// filenames collide for every ADR written the same day. The colliding records
// still dedupe to one finding (a reader cannot tell a genuine duplicate from the
// same ADR copied into two homes), but the drop must be announced on the finding
// that won rather than vanishing with no marker anywhere in the lifeboat.
func TestAbandonedSupersededADRDuplicateNumberAnnouncesShadow(t *testing.T) {
	dir, write := abandonedWriter(t)
	write("docs/adr/0007-use-kafka.md", "---\nstatus: superseded\n---\n\n# 7. Kafka\n")
	write("docs/adr/0007-use-rabbitmq.md", "---\nstatus: superseded\n---\n\n# 7. RabbitMQ\n")

	fs := gvSupersededADRs(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalSupersededADR) != 1 {
		t.Fatalf("two records claiming adr-7 must yield one finding, got %d (%v)", len(fs), fs)
	}
	f, ok := gvFindingByID(fs, "adr-7")
	if !ok {
		t.Fatalf("want adr-7, got %v", fs)
	}
	if !gvNoticesContain(f, "docs/adr/0007-use-rabbitmq.md") {
		t.Errorf("the shadowed claimant was dropped silently: notices = %v", f.Notices)
	}
}

// TestAbandonedSupersededADRPaddedFrontmatterIDDedupesAcrossHomes is the inverse
// failure: gvSupersededADRs documents a first-wins dedup across homes, but the
// two id-derivation paths canonicalised differently — a native `id: adr-012` and
// a filename-derived adr-12 for the SAME ADR read as two records, so the copy was
// reported twice. adr-0012 and adr-12 are one handle everywhere else in the
// repository (record dispatch, the citation resolver); layer 2 must agree.
func TestAbandonedSupersededADRPaddedFrontmatterIDDedupesAcrossHomes(t *testing.T) {
	dir, write := abandonedWriter(t)
	write(".abcd/development/decisions/adrs/0012-thing.md",
		"---\nid: adr-012\nstatus: superseded\n---\n\n# Native copy\n")
	write("docs/adr/0012-thing.md",
		"---\nstatus: superseded\n---\n\n# Conventional copy\n")

	fs := gvSupersededADRs(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalSupersededADR) != 1 {
		t.Fatalf("one ADR in two homes must dedupe to one finding, got %d (%v)", len(fs), fs)
	}
	if _, ok := gvFindingByID(fs, "adr-12"); !ok {
		t.Errorf("want the canonical id adr-12, got %v", fs)
	}
}

// TestAbandonedAlternativesConsideredDuplicateNumberAnnouncesShadow: the
// alternatives signal shares gvADRID and shadowed identically.
func TestAbandonedAlternativesConsideredDuplicateNumberAnnouncesShadow(t *testing.T) {
	dir, write := abandonedWriter(t)
	body := "\n## Alternatives Considered\n\n- %s — rejected.\n\n## Decision\n\nx.\n"
	write("docs/adr/20200926-use-log4brains.md",
		"---\nstatus: accepted\n---\n\n# log4brains\n"+fmt.Sprintf(body, "a docs folder"))
	write("docs/adr/20200926-use-markdown-adrs.md",
		"---\nstatus: accepted\n---\n\n# markdown adrs\n"+fmt.Sprintf(body, "a wiki"))

	fs := gvAlternativesConsidered(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalAlternativesConsidered) != 1 {
		t.Fatalf("two records claiming one ADR number must yield one finding, got %d (%v)", len(fs), fs)
	}
	if !gvNoticesContain(fs[0], "docs/adr/20200926-use-markdown-adrs.md") {
		t.Errorf("the shadowed claimant was dropped silently: notices = %v", fs[0].Notices)
	}
}

// TestGvCanonADRIDStripsPadding pins the canonical spelling both derivation paths
// must resolve to, rebuilt from the parsed integer so a padded id can never mint
// a second spelling of one record.
func TestGvCanonADRIDStripsPadding(t *testing.T) {
	for in, want := range map[string]string{
		"adr-12":   "adr-12",
		"adr-012":  "adr-12",
		"adr-0012": "adr-12",
		"adr-x":    "",
		"itd-12":   "",
		"":         "",
		// All zeros is one handle, not the empty id.
		"adr-0":   "adr-0",
		"adr-000": "adr-0",
		// Wider than any integer type: still a well-formed id, so it must keep an
		// identity rather than canonicalise to "" and be skipped in silence.
		"adr-99999999999999999999":   "adr-99999999999999999999",
		"adr-0099999999999999999999": "adr-99999999999999999999",
	} {
		if got := gvCanonADRID(in); got != want {
			t.Errorf("gvCanonADRID(%q) = %q, want %q", in, got, want)
		}
	}
}

// gvHasTruncationNotice reports whether fs carries a per-scan listing-truncation
// notice for signal sig.
func gvHasTruncationNotice(fs []Finding, sig Signal) bool {
	for _, f := range fs {
		if f.Signal == sig && strings.Contains(f.Summary, "truncated") {
			return true
		}
	}
	return false
}

// --- 11. filename-ordinal identity (iss-2608270945469978) -------------------

// TestAbandonedSupersededADRHugeFilenameOrdinalKeepsIdentity: an ADR whose id is
// DERIVED from a filename ordinal wider than any integer type must keep an
// identity. gvADRIDFromFilename parsed the digit run with strconv.Atoi and returned
// "" on overflow, so gvADRID yielded "" and the record was skipped in silence — the
// filename-fallback twin of the resolved textual-canonicalisation fix.
func TestAbandonedSupersededADRHugeFilenameOrdinalKeepsIdentity(t *testing.T) {
	dir, write := abandonedWriter(t)
	huge := "99999999999999999999" // 20 digits: beyond int64
	// No frontmatter id, so the id must be derived from the filename ordinal.
	write("docs/adr/"+huge+"-thing.md", "---\nstatus: superseded\n---\n\n# huge\n")
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	if _, ok := gvFindingByID(fs, "adr-"+huge); !ok {
		t.Fatalf("want adr-%s derived from a huge filename ordinal, got %v", huge, fs)
	}
}

// TestAbandonedSupersededADRHugeFilenamePaddedDedupes: a padded and a bare huge
// filename ordinal are one handle — the textual canonicaliser trims the padding, so
// the two homes dedupe to one finding rather than minting two.
func TestAbandonedSupersededADRHugeFilenamePaddedDedupes(t *testing.T) {
	dir, write := abandonedWriter(t)
	huge := "99999999999999999999"
	write("docs/adr/"+huge+"-a.md", "---\nstatus: superseded\n---\n\n# a\n")
	write("docs/adrs/00"+huge+"-b.md", "---\nstatus: superseded\n---\n\n# b\n")
	fs := gvSupersededADRs(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalSupersededADR) != 1 {
		t.Fatalf("padded and bare huge filename ordinals must dedupe to one finding, got %d (%v)", len(fs), fs)
	}
	if _, ok := gvFindingByID(fs, "adr-"+huge); !ok {
		t.Errorf("want the canonical id adr-%s, got %v", huge, fs)
	}
}

// --- 12. intent + issue canonicalisation and shadows (iss-2608270908349721) --

func TestGvCanonIntentAndIssueIDStripPadding(t *testing.T) {
	for in, want := range map[string]string{
		"itd-7": "itd-7", "itd-007": "itd-7", "itd-0": "itd-0", "itd-000": "itd-0",
		"itd-x": "", "iss-9": "", "": "",
	} {
		if got := gvCanonIntentID(in); got != want {
			t.Errorf("gvCanonIntentID(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{
		"iss-9": "iss-9", "iss-009": "iss-9", "iss-0": "iss-0", "iss-000": "iss-0",
		"iss-x": "", "itd-9": "", "": "",
	} {
		if got := gvCanonIssueID(in); got != want {
			t.Errorf("gvCanonIssueID(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestAbandonedSupersededIntentPaddedIDDedupes: itd-7 and itd-007 are one intent
// everywhere the record is read, but the intent signal deduped on the raw
// frontmatter id with no canonicaliser and no shadow, so the two spellings keyed
// two findings and neither drop was announced.
func TestAbandonedSupersededIntentPaddedIDDedupes(t *testing.T) {
	dir, write := abandonedWriter(t)
	write(".abcd/development/intents/superseded/itd-7-bare.md", "---\nid: itd-7\n---\n")
	write(".abcd/development/intents/superseded/itd-007-padded.md", "---\nid: itd-007\n---\n")
	fs := gvSupersededIntents(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalSupersededIntent) != 1 {
		t.Fatalf("itd-7 and itd-007 are one intent and must dedupe to one finding, got %d (%v)", len(fs), fs)
	}
	f, ok := gvFindingByID(fs, "itd-7")
	if !ok {
		t.Fatalf("want the canonical id itd-7, got %v", fs)
	}
	if !gvNoticesContain(f, "shadowed") {
		t.Errorf("the dropped duplicate must be announced as a shadow: notices = %v", f.Notices)
	}
}

// TestAbandonedWontfixIssuePaddedIDDedupes: the issue analogue of the above.
func TestAbandonedWontfixIssuePaddedIDDedupes(t *testing.T) {
	dir, write := abandonedWriter(t)
	write(".abcd/work/issues/wontfix/iss-7-bare.md", "---\nid: \"iss-7\"\n---\n")
	write(".abcd/work/issues/wontfix/iss-007-padded.md", "---\nid: \"iss-007\"\n---\n")
	fs := gvWontfixIssues(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalWontfixIssue) != 1 {
		t.Fatalf("iss-7 and iss-007 are one issue and must dedupe to one finding, got %d (%v)", len(fs), fs)
	}
	f, ok := gvFindingByID(fs, "iss-7")
	if !ok {
		t.Fatalf("want the canonical id iss-7, got %v", fs)
	}
	if !gvNoticesContain(f, "shadowed") {
		t.Errorf("the dropped duplicate must be announced as a shadow: notices = %v", f.Notices)
	}
}

// --- 13. per-scan listing-truncation notice (iss-2608270908348796) -----------

// TestAbandonedSupersededIntentListingTruncationNoticed: a superseded/ home holding
// more entries than the per-directory listing cap is read only up to the cap, so the
// scan is INCOMPLETE and must say so with a per-scan notice rather than present a
// truncated input as a complete graveyard.
func TestAbandonedSupersededIntentListingTruncationNoticed(t *testing.T) {
	dir, write := abandonedWriter(t)
	for i := 1; i <= 3; i++ {
		write(fmt.Sprintf(".abcd/development/intents/superseded/itd-%d-x.md", i),
			fmt.Sprintf("---\nid: itd-%d\n---\n", i))
	}
	ctx := abandonedCtx(t, dir)
	ctx.listCap = 2 // force truncation on a small tree
	fs := gvSupersededIntents(ctx)
	if !gvHasTruncationNotice(fs, SignalSupersededIntent) {
		t.Fatalf("a truncated superseded/ listing must emit a per-scan notice, got %v", fs)
	}
}

// TestAbandonedWontfixIssueListingTruncationNoticed: the wontfix/ ledger analogue.
func TestAbandonedWontfixIssueListingTruncationNoticed(t *testing.T) {
	dir, write := abandonedWriter(t)
	for i := 1; i <= 3; i++ {
		write(fmt.Sprintf(".abcd/work/issues/wontfix/iss-%d-x.md", i),
			fmt.Sprintf("---\nid: \"iss-%d\"\n---\n", i))
	}
	ctx := abandonedCtx(t, dir)
	ctx.listCap = 2
	fs := gvWontfixIssues(ctx)
	if !gvHasTruncationNotice(fs, SignalWontfixIssue) {
		t.Fatalf("a truncated wontfix/ listing must emit a per-scan notice, got %v", fs)
	}
}

// TestAbandonedSupersededADRListingTruncationNoticed: an ADR home past the cap must
// note its truncation on the signals gvEachADR feeds.
func TestAbandonedSupersededADRListingTruncationNoticed(t *testing.T) {
	dir, write := abandonedWriter(t)
	for i := 1; i <= 3; i++ {
		write(fmt.Sprintf("docs/adr/%04d-x.md", i),
			fmt.Sprintf("---\nid: adr-%d\nstatus: superseded\n---\n\n# %d\n", i, i))
	}
	ctx := abandonedCtx(t, dir)
	ctx.listCap = 2
	fs := gvSupersededADRs(ctx)
	if !gvHasTruncationNotice(fs, SignalSupersededADR) {
		t.Fatalf("a truncated ADR home must emit a per-scan notice, got %v", fs)
	}
}

// TestAbandonedListingUnderCapEmitsNoNotice: the notice fires ONLY on truncation —
// a home within the cap yields exactly its records and no spurious notice.
func TestAbandonedListingUnderCapEmitsNoNotice(t *testing.T) {
	dir, write := abandonedWriter(t)
	write(".abcd/development/intents/superseded/itd-1-x.md", "---\nid: itd-1\n---\n")
	ctx := abandonedCtx(t, dir)
	ctx.listCap = 50
	fs := gvSupersededIntents(ctx)
	if gvHasTruncationNotice(fs, SignalSupersededIntent) {
		t.Fatalf("a listing within the cap must not emit a truncation notice, got %v", fs)
	}
}

// TestAbandonedSupersededADROversizeOrdinalKeepsIdentity: an ADR ordinal wider
// than any integer type is still a well-formed adr-N and must keep an identity.
// Canonicalising through an integer parse would fail on it and return "", and a
// record with no id is skipped in silence — the drop this file's dedup exists to
// announce, reintroduced at the identity step.
func TestAbandonedSupersededADROversizeOrdinalKeepsIdentity(t *testing.T) {
	dir, write := abandonedWriter(t)
	huge := "99999999999999999999" // 20 digits: beyond int64
	// Filenames start with a letter, so the frontmatter id is the only id source.
	write("docs/adr/a-padded.md", "---\nid: adr-00"+huge+"\nstatus: superseded\n---\n\n# padded\n")
	write("docs/adr/b-bare.md", "---\nid: adr-"+huge+"\nstatus: superseded\n---\n\n# bare\n")

	fs := gvSupersededADRs(abandonedCtx(t, dir))
	if gvCountSignal(fs, SignalSupersededADR) != 1 {
		t.Fatalf("padded and bare spellings of one huge ordinal must dedupe to one finding, got %d (%v)", len(fs), fs)
	}
	f, ok := gvFindingByID(fs, "adr-"+huge)
	if !ok {
		t.Fatalf("want the canonical id adr-%s, got %v", huge, fs)
	}
	if !gvNoticesContain(f, "docs/adr/b-bare.md") {
		t.Errorf("the shadowed claimant was dropped silently: notices = %v", f.Notices)
	}
}
