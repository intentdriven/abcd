package lifeboat

import (
	"os"
	"path/filepath"
	"testing"
)

// nativeTierFixture builds a small abcd record under .abcd/ exercising the
// Tier-2 native signals: an authored product/context brief file, a STUB
// mental-model brief file, a numbered ADR, an open issue, an intent, and a
// conventions router with a lint config. It deliberately carries NO personas
// file, so that section is a genuine absent-material blank. Returns the repo
// root directory.
func nativeTierFixture(t *testing.T) string {
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

	// An authored brief section: comfortably over the body-prose threshold.
	write(".abcd/development/brief/01-product/02-context.md", "# Context\n\n"+
		"abcd is a host-agnostic configuration layer for development, carried in a "+
		"single Go binary. It reads a project's record and, where none exists, its "+
		"git history, and reports what a lifeboat could reconstruct. The record "+
		"lives under .abcd/ and is the single source of truth for the project's "+
		"durable decisions, intents, and specifications.\n")

	// A STUB brief section: a heading and a placeholder line, well under the
	// threshold — must NOT ground.
	write(".abcd/development/brief/01-product/03-mental-model.md", "# Mental model\n\nTODO.\n")

	// A numbered ADR carrying an Alternatives Considered section.
	write(".abcd/development/decisions/adrs/0001-record-architecture-decisions.md",
		"# 1. Record architecture decisions\n\n## Context\n\nWe need a durable log.\n\n"+
			"## Alternatives Considered\n\nA wiki; a spreadsheet. Both drift.\n")

	// An open issue in the capture ledger.
	write(".abcd/work/issues/open/iss-001-example.md", "# iss-001\n\nAn open question.\n")

	// An intent in the corpus.
	write(".abcd/development/intents/drafts/itd-001-example.md", "# itd-001\n\nAn intent.\n")

	// A conventions router and a lint config.
	write("AGENTS.md", "# AGENTS\n\nInvariants: the core never writes to stdout.\n")
	write(".abcd/rules.json", "{\"schema_version\": 1, \"disabled\": false, \"domains\": {}}\n")

	return dir
}

// nativeSourceForSection returns the Tier-2 adapter that speaks for section,
// failing the test if none does.
func nativeSourceForSection(t *testing.T, section Section) Source {
	t.Helper()
	for _, s := range nativeSources() {
		if s.Section() == section {
			return s
		}
	}
	t.Fatalf("no native source for section %s", section)
	return nil
}

