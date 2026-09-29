package lifeboat

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// convTierFixture builds a plain (non-git) directory that exercises the Tier-1
// conventions signals: a README with real prose and a features section, a
// go.mod/go.sum manifest pair, an ADR under docs/adr, and a CHANGELOG. It
// deliberately carries no glossary, so that section is a genuine absent-material
// blank. Returns the directory.
func convTierFixture(t *testing.T) string {
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

	write("README.md", "# Widget\n\n"+
		"Widget is a small tool that turns sprockets into cogs. It is designed "+
		"for people who need cogs quickly and cannot afford a full foundry, and "+
		"it has been used in production by several teams.\n\n"+
		"## Features\n\n- Fast sprocket conversion\n- Deterministic output\n\n"+
		"## Usage\n\n```\nwidget convert in.spr\n```\n\n"+
		"## Non-goals\n\nWidget will never smelt raw ore.\n")
	write("go.mod", "module example.com/widget\n\ngo 1.22\n")
	write("go.sum", "example.com/dep v1.0.0 h1:abc=\n")
	write("docs/adr/0001-record-architecture-decisions.md", "# 1. Record ADRs\n\nContext.\n")
	write("CHANGELOG.md", "# Changelog\n\n## 1.0.0\n\n- Initial release\n")

	return dir
}

// convSourceForSection returns the Tier-1 adapter that speaks for section,
// failing the test if none does.
func convSourceForSection(t *testing.T, section Section) Source {
	t.Helper()
	for _, s := range conventionSources() {
		if s.Section() == section {
			return s
		}
	}
	t.Fatalf("no conventions source for section %s", section)
	return nil
}

// TestHasConventionsMatchesAdapterEvidence guards that the tier gate is not
// narrower than what the Tier-1 adapters read: a repo carrying build manifests
// and CI workflows but none of the headline docs (README/LICENSE/CHANGELOG/
// CONTRIBUTING) must still count as having conventions, so its platform and
// dependency adapters run instead of being skipped and reported as false blanks.
func TestHasConventionsMatchesAdapterEvidence(t *testing.T) {
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
	// Only manifest + lockfile + CI workflows — none of the headline doc sentinels.
	write("go.mod", "module example.com/tool\n\ngo 1.22\n")
	write("go.sum", "example.com/dep v1.0.0 h1:abc=\n")
	write(".github/workflows/ci.yml", "name: ci\n")

	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	if !hasConventions(ctx) {
		t.Fatal("hasConventions=false for a go.mod/go.sum/.github/workflows repo; Tier-1 adapters would be skipped and blanked")
	}
	present := tiersPresent(ctx)
	found := false
	for _, tr := range present {
		if tr == TierConventions {
			found = true
		}
	}
	if !found {
		t.Fatalf("tiersPresent=%v omits TierConventions for a manifest+CI repo", present)
	}

	// The gated adapter must actually ground, proving the tier was not skipped.
	if ev := convSourceForSection(t, "constraints/platform").Probe(ctx); ev.Status != StatusGrounded {
		t.Fatalf("constraints/platform status = %s, want grounded", ev.Status)
	}
}

