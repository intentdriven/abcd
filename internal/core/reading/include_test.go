package reading

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up from the test's working directory to the directory holding
// go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test working directory")
		}
		dir = parent
	}
}

// recordFamilyRoots are the directories that hold a record family. Assembler
// rule 1 is stated against this list: no include may name a directory that
// CONTAINS one of them.
var recordFamilyRoots = []string{
	".abcd/development/decisions",
	".abcd/development/intents",
	".abcd/development/specs",
	".abcd/development/readings",
	".abcd/work/issues",
}

// TestReadingsCharterCarriesTheRenderedIncludeTable is the anti-drift detector:
// the charter and the Go table are one contract, and the code is its source of
// truth. Editing either alone fails here.
func TestReadingsCharterCarriesTheRenderedIncludeTable(t *testing.T) {
	path := filepath.Join(repoRoot(t), CharterPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", CharterPath, err)
	}
	doc := string(raw)

	begin := strings.Index(doc, MarkerBegin)
	end := strings.Index(doc, MarkerEnd)
	if begin < 0 || end < 0 {
		t.Fatalf("%s does not carry the generated-table markers %q / %q; the charter "+
			"is the include table's rendering, so it must render it", CharterPath, MarkerBegin, MarkerEnd)
	}
	if end < begin {
		t.Fatalf("%s: end marker precedes begin marker", CharterPath)
	}
	got := strings.TrimSpace(doc[begin+len(MarkerBegin) : end])
	want := strings.TrimSpace(Render())
	if got != want {
		t.Errorf("%s has drifted from reading.Table.\n\n--- charter has ---\n%s\n\n--- Render() wants ---\n%s",
			CharterPath, got, want)
	}
}

// TestEveryFramingRowCitesAdr55 holds adr-55's own consequence: the framing
// section's construal statement is admissible only because that ADR says so, so
// every row that admits it must name the rule admitting it.
func TestEveryFramingRowCitesAdr55(t *testing.T) {
	const framing = ".abcd/development/brief/01-product/06-framing.md"
	admitting := 0
	for _, row := range Table {
		if !row.Reaches(framing) {
			continue
		}
		admitting++
		if !strings.Contains(row.Rule, "adr-55") {
			t.Errorf("row %q admits the framing section but its rule %q does not cite adr-55",
				row.Source, row.Rule)
		}
	}
	if admitting == 0 {
		t.Fatalf("no row admits %s; the construal statement is committed record and must be passed", framing)
	}
}

// TestNoIncludeNamesARecordFamilyDirectory is assembler rule 1 as an executable
// property of the table: a row may name a family's own bucket individually, but
// no row may reach into a family from above.
func TestNoIncludeNamesARecordFamilyDirectory(t *testing.T) {
	for _, family := range recordFamilyRoots {
		for _, row := range Table {
			src := strings.TrimSuffix(row.Source, "/")
			if src == family || strings.HasPrefix(src+"/", family+"/") {
				continue // named individually, which rule 1 permits
			}
			for _, probe := range []string{family + "/probe.md", family + "/bucket/probe.md"} {
				if row.Reaches(probe) {
					t.Errorf("row %q reaches %s: an include may not name a directory containing a record family",
						row.Source, probe)
				}
			}
		}
	}
}