// TestNativeContextGroundsFromBrief is the flagship Tier-2 assertion: an authored
// brief file grounds "product/context" and cites the brief path.
func TestNativeContextGroundsFromBrief(t *testing.T) {
	ctx, err := newSourceContext(nativeTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	src := nativeSourceForSection(t, "product/context")
	if src.Tier() != TierNative {
		t.Fatalf("product/context tier = %s, want %s", src.Tier(), TierNative)
	}
	ev := src.Probe(ctx)
	if ev.Status != StatusGrounded {
		t.Fatalf("product/context status = %s, want grounded", ev.Status)
	}
	if !containsSource(ev.Sources, ".abcd/development/brief/01-product/02-context.md") {
		t.Errorf("product/context evidence = %v, want the brief file cited", ev.Sources)
	}
	if ev.Confidence == "" {
		t.Error("grounded product/context has no confidence")
	}
}

// TestNativeStubBriefDoesNotGround holds the "do not ground a stub" contract: a
// near-empty brief file yields partial (not grounded), still citing the file.
func TestNativeStubBriefDoesNotGround(t *testing.T) {
	ctx, err := newSourceContext(nativeTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := nativeSourceForSection(t, "product/mental-model").Probe(ctx)
	if ev.Status == StatusGrounded {
		t.Fatalf("stub product/mental-model grounded; want partial/blank")
	}
	if ev.Status != StatusPartial {
		t.Fatalf("stub product/mental-model status = %s, want partial", ev.Status)
	}
	if len(ev.Sources) == 0 {
		t.Error("partial mental-model cites no evidence")
	}
}

// TestNativeMissingBriefIsBlank confirms a section whose brief file is absent
// returns a blank carrying what was searched and a question a human must answer.
func TestNativeMissingBriefIsBlank(t *testing.T) {
	ctx, err := newSourceContext(nativeTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	// No press-release file in the fixture.
	ev := nativeSourceForSection(t, "product/press-release").Probe(ctx)
	if ev.Status != StatusBlank {
		t.Fatalf("absent product/press-release status = %s, want blank", ev.Status)
	}
	if ev.Question == "" {
		t.Error("blank product/press-release carries no question for a human")
	}
	if len(ev.Searched) == 0 {
		t.Error("blank product/press-release names nothing it searched")
	}
}

// TestNativeADRsGround confirms the record's ADR directory grounds "docs/adrs"
// and the citation lists the ADR file.
func TestNativeADRsGround(t *testing.T) {
	ctx, err := newSourceContext(nativeTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := nativeSourceForSection(t, "docs/adrs").Probe(ctx)
	if ev.Status != StatusGrounded {
		t.Fatalf("docs/adrs status = %s, want grounded", ev.Status)
	}
	if !containsSource(ev.Sources, ".abcd/development/decisions/adrs/0001-record-architecture-decisions.md") {
		t.Errorf("docs/adrs evidence = %v, want the ADR file listed", ev.Sources)
	}
}

// TestNativeIssuesGround confirms the capture ledger grounds "activity/issues".
func TestNativeIssuesGround(t *testing.T) {
	ctx, err := newSourceContext(nativeTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := nativeSourceForSection(t, "activity/issues").Probe(ctx)
	if ev.Status != StatusGrounded {
		t.Fatalf("activity/issues status = %s, want grounded", ev.Status)
	}
	if len(ev.Sources) == 0 {
		t.Fatal("grounded activity/issues cites no evidence")
	}
}

// TestNativeSpineGroundsFromIntents confirms the intent corpus grounds
// "rescue/spine" and cites the intents directory.
func TestNativeSpineGroundsFromIntents(t *testing.T) {
	ctx, err := newSourceContext(nativeTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := nativeSourceForSection(t, "rescue/spine").Probe(ctx)
	if ev.Status != StatusGrounded {
		t.Fatalf("rescue/spine status = %s, want grounded", ev.Status)
	}
	if len(ev.Sources) == 0 {
		t.Fatal("grounded rescue/spine cites no evidence")
	}
}

// TestNativeInvariantsGround confirms the conventions router grounds
// "constraints/invariants" with AGENTS.md cited.
func TestNativeInvariantsGround(t *testing.T) {
	ctx, err := newSourceContext(nativeTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := nativeSourceForSection(t, "constraints/invariants").Probe(ctx)
	if ev.Status != StatusGrounded {
		t.Fatalf("constraints/invariants status = %s, want grounded", ev.Status)
	}
	if !containsSource(ev.Sources, "AGENTS.md") {
		t.Errorf("constraints/invariants evidence = %v, want AGENTS.md cited", ev.Sources)
	}
}

// TestNativePersonasBlank holds the human-owned case: with no personas file in
// the fixture, "product/personas" returns a blank naming what it searched and the
// question a human must answer.
func TestNativePersonasBlank(t *testing.T) {
	ctx, err := newSourceContext(nativeTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := nativeSourceForSection(t, "product/personas").Probe(ctx)
	if ev.Status != StatusBlank {
		t.Fatalf("product/personas status = %s, want blank (no personas file in fixture)", ev.Status)
	}
	if ev.Question == "" {
		t.Error("blank product/personas carries no question for a human")
	}
	if len(ev.Searched) == 0 {
		t.Error("blank product/personas names nothing it searched")
	}
}

// TestNativeAdaptersAllReportTierNative guards the tier contract: every Tier-2
// adapter reports TierNative.
func TestNativeAdaptersAllReportTierNative(t *testing.T) {
	for _, s := range nativeSources() {
		if s.Tier() != TierNative {
			t.Errorf("source for %s reports tier %s, want %s", s.Section(), s.Tier(), TierNative)
		}
	}
}

// TestNativeProbeIsDeterministic asserts the Tier-2 adapters are byte-stable
// across runs against the same record.
func TestNativeProbeIsDeterministic(t *testing.T) {
	dir := nativeTierFixture(t)
	ctxA, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctxA.Close()
	ctxB, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctxB.Close()

	for _, s := range nativeSources() {
		a := s.Probe(ctxA)
		b := s.Probe(ctxB)
		if len(a.Sources) != len(b.Sources) {
			t.Fatalf("%s: source count differs across runs", s.Section())
		}
		for i := range a.Sources {
			if a.Sources[i] != b.Sources[i] {
				t.Errorf("%s: source[%d] differs: %q vs %q", s.Section(), i, a.Sources[i], b.Sources[i])
			}
		}
	}
}

// TestNativeEvidenceSectionsGroundFromTheirBriefFiles: the two evidence sections
// a dedicated adapter owns ground from the section's own authored brief file, as
// every other brief section does, even when the record holds no open issue, no
// intent, no ADR alternative and no decision log (iss-2610040758387938).
func TestNativeEvidenceSectionsGroundFromTheirBriefFiles(t *testing.T) {
	prose := "This section is authored by the project, not scaffolded. It names the " +
		"questions the team still carries and the trade-offs it weighed, each with " +
		"the reason it was left open or settled, so a reader can pick the work up " +
		"without reconstructing it from history.\n"
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		".abcd/development/brief/03-evidence/03-open-questions.md": "# Open questions\n\n" + prose,
		".abcd/development/brief/03-evidence/04-tradeoffs.md":      "# Trade-offs\n\n" + prose,
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	for section, file := range map[Section]string{
		"evidence/open-questions": ".abcd/development/brief/03-evidence/03-open-questions.md",
		"evidence/tradeoffs":      ".abcd/development/brief/03-evidence/04-tradeoffs.md",
	} {
		ev := nativeSourceForSection(t, section).Probe(ctx)
		if ev.Status != StatusGrounded {
			t.Errorf("%s = %s with an authored %s, want grounded", section, ev.Status, file)
		}
		if !containsSource(ev.Sources, file) {
			t.Errorf("%s evidence = %v, want %s cited", section, ev.Sources, file)
		}
	}

	// A stub brief file alone is partial, never grounded, and says why.
	stub := t.TempDir()
	writeTree(t, stub, map[string]string{
		".abcd/development/brief/03-evidence/03-open-questions.md": "# Open questions\n\nTODO.\n",
	})
	sctx, err := newSourceContext(stub)
	if err != nil {
		t.Fatal(err)
	}
	defer sctx.Close()
	ev := nativeSourceForSection(t, "evidence/open-questions").Probe(sctx)
	if ev.Status != StatusPartial || ev.Reason == "" {
		t.Errorf("stub open-questions = %s (reason %q), want partial with a reason", ev.Status, ev.Reason)
	}
}

// TestNativeConstraintAndPersonaSectionsGroundFromTheirBriefFiles: the
// invariants, naming and personas adapters read the section's own brief file,
// as the evidence adapters do — an authored file grounds the section even when
// the record holds no conventions router, glossary or other signal, and a stub
// one alone is partial and says why (iss-2610072347247487).
func TestNativeConstraintAndPersonaSectionsGroundFromTheirBriefFiles(t *testing.T) {
	prose := "This section is authored by the project, not scaffolded. It states what " +
		"the team holds fixed and who the work is for, each with the reason it was " +
		"settled that way, so a reader can pick the work up without reconstructing " +
		"it from history or asking the people who wrote it.\n"
	files := map[Section]string{
		"constraints/invariants": ".abcd/development/brief/02-constraints/03-invariants.md",
		"constraints/naming":     ".abcd/development/brief/02-constraints/04-naming.md",
		"product/personas":       ".abcd/development/brief/01-product/05-personas.md",
	}

	authored := map[string]string{}
	stubbed := map[string]string{}
	for _, file := range files {
		authored[file] = "# Section\n\n" + prose
		stubbed[file] = "# Section\n\nTODO.\n"
	}
	dir := t.TempDir()
	writeTree(t, dir, authored)
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	stub := t.TempDir()
	writeTree(t, stub, stubbed)
	sctx, err := newSourceContext(stub)
	if err != nil {
		t.Fatal(err)
	}
	defer sctx.Close()

	for section, file := range files {
		ev := nativeSourceForSection(t, section).Probe(ctx)
		if ev.Status != StatusGrounded {
			t.Errorf("%s = %s with an authored %s, want grounded", section, ev.Status, file)
		}
		if !containsSource(ev.Sources, file) {
			t.Errorf("%s evidence = %v, want %s cited", section, ev.Sources, file)
		}

		ev = nativeSourceForSection(t, section).Probe(sctx)
		if ev.Status != StatusPartial || ev.Reason == "" {
			t.Errorf("stub %s = %s (reason %q), want partial with a reason", section, ev.Status, ev.Reason)
		}
		if !containsSource(ev.Sources, file+" (stub)") {
			t.Errorf("stub %s evidence = %v, want %s cited as a stub", section, ev.Sources, file)
		}
	}
}