// TestConvContextGroundsFromREADME is the flagship Tier-1 assertion: a README
// with real prose grounds "product/context" and cites the README file.
func TestConvContextGroundsFromREADME(t *testing.T) {
	ctx, err := newSourceContext(convTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	src := convSourceForSection(t, "product/context")
	if src.Tier() != TierConventions {
		t.Fatalf("product/context tier = %s, want %s", src.Tier(), TierConventions)
	}
	ev := src.Probe(ctx)
	if ev.Status != StatusGrounded {
		t.Fatalf("product/context status = %s, want grounded", ev.Status)
	}
	if len(ev.Sources) == 0 {
		t.Fatal("grounded product/context cites no evidence")
	}
	if ev.Sources[0] != "README.md" {
		t.Errorf("product/context evidence = %v, want README.md cited", ev.Sources)
	}
	if ev.Confidence == "" {
		t.Error("grounded product/context has no confidence")
	}
}

// TestConvPlatformGroundsFromManifests confirms build manifests ground
// "constraints/platform" with the manifest cited.
func TestConvPlatformGroundsFromManifests(t *testing.T) {
	ctx, err := newSourceContext(convTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "constraints/platform").Probe(ctx)
	if ev.Status != StatusGrounded {
		t.Fatalf("constraints/platform status = %s, want grounded", ev.Status)
	}
	if !containsSource(ev.Sources, "go.mod") {
		t.Errorf("constraints/platform evidence = %v, want go.mod cited", ev.Sources)
	}
}

// TestConvDependenciesGroundsFromPair confirms a manifest+lockfile pair grounds
// "constraints/dependencies" and cites both files.
func TestConvDependenciesGroundsFromPair(t *testing.T) {
	ctx, err := newSourceContext(convTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "constraints/dependencies").Probe(ctx)
	if ev.Status != StatusGrounded {
		t.Fatalf("constraints/dependencies status = %s, want grounded", ev.Status)
	}
	if !containsSource(ev.Sources, "go.mod") || !containsSource(ev.Sources, "go.sum") {
		t.Errorf("constraints/dependencies evidence = %v, want go.mod and go.sum", ev.Sources)
	}
}

// TestConvDependenciesGroundsFromPythonManifest is the fix for the manifest
// under-detection the M2 cross-repo probe exposed: a real Python project
// (pyproject.toml + uv.lock) was reported blank because the adapter only knew
// Go/Node/Rust/pip. It must now ground, citing both files.
func TestConvDependenciesGroundsFromPythonManifest(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"pyproject.toml": "[project]\nname = \"x\"\ndependencies = [\"requests\"]\n",
		"uv.lock":        "version = 1\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "constraints/dependencies").Probe(ctx)
	if ev.Status != StatusGrounded {
		t.Fatalf("python deps status = %s, want grounded", ev.Status)
	}
	if !containsSource(ev.Sources, "pyproject.toml") || !containsSource(ev.Sources, "uv.lock") {
		t.Errorf("python deps evidence = %v, want pyproject.toml and uv.lock", ev.Sources)
	}
}

// TestConvADRsGroundFromDocsAdr confirms ADRs under docs/adr ground "docs/adrs"
// and the citation lists the ADR file.
func TestConvADRsGroundFromDocsAdr(t *testing.T) {
	ctx, err := newSourceContext(convTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "docs/adrs").Probe(ctx)
	if ev.Status != StatusGrounded {
		t.Fatalf("docs/adrs status = %s, want grounded", ev.Status)
	}
	if !containsSource(ev.Sources, "docs/adr/0001-record-architecture-decisions.md") {
		t.Errorf("docs/adrs evidence = %v, want the ADR file listed", ev.Sources)
	}
}

// TestConvWhatWorkedPartialFromChangelog confirms a CHANGELOG partially grounds
// "evidence/what-worked" with the file cited.
func TestConvWhatWorkedPartialFromChangelog(t *testing.T) {
	ctx, err := newSourceContext(convTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "evidence/what-worked").Probe(ctx)
	if ev.Status != StatusPartial {
		t.Fatalf("evidence/what-worked status = %s, want partial", ev.Status)
	}
	if !containsSource(ev.Sources, "CHANGELOG.md") {
		t.Errorf("evidence/what-worked evidence = %v, want CHANGELOG.md", ev.Sources)
	}
}

// TestConvGlossaryBlankWhenAbsent holds the "a blank is a result" contract for
// Tier-1: with no glossary in the fixture, the section returns a blank that names
// what it searched and the question a human must answer.
func TestConvGlossaryBlankWhenAbsent(t *testing.T) {
	ctx, err := newSourceContext(convTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "glossary").Probe(ctx)
	if ev.Status != StatusBlank {
		t.Fatalf("glossary status = %s, want blank (no glossary in fixture)", ev.Status)
	}
	if ev.Question == "" {
		t.Error("blank glossary carries no question for a human")
	}
	if len(ev.Searched) == 0 {
		t.Error("blank glossary names nothing it searched")
	}
}

// TestConvOutOfScopePartialFromNonGoals confirms a README non-goals section
// partially grounds "delivery/out-of-scope".
func TestConvOutOfScopePartialFromNonGoals(t *testing.T) {
	ctx, err := newSourceContext(convTierFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "delivery/out-of-scope").Probe(ctx)
	if ev.Status != StatusPartial {
		t.Fatalf("delivery/out-of-scope status = %s, want partial", ev.Status)
	}
	if len(ev.Sources) == 0 {
		t.Fatal("partial delivery/out-of-scope cites no evidence")
	}
}

// TestConvAdaptersAllReportTierConventions guards the tier contract: every Tier-1
// adapter reports TierConventions.
func TestConvAdaptersAllReportTierConventions(t *testing.T) {
	for _, s := range conventionSources() {
		if s.Tier() != TierConventions {
			t.Errorf("source for %s reports tier %s, want %s", s.Section(), s.Tier(), TierConventions)
		}
	}
}

// TestConvProbeIsDeterministic asserts the Tier-1 adapters are byte-stable across
// runs against the same directory.
func TestConvProbeIsDeterministic(t *testing.T) {
	dir := convTierFixture(t)
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

	for _, s := range conventionSources() {
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

// convMarkerFixture builds the intent's motivating repository: no .abcd record,
// no git, but source files carrying the work markers the last team left behind.
// go.mod is deliberate — it is the conventions tier gate's sentinel, without
// which the whole Tier-1 set is skipped and every conventions section blanks.
func convMarkerFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":   "module example.com/wreck\n\ngo 1.22\n",
		"retry.go": "package wreck\n\nfunc retry() {\n\t// TODO: handle the retry case\n}\n",
		"store.go": "package wreck\n\n// FIXME(alice): this leaks a connection\nfunc store() {}\n",
	})
	return dir
}