// TestTheAssemblersOwnOutputIsNeverItsInput is ruling (18): the instrument's own
// prior outputs, its definitions, the evals that guard it and the include table
// that decides what it sees are all unreachable at every position.
func TestTheAssemblersOwnOutputIsNeverItsInput(t *testing.T) {
	own := []string{
		".abcd/development/readings/rdg-2608301200000001/manifest.json",
		".abcd/development/readings/rdg-2608301200000001/run.json",
		".abcd/development/readings/README.md",
		"agents/cold-reading-widening.md",
		"evals/cold_reading_readblock_test.go",
		"internal/core/reading/include.go",
		"internal/core/reading/testdata/fixture.md",
	}
	for _, p := range Positions() {
		for _, rel := range own {
			if Admits(p, rel) {
				t.Errorf("position %s admits %s; the assembler's own output, definition, "+
					"eval and include table must never become its input", p, rel)
			}
		}
	}

	// The ONE positional exception, and its limit. A reading record is the
	// instrument's own output, and adr-2609021016272867 admits two body fields of
	// one derived widening run's items at the comparative position — so the path
	// is admitted THERE and refused at the other three. itd-186's fourth
	// criterion gains the same exception, and the read-block eval carries the
	// case. Both halves are asserted: an exception that widened to another
	// position would be caught by the loop below, and one that vanished would be
	// caught by the first check.
	const candidateRecord = ".abcd/work/issues/readings/rdg-2608301200000001/rdi-1.md"
	if !Admits(PositionComparative, candidateRecord) {
		t.Errorf("the comparative position does not admit %s; the candidate channel is the one "+
			"exception to the prior-run exhaust, and without it the position has no object",
			candidateRecord)
	}
	for _, p := range Positions() {
		if p == PositionComparative {
			continue
		}
		if Admits(p, candidateRecord) {
			t.Errorf("position %s admits %s; the exception is the COMPARATIVE position's alone",
				p, candidateRecord)
		}
	}
	// And the limit inside the exception: the run's own artefacts and the
	// dispositions, admissions and surprises keyed to its items stay out at every
	// position, comparative included.
	for _, rel := range []string{
		".abcd/work/issues/dispositions/rdi-1/dsp-1.md",
		".abcd/work/issues/admissions/rdg-2608301200000001/adm-1.md",
		".abcd/work/issues/surprises/srp-1.md",
		".abcd/work/issues/open/iss-1-a-defect.md",
	} {
		for _, p := range Positions() {
			if Admits(p, rel) {
				t.Errorf("position %s admits %s; the comparative reading receives candidates and "+
					"never their fate", p, rel)
			}
		}
	}

	// The path half is not the whole of ruling (18). An artefact COMMITTED where
	// a root row reaches it — the plugin page's own `--out ./run-dir` example,
	// resolved against the repository root — is admitted by shape, so the
	// artefacts have to be recognised by what they are. They self-identify: both
	// carry a top-level `_type`.
	root := fixtureRepo(t)
	priorBundle, err := EncodeBundle(Bundle{
		Type: BundleType, SchemaVersion: SchemaVersion, Position: PositionWidening,
		Items: []BundleItem{{ItemKey: "itm-0001", Kind: KindDoc, Text: "a prior run's passed item"}},
	})
	if err != nil {
		t.Fatalf("encode a prior bundle: %v", err)
	}
	writeFile(t, root, "run-dir/bundle.json", string(priorBundle))
	gitCommitAll(t, root)

	if _, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	}); err == nil {
		t.Fatal("a prior run's assembled input committed under an admitted root was passed as config")
	} else if !strings.Contains(err.Error(), "run-dir/bundle.json") {
		t.Errorf("the refusal does not name the prior artefact: %v", err)
	}
}

// TestAssembleRefusesAnOutputDirectoryTheTableAdmits closes the same breach at
// its source: an operator who writes a run where the table can reach it has
// committed the next run's contamination, and the refusal belongs at the moment
// the directory is named rather than one run later.
func TestAssembleRefusesAnOutputDirectoryTheTableAdmits(t *testing.T) {
	root := fixtureRepo(t)
	for _, out := range []string{"run-dir", "./run-dir", "docs/runs", filepath.Join(root, "run-dir")} {
		_, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: PositionWidening, Target: "HEAD", OutDir: out,
		})
		if err == nil {
			t.Errorf("--out %q was accepted; the table admits the artefacts it would write there", out)
		}
	}
	for _, out := range []string{DefaultRunDir + "/rdg-2608301200000001", filepath.Join(t.TempDir(), "run")} {
		if _, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: PositionWidening, Target: "HEAD", OutDir: out,
		}); err != nil {
			t.Errorf("--out %q was refused: %v", out, err)
		}
	}
}

