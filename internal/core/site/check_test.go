package site

// Fixture-driven tests for `abcd site check`.
//
// Each test renders a small but REAL repository, then breaks exactly one thing
// and asserts the gate that owns it says so. The break is applied to the
// rendered OUTPUT wherever the failure is a property of the output — an
// unsourced node, a smuggled token, a missing viewport, an unwrapped table —
// because that is where a published page's faults actually live, and because a
// check that can only be driven through the composer is a check that cannot
// catch the composer.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checkRepo is a minimal repository the site can be rendered from.
type checkRepo struct {
	t    *testing.T
	root string
	out  string
}

func newCheckRepo(t *testing.T) *checkRepo {
	t.Helper()
	r := &checkRepo{t: t, root: t.TempDir(), out: filepath.Join(t.TempDir(), "out")}
	for rel, body := range checkFixtureFiles {
		r.write(rel, body)
	}
	r.writeBytes("docs/assets/img/mark.png", fixturePNG(200, 120))
	// The interface-string allowlist is THIS repository's own. It is a closed
	// struct decoded with unknown fields refused, so a hand-written fixture copy
	// would fail the moment a field is added — and it would fail in this file,
	// which is not where the field was added. Reading the committed one keeps the
	// fixture honest about what the generator may say.
	r.write("site-src/ui.json", readFile(t, filepath.Join(repoRoot(), "site-src", "ui.json")))
	return r
}