// TestConvOpenQuestionsPartialFromWorkMarkers is the intent's headline
// behaviour: a record-less repository whose source is dense with TODO/FIXME no
// longer reports "no open questions on record" — the section comes back
// non-blank at the conventions tier, citing the file and line of each marker.
func TestConvOpenQuestionsPartialFromWorkMarkers(t *testing.T) {
	cov, err := Probe(convMarkerFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	sc := findSection(t, cov, "evidence/open-questions")
	if sc.Status != StatusPartial {
		t.Fatalf("evidence/open-questions status = %s, want partial", sc.Status)
	}
	if sc.Tier != TierConventions {
		t.Errorf("evidence/open-questions tier = %s, want %s", sc.Tier, TierConventions)
	}
	if !containsSource(sc.Evidence, "retry.go:4 (TODO)") {
		t.Errorf("evidence = %v, want the TODO cited at retry.go:4", sc.Evidence)
	}
	if !containsSource(sc.Evidence, "store.go:3 (FIXME)") {
		t.Errorf("evidence = %v, want the FIXME cited at store.go:3", sc.Evidence)
	}
	if sc.Confidence == "" {
		t.Error("non-blank evidence/open-questions carries no confidence")
	}
}

// TestConvOpenQuestionsCeilingIsPartial holds the status ceiling: markers are a
// thread, not a framed set of open questions, so no quantity of them ever
// grounds the section. Volume moves the confidence instead.
func TestConvOpenQuestionsCeilingIsPartial(t *testing.T) {
	dir := t.TempDir()
	var body strings.Builder
	body.WriteString("package wreck\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&body, "// TODO: unfinished thing %d\n", i)
	}
	writeTree(t, dir, map[string]string{
		"go.mod":  "module example.com/wreck\n\ngo 1.22\n",
		"many.go": body.String(),
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "evidence/open-questions").Probe(ctx)
	if ev.Status != StatusPartial {
		t.Fatalf("20 markers give status %s, want partial (grounded is not reachable)", ev.Status)
	}
	if ev.Confidence != ConfidenceMedium {
		t.Errorf("confidence = %s at 20 markers, want %s", ev.Confidence, ConfidenceMedium)
	}
}

// TestConvOpenQuestionsBlankWithoutMarkers holds the blank contract for the
// marker scan: a tree with no work markers is an honest blank naming the markers
// and the trees it searched, never a fabricated result.
func TestConvOpenQuestionsBlankWithoutMarkers(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":    "module example.com/tidy\n\ngo 1.22\n",
		"README.md": "# Tidy\n\nA project that left nothing unfinished.\n",
		"tidy.go":   "package tidy\n\n// NOTE: this explains, it does not ask.\nfunc tidy() {}\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "evidence/open-questions").Probe(ctx)
	if ev.Status != StatusBlank {
		t.Fatalf("marker-free tree gives status %s, want blank (evidence %v)", ev.Status, ev.Sources)
	}
	if len(ev.Searched) == 0 {
		t.Error("blank evidence/open-questions names nothing it searched")
	}
	if ev.Question == "" {
		t.Error("blank evidence/open-questions carries no question for a human")
	}
}

// TestConvOpenQuestionsIgnoresRedactionPlaceholders holds the honest-blank
// contract against the one shape that reads like a marker and is not one: the
// XXX-XXX-XXX redaction placeholder. Fabricated evidence is the failure a tool
// built around honest blanks cannot afford, so a tree whose only uppercase
// triples are redactions must still come back blank.
func TestConvOpenQuestionsIgnoresRedactionPlaceholders(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":            "module example.com/support\n\ngo 1.22\n",
		"docs/support.md":   "# Support\n\nCall us on XXX-XXX-XXX for details.\n",
		"docs/redacted.md":  "account: XXX-XXX-XXX\n",
		"docs/trailing.md":  "XXX-XXX-XXX\n",
		"docs/customers.md": "Reach the team on XXX-XXX-XXX\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "evidence/open-questions").Probe(ctx)
	if ev.Status != StatusBlank {
		t.Fatalf("redaction placeholders give status %s, want blank; fabricated evidence %v", ev.Status, ev.Sources)
	}
}

// TestConvOpenQuestionsIgnoresProseMentioningMarkers holds the iss-111 contract:
// a repository that documents its own conventions mentions TODO and FIXME by name
// in prose, and those mentions are not the team's open questions. A bare TODO or
// FIXME followed by whitespace reads exactly like such a mention, so the scan must
// come back blank rather than fabricate open questions out of the documentation.
func TestConvOpenQuestionsIgnoresProseMentioningMarkers(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod": "module example.com/documented\n\ngo 1.22\n",
		"CONTRIBUTING.md": "# Conventions\n\n" +
			"Leave a TODO for unfinished work and a FIXME where a bug hides.\n" +
			"We grep for TODO and FIXME markers before every release.\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "evidence/open-questions").Probe(ctx)
	if ev.Status != StatusBlank {
		t.Fatalf("prose that merely names markers gives status %s, want blank; fabricated evidence %v", ev.Status, ev.Sources)
	}
}

// TestConvMarkerRePinsTheRecognisedSpellings pins the marker pattern against the
// spellings it must accept and the near-misses it must reject, so the word
// boundary cannot be loosened or tightened without a test saying so.
func TestConvMarkerRePinsTheRecognisedSpellings(t *testing.T) {
	match := []string{
		"// TODO: handle the retry case",
		"//TODO: no space after the slashes",
		"- TODO: a list item",
		"FIXME(alice): leaks a connection",
		"-- BUG --",
		"// HACK around the driver",
		"XXX",
	}
	reject := []string{
		"redacted: XXX-XXX-XXX",
		"call XXX-XXX-XXX for details",
		"XXX-XXX-XXX",
		"TODOS are not markers",
		"todo_list := nil",
		// iss-111: TODO and FIXME require a trailing ':' or '('. A bare word is
		// indistinguishable from prose that merely names the marker — the
		// false-positive class that swamped a repo documenting its own
		// conventions — so these documentation mentions must not match.
		"# TODO",
		"* FIXME check this",
		"names TODO and FIXME markers as a read",
		"a codebase dense with TODO markers",
		"the TODO and FIXME conventions this project follows",
	}
	for _, line := range match {
		if convMarkerRe.FindStringSubmatch(line) == nil {
			t.Errorf("convMarkerRe rejects the marker line %q", line)
		}
	}
	for _, line := range reject {
		if m := convMarkerRe.FindStringSubmatch(line); m != nil {
			t.Errorf("convMarkerRe matches %q as a %s marker", line, convMarkerName(m))
		}
	}
}

// TestConvOpenQuestionsIgnoresSkippedTrees proves the adapter inherits the
// walk's skip set: a dependency's or a generator's TODO is not this team's open
// question, so it is never cited.
func TestConvOpenQuestionsIgnoresSkippedTrees(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":                    "module example.com/wreck\n\ngo 1.22\n",
		"keep.go":                   "package wreck\n\n// TODO: our own question\n",
		".git/hooks/pre-commit":     "# TODO: git internals\n",
		"node_modules/pkg/index.js": "// TODO: a dependency's marker\n",
		"vendor/dep/dep.go":         "// TODO: vendored\n",
		"generated/api.pb.go":       "// TODO: generated\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "evidence/open-questions").Probe(ctx)
	if ev.Status != StatusPartial {
		t.Fatalf("status = %s, want partial", ev.Status)
	}
	if !containsSource(ev.Sources, "keep.go:3 (TODO)") {
		t.Errorf("evidence = %v, want keep.go:3 cited", ev.Sources)
	}
	for _, s := range ev.Sources {
		for _, skipped := range walkSkipDirs {
			if strings.Contains(s, skipped+"/") {
				t.Errorf("evidence cites %q from the skipped %s/ tree", s, skipped)
			}
		}
	}
}

// TestConvOpenQuestionsStopsAtTheScanBudget proves the scan is bounded in bytes,
// not only in files: the per-file cap and the walk cap multiply, so without an
// aggregate budget a hostile tree of large files holds the probe for hours. The
// scan must stop when the budget is spent and say so in its own evidence, so a
// rescuer never mistakes a partial count for the whole tree. The branch is
// exercised through the same code path at an affordable scale — the shipped
// budget stays a const, so concurrent adapters share no mutable state.
func TestConvOpenQuestionsStopsAtTheScanBudget(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{"go.mod": "module example.com/vast\n\ngo 1.22\n"}
	for i := 0; i < 6; i++ {
		files[fmt.Sprintf("f%d.go", i)] = strings.Repeat("x", 1000) + "\n// TODO: unfinished\n"
	}
	writeTree(t, dir, files)
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	src := convSourceForSection(t, "evidence/open-questions").(convOpenQuestionsSource)
	full := src.probeLimited(ctx, maxMarkerScanBytes)
	if !containsSource(full.Sources, "6 work marker(s) across 6 file(s)") {
		t.Fatalf("unbudgeted scan evidence = %v, want all 6 markers counted", full.Sources)
	}

	// A budget of two files' worth stops the scan partway through the walk.
	capped := src.probeLimited(ctx, 2100)
	if containsSource(capped.Sources, "6 work marker(s) across 6 file(s)") {
		t.Errorf("scan ran past its 2100-byte budget: %v", capped.Sources)
	}
	said := false
	for _, s := range capped.Sources {
		if strings.Contains(s, "read budget") {
			said = true
		}
	}
	if !said {
		t.Errorf("budget-exhausted scan does not report it in its evidence: %v", capped.Sources)
	}
}

// TestConvOpenQuestionsBlankAdmitsAPartialScan closes the honesty gap the scan
// bounds open up: when the budget or the walk cap stops the scan before the end
// of the tree and nothing was found in the part that was read, "its source
// carries no work markers" is a claim about files the adapter never opened. A
// blank is a first-class result (adr-35) precisely because it is trustworthy, so
// a blank drawn from a partial read must say the read was partial.
func TestConvOpenQuestionsBlankAdmitsAPartialScan(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{}
	// Marker-free files sort first, so a small budget is spent on them and the
	// one marker-bearing file is never reached.
	for i := 0; i < 4; i++ {
		files[fmt.Sprintf("a%d.go", i)] = strings.Repeat("x", 1000) + "\n"
	}
	files["z.go"] = "// TODO: the marker the scan never reaches\n"
	writeTree(t, dir, files)
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	src := convSourceForSection(t, "evidence/open-questions").(convOpenQuestionsSource)
	capped := src.probeLimited(ctx, 2100)
	if capped.Status != StatusBlank {
		t.Fatalf("budget-exhausted marker-free prefix = %s, want blank", capped.Status)
	}
	if strings.Contains(capped.Question, "carries no work markers") {
		t.Errorf("blank claims the whole tree is marker-free after a partial scan: %q", capped.Question)
	}
	said := false
	for _, s := range capped.Searched {
		if strings.Contains(s, "read budget") || strings.Contains(s, "walk cap") {
			said = true
		}
	}
	if !said {
		t.Errorf("blank from a partial scan does not say the scan was partial: searched = %v", capped.Searched)
	}

	// The unbudgeted scan over the same tree still finds the marker, so the
	// fixture proves truncation and not an unrelated miss.
	if full := src.probeLimited(ctx, maxMarkerScanBytes); full.Status != StatusPartial {
		t.Fatalf("unbudgeted scan = %s, want partial (z.go carries a TODO)", full.Status)
	}
}

// convArchitectureProse is an ARCHITECTURE.md body comfortably above the
// convGroundedProseBytes threshold — real architecture prose rather than a stub.
const convArchitectureProse = "# Architecture\n\n" +
	"The wreck is split into a transport-agnostic core and a thin CLI shell. " +
	"The core owns every decision; the shell only formats what the core returns, " +
	"so a second front door costs nothing but its formatter.\n"

// TestConvNamingPartialFromNamingDoc is the intent's headline naming behaviour: a
// record-less repository carrying a dedicated NAMING.md no longer blanks
// "constraints/naming" — the section comes back non-blank at the conventions
// tier, citing the file it read.
func TestConvNamingPartialFromNamingDoc(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":    "module example.com/wreck\n\ngo 1.22\n",
		"NAMING.md": "# Naming\n\nA voyage is never called a run.\n",
	})
	cov, err := Probe(dir)
	if err != nil {
		t.Fatal(err)
	}
	sc := findSection(t, cov, "constraints/naming")
	if sc.Status != StatusPartial {
		t.Fatalf("constraints/naming status = %s, want partial", sc.Status)
	}
	if sc.Tier != TierConventions {
		t.Errorf("constraints/naming tier = %s, want %s", sc.Tier, TierConventions)
	}
	if !containsSource(sc.Evidence, "NAMING.md") {
		t.Errorf("evidence = %v, want NAMING.md cited", sc.Evidence)
	}
	if sc.Confidence != ConfidenceMedium {
		t.Errorf("confidence = %s for a dedicated naming document, want %s", sc.Confidence, ConfidenceMedium)
	}
}