// TestWideningExcludesDraftsAndPlannedEntailmentIncludesThem holds the one
// asymmetry the record insists is stated rather than remembered: a widening
// reading must not see the candidate set it is asked to widen, and articulation
// precedes selection.
//
// It carries the shipped intents too, which is itd-194 ac-7 at table grain. The
// framework's widening object and the readings companion's section 5.2 both
// state that object without the shipped intents, and the maintainer's ruling of
// 2026-09-02 takes the row's positions accordingly (iss-2609012259587904). The
// shipped path sits here rather than in a test of its own because it is the
// same asymmetry: the widening position is the one that does not receive what
// it is asked to widen past.
func TestWideningExcludesDraftsAndPlannedEntailmentIncludesThem(t *testing.T) {
	candidates := []string{
		".abcd/development/intents/drafts/itd-900-a-draft.md",
		".abcd/development/intents/planned/itd-901-a-planned.md",
	}
	for _, rel := range candidates {
		if Admits(PositionWidening, rel) {
			t.Errorf("the widening position admits %s; it is the candidate set the reading is asked to widen", rel)
		}
		if Admits(PositionDetection, rel) {
			t.Errorf("the detection position admits %s; only entailment sees the candidate set", rel)
		}
		if !Admits(PositionEntailment, rel) {
			t.Errorf("the entailment position excludes %s; articulation precedes selection", rel)
		}
	}

	const shipped = ".abcd/development/intents/shipped/itd-902-a-shipped.md"
	if Admits(PositionWidening, shipped) {
		t.Errorf("the widening position admits %s; neither design document lists the shipped "+
			"intents in the widening object", shipped)
	}
	for _, p := range []Position{PositionEntailment, PositionDetection} {
		if !Admits(p, shipped) {
			t.Errorf("position %s excludes %s; only widening withdraws from the shipped row", p, shipped)
		}
		for _, e := range ExclusionsFor(p) {
			if e.Detail == ".abcd/development/intents/shipped" {
				t.Errorf("position %s asserts the shipped intents excluded while its rows admit them", p)
			}
		}
	}
}

// TestSupersededIntentsAreNeverAdmitted holds the exclusion floor's directory
// half at the one place the intent store makes it easy to get wrong.
func TestSupersededIntentsAreNeverAdmitted(t *testing.T) {
	const superseded = ".abcd/development/intents/superseded/itd-47-a-retired.md"
	for _, p := range Positions() {
		if Admits(p, superseded) {
			t.Errorf("position %s admits %s; superseded records are deliberation", p, superseded)
		}
	}
}

// TestBriefEvidenceChapterIsNeverAdmitted holds the chapter-level include bound:
// 03-evidence is deliberation, and a future brief-homed warm record must not
// walk in as "brief text".
func TestBriefEvidenceChapterIsNeverAdmitted(t *testing.T) {
	const evidence = ".abcd/development/brief/03-evidence/01-open-questions.md"
	for _, p := range Positions() {
		if Admits(p, evidence) {
			t.Errorf("position %s admits %s; the evidence chapter is deliberation", p, evidence)
		}
	}
}

// includeTableDigest is the sha256 of Render() at AssemblerVersionCore.
//
// The digest the manifest carries is COMPUTED, so this literal is not what
// makes a version honest — TestATableChangeMovesTheStampedVersion holds that,
// structurally. This literal does the one job the computed digest cannot: it
// fails on a table change and names AssemblerVersionCore while doing so, which
// is what puts the hand-set semver in front of a human at the moment it should
// move.
//
// That forcing function is worth keeping explicitly, because removing the old
// gate removed it by accident. The old gate was the only thing that ever named
// the version constant, and replacing it with a structural property left the
// core semver advisory with NOTHING pointing at it — weaker than before, not
// equivalent. Restated here, and the weakness that made the old gate
// insufficient no longer matters: updating this literal without moving the core
// can no longer make a manifest lie, because the manifest's digest is not this
// literal.
const includeTableDigest = "cacc591ff9d463fe20118ddc72b5a12363aaca145d3810c3c0221f416114a75a"