// fixturePNG is a file the raster pipeline reads a size out of. Only the
// signature and the IHDR header matter to the build, and inventing a valid
// image would be inventing a fact the test does not use.
func fixturePNG(w, h int) []byte {
	b := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR")
	for _, v := range []int{w, h} {
		b = append(b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
	}
	return append(b, 8, 6, 0, 0, 0)
}

func (r *checkRepo) writeBytes(rel string, body []byte) {
	r.t.Helper()
	dest := filepath.Join(r.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(dest, body, 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *checkRepo) write(rel, body string) {
	r.t.Helper()
	dest := filepath.Join(r.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte(body), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// build renders the fixture, failing the test on a build error.
func (r *checkRepo) build() {
	r.t.Helper()
	if _, err := Build(Request{RepoRoot: r.root, OutDir: r.out,
		Stamp: BuildStamp{Version: "0.1.0", Commit: "abc1234", GeneratedAt: "2026-01-02"}}); err != nil {
		r.t.Fatalf("Build: %v", err)
	}
}

// breakOutput rewrites the rendered landing page, replacing old with new.
func (r *checkRepo) breakOutput(old, new string) {
	r.t.Helper()
	page := filepath.Join(r.out, "index.html")
	data, err := os.ReadFile(page)
	if err != nil {
		r.t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		r.t.Fatalf("the rendered page carries no %q to break", old)
	}
	if err := os.WriteFile(page, []byte(strings.Replace(string(data), old, new, 1)), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// check runs the gates over the already-built output.
func (r *checkRepo) check() CheckResult {
	r.t.Helper()
	res, err := Check(CheckRequest{RepoRoot: r.root, OutDir: r.out})
	if err != nil {
		r.t.Fatalf("Check: %v", err)
	}
	return res
}

// findingsFor returns the findings one gate raised.
func findingsFor(res CheckResult, check string) []CheckFinding {
	var out []CheckFinding
	for _, f := range res.Findings {
		if f.Check == check {
			out = append(out, f)
		}
	}
	return out
}

// wantFinding asserts one gate failed, with a detail naming want.
func wantFinding(t *testing.T, res CheckResult, check, want string) {
	t.Helper()
	got := findingsFor(res, check)
	if len(got) == 0 {
		t.Fatalf("%s reported nothing; want a finding naming %q. All findings: %+v", check, want, res.Findings)
	}
	for _, f := range got {
		if strings.Contains(f.Detail, want) || strings.Contains(f.Where, want) || strings.Contains(f.Source, want) {
			return
		}
	}
	t.Fatalf("%s findings do not name %q: %+v", check, want, got)
}

// wantClean asserts one gate found nothing.
func wantClean(t *testing.T, res CheckResult, check string) {
	t.Helper()
	if got := findingsFor(res, check); len(got) > 0 {
		t.Fatalf("%s reported %+v; want none", check, got)
	}
}

// TestCheckPassesAWellFormedRender is the baseline every other test breaks: a
// repository composed under the single-source rule passes every gate.
func TestCheckPassesAWellFormedRender(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	res := r.check()
	if !res.OK() {
		t.Fatalf("a well-formed render failed: %+v", res.Findings)
	}
	// adr-47 decision 3's scope, asserted rather than assumed: the landing page
	// and the explorer pages that render a span the manifest selects are
	// composed surfaces; the verbatim record rendering is not.
	if !containsString(res.Composed, "index.html") || !containsString(res.Composed, "contributors/index.html") {
		t.Errorf("composed surfaces = %v, want the landing and contributors pages", res.Composed)
	}
	for _, p := range res.Composed {
		if strings.HasPrefix(p, "record/") {
			t.Errorf("composed surfaces include the exempt record rendering: %q", p)
		}
	}
	if len(res.Pages) <= len(res.Composed) {
		t.Errorf("pages = %d, composed = %d; the record rendering should be read for the mobile gate",
			len(res.Pages), len(res.Composed))
	}
	if len(res.Checks) != 8 {
		t.Errorf("checks run = %v, want eight", res.Checks)
	}
}

// TestCheckPassesAPreviewRender is the constraint that keeps preview mode
// honest without loosening anything.
//
// The stamp is the one place the build writes words that are not a span of a
// repository file, so a new word there is exactly the kind of thing the
// provenance gate exists to catch. "unreleased" passes because it is a ui.json
// interface string, and the short sha passes because a commit is already
// something the generator may add — the same two routes every other stamp word
// takes. Nothing about the gate changes for a preview, and this test is what
// says so: if a future stamp rendered a word from neither source, this fails.
func TestCheckPassesAPreviewRender(t *testing.T) {
	r := newCheckRepo(t)
	if _, err := Build(Request{RepoRoot: r.root, OutDir: r.out,
		Stamp: BuildStamp{Commit: "abc1234", GeneratedAt: "2026-01-02", Preview: true}}); err != nil {
		t.Fatalf("Build: %v", err)
	}

	res := r.check()
	if !res.OK() {
		t.Fatalf("a preview render failed the gates: %+v", res.Findings)
	}
	wantClean(t, res, CheckProvenance)

	// And it really is the preview that was checked, not a release build.
	page, err := os.ReadFile(filepath.Join(r.out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "unreleased") {
		t.Error("the checked render is not a preview")
	}
}

// TestCheckBuildsWhenTheOutputIsAbsent asserts the verb answers the same
// question for a caller who has not rendered yet.
func TestCheckBuildsWhenTheOutputIsAbsent(t *testing.T) {
	r := newCheckRepo(t)
	res := r.check()
	if !res.Built {
		t.Fatal("Check did not render the absent output directory")
	}
	if len(res.Pages) == 0 {
		t.Fatal("Check rendered nothing to read")
	}
}

// --- 1. provenance ---------------------------------------------------------

// TestCheckRefusesAnUnsourcedTextNode is itd-135 AC 2: a visible word in no
// data-src element and in no allowlisted category fails, naming the node.
func TestCheckRefusesAnUnsourcedTextNode(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`<footer class="site-foot">`, `<footer class="site-foot"><p>Trusted by dozens of teams.</p>`)
	wantFinding(t, r.check(), CheckProvenance, "Trusted by dozens of teams.")
}

// TestCheckRefusesADataSrcNamingAMissingFile asserts a provenance attribute is
// RESOLVED, not merely present: an attribute naming nothing is a claim nobody
// can follow.
func TestCheckRefusesADataSrcNamingAMissingFile(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`data-src="docs/explanation/roles.md#roles"`, `data-src="docs/explanation/gone.md#roles"`)
	wantFinding(t, r.check(), CheckProvenance, "docs/explanation/gone.md")
}

// TestCheckRefusesADataSrcNamingAMissingAnchor is the same claim one level down.
func TestCheckRefusesADataSrcNamingAMissingAnchor(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`data-src="docs/explanation/roles.md#roles"`, `data-src="docs/explanation/roles.md#not-a-heading"`)
	wantFinding(t, r.check(), CheckProvenance, "#not-a-heading")
}

// TestCheckReadsWordsInEveryScript asserts the walk's claim is about every
// visible word, not about Latin script: a node with no ASCII letters is still a
// node full of words.
func TestCheckReadsWordsInEveryScript(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`<footer class="site-foot">`, `<footer class="site-foot"><p>Мы лучшие в мире</p>`)
	wantFinding(t, r.check(), CheckProvenance, "Мы лучшие в мире")
}

// TestGeneratorWordsReadNamesAndCounts pins the vocabulary's edges directly,
// because each of them arrived as a page reporting text it had every right to
// print. A dotfile path keeps its leading dot; a proportion is a count; a
// separator between two counts is punctuation.
func TestGeneratorWordsReadNamesAndCounts(t *testing.T) {
	g := generatorWords{
		phrases: map[string]bool{}, names: map[string]bool{}, attribution: map[string]bool{},
		exists: func(s string) bool { return s == ".abcd/site.json" },
	}
	for _, ok := range []string{".abcd/site.json", "70%", "976 / 1377", "v0.6.1", "2026-08-22", "abc1234"} {
		if !g.covers(ok) {
			t.Errorf("covers(%q) = false; the generator may print it", ok)
		}
	}
	for _, bad := range []string{"Trusted by dozens of teams.", ".abcd/absent.json", "Мы лучшие"} {
		if g.covers(bad) {
			t.Errorf("covers(%q) = true; nothing entitles the generator to say it", bad)
		}
	}
}

// TestCheckAccountsForAPageThatNamesItself covers the explorer's title shape:
// a page that names itself reads `<page> · <project>`, and BOTH halves are
// held up — the tail against the Identity block, the head against what the
// generator may add or what the page renders from a named source.
func TestCheckAccountsForAPageThatNamesItself(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	page := filepath.Join(r.out, "contributors", "index.html")
	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<title>Contributors · Probe — the fixture project</title>") {
		t.Fatalf("the contributors page does not carry the expected title shape")
	}
	wantClean(t, r.check(), CheckProvenance)

	// A page name that is neither an interface string nor anything the page
	// sources fails, and so does a tail that is not the Identity title.
	for _, bad := range []struct{ title, want string }{
		{"<title>The best tool ever · Probe — the fixture project</title>", "The best tool ever"},
		{"<title>Contributors · Probe, the fastest tool alive</title>", "Identity block's title"},
	} {
		broken := strings.Replace(string(data),
			"<title>Contributors · Probe — the fixture project</title>", bad.title, 1)
		if err := os.WriteFile(page, []byte(broken), 0o644); err != nil {
			t.Fatal(err)
		}
		wantFinding(t, r.check(), CheckProvenance, bad.want)
	}
}

// TestCheckReadsARootAbsoluteStylesheet is the resolution the explorer depends
// on: the shared stylesheet is linked from every depth the site has, so a
// root-absolute href resolves against the SERVED ROOT. Resolving it per-page
// would look for one stylesheet in as many places as there are routes, find it
// in none, and report every page as unstyled.
func TestCheckReadsARootAbsoluteStylesheet(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	deep := filepath.Join(r.out, "record", "adr", "adr-1", "index.html")
	data, err := os.ReadFile(deep)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `href="/site.css"`) {
		t.Fatalf("the deep record page does not link the stylesheet root-absolutely")
	}
	wantClean(t, r.check(), CheckMobile)
}

// TestCheckRefusesADataSrcThatEscapesTheRepository asserts the check never
// follows a rendered attribute out of the tree: the path is held to a
// repo-relative shape before anything opens it.
func TestCheckRefusesADataSrcThatEscapesTheRepository(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`data-src="docs/explanation/roles.md#roles"`, `data-src="../../../etc/passwd"`)
	wantFinding(t, r.check(), CheckProvenance, "not a repo-relative path")
}

// TestCheckVerifiesTheTitleAgainstIdentity covers the documented special case:
// `<title>` carries Identity text with no attribute to name it, so it is
// checked against the block rather than skipped (brief 04-surfaces/22-site.md).
func TestCheckVerifiesTheTitleAgainstIdentity(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput("<title>Probe — the fixture project</title>", "<title>Probe — the best tool ever</title>")
	wantFinding(t, r.check(), CheckProvenance, "<title>")
}

// TestCheckStillReadsTheTitleWithoutAnIdentityBlock asserts the documented
// exception has no hole in it: with no block to check against, the title falls
// back to the generator's own vocabulary rather than going unread.
func TestCheckStillReadsTheTitleWithoutAnIdentityBlock(t *testing.T) {
	r := newCheckRepo(t)
	r.write(".abcd/development/brief/01-product/README.md", "# Product\n\nNo identity block here.\n")
	r.build()
	res := r.check()
	// The composer falls back to the package name, which the vocabulary covers.
	wantClean(t, res, CheckProvenance)
	r.breakOutput("<title>probe</title>", "<title>The best tool ever</title>")
	wantFinding(t, r.check(), CheckProvenance, "no Identity block")
}

// TestCheckRefusesAnUnparseablePage asserts the tokenizer fails loudly rather
// than skipping what it cannot read.
func TestCheckRefusesAnUnparseablePage(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`<footer class="site-foot">`, `<footer class="site-foot"><!-- smuggled -->`)
	wantFinding(t, r.check(), CheckProvenance, "comment")
}