// TestConvNamingPartialFromDocsNamingPage confirms the docs/ prefix idiom: a
// naming page under docs/ grounds the section when no root NAMING.md exists.
func TestConvNamingPartialFromDocsNamingPage(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":                     "module example.com/wreck\n\ngo 1.22\n",
		"docs/naming-conventions.md": "# Naming conventions\n\nPackages are singular nouns.\n",
		"docs/unrelated-appendix.md": "# Appendix\n\nNothing about naming.\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "constraints/naming").Probe(ctx)
	if ev.Status != StatusPartial {
		t.Fatalf("constraints/naming status = %s, want partial", ev.Status)
	}
	if !containsSource(ev.Sources, "docs/naming-conventions.md") {
		t.Errorf("evidence = %v, want the docs/ naming page cited", ev.Sources)
	}
	if ev.Confidence != ConfidenceMedium {
		t.Errorf("confidence = %s for a docs/ naming page, want %s", ev.Confidence, ConfidenceMedium)
	}
}

// TestConvNamingFallsBackToGlossaryWithoutDisplacingIt holds the distinctness
// contract on the fixture that could collapse the two sections into one: a
// repository whose only vocabulary signal is a GLOSSARY.md. The glossary section
// stays exactly what convGlossarySource already made it, and naming is a visibly
// weaker derived reading — lower confidence, and a citation qualified as the
// fallback it is — never a duplicate row.
func TestConvNamingFallsBackToGlossaryWithoutDisplacingIt(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"GLOSSARY.md": "# Glossary\n\n**Voyage** — one packed lifeboat.\n",
	})
	cov, err := Probe(dir)
	if err != nil {
		t.Fatal(err)
	}

	naming := findSection(t, cov, "constraints/naming")
	if naming.Status != StatusPartial {
		t.Fatalf("constraints/naming status = %s, want partial", naming.Status)
	}
	if naming.Confidence != ConfidenceLow {
		t.Errorf("naming confidence = %s on a glossary fallback, want %s", naming.Confidence, ConfidenceLow)
	}
	if len(naming.Evidence) != 1 || !strings.Contains(naming.Evidence[0], "GLOSSARY.md") {
		t.Fatalf("naming evidence = %v, want the glossary cited", naming.Evidence)
	}
	if !strings.Contains(naming.Evidence[0], "glossary fallback") {
		t.Errorf("naming cites %q without the glossary-fallback qualifier", naming.Evidence[0])
	}

	glossary := findSection(t, cov, "glossary")
	if glossary.Status != StatusPartial || glossary.Confidence != ConfidenceMedium {
		t.Fatalf("glossary = %s/%s, want partial/medium (convGlossarySource unchanged)",
			glossary.Status, glossary.Confidence)
	}
	if !containsSource(glossary.Evidence, "GLOSSARY.md") {
		t.Errorf("glossary evidence = %v, want GLOSSARY.md cited bare", glossary.Evidence)
	}

	// The two rows must differ in both dimensions: same file, different reading.
	if naming.Confidence == glossary.Confidence {
		t.Errorf("naming and glossary share confidence %s; naming must be the weaker reading", naming.Confidence)
	}
	if naming.Evidence[0] == glossary.Evidence[0] {
		t.Errorf("naming and glossary cite the identical string %q; naming must be visibly derived", naming.Evidence[0])
	}
}