// TestAssemblerVersionCoversTheIncludeTable puts the core semver in front of
// whoever changed the table. It is ADVISORY by construction — the fix for a red
// run may legitimately be to restate the digest — and it is not the gate that
// makes the stamped version trustworthy. Do not read it as one.
func TestAssemblerVersionCoversTheIncludeTable(t *testing.T) {
	sum := sha256.Sum256([]byte(Render()))
	got := hex.EncodeToString(sum[:])
	if got != includeTableDigest {
		t.Errorf("the include table has changed while AssemblerVersionCore is still %s.\n"+
			"Decide whether the CONTRACT moved: if it did, bump AssemblerVersionCore.\n"+
			"Either way set includeTableDigest to %q.", AssemblerVersionCore, got)
	}
}

// TestAssemblerVersionCarriesTheTableDigest checks the stamped version is the
// core composed with the digest.
//
// It is a composition check and nothing more: it re-derives what
// AssemblerVersion() computes, so it cannot fail for any table change. The
// property that matters — that the version MOVES when the table does — is held
// by TestATableChangeMovesTheStampedVersion below, which performs the change
// rather than asserting about it. Both are here because a reader who finds only
// the composition check would reasonably think it was the gate.
func TestAssemblerVersionCarriesTheTableDigest(t *testing.T) {
	sum := sha256.Sum256([]byte(Render()))
	want := AssemblerVersionCore + "+" + hex.EncodeToString(sum[:])
	if got := AssemblerVersion(); got != want {
		t.Errorf("AssemblerVersion() = %q, want %q — the stamped version must carry the "+
			"digest of the table it was built from", got, want)
	}
}

// TestATableChangeMovesTheStampedVersion is the mutation proof for the test
// above. A passing digest comparison proves nothing about whether a CHANGE
// would be caught, which is exactly how the old gate stayed vacuous, so the
// change is performed here rather than assumed: the table is mutated, the
// stamped version is re-read, and it must differ.
func TestATableChangeMovesTheStampedVersion(t *testing.T) {
	before := AssemblerVersion()

	restore := Table
	t.Cleanup(func() { Table = restore })

	mutated := make([]Row, len(Table))
	copy(mutated, Table)
	mutated[0].Kind = KindConfig // a kind reassignment: invisible to the OLD gate
	Table = mutated

	if after := AssemblerVersion(); after == before {
		t.Errorf("reassigning a row's kind left the stamped version at %q; a change to "+
			"the table a reading is decided by must move the version a manifest names", after)
	}
}

// TestEveryRowNamesAKnownKindAndPosition holds the two closed vocabularies.
func TestEveryRowNamesAKnownKindAndPosition(t *testing.T) {
	if len(Table) == 0 {
		t.Fatal("the include table is empty; a reading with no input is not a blind reading")
	}
	kinds := map[Kind]bool{}
	for _, k := range Kinds() {
		kinds[k] = true
	}
	scans := map[Scan]bool{}
	for _, s := range Scans() {
		scans[s] = true
	}
	for _, row := range Table {
		if !kinds[row.Kind] {
			t.Errorf("row %q carries kind %q, which is not in the closed vocabulary", row.Source, row.Kind)
		}
		if len(row.Positions) == 0 {
			t.Errorf("row %q is admitted at no position", row.Source)
		}
		for _, p := range row.Positions {
			if _, err := ParsePosition(string(p)); err != nil {
				t.Errorf("row %q names position %q: %v", row.Source, p, err)
			}
		}
		if row.Rule == "" {
			t.Errorf("row %q states no admitting rule", row.Source)
		}
		// The third closed vocabulary (itd-194). A row's Scan is the table's
		// promise about whether the exclusion floor examines what the row
		// admits, and an unset or unknown value is a promise nobody made —
		// which is the state brief invariant 16 refuses on the manifest side
		// and refuses here for the same reason.
		if !scans[row.Scan] {
			t.Errorf("row %q declares the scan %q, which is not in the closed vocabulary %v",
				row.Source, row.Scan, Scans())
		}
		// A row must select positively by SOMETHING. Either match form
		// satisfies that; neither does not. The guard reads both fields
		// because a row selecting only by suffix is positive at exactly the
		// same grain as one selecting only by extension (spc-68).
		if len(row.Match) == 0 && len(row.MatchSuffix) == 0 {
			t.Errorf("row %q matches every file; inclusion is positive at every grain", row.Source)
		}
	}
}