// TestCheckUnparseablePageDeniesEveryGateAPass proves a page that fails to parse
// cannot silently drop out of the other gates: each page-walking gate raises a
// finding naming the page rather than printing ok having examined nothing, so a
// real finding on that page cannot disappear behind the parse fault.
func TestCheckUnparseablePageDeniesEveryGateAPass(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`<footer class="site-foot">`, `<footer class="site-foot"><!-- smuggled -->`)
	res := r.check()
	for _, gate := range []string{CheckHero, CheckBannedTokens, CheckSnippets, CheckMobile, CheckFigureLabels} {
		wantFinding(t, res, gate, "index.html")
	}
}

// --- 2. hero-vs-identity ---------------------------------------------------

// TestCheckRefusesADriftedHero is itd-135 AC 1: the rendered hero is compared
// against the Identity block through the canonical parser.
func TestCheckRefusesADriftedHero(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(">A configuration layer for the fixture.</p>", ">The fastest configuration layer on earth.</p>")
	wantFinding(t, r.check(), CheckHero, "fastest configuration layer")
}

// TestCheckRefusesAHeroMissingItsPitch asserts an absent span is drift too: the
// block declares a pitch and the page renders none.
func TestCheckRefusesAHeroMissingItsPitch(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	page := filepath.Join(r.out, "index.html")
	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	i := strings.Index(string(data), `<p class="pitch"`)
	j := strings.Index(string(data)[i:], "</p>")
	if i < 0 || j < 0 {
		t.Fatal("the rendered hero carries no pitch to remove")
	}
	stripped := string(data)[:i] + string(data)[i+j+4:]
	if err := os.WriteFile(page, []byte(stripped), 0o644); err != nil {
		t.Fatal(err)
	}
	wantFinding(t, r.check(), CheckHero, "renders no .pitch")
}