// TestConventionSourcesHaveOneAdapterPerSection guards the registry against a
// duplicate adapter: two adapters for one section race for the same coverage row
// and the loser's evidence silently vanishes. It pins the sections this intent
// touches — including glossary, which keeps its single pre-existing adapter.
func TestConventionSourcesHaveOneAdapterPerSection(t *testing.T) {
	count := map[Section]int{}
	for _, s := range conventionSources() {
		count[s.Section()]++
	}
	for _, section := range []Section{
		"glossary", "constraints/naming", "internals", "evidence/open-questions",
	} {
		if count[section] != 1 {
			t.Errorf("conventionSources has %d adapters for %s, want exactly 1", count[section], section)
		}
	}
	for section, n := range count {
		if n > 1 {
			t.Errorf("conventionSources has %d adapters for %s, want at most 1", n, section)
		}
	}
}

// TestConvInternalsCitesArchitectureAndLayout is the intent's headline internals
// behaviour: a record-less repository with an ARCHITECTURE.md and a package tree
// grounds "internals" from both signals at once, citing the document and the
// packages a rescuer must navigate.
func TestConvInternalsCitesArchitectureAndLayout(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":                       "module example.com/wreck\n\ngo 1.22\n",
		"ARCHITECTURE.md":              convArchitectureProse,
		"internal/core/voyage.go":      "package core\n",
		"internal/surface/cli/main.go": "package cli\n",
		"cmd/wreck/main.go":            "package main\n",
	})
	cov, err := Probe(dir)
	if err != nil {
		t.Fatal(err)
	}
	sc := findSection(t, cov, "internals")
	if sc.Status != StatusPartial {
		t.Fatalf("internals status = %s, want partial", sc.Status)
	}
	if sc.Tier != TierConventions {
		t.Errorf("internals tier = %s, want %s", sc.Tier, TierConventions)
	}
	if sc.Confidence != ConfidenceHigh {
		t.Errorf("internals confidence = %s with real architecture prose, want %s", sc.Confidence, ConfidenceHigh)
	}
	if !containsSource(sc.Evidence, "ARCHITECTURE.md") {
		t.Errorf("evidence = %v, want ARCHITECTURE.md cited", sc.Evidence)
	}
	for _, want := range []string{"internal/core/", "internal/surface/", "cmd/wreck/"} {
		if !containsSource(sc.Evidence, want) {
			t.Errorf("evidence = %v, want the layout entry %s cited", sc.Evidence, want)
		}
	}
}