// TestPositionTokenIsClosed holds the invocation interface's first operand.
func TestPositionTokenIsClosed(t *testing.T) {
	for _, p := range Positions() {
		if _, err := ParsePosition(string(p)); err != nil {
			t.Errorf("ParsePosition(%q): %v", p, err)
		}
	}
	for _, bad := range []string{"", "Widening", "widening ", "framing", "../widening", "widening; drop table"} {
		if _, err := ParsePosition(bad); err == nil {
			t.Errorf("ParsePosition(%q) accepted an unknown token", bad)
		}
	}
}

// TestExclusionFloorNamesEveryRecordedExclusion holds itd-183's exclusion list
// against the floor the manifest asserts, so an exclusion the record states
// cannot quietly stop being declared.
func TestExclusionFloorNamesEveryRecordedExclusion(t *testing.T) {
	want := []string{
		"origin",
		"production_mode",
		"Audit Notes",
		".abcd/development/brief/03-evidence",
		".abcd/development/decisions",
		".abcd/development/roadmap/rfcs",
		".abcd/development/intents/superseded",
		".abcd/work/issues",
		".abcd/development/plans",
		".abcd/development/research/notes",
		".abcd/development/readings",
		// itd-194: the widening position withdraws from the shipped intents, and
		// the floor asserts the withdrawal there so a reader can check it rather
		// than infer it from a row's absence (iss-2609012259587904).
		".abcd/development/intents/shipped",
	}
	joined := ""
	for _, e := range ExclusionsFor(PositionWidening) {
		joined += e.Rule + "\x00" + e.Signal + "\x00" + e.Detail + "\n"
	}
	for _, w := range want {
		if !strings.Contains(joined, "\x00"+w+"\n") {
			t.Errorf("the exclusion floor does not name %q", w)
		}
	}
}

// TestParsedRowsAdmitOnlyMarkdown is the point at which the include table's
// admission and the exclusion floor's examination are proved to describe one
// set, whichever of them moves later — brief invariant 16's third clause, and
// adr-56's third rule (spc-2609021003136831, "The vocabulary").
//
// The declaration is held to the parser rather than to itself: a row that says
// the floor parses what it admits must admit markdown and nothing else, because
// the floor's key and heading signals are record shapes only a markdown file
// carries; and a row that says the floor does not parse what it admits must
// admit no markdown, because an unparsed markdown document is exactly the
// silent-admission path this intent closes.
func TestParsedRowsAdmitOnlyMarkdown(t *testing.T) {
	const markdown = "a-chapter.md"
	nonMarkdown := []string{
		"widget.go", "widget_test.go", "config.json", "ci.yml", "compose.yaml",
		"tool.toml", "go.mod", "go.sum", "Makefile", "notes.txt",
	}
	for _, row := range Table {
		// Every selector the row declares, so the property is read off the row
		// rather than off a probe list that might miss a form.
		selectors := append(append([]string{}, row.Match...), row.MatchSuffix...)
		switch row.Scan {
		case ScanParsed:
			if len(selectors) == 0 {
				t.Errorf("row %q declares %q and selects by nothing; a parsed row that admits "+
					"no markdown declares a scan that never runs", row.Source, row.Scan)
			}
			for _, sel := range selectors {
				if !strings.EqualFold(filepath.Ext(sel), ".md") {
					t.Errorf("row %q declares %q and selects %q, which is not markdown; the "+
						"floor parses markdown and nothing else", row.Source, row.Scan, sel)
				}
			}
			if !row.matches(markdown) && !row.matches("00-meta.md") {
				t.Errorf("row %q declares %q and admits no markdown probe; a parsed row that "+
					"admits no markdown declares a scan that never runs", row.Source, row.Scan)
			}
			for _, base := range nonMarkdown {
				if row.matches(base) {
					t.Errorf("row %q declares %q and admits %s, which the floor cannot parse; "+
						"admission and examination describe one set (adr-56, rule 3)",
						row.Source, row.Scan, base)
				}
			}
		case ScanUnscanned:
			if row.matches(markdown) {
				t.Errorf("row %q declares %q and admits %s; a markdown document the floor does "+
					"not examine is the silent-admission path itd-194 closes",
					row.Source, row.Scan, markdown)
			}
		default:
			t.Errorf("row %q declares the scan %q, which is not in the closed vocabulary %v",
				row.Source, row.Scan, Scans())
		}
	}
}