// TestCheckRefusesAMissingHero asserts the gate catches the largest drift there
// is. A hero that reads the wrong words fails; a hero that is gone must not
// pass for having nothing left to compare.
func TestCheckRefusesAMissingHero(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	page := filepath.Join(r.out, "index.html")
	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	i := strings.Index(string(data), `<section class="hero">`)
	j := strings.Index(string(data), `<section class="chapter"`)
	if i < 0 || j < i {
		t.Fatal("the rendered page carries no hero to remove")
	}
	if err := os.WriteFile(page, []byte(string(data)[:i]+string(data)[j:]), 0o644); err != nil {
		t.Fatal(err)
	}
	wantFinding(t, r.check(), CheckHero, "renders no .hero")
}

// --- 3. banned tokens ------------------------------------------------------

// TestCheckRefusesABannedTokenInAComposedSpan is itd-135 AC 5: the docs-lint
// bans run over the text the site publishes, and a token smuggled into a
// composed span fails naming the rule.
func TestCheckRefusesABannedTokenInAComposedSpan(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput("Two roles carry the work.", "The flag was previously named --old.")
	res := r.check()
	wantFinding(t, res, CheckBannedTokens, "present_tense/previously")
	wantFinding(t, res, CheckBannedTokens, "docs/explanation/roles.md")
}

// TestCheckHonoursTheSourceSideAllowEscape asserts the escape survives the lift.
//
// The composed page cannot carry the escape — an HTML comment is not in the
// rendered markdown subset, and a reader would never see one — so it is read
// SOURCE-side. The chapter heading's span is the WHOLE page file, and that file
// carries a line where the token is declared legitimate, so the same words that
// fail inside a section-scoped span pass inside this one. The difference
// between this test and the one above it is the span, and nothing else.
func TestCheckHonoursTheSourceSideAllowEscape(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(">Roles</h2>", ">Roles, previously Positions</h2>")
	wantClean(t, r.check(), CheckBannedTokens)
}

// TestCheckDoesNotTreatAnInlineCodeSpanAsAFence asserts monospace is not an
// escape: the file walk masks fenced BLOCKS, so wrapping a banned token in a
// `<code>` span must not exempt it. A rendered command block still is exempt,
// which is the distinction being drawn.
func TestCheckDoesNotTreatAnInlineCodeSpanAsAFence(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput("Two roles carry the work.", "The flag was <code>previously</code> named --old.")
	wantFinding(t, r.check(), CheckBannedTokens, "present_tense/previously")
}

// TestCheckSkipsABannedTokenInsideACodeBlock is the other half of the same
// distinction: a ban whose config skips code fences skips a rendered one too.
func TestCheckSkipsABannedTokenInsideACodeBlock(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`abcd capture "a thought"`, `abcd capture "previously a thought"`)
	wantClean(t, r.check(), CheckBannedTokens)
}

// TestCheckGrantsTheAttributionEscape is adr-47 decision 3's carve-out: naming
// a tool is confined to credit, so a span selected from the acknowledgement
// file is where naming one is the sanctioned use. The same words fail when they
// are composed from anywhere else.
func TestCheckGrantsTheAttributionEscape(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	// Credit: exempt. The span names the acknowledgement file, which is where
	// the conventions confine naming a tool.
	r.breakOutput(`<footer class="site-foot">`,
		`<div data-src="ACKNOWLEDGEMENTS.md#inspirations"><p>Claude Code's sandboxing.</p></div><footer class="site-foot">`)
	wantClean(t, r.check(), CheckBannedTokens)

	// The same sentence under a documentation span: a finding.
	r2 := newCheckRepo(t)
	r2.build()
	r2.breakOutput("Two roles carry the work.", "Claude Code's sandboxing.")
	wantFinding(t, r2.check(), CheckBannedTokens, "harness/claude-code")
}

// TestCheckVerifiesAttributionNamesAgainstGit asserts the escape is a
// VERIFICATION, not an exemption: the attribution page's model names are
// matched against the trailers the history actually carries, so a name no
// commit declared is still untraceable text.
func TestCheckVerifiesAttributionNamesAgainstGit(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	page := filepath.Join(r.out, "contributors", "index.html")
	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(data), "</body>",
		`<p>Endorsed by Acme Analytics Inc.</p></body>`, 1)
	if err := os.WriteFile(page, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	wantFinding(t, r.check(), CheckProvenance, "Endorsed by Acme Analytics Inc.")
}

// --- 4. snippet pinning ----------------------------------------------------

// TestCheckRefusesAStaleSnippet is itd-135 AC 4: a command the site shows that
// the generated CLI reference does not document fails, naming the snippet and
// the span it came from.
func TestCheckRefusesAStaleSnippet(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput("abcd capture", "abcd captur")
	res := r.check()
	wantFinding(t, res, CheckSnippets, "abcd captur")
	if findingsFor(res, CheckSnippets)[0].Source == "" {
		t.Error("the snippet finding names no data-src")
	}
}