// TestConvInternalsLayoutOnlyIsLowConfidence holds the weakest internals signal:
// a package tree with no architecture document still says something about the
// shape of the system, at low confidence and citing the layout alone. The fixture
// carries no other conventional sentinel, so it also proves the tier gate admits
// a repository whose only Tier-1 signal is its layout.
func TestConvInternalsLayoutOnlyIsLowConfidence(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"internal/store/store.go": "package store\n",
		"pkg/api/api.go":          "package api\n",
	})
	cov, err := Probe(dir)
	if err != nil {
		t.Fatal(err)
	}
	sc := findSection(t, cov, "internals")
	if sc.Status != StatusPartial {
		t.Fatalf("internals status = %s, want partial (evidence %v)", sc.Status, sc.Evidence)
	}
	if sc.Confidence != ConfidenceLow {
		t.Errorf("internals confidence = %s with layout only, want %s", sc.Confidence, ConfidenceLow)
	}
	for _, want := range []string{"internal/store/", "pkg/api/"} {
		if !containsSource(sc.Evidence, want) {
			t.Errorf("evidence = %v, want the layout entry %s cited", sc.Evidence, want)
		}
	}
	for _, got := range sc.Evidence {
		if strings.Contains(strings.ToLower(got), "architecture") {
			t.Errorf("layout-only internals cites %q; no architecture document exists", got)
		}
	}
}