// admittingRow returns the first row that admits rel at p, which is the row
// that owns the projection and the kind applied to it.
func admittingRow(t *testing.T, p Position, rel string) (Row, bool) {
	t.Helper()
	for _, row := range Table {
		if row.AdmittedAt(p) && row.Reaches(rel) {
			return row, true
		}
	}
	return Row{}, false
}

// TestBriefChaptersAreAdmittedAsBriefSections is ac-8's positive half: the
// design documents name "brief current text" as a reading's object (framework
// 7.2, companion 5.2), and on the corrections ruling of 2026-09-02 that is the
// whole brief bar the evidence chapter and bar the glossary, which is a record
// family with its own row (divergence register entry 26).
func TestBriefChaptersAreAdmittedAsBriefSections(t *testing.T) {
	const product = ".abcd/development/brief/01-product/06-framing.md"
	var productPositions []Position
	for _, p := range Positions() {
		if Admits(p, product) {
			productPositions = append(productPositions, p)
		}
	}
	if len(productPositions) == 0 {
		t.Fatal("no position admits the product chapter; the brief rows are stated against it")
	}

	chapters := []string{
		".abcd/development/brief/00-meta.md",
		".abcd/development/brief/01-product/06-framing.md",
		".abcd/development/brief/02-constraints/03-invariants.md",
		".abcd/development/brief/04-surfaces/23-reading.md",
		".abcd/development/brief/05-internals/03-configuration.md",
		".abcd/development/brief/06-delivery/01-shipping.md",
	}
	for _, rel := range chapters {
		for _, p := range productPositions {
			row, ok := admittingRow(t, p, rel)
			if !ok {
				t.Errorf("position %s admits no row reaching %s; brief current text is a "+
					"reading's object", p, rel)
				continue
			}
			if row.Kind != KindBriefSection {
				t.Errorf("position %s admits %s as %q; a brief chapter travels as %q",
					p, rel, row.Kind, KindBriefSection)
			}
			if row.Scan != ScanParsed {
				t.Errorf("position %s admits %s under a row declaring %q; a brief chapter is "+
					"markdown the floor parses", p, rel, row.Scan)
			}
		}
	}

	// The meta row's source is the brief directory, so its match is what bounds
	// it: it reaches the one file 00-meta.md and nothing beside it.
	for _, beside := range []string{
		".abcd/development/brief/00-orientation.md",
		".abcd/development/brief/07-appendix/00-meta-notes.md",
	} {
		row, ok := admittingRow(t, productPositions[0], beside)
		if ok && row.Source == ".abcd/development/brief" {
			t.Errorf("the meta row reaches %s; its Match is the exact basename so it reaches "+
				"00-meta.md and nothing beside it", beside)
		}
	}
}

// TestTheEvidenceChapterIsExcludedAsVerdictMaterial is ac-8's negative half.
// The ground moves into the table: framework 7.1 excludes Audit Notes because a
// prior verdict is revision history, and the evidence chapter is the same
// material at chapter grain (divergence register entry 26).
func TestTheEvidenceChapterIsExcludedAsVerdictMaterial(t *testing.T) {
	const evidence = ".abcd/development/brief/03-evidence/01-open-questions.md"
	for _, p := range Positions() {
		if Admits(p, evidence) {
			t.Errorf("position %s admits %s; the evidence chapter is verdict material", p, evidence)
		}
	}
	found := false
	for _, e := range Exclusions {
		if e.Detail != ".abcd/development/brief/03-evidence" {
			continue
		}
		found = true
		if !strings.Contains(e.Rule, "verdict material") {
			t.Errorf("the evidence chapter's exclusion rule is %q; the table states the ground "+
				"the Audit Notes exclusion rests on", e.Rule)
		}
	}
	if !found {
		t.Error("the exclusion floor names no entry for the brief's evidence chapter")
	}
}