// TestCheckRefusesAnUndocumentedFlag asserts the pin reaches the flags too.
func TestCheckRefusesAnUndocumentedFlag(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`abcd capture "a thought"`, "abcd capture --nonesuch")
	wantFinding(t, r.check(), CheckSnippets, "--nonesuch")
}

// --- 5. the baseline ratchet -----------------------------------------------

// TestCheckRefusesAGrownBaseline asserts the ratchet's failing direction: an
// unresolved reference outside the committed baseline fails, naming it.
func TestCheckRefusesAGrownBaseline(t *testing.T) {
	r := newCheckRepo(t)
	r.write(".abcd/site-baseline.json", `{"schema_version": 1, "unresolved_references": []}`)
	r.build()
	wantFinding(t, r.check(), CheckBaseline, "adr-2")
}

// TestCheckInvitesAShrinkingBaseline asserts the other direction: a baseline
// entry whose reference resolves is news, not a failure.
func TestCheckInvitesAShrinkingBaseline(t *testing.T) {
	r := newCheckRepo(t)
	r.write(".abcd/site-baseline.json",
		`{"schema_version": 1, "unresolved_references": [{"from":"adr-1","to":"adr-2"},{"from":"adr-1","to":"adr-99"}]}`)
	r.build()
	res := r.check()
	wantClean(t, res, CheckBaseline)
	found := false
	for _, n := range res.Notes {
		if n.Check == CheckBaseline && strings.Contains(n.Detail, "adr-99") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no shrink invitation for the fixed reference: %+v", res.Notes)
	}
}

// supersedesDangle is a decision whose `supersedes` names a record that is not
// in the tree — the everyday shape of the backlog the ratchet carries, since the
// ADR lifecycle deletes a superseded file and keeps the trace in its successor.
const supersedesDangle = `---
id: adr-3
status: accepted
date: 2026-01-04
supersedes: adr-8
superseded_by: null
related_adrs: []
related_intents: []
related_rfcs: []
---

# ADR-3: the third decision

## Context

A decision that names a predecessor no longer in the tree.

## Decision

Keep it.

## Consequences

The reference dangles.
`

// specTargetDangle is an intent whose `spec_id` names a spec with no file. It is
// the second kind of dangle the detector has to see, and it reaches the gate by
// a different route: `supersedes` is a handle field the record rule parses,
// `spec_id` a graph field it carries without judging.
const specTargetDangle = `---
id: itd-1
slug: the-unspecced-one
spec_id: spc-404
kind: standalone
builds_on: []
severity: minor
---

# The Unspecced One

## Press Release

The intent naming a spec that was never written.

## Acceptance Criteria

- Given a named spec, when it has no file, then the reference dangles.
`

// withIntentStore adds an intent store to the fixture's record-lint
// configuration, so a record carrying a `spec_id` is scanned at all.
const withIntentStore = `{
  "roots": ["records"],
  "banned_tokens": [],
  "rules": {
    "record_schema": {
      "enabled": true,
      "severity": "blocker",
      "record_stores": {"adr": "records/adrs", "itd": "records/intents"}
    }
  }
}
`

// TestCheckRefusesANewSupersedesDangle is the ratchet's whole purpose: nothing
// checked supersedes targets, so a decision could name a predecessor that had
// never existed and the build shipped green. The fixture's own dangle stays
// admitted, so what fails here is the NEW one and only the new one.
func TestCheckRefusesANewSupersedesDangle(t *testing.T) {
	r := newCheckRepo(t)
	r.write("records/adrs/0003-third.md", supersedesDangle)
	r.build()
	res := r.check()
	wantFinding(t, res, CheckBaseline, "adr-8")
	for _, f := range res.Findings {
		if f.Check == CheckBaseline && strings.Contains(f.Detail, "adr-2") {
			t.Errorf("the admitted backlog entry failed too: %s", f.Detail)
		}
	}
}

// TestCheckRefusesADanglingSpecTarget is the same gate on the other field kind.
func TestCheckRefusesADanglingSpecTarget(t *testing.T) {
	r := newCheckRepo(t)
	r.write(".abcd/record-lint.json", withIntentStore)
	r.write("records/intents/planned/itd-1-the-unspecced-one.md", specTargetDangle)
	r.build()
	wantFinding(t, r.check(), CheckBaseline, "spc-404")
}

// TestCheckPassesABaselinedBacklog is the ratchet's admitting direction, over
// both field kinds at once: a dangle the repository has deliberately admitted is
// carried, not failed. It is what lets a detector be armed at all against a tree
// that already has a backlog in it.
func TestCheckPassesABaselinedBacklog(t *testing.T) {
	r := newCheckRepo(t)
	r.write(".abcd/record-lint.json", withIntentStore)
	r.write("records/adrs/0003-third.md", supersedesDangle)
	r.write("records/intents/planned/itd-1-the-unspecced-one.md", specTargetDangle)
	r.write(".abcd/site-baseline.json", `{"schema_version": 1, "unresolved_references": [
		{"from": "adr-1", "to": "adr-2"},
		{"from": "adr-3", "to": "adr-8"},
		{"from": "itd-1", "to": "spc-404"}
	]}`)
	r.build()
	wantClean(t, r.check(), CheckBaseline)
}