// TestConvInternalsThinArchitectureIsMedium separates a written architecture
// chapter from a placeholder: a document below the prose threshold is still
// evidence, but only at medium confidence.
func TestConvInternalsThinArchitectureIsMedium(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":          "module example.com/wreck\n\ngo 1.22\n",
		"ARCHITECTURE.md": "# Architecture\n\nTBD.\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "internals").Probe(ctx)
	if ev.Status != StatusPartial {
		t.Fatalf("internals status = %s, want partial", ev.Status)
	}
	if ev.Confidence != ConfidenceMedium {
		t.Errorf("internals confidence = %s for a thin architecture doc, want %s", ev.Confidence, ConfidenceMedium)
	}
	if len(ev.Sources) == 0 {
		t.Fatal("partial internals cites no evidence")
	}
}

// TestConvInternalsMediumFromArchitectureDirectory confirms the Diataxis
// fallback: a docs/explanation tree holding Markdown is architecture prose spread
// across files, and counts at medium confidence.
func TestConvInternalsMediumFromArchitectureDirectory(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":                     "module example.com/wreck\n\ngo 1.22\n",
		"docs/explanation/model.md":  "# The model\n\nHow the pieces fit.\n",
		"docs/explanation/notes.txt": "not markdown\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	ev := convSourceForSection(t, "internals").Probe(ctx)
	if ev.Status != StatusPartial {
		t.Fatalf("internals status = %s, want partial", ev.Status)
	}
	if ev.Confidence != ConfidenceMedium {
		t.Errorf("internals confidence = %s for a doc directory, want %s", ev.Confidence, ConfidenceMedium)
	}
	if !containsSource(ev.Sources, "docs/explanation/") {
		t.Errorf("evidence = %v, want the docs/explanation/ tree cited", ev.Sources)
	}
}

// TestConvInternalsBoundsItsLayoutCitations holds both layout bounds at once: the
// citation cap keeps a vast monorepo from dumping every package into a section
// while the count stays truthful, and a walk cut short by the file cap says so in
// its own evidence rather than reading as a complete survey.
func TestConvInternalsBoundsItsLayoutCitations(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{"go.mod": "module example.com/vast\n\ngo 1.22\n"}
	for i := 0; i < maxLayoutCitations+5; i++ {
		files[fmt.Sprintf("internal/p%03d/p.go", i)] = "package p\n"
	}
	writeTree(t, dir, files)
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	src := convSourceForSection(t, "internals").(convInternalsSource)
	full := src.probeLimited(ctx, maxWalkFiles)
	cited := 0
	for _, s := range full.Sources {
		if strings.HasPrefix(s, "internal/p") {
			cited++
		}
	}
	if cited != maxLayoutCitations {
		t.Errorf("cited %d layout entries, want the cap of %d", cited, maxLayoutCitations)
	}
	if !containsSource(full.Sources, fmt.Sprintf("5 further package(s) counted but not cited (citation cap %d)", maxLayoutCitations)) {
		t.Errorf("evidence = %v, want the 5 uncited packages reported as a count", full.Sources)
	}

	// A walk cut short must be visible in the evidence, exactly as the marker
	// scan's truncation is.
	capped := src.probeLimited(ctx, 3)
	said := false
	for _, s := range capped.Sources {
		if strings.Contains(s, "walk cap") {
			said = true
		}
	}
	if !said {
		t.Errorf("truncated layout scan does not report it in its evidence: %v", capped.Sources)
	}
}