// TestSupersedesToAPrunedRecordStillCounts pins the load-bearing exception the
// whole detector rests on. An absent target is normally excused when some record
// declares it PRUNED — the ADR lifecycle deletes a superseded file and keeps the
// trace in its successor — but the declaration itself cannot be excused by what
// it declares, or `supersedes` would be the one typed reference nothing could
// ever check. Here adr-3 lifts the store's high-water mark past adr-2, so adr-2
// reads as pruned: the `related_adrs` reference to it is excused and the
// `supersedes` one is not.
func TestSupersedesToAPrunedRecordStillCounts(t *testing.T) {
	r := newCheckRepo(t)
	r.write("records/adrs/0003-third.md", `---
id: adr-3
status: accepted
date: 2026-01-04
supersedes: null
superseded_by: null
related_adrs: [adr-2]
related_intents: []
related_rfcs: []
---

# ADR-3: the third decision

## Context

A decision relating itself to one that has been pruned.

## Decision

Keep it.

## Consequences

The relation is excused; the supersedes declaration is not.
`)
	r.build()
	var export RecordExport
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(r.out, "record.json"))), &export); err != nil {
		t.Fatalf("record.json: %v", err)
	}
	var pruned bool
	for _, id := range export.Health.Retired {
		if id == "adr-2" {
			pruned = true
		}
	}
	if !pruned {
		t.Fatalf("adr-2 does not read as pruned, so the exception is not under test: %+v", export.Health.Retired)
	}
	// Accumulated, not assigned: a record may carry more than one unresolved
	// reference, and an assignment would let a later one erase the finding this
	// test is about.
	var supersedes, related bool
	for _, e := range export.Health.Unresolved {
		switch e.From {
		case "adr-1":
			supersedes = supersedes || e.To == "adr-2"
		case "adr-3":
			related = related || e.To == "adr-2"
		}
	}
	if !supersedes {
		t.Error("a supersedes naming a pruned record is excused; nothing would ever check one")
	}
	if related {
		t.Error("a related_adrs naming a pruned record counts; every retired id would fail the gate")
	}
}

// TestCheckPassesWhenABaselinedTargetArrives is the fourth ratchet behaviour: a
// baselined reference whose target is later written stops dangling, and the gate
// says so as an invitation to shrink rather than as a failure. A ratchet that
// failed here would teach people to stop fixing references.
func TestCheckPassesWhenABaselinedTargetArrives(t *testing.T) {
	r := newCheckRepo(t)
	r.write("records/adrs/0002-second.md", `---
id: adr-2
status: superseded
date: 2026-01-01
supersedes: null
superseded_by: adr-1
related_adrs: []
related_intents: []
related_rfcs: []
---

# ADR-2: the superseded decision

## Context

The decision adr-1 supersedes, back in the tree.

## Decision

Keep it.

## Consequences

The baseline can shrink.
`)
	r.build()
	res := r.check()
	wantClean(t, res, CheckBaseline)
	found := false
	for _, n := range res.Notes {
		if n.Check == CheckBaseline && strings.Contains(n.Detail, "adr-2") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no shrink invitation once the target arrived: %+v", res.Notes)
	}
}

// --- 6. static mobile checks -----------------------------------------------

// TestCheckRefusesAMissingViewport is the first half of itd-135 AC 7.
func TestCheckRefusesAMissingViewport(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`<meta name="viewport" content="width=device-width, initial-scale=1">`, "")
	wantFinding(t, r.check(), CheckMobile, "viewport")
}

// TestCheckRefusesAnUnwrappedTable asserts a wide element with no scroll above
// it fails: on a narrow viewport it widens the page instead of scrolling.
func TestCheckRefusesAnUnwrappedTable(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`<div class="tablewrap"`, `<div class="plain"`)
	wantFinding(t, r.check(), CheckMobile, "<table>")
}

// TestCheckRefusesAnUnscrollableCodeBlock asserts a command block is held to
// the same rule as a table: a fenced command is the other thing the corpus
// renders that is wider than a phone.
func TestCheckRefusesAnUnscrollableCodeBlock(t *testing.T) {
	r := newCheckRepo(t)
	r.write("site-src/site.css", strings.Replace(checkFixtureFiles["site-src/site.css"],
		"pre{overflow-x:auto}\n", "", 1))
	r.build()
	wantFinding(t, r.check(), CheckMobile, "<pre>")
}

// TestCheckRefusesAnUnconstrainedImage asserts the stylesheet half of the
// claim: a page that renders pictures and links no max-width rule fails.
func TestCheckRefusesAnUnconstrainedImage(t *testing.T) {
	r := newCheckRepo(t)
	r.write("site-src/site.css", strings.Replace(checkFixtureFiles["site-src/site.css"],
		"img{max-width:100%;height:auto}", "img{height:auto}", 1))
	r.build()
	wantFinding(t, r.check(), CheckMobile, "max-width")
}

// TestCheckRefusesAnOversizedImage asserts the per-page half: a picture that
// pins itself wider than the widest column the design offers is the width it
// would take if the max-width rule were ever dropped.
func TestCheckRefusesAnOversizedImage(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`width="200"`, `width="1600"`)
	wantFinding(t, r.check(), CheckMobile, "1600")
}