// TestConvInternalsStagesAScanThatReachedNoPackage holds the loud staging where
// it is easiest to lose: a walk the cap cut short before it ever reached a
// source root found no packages at all, so both the blank and the
// architecture-only reading would otherwise describe a tree the scan never
// finished reading. A blank is a first-class result only while it is trustworthy
// (adr-35), so it must say the layout scan stopped early — and so must the
// non-blank reading beside it.
func TestConvInternalsStagesAScanThatReachedNoPackage(t *testing.T) {
	// assets/ sorts before every source root, so a cap of ten files is spent
	// before the walk reaches src/ — exactly the shape of a repository whose
	// code sits behind a large data or frontend tree.
	bulk := func(extra map[string]string) map[string]string {
		files := map[string]string{"src/pkg/code.go": "package pkg\n"}
		for i := 0; i < 40; i++ {
			files[fmt.Sprintf("assets/a%02d.bin", i)] = "x\n"
		}
		for rel, content := range extra {
			files[rel] = content
		}
		return files
	}

	t.Run("blank", func(t *testing.T) {
		dir := t.TempDir()
		writeTree(t, dir, bulk(nil))
		ctx, err := newSourceContext(dir)
		if err != nil {
			t.Fatal(err)
		}
		defer ctx.Close()

		ev := convSourceForSection(t, "internals").(convInternalsSource).probeLimited(ctx, 10)
		if ev.Status != StatusBlank {
			t.Fatalf("internals status = %s, want blank (evidence %v)", ev.Status, ev.Sources)
		}
		said := false
		for _, s := range ev.Searched {
			if strings.Contains(s, "walk cap") {
				said = true
			}
		}
		if !said {
			t.Errorf("blank internals searched %v without admitting the walk cap cut the layout scan short", ev.Searched)
		}
		if !strings.Contains(ev.Question, "reached") {
			t.Errorf("blank internals asks %q, claiming no layout in a tree it did not finish reading", ev.Question)
		}
	})

	t.Run("architecture only", func(t *testing.T) {
		dir := t.TempDir()
		writeTree(t, dir, bulk(map[string]string{"ARCHITECTURE.md": convArchitectureProse}))
		ctx, err := newSourceContext(dir)
		if err != nil {
			t.Fatal(err)
		}
		defer ctx.Close()

		ev := convSourceForSection(t, "internals").(convInternalsSource).probeLimited(ctx, 10)
		if ev.Status != StatusPartial {
			t.Fatalf("internals status = %s, want partial", ev.Status)
		}
		said := false
		for _, s := range ev.Sources {
			if strings.Contains(s, "walk cap") {
				said = true
			}
		}
		if !said {
			t.Errorf("evidence = %v, want the truncated layout scan reported even though it found no package", ev.Sources)
		}
	})
}

// TestConvNamingAndInternalsBlankWithoutSignals holds the "a blank is a result"
// contract for both new adapters: a repository with neither naming nor
// architecture documentation and no recognisable layout returns honest blanks
// naming what was searched and the question a human must answer — never a
// fabricated section.
func TestConvNamingAndInternalsBlankWithoutSignals(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"README.md": "# Bare\n\nA project that documented nothing else.\n",
		"main.go":   "package main\n\nfunc main() {}\n",
	})
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()

	for _, section := range []Section{"constraints/naming", "internals"} {
		ev := convSourceForSection(t, section).Probe(ctx)
		if ev.Status != StatusBlank {
			t.Fatalf("%s status = %s, want blank (evidence %v)", section, ev.Status, ev.Sources)
		}
		if len(ev.Searched) == 0 {
			t.Errorf("blank %s names nothing it searched", section)
		}
		if ev.Question == "" {
			t.Errorf("blank %s carries no question for a human", section)
		}
	}
}

// TestHasConventionsCoversNamingAndArchitecture is the tier-gate regression the
// new adapters open up: the gate skips every adapter of an absent tier, so a
// repository whose only conventional signal is a NAMING.md or an ARCHITECTURE.md
// would have the whole Tier-1 set skipped and its section blanked falsely.
func TestHasConventionsCoversNamingAndArchitecture(t *testing.T) {
	for _, tc := range []struct {
		file    string
		content string
		section Section
	}{
		{"NAMING.md", "# Naming\n\nA voyage is never called a run.\n", "constraints/naming"},
		{"ARCHITECTURE.md", convArchitectureProse, "internals"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			dir := t.TempDir()
			writeTree(t, dir, map[string]string{tc.file: tc.content})
			cov, err := Probe(dir)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, tr := range cov.TiersPresent {
				if tr == TierConventions {
					found = true
				}
			}
			if !found {
				t.Fatalf("tiers_present = %v omits TierConventions for a %s-only repo", cov.TiersPresent, tc.file)
			}
			sc := findSection(t, cov, tc.section)
			if sc.Status == StatusBlank {
				t.Errorf("%s blanked in a %s-only repo; the tier gate skipped its adapter", tc.section, tc.file)
			}
		})
	}
}

// containsSource reports whether want appears in sources.
func containsSource(sources []string, want string) bool {
	for _, s := range sources {
		if s == want {
			return true
		}
	}
	return false
}

// TestHasConventionsReadsTheGithubContributionGuide: the forge reads a
// repository's contribution guide from `.github/` as readily as from the root,
// so a repository whose only convention document is `.github/CONTRIBUTING.md`
// still carries the conventions tier (iss-2608270540523859).
func TestHasConventionsReadsTheGithubContributionGuide(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, ".github", "CONTRIBUTING.md")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("# Contributing\n\nOpen a pull request.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, err := newSourceContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	if !hasConventions(ctx) {
		t.Fatal("hasConventions=false for a repository whose contribution guide is in .github/")
	}
}