// TestCheckRefusesAnInlineFixedWidth asserts the absolute half: a stylesheet
// rule yields to a media query, and an inline width does not.
func TestCheckRefusesAnInlineFixedWidth(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`<main id="app">`, `<main id="app" style="width:900px">`)
	wantFinding(t, r.check(), CheckMobile, "900px")
}

// TestCheckReadsTheRecordRenderingForMobileOnly asserts the scope split: the
// verbatim record rendering is exempt from the composed-surface gates and is
// still held to the mobile ones (adr-47 decision 3).
func TestCheckReadsTheRecordRenderingForMobileOnly(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	if err := os.MkdirAll(filepath.Join(r.out, "record"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A page with change-narration, an unsourced node and no viewport: only the
	// last of the three is this page's business.
	page := "<!doctype html>\n<html lang=\"en\">\n<head>\n<title>Record</title>\n</head>\n<body>\n" +
		"<p>The rule was previously written down elsewhere.</p>\n</body>\n</html>\n"
	if err := os.WriteFile(filepath.Join(r.out, "record", "index.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	res := r.check()
	wantClean(t, res, CheckBannedTokens)
	for _, f := range findingsFor(res, CheckProvenance) {
		if strings.HasPrefix(f.Where, "record/") {
			t.Fatalf("the record rendering was held to the provenance walk: %+v", f)
		}
	}
	wantFinding(t, res, CheckMobile, "record/index.html")
}

// --- 7. loop-figure labels -------------------------------------------------

// TestCheckRefusesAnAlienFigureLabel consumes the manifest's
// `figure.labels-from-page`: a label the page does not carry fails, naming it.
func TestCheckRefusesAnAlienFigureLabel(t *testing.T) {
	r := newCheckRepo(t)
	r.write("docs/assets/img/loop.svg", strings.Replace(checkFixtureFiles["docs/assets/img/loop.svg"],
		"<tspan>the</tspan><tspan>verdict</tspan>", "<tspan>synergy</tspan><tspan>quadrant</tspan>", 1))
	r.build()
	wantFinding(t, r.check(), CheckFigureLabels, "synergy quadrant")
}

// TestCheckJoinsFigureTspansAsOneLabel asserts a wrapped label is one phrase:
// the drawings break a phrase across tspans, and reading each as a label of its
// own would fail every wrapped one.
func TestCheckJoinsFigureTspansAsOneLabel(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	wantClean(t, r.check(), CheckFigureLabels)
}

// TestCheckRefusesADeclaredFigureThatIsNotRendered asserts the manifest key is
// CONSUMED rather than merely validated: asking for a label check on a figure
// nothing renders is a manifest that cannot be honoured.
func TestCheckRefusesADeclaredFigureThatIsNotRendered(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	r.breakOutput(`<figure class="card loopfig"`, `<div class="card loopfig"`)
	r.breakOutput("</figure>", "</div>")
	wantFinding(t, r.check(), CheckFigureLabels, "carries no figure")
}

// --- the fixture -----------------------------------------------------------

// checkFixtureFiles is one small repository composed under the single-source
// rule: an Identity block, three documentation pages, a drawing whose labels
// are phrases on its page, a generated-shaped CLI reference, and the committed
// configuration the build and the check read.
var checkFixtureFiles = map[string]string{
	".claude-plugin/plugin.json": `{
  "name": "probe",
  "repository": "https://example.invalid/acme/probe",
  "license": "MIT",
  "author": {"name": "Acme"}
}
`,

	".abcd/site.json": `{
  "schema_version": 1,
  "purpose": "The fixture composition.",
  "identity": {
    "file": ".abcd/development/brief/01-product/README.md",
    "heading": "Identity (canonical)"
  },
  "ui_strings": "site-src/ui.json",
  "home": {
    "hero": {"page": "docs/explanation/rationale.md", "figure": "first-image"},
    "chapters": [
      {"letter": "a", "page": "docs/explanation/roles.md", "layout": "lead-in-cards"},
      {"letter": "b", "page": "docs/explanation/process.md", "layout": "prose",
       "figure": {"kind": "first-image", "labels-from-page": true}}
    ]
  },
  "record": {"issue_ledger": false},
  "docs": {"index": "docs/README.md", "cli": "docs/reference/cli/commands.md"},
  "checks": {
    "every_text_node_has_source": true,
    "docs_lint_on_rendered_text": true,
    "command_snippets_pinned_to_cli_reference": true,
    "unresolved_reference_baseline": ".abcd/site-baseline.json"
  }
}
`,

	"site-src/site.css": `.wrap{max-width:1000px;margin:0 auto}
img{max-width:100%;height:auto}
pre{overflow-x:auto}
.tablewrap{overflow-x:auto}
@media (max-width:700px){.wrap{padding:0 12px}}
`,
	"site-src/site.js":   "/* the fixture ships no behaviour */\n",
	"site-src/redirects": "/old /new 301\n",
	"site-src/headers":   "/*\n  X-Frame-Options: DENY\n",
	"CHANGELOG.md":       "# Changelog\n\n## [0.1.0] - 2026-01-02\n\n- The first cut (adr-1).\n",
	"docs/README.md":     "# Documentation\n\nThe fixture's documentation index.\n",
	// The attribution file. Naming a tool is confined to credit, and this is
	// where credit is written.
	"ACKNOWLEDGEMENTS.md": "# Acknowledgements\n\n## Inspirations\n\nWhere the ideas came from.\n",
	".abcd/record-lint.json": `{
  "roots": ["records"],
  "banned_tokens": [],
  "rules": {
    "record_schema": {
      "enabled": true,
      "severity": "blocker",
      "record_stores": {"adr": "records/adrs"}
    }
  }
}
`,
	".abcd/docs-lint.json": `{
  "roots": ["docs"],
  "banned_tokens": [
    {"id": "present_tense/previously", "pattern": "(?i)\\bpreviously\\b", "severity": "blocker",
     "successor": "present-tense phrasing", "message": "change-narration in a doc body",
     "allow_context": ["(?i)<!--\\s*allow-here\\s*-->"]},
    {"id": "harness/claude-code", "pattern": "(?i)\\bclaude[ -]?code\\b", "severity": "blocker",
     "successor": "a generic term (the agent harness)", "message": "names a specific agent harness",
     "allow_context": ["(?i)<!--\\s*allow-here\\s*-->"]}
  ],
  "rules": {}
}
`,
	".abcd/site-baseline.json": `{
  "schema_version": 1,
  "unresolved_references": [{"from": "adr-1", "to": "adr-2"}]
}
`,

	".abcd/development/brief/01-product/README.md": `# Product

## Identity (canonical)

- **Title:** Probe — the fixture project
- **Tagline:** A configuration layer for the fixture.
- **Pitch:** One binary that carries the why from idea to shipped reality.
`,

	"records/adrs/0001-first.md": `---
id: adr-1
status: accepted
date: 2026-01-01
supersedes: adr-2
superseded_by: null
related_adrs: []
related_intents: []
related_rfcs: []
---

# ADR-1: the first decision

## Context

The fixture needs one decision so the record graph is not empty.

## Decision

Keep it.

## Consequences

None.
`,

	"docs/explanation/rationale.md": `# Who the fixture is for

The fixture exists so the gates have something to read.

![The loop](../assets/img/loop.svg)
`,

	"docs/explanation/roles.md": `# Roles

Two roles carry the work.

![The mark](../assets/img/mark.png)

**Product thinker.** They decide what is worth building.

**Facilitator.** They turn that into engineering work.

| Role | Owns |
|---|---|
| Product thinker | The why |
| Facilitator | The how |

## Aside

This section is not composed onto the landing page; the lead-in-cards layout
reads the first section only. It exists so the file carries a line where the
banned token is declared legitimate, which is what the chapter heading's
whole-file span reads.

The setting was previously spelled --old. <!-- allow-here -->
`,

	"docs/explanation/process.md": `# Process

The loop runs from the brief to the verdict and back.

![The loop](../assets/img/loop.svg)

## Capturing intents

Write an intent down the moment it exists.

` + "```bash\nabcd capture \"a thought\"\n```" + `

Nothing else is required.
`,

	"docs/assets/img/loop.svg": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 60" width="100" height="60" role="img">
  <title>The loop</title>
  <rect x="1" y="1" width="98" height="58" fill="var(--surface, #fff)"/>
  <text x="10" y="20">the brief</text>
  <text x="10" y="40"><tspan>the</tspan><tspan>verdict</tspan></text>
</svg>
`,

	"docs/reference/cli/commands.md": "# CLI command reference\n\n" +
		"This page is generated from the command tree.\n\n" +
		"## `abcd`\n\nThe root command\n\n**Usage:** `abcd [<record-id>]`\n\n" +
		"**Flags:**\n\n```\n      --json   emit machine-readable JSON\n```\n\n" +
		"### `abcd capture`\n\nCapture an issue\n\n**Usage:** `abcd capture [text] [flags]`\n\n" +
		"**Flags:**\n\n```\n      --impact string   the impact class\n```\n",
}

// TestCheckSkipsTheDocsTree pins the declared scope: `docs/` is MkDocs' own
// output, not this build's, so its pages are excluded from the walk — never
// parsed against the generator's strict grammar (mkdocs emits HTML comments),
// never examined by gates whose rules were written for generator output. Its
// words are gated at the source by docs-lint.
func TestCheckSkipsTheDocsTree(t *testing.T) {
	r := newCheckRepo(t)
	r.build()
	docs := filepath.Join(r.out, "docs")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	page := "<!doctype html>\n<html><head><title>d</title></head>" +
		"<body><!-- mkdocs writes comments --><p>docs text</p></body></html>\n"
	if err := os.WriteFile(filepath.Join(docs, "index.html"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	res := r.check()
	for _, f := range res.Findings {
		if strings.HasPrefix(f.Where, "docs/") {
			t.Errorf("a gate examined the docs tree: %s — %s", f.Where, f.Detail)
		}
	}
	for _, p := range res.Pages {
		if strings.HasPrefix(p, "docs/") {
			t.Errorf("the docs tree entered the page list: %s", p)
		}
	}
}
