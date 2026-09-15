package reading

// window_test.go holds spc-2609020626048722: the three-part committed entry
// (the object set, the kinds within it, the window it measures at), the
// declaration on the size report, the entailment mechanism proportion, and the
// bounds the run record states.
//
// Every test here cites the section it holds. The design framework v4's
// section 13 fixes the object set and the closing run's identity; the readings
// companion v4's sections 5.2, 5.6 and 6.6 fix the widening object, the
// glossary bound and the mechanism proportion; adr-2609021016286571 fixes the
// invocation; the divergence register's entries 1, 16, 17, 18, 24 and 25 record
// what the materials chose where the documents are open.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// v2Preset renders a schema version 2 preset file from one position's entry, so
// a test states the entry it is about and nothing else.
func v2Preset(entries string) string {
	return "{\n  \"schema_version\": 2,\n  \"positions\": {\n" + entries + "\n  }\n}\n"
}

// v2Entry renders one position entry with a declared window.
func v2Entry(position, kinds, records, paths string, tokens int) string {
	return fmt.Sprintf(`    %q: {"object": {"records": [%s], "paths": [%s]}, "kinds": [%s], `+
		`"window": {"tokens_est": %d, "measured_tokens_est": 1, "measured_bytes": 4, `+
		`"measured_at": "0abc123"}}`, position, records, paths, kinds, tokens)
}

// TestPresetV2RefusesAPositionWithoutAWindow is ac-1 (spc-2609020626048722,
// "Loading: two versions, one strict").
//
// A declaration is what the window eval holds an entry to, so an entry carrying
// none is one nothing can be held to — and a silent absence is exactly how the
// instrument reached release measured by nobody. The refusal names the POSITION,
// because a file with three entries and one omission is a file whose author has
// to be told which.
func TestPresetV2RefusesAPositionWithoutAWindow(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath, `{
  "schema_version": 2,
  "positions": {
    "widening": {"object": {"records": [], "paths": []}, "kinds": ["brief-section"],
      "window": {"tokens_est": 10, "measured_tokens_est": 1, "measured_bytes": 4, "measured_at": "0abc123"}},
    "entailment": {"object": {"records": [], "paths": []}, "kinds": ["spec"]}
  }
}`)
	gitCommitAll(t, root)

	_, err := LoadPresets(root)
	if err == nil {
		t.Fatal("a version 2 entry declaring no window loaded; the eval has nothing to hold " +
			"that position to, and an undeclared window is how the instrument reached release " +
			"measured by nobody")
	}
	for _, want := range []string{string(PositionEntailment), "window"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), string(PositionWidening)) {
		t.Errorf("the refusal names the position that DOES declare a window, so it does not "+
			"tell its reader which entry to fix: %v", err)
	}
}

// TestPresetV1LoadsAndReportsNoWindow is ac-2 (spc-2609020626048722, "Loading:
// two versions, one strict"; cond-2609020626048715, which is why this change's
// impact is `fix`).
//
// The schema move is opt-in by version: an adopter's committed version 1 file
// goes on loading, its kinds, records and paths read as the entry set, and the
// size report says no window is declared rather than rendering a zero a reader
// would take for a bound.
func TestPresetV1LoadsAndReportsNoWindow(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath, `{
  "schema_version": 1,
  "presets": {
    "default": {"positions": {"widening":
      {"kinds": ["brief-section", "glossary-term"], "records": [], "paths": []}}}
  }
}`)
	gitCommitAll(t, root)

	pf, err := LoadPresets(root)
	if err != nil {
		t.Fatalf("a version 1 file holding one preset was refused: %v", err)
	}
	if pf.SchemaVersion != 1 {
		t.Errorf("the loaded file reports schema version %d, want 1", pf.SchemaVersion)
	}
	entry, err := PresetFor(pf, PositionWidening)
	if err != nil {
		t.Fatalf("the version 1 entry did not resolve: %v", err)
	}
	if len(entry.Kinds) != 2 {
		t.Errorf("the version 1 entry carries %d kind(s), want the two it names", len(entry.Kinds))
	}
	if entry.Window != nil {
		t.Errorf("a version 1 entry declared a window: %+v", entry.Window)
	}
	if w := PresetWindow(pf, PositionWidening); w != nil {
		t.Errorf("PresetWindow returned %+v for a version 1 file; there is no declaration to "+
			"return, and a zero would read as a bound", w)
	}

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble under a version 1 file: %v", err)
	}
	if res.Size.Window != nil {
		t.Errorf("the size report of a version 1 run carries a window: %+v", res.Size.Window)
	}
	if res.Size.ExceedsWindow {
		t.Error("the size report of a version 1 run reports exceeding a window it does not have")
	}
}

// TestPresetV1WithTwoPresetsRefusesNamingThem is ac-2's other half and
// cond-2609021004074586: nothing at the invocation can choose between two
// presets, and the design admits no operand that could, so a version 1 file
// holding two refuses NAMING them rather than picking one by map order.
func TestPresetV1WithTwoPresetsRefusesNamingThem(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath, `{
  "schema_version": 1,
  "presets": {
    "alpha": {"positions": {"widening": {"kinds": ["doc"], "records": [], "paths": ["docs"]}}},
    "beta": {"positions": {"widening": {"kinds": ["spec"], "records": [], "paths": []}}}
  }
}`)
	gitCommitAll(t, root)

	_, err := LoadPresets(root)
	if err == nil {
		t.Fatal("a version 1 file holding two presets loaded; whichever the loader picked would " +
			"be a resolution order deciding silently")
	}
	for _, want := range []string{"alpha", "beta"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
}

// TestPresetV1RefusesAWindow is the guard that keeps the two shapes from being
// mixed into a third nobody specified (spc-2609020626048722, "Loading: two
// versions, one strict"). A version 1 file declaring a window is refused as an
// unknown field by the decoder for the version it claims.
func TestPresetV1RefusesAWindow(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath, `{
  "schema_version": 1,
  "presets": {
    "default": {"positions": {"widening":
      {"kinds": ["brief-section"], "records": [], "paths": [],
       "window": {"tokens_est": 10, "measured_tokens_est": 1, "measured_bytes": 4,
                  "measured_at": "0abc123"}}}}
  }
}`)
	gitCommitAll(t, root)

	if _, err := LoadPresets(root); err == nil {
		t.Fatal("a version 1 file declaring a window loaded; the declaration would be read by " +
			"nothing, which is a preset that silently does less than it says")
	}
}

// TestSizeReportCarriesTheDeclarationAndTheVerdict is ac-3 and ac-4 at the
// report (spc-2609020626048722, "The size report carries the declaration").
//
// The mutation is the point: a declaration one below the measured figure flips
// ExceedsWindow, so a report that stopped comparing fails here rather than
// passing quietly with the declaration echoed back.
func TestSizeReportCarriesTheDeclarationAndTheVerdict(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath,
		v2Preset(v2Entry("widening", `"brief-section"`, "", "", 1_000_000)))
	gitCommitAll(t, root)

	within, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if within.Size.Window == nil || within.Size.Window.TokensEst != 1_000_000 {
		t.Fatalf("the size report does not carry the declaration: %+v", within.Size.Window)
	}
	if within.Size.ExceedsWindow {
		t.Errorf("a run of ~%d estimated tokens reports exceeding a declaration of 1,000,000",
			within.Size.TokensEst)
	}

	writeFile(t, root, PresetConfigPath,
		v2Preset(v2Entry("widening", `"brief-section"`, "", "", within.Size.TokensEst-1)))
	gitCommitAll(t, root)

	over, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble under the lowered declaration: %v", err)
	}
	if !over.Size.ExceedsWindow {
		t.Errorf("a run of ~%d estimated tokens against a declaration of %d does not report "+
			"exceeding it; the report echoes the declaration without comparing it",
			over.Size.TokensEst, over.Size.Window.TokensEst)
	}
}

// TestSizeReportNamesAnOverTargetTotal is ac-8 (spc-2609020626048722, "The
// target is stated, not enforced"; the maintainer's ruling of 2026-09-02,
// divergence register 24).
//
// Two hundred thousand estimated tokens is a target an entry aims at, never a
// limit: an assembly over it is stated to the operator and nothing is refused.
func TestSizeReportNamesAnOverTargetTotal(t *testing.T) {
	if TargetTokens != 200_000 {
		t.Fatalf("the target is %d; the maintainer's ruling of 2026-09-02 fixes it at two "+
			"hundred thousand estimated tokens", TargetTokens)
	}
	under := sizeReport([]candidate{{kind: KindDoc, text: strings.Repeat("x", 1000)}},
		PositionWidening, nil)
	if under.OverTarget {
		t.Errorf("a report of ~%d estimated tokens is marked over a target of %d",
			under.TokensEst, TargetTokens)
	}
	over := sizeReport([]candidate{{
		kind: KindDoc,
		text: strings.Repeat("x", int(float64(TargetTokens)*tokenBytesPerToken)+1000),
	}}, PositionWidening, nil)
	if !over.OverTarget {
		t.Errorf("a report of ~%d estimated tokens is not marked over a target of %d",
			over.TokensEst, TargetTokens)
	}
}

// TestAnObjectSetNamingOneRecordCarriesThatRecordAlone is ac-5
// (spc-2609020626048722, "The three parts, and how they select"; framework 13's
// object set).
//
// itd-199 shipped with every committed entry carrying an empty record list, so
// the POSITIVE half of the selector grammar had never carried a record's
// material. Here two shipped intents stand in the fixture and the entry names
// one: the bundle carries that intent's projected fields, no other intent's,
// and the manifest lists exactly those items.
func TestAnObjectSetNamingOneRecordCarriesThatRecordAlone(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/intents/shipped/itd-7-a-second-shipped-intent.md",
		"---\nid: itd-7\nspec_id: spc-1\n---\n\n# A second shipped intent\n\n"+
			"## Press Release\n\nThe second promise.\n\n"+
			"## Acceptance Criteria\n\n- Given a second state, when it runs, then it holds.\n")
	writeFile(t, root, PresetConfigPath,
		v2Preset(v2Entry("detection", `"intent-projection"`, `"itd-1"`, "", 1_000_000)))
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionDetection, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble under an object set naming one record: %v", err)
	}
	if len(res.Manifest.Items) == 0 {
		t.Fatal("an object set naming one record assembled nothing")
	}
	fields := []string{}
	for _, m := range res.Manifest.Items {
		if !pathNamesRecord(m.Path, "itd-1") {
			t.Errorf("the assembly passed %s, which the object set does not name; an entry "+
				"naming one record carries that record's material and no other's", m.Path)
		}
		fields = append(fields, m.Field)
	}
	sort.Strings(fields)
	// itd-1 carries four of the five projected fields; the fixture's shipped
	// intent has no spec_id heading, so `spec_id` resolves as a frontmatter key.
	want := []string{"Acceptance Criteria", "Mechanism", "Press Release", "Scope Conditions", "spec_id"}
	if strings.Join(fields, ",") != strings.Join(want, ",") {
		t.Errorf("the bundle carries the fields %v, want %v; the projection is a contract and "+
			"the manifest lists exactly the items it yields", fields, want)
	}
	if len(res.Bundle.Items) != len(res.Manifest.Items) {
		t.Errorf("the bundle carries %d item(s) and the manifest lists %d",
			len(res.Bundle.Items), len(res.Manifest.Items))
	}
}

// TestAnObjectSetSelectsOnlyListedRecords is ac-6 (spc-2609020626048722, "The
// three parts, and how they select").
//
// The narrowing is per ROW: a record row the object set reaches is narrowed to
// the records it names, and a constraint source no part of the object set names
// travels whole. That rule is what lets one object set stay one fact in every
// entry while each position's kinds follow its own definition.
func TestAnObjectSetSelectsOnlyListedRecords(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/intents/shipped/itd-7-a-second-shipped-intent.md",
		"---\nid: itd-7\nspec_id: spc-1\n---\n\n# A second shipped intent\n\n"+
			"## Press Release\n\nThe second promise.\n")
	writeFile(t, root, PresetConfigPath,
		v2Preset(v2Entry("entailment", `"intent-projection", "glossary-term"`,
			`"itd-1"`, "", 1_000_000)))
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionEntailment, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	glossary, shipped := 0, 0
	const shippedDir = ".abcd/development/intents/shipped/"
	for _, m := range res.Manifest.Items {
		switch {
		case m.Kind == KindGlossaryTerm:
			glossary++
		case strings.HasPrefix(m.Path, shippedDir):
			shipped++
			if !pathNamesRecord(m.Path, "itd-1") {
				t.Errorf("the assembly passed %s from the shipped row, which the object set does "+
					"not name", m.Path)
			}
		}
	}
	if shipped == 0 {
		t.Error("the assembly passed no item from the shipped row, so the record narrowing " +
			"was asserted over nothing")
	}
	if glossary == 0 {
		t.Error("the assembly passed no glossary term; a constraint source no part of the " +
			"object set names is handed whole, and an entry naming the kind must receive it")
	}
}

// TestAConstraintSourceKindIsHandedWhole is ac-6's other half and the sentence
// the object-set rule turns on (spc-2609020626048722): "a record row is
// admitted whole when the object set names none of its records".
//
// The fixture's disciplines are named by no object set here, so they arrive
// whole: a discipline the run is not about is context the reading reads
// AGAINST, which is what a constraint source is.
//
// The drafts and planned rows are the ruled exception, and this test states
// both halves so the rule and its exception cannot drift apart. The maintainer
// ruled on 2026-09-02, at the Phase A review, that the entailment reading is
// handed the object set only: a draft or a planned intent is the claim record
// the position reads rather than context it reads against, so those two rows
// narrow by the entry's record list always and the companion's section 6.2
// admissibility survives as the `admit_drafts_and_planned` switch, default off
// (divergence register 1 as corrected; objectset_test.go holds the switch).
func TestAConstraintSourceKindIsHandedWhole(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath,
		v2Preset(v2Entry("entailment", `"intent-projection", "discipline"`,
			`"itd-1"`, "", 1_000_000)))
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionEntailment, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	whole := map[string]bool{}
	for _, m := range res.Manifest.Items {
		whole[path.Dir(m.Path)] = true
	}
	if !whole[".abcd/development/intents/disciplines"] {
		t.Error("no item arrived from .abcd/development/intents/disciplines; the object set " +
			"names no record under that row's source, so the row is admitted whole")
	}
	for _, dir := range []string{
		".abcd/development/intents/drafts",
		".abcd/development/intents/planned",
	} {
		if whole[dir] {
			t.Errorf("an item arrived from %s under an entry naming no record there; those "+
				"two rows narrow by the entry's record list always, so a draft or a planned "+
				"intent travels only when the entry names it (ruled 2026-09-02)", dir)
		}
	}
}

// TestAPathSelectsOnlyTheEntrysKinds is ac-6's third half and the rule that
// replaces spc-69's union (spc-2609020626048722): a path in the object set now
// selects only the kinds the entry admits, where a path clause used to select
// every kind beneath it.
func TestAPathSelectsOnlyTheEntrysKinds(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, "build/generated.go", "package build\n\nfunc Generated() {}\n")
	writeFile(t, root, "build/notes.md", "# Build notes\n\nHow it is generated.\n")
	writeFile(t, root, PresetConfigPath,
		v2Preset(v2Entry("detection", `"doc"`, "", `"build"`, 1_000_000)))
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionDetection, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	got := map[string]Kind{}
	for _, m := range res.Manifest.Items {
		got[m.Path] = m.Kind
	}
	if got["build/notes.md"] != KindDoc {
		t.Errorf("build/notes.md arrived as %q, want %q; the entry names the path and the doc "+
			"kind", got["build/notes.md"], KindDoc)
	}
	if _, present := got["build/generated.go"]; present {
		t.Error("build/generated.go arrived under an entry naming only the doc kind; a path " +
			"selects the kinds the entry admits and no others")
	}
}

// TestShippedObjectRecordsAllResolve holds the committed object set to ids that
// exist (spc-2609020626048722, "The default object set, and why").
//
// The list is maintained by hand and held by review, and the eval is silent on
// whether it covers the workstream. What it cannot be silent about is a
// MISTYPED id, which would select nothing under cover of the kinds and quietly
// narrow what a position reads.
func TestShippedObjectRecordsAllResolve(t *testing.T) {
	root := repoRoot(t)
	pf, err := LoadPresets(root)
	if err != nil {
		t.Fatalf("the committed preset file does not load: %v", err)
	}
	seen := map[string]bool{}
	for _, pos := range AssemblingPositions() {
		entry, err := PresetFor(pf, pos)
		if err != nil {
			t.Fatalf("the committed file names no entry for %s: %v", pos, err)
		}
		for _, id := range entry.Object.Records {
			if seen[id] {
				continue
			}
			seen[id] = true
			if !recordFileExists(t, root, id) {
				t.Errorf("the committed object set names %s, which no record file is named for; "+
					"a mistyped id selects nothing under cover of the kinds", id)
			}
		}
		for _, p := range entry.Object.Paths {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
				t.Errorf("the committed object set names the path %q at %s, which does not "+
					"exist: %v", p, pos, err)
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("the committed object set names no record, so this test proves nothing")
	}
}

// recordFileExists reports whether a record store holds a file named for the id.
func recordFileExists(t *testing.T, root, id string) bool {
	t.Helper()
	found := false
	for _, store := range []string{".abcd/development/intents", ".abcd/development/specs"} {
		base := filepath.Join(root, filepath.FromSlash(store))
		_ = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr // a missing store is reported by the caller's assertion
			}
			if pathNamesRecord(filepath.ToSlash(d.Name()), id) {
				found = true
			}
			return nil
		})
	}
	return found
}

// TestShippedEntriesDeclareMeasuredFigures is ac-7 (spc-2609020626048722, "The
// preset file at schema version 2").
//
// A declaration without the measurement behind it is a number nobody can check.
// The three measured_* keys are disclosure and nothing gates on them beyond
// shape — measured_at's reachability is deliberately unchecked, because a squash
// or rebase merge rewrites a branch sha out of existence and a disclosure that
// failed the build after one would teach people to omit it.
func TestShippedEntriesDeclareMeasuredFigures(t *testing.T) {
	pf, err := LoadPresets(repoRoot(t))
	if err != nil {
		t.Fatalf("the committed preset file does not load: %v", err)
	}
	if pf.SchemaVersion != PresetSchemaVersion {
		t.Errorf("the committed file is at schema version %d, want %d",
			pf.SchemaVersion, PresetSchemaVersion)
	}
	for _, pos := range AssemblingPositions() {
		w := PresetWindow(pf, pos)
		if w == nil {
			t.Errorf("the committed entry for %s declares no window", pos)
			continue
		}
		if w.TokensEst <= 0 {
			t.Errorf("the committed entry for %s declares a window of %d", pos, w.TokensEst)
		}
		if w.MeasuredTokensEst <= 0 || w.MeasuredBytes <= 0 {
			t.Errorf("the committed entry for %s declares the measurement %d token(s) over %d "+
				"byte(s); a declaration states the figure it was measured at",
				pos, w.MeasuredTokensEst, w.MeasuredBytes)
		}
		if !targetRe.MatchString(w.MeasuredAt) {
			t.Errorf("the committed entry for %s names %q as the commit it was measured on",
				pos, w.MeasuredAt)
		}
		if w.TokensEst < w.MeasuredTokensEst {
			t.Errorf("the committed entry for %s declares %d and measured %d; the declaration "+
				"is the figure measured, rounded up", pos, w.TokensEst, w.MeasuredTokensEst)
		}
	}
}

// TestEntailmentSizeReportStatesTheMechanismProportion is ac-9 (the readings
// companion's section 6.6; divergence register 16; iss-2609012259585189).
//
// The companion bounds the entailment reading by how many intents carry a
// mechanism claim and asks that the proportion be reported beside the findings.
// The count is per FILE and exhaustive: a projected intent either stated a
// claim, stated the nullity, or carried no section at all.
func TestEntailmentSizeReportStatesTheMechanismProportion(t *testing.T) {
	root := fixtureRepo(t)
	// itd-1, the fixture's own shipped intent, states a mechanism. Two more
	// stand beside it: one carrying the nullity and one carrying no section.
	writeFile(t, root, ".abcd/development/intents/shipped/itd-7-none-stated.md",
		"---\nid: itd-7\n---\n\n# None stated\n\n## Press Release\n\nA promise.\n\n"+
			"## Mechanism\n\nNone stated.\n")
	writeFile(t, root, ".abcd/development/intents/shipped/itd-8-no-section.md",
		"---\nid: itd-8\n---\n\n# No section\n\n## Press Release\n\nAnother promise.\n")
	writeFile(t, root, PresetConfigPath,
		v2Preset(v2Entry("entailment", `"intent-projection"`,
			`"itd-1", "itd-7", "itd-8"`, "", 1_000_000)))
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionEntailment, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	m := res.Size.Mechanism
	if m == nil {
		t.Fatal("the entailment size report states no mechanism proportion; the readings " +
			"companion's section 6.6 asks for it beside the findings")
	}
	// The entry names three shipped intents and no draft or planned one, and
	// since the ruling of 2026-09-02 those two rows narrow by the entry's
	// record list, so the fixture's draft and planned intent do not travel
	// here. What is asserted is the three counts against the three intents the
	// object set names, summing exactly over whatever the entry hands.
	if m.Stated != 1 {
		t.Errorf("the report states %d intent(s) carrying a mechanism claim, want 1", m.Stated)
	}
	if m.NoneStated != 1 {
		t.Errorf("the report states %d intent(s) carrying the nullity, want 1", m.NoneStated)
	}
	if m.Absent < 1 {
		t.Errorf("the report states %d intent(s) carrying neither, want at least 1", m.Absent)
	}
	if got := m.Stated + m.NoneStated + m.Absent; got != m.Intents {
		t.Errorf("the three counts sum to %d over %d projected intent(s); the vocabulary is "+
			"exhaustive over the files", got, m.Intents)
	}
}

// TestMechanismProportionIsAbsentAtOtherPositions is ac-9's other half: the
// companion asks for the proportion beside the ENTAILMENT reading's findings
// and nowhere else, so a report at any other position carries no such
// statement.
func TestMechanismProportionIsAbsentAtOtherPositions(t *testing.T) {
	root := fixtureRepo(t)
	for _, p := range AssemblingPositions() {
		res := assembleFixture(t, root, p)
		if p == PositionEntailment {
			if res.Size.Mechanism == nil {
				t.Error("the entailment report states no mechanism proportion")
			}
			continue
		}
		if res.Size.Mechanism != nil {
			t.Errorf("the %s report states a mechanism proportion (%+v); the readings "+
				"companion's section 6.6 asks for it beside the entailment reading's findings "+
				"alone", p, res.Size.Mechanism)
		}
	}
}

// TestDefaultItemSetsMatchTheRecordedDigests pins the item set each committed
// entry hands, so what a position reads moves by a COMMIT to the preset file and
// by nothing else — which is what resolves iss-2608311501240566 (divergence
// register 25; the maintainer's ruling of 2026-09-02 that the preset entry is
// the one configuration surface for a position's object set, kinds and window).
//
// The digest is taken over the fixture repository carrying the committed
// entries verbatim, not over this repository: a unit test cannot assume the
// working tree is clean, and the assembler rightly refuses a dirty one. What is
// pinned is therefore the entry's own selection behaviour, which is the half
// that moves when an entry moves; the figures over this repository's own tree
// are the window eval's, which assembles a clean clone of HEAD.
func TestDefaultItemSetsMatchTheRecordedDigests(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(PresetConfigPath)))
	if err != nil {
		t.Fatalf("read %s: %v", PresetConfigPath, err)
	}
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath, string(raw))
	writeFile(t, root, "internal/core/lint/lint.go", "package lint\n\nfunc Lint() {}\n")
	writeFile(t, root, "internal/core/lint/lint_test.go", "package lint\n\nfunc TestLint() {}\n")
	writeFile(t, root, "commands/reading.md", "# reading\n\nThe verb's page.\n")
	gitCommitAll(t, root)

	// Recorded before the change landed, and moved only in the same diff as a
	// change to that position's entry.
	// All four moved with the comparative channel (adr-2609021016272867,
	// spc-2609020626039834): the fixture gained the criteria discipline itd-191,
	// which every entry admitting the discipline kind now hands, and the
	// comparative position has an item set for the first time — its criteria
	// discipline and the derived run's six candidate items.
	// The entailment digest moved again on 2026-09-02 with the maintainer's
	// Phase A ruling: the drafts and planned rows narrow by the entry's record
	// list, so the fixture's draft and planned intent — which that entry does
	// not name — no longer travel there (divergence register 1 as corrected).
	want := map[Position]string{
		PositionWidening:    "e5d389881e1c7b30720f31c9a5374755249de77a76f514b37dab82d8f22891bf",
		PositionEntailment:  "43b59fb91071189ee107468272336d20dafbbd53785efb5bc1491b8a3a1fe878",
		PositionComparative: "5559b8a5d6126cdfa831c866394e5d15a3995f5e87a5f09268883679d2b61cfc",
		PositionDetection:   "7b37ef14556a385fc65fac148059c815799bfe2aa17a732cd9799ee30aecdb85",
	}
	seen := map[string]Position{}
	for _, p := range AssemblingPositions() {
		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: p, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			t.Fatalf("assemble at %s under the committed entry: %v", p, err)
		}
		got := itemSetDigest(res.Manifest)
		if got != want[p] {
			t.Errorf("the %s entry now hands a different item set: digest %s, recorded %s. "+
				"The item set a position is handed moves by a commit to %s and by nothing "+
				"else, so move this digest in the same diff as the entry",
				p, got, want[p], PresetConfigPath)
		}
		if other, dup := seen[got]; dup {
			t.Errorf("the %s and %s entries hand the same item set; the per-position entry is "+
				"what makes each position's item set its own", other, p)
		}
		seen[got] = p
	}
}

// itemSetDigest hashes a manifest's item set: path, field and kind, in order.
// It is deliberately not the manifest hash, which carries the run id and the
// assembler version and would move on every unrelated change.
func itemSetDigest(m Manifest) string {
	var b strings.Builder
	for _, it := range m.Items {
		fmt.Fprintf(&b, "%s\x00%s\x00%s\n", it.Path, it.Field, it.Kind)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// TestAnEntrysItemSetHoldsTheExclusionFloor assembles under each committed
// entry over the fixture, which plants every warm class the record names, and
// requires none of them in the bundle.
//
// The entries are what a run of this repository applies, and an entry cannot
// quiet a floor breach: the deny, the floor and the dirty gate all run over the
// UNFILTERED walk before the entry narrows it (spc-69, kept here). This is that
// property held against the entries as committed rather than against a
// hand-written one.
func TestAnEntrysItemSetHoldsTheExclusionFloor(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(PresetConfigPath)))
	if err != nil {
		t.Fatalf("read %s: %v", PresetConfigPath, err)
	}
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath, string(raw))
	writeFile(t, root, "internal/core/lint/lint.go", "package lint\n\nfunc Lint() {}\n")
	gitCommitAll(t, root)

	planted := []string{
		sentinelEvidence, sentinelDecision, sentinelIssue, sentinelAuditNotes,
		sentinelWhyItMatter, sentinelOrigin, sentinelSuperseded, sentinelPlan,
		sentinelPriorRun, sentinelDefinition, sentinelLapse,
	}
	for _, p := range AssemblingPositions() {
		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: p, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			t.Fatalf("assemble at %s under the committed entry: %v", p, err)
		}
		text := bundleText(res.Bundle)
		for _, token := range planted {
			if strings.Contains(text, token) {
				t.Errorf("the committed %s entry passed the warm class %s; an entry narrows "+
					"what a reading is handed and never widens what the floor refuses", p, token)
			}
		}
	}
}

// TestTwoAssembliesOfOneEntryAreByteIdentical holds the reproducibility the
// manifest promises: a run is reproducible from the commit it names and the
// entry it records, and the invocation carries nothing a re-run could differ on
// (adr-2609021016286571).
func TestTwoAssembliesOfOneEntryAreByteIdentical(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(PresetConfigPath)))
	if err != nil {
		t.Fatalf("read %s: %v", PresetConfigPath, err)
	}
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath, string(raw))
	gitCommitAll(t, root)

	for _, p := range AssemblingPositions() {
		first := assembleFixture(t, root, p)
		second := assembleFixture(t, root, p)
		if string(mustEncodeBundle(t, first.Bundle)) != string(mustEncodeBundle(t, second.Bundle)) {
			t.Errorf("two assemblies of the committed %s entry produced different bundles", p)
		}
		a := decodedManifest(t, first.Manifest)
		b := decodedManifest(t, second.Manifest)
		for _, key := range []string{"preset", "preset_hash", "items", "exclusions"} {
			if string(a[key]) != string(b[key]) {
				t.Errorf("two assemblies of the committed %s entry disagree on %q", p, key)
			}
		}
	}
}

// TestAppliedEntryHashIsTheEntrysOwn holds the pin between the opening and the
// closing run (framework 13, "the same object set"; cond-2609021140328523;
// divergence register 18). The manifest records the entry applied and its hash,
// and the hash is the entry's own content — so a commit that moves any part of
// an entry moves the hash a later run compares against.
func TestAppliedEntryHashIsTheEntrysOwn(t *testing.T) {
	before := PositionEntry{
		Kinds:  []Kind{KindSpec},
		Object: ObjectSet{Records: []string{"itd-1"}, Paths: []string{"docs"}},
	}
	after := PositionEntry{
		Kinds:  []Kind{KindSpec},
		Object: ObjectSet{Records: []string{"itd-1", "itd-2"}, Paths: []string{"docs"}},
	}
	ha, err := before.Applied().Hash()
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	hb, err := after.Applied().Hash()
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if ha == hb {
		t.Error("adding a record to the object set left the applied entry's hash unmoved; the " +
			"closing run is pinned to the opening run's hash, so a moved object set has to move it")
	}
	// And a window is not part of it: the declaration rides on the size report,
	// not on the manifest, so recalibrating a window does not unpin a closing
	// run from the object set the opening run read.
	windowed := before
	windowed.Window = &Window{TokensEst: 1, MeasuredTokensEst: 1, MeasuredBytes: 1, MeasuredAt: "0abc123"}
	hw, err := windowed.Applied().Hash()
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hw != ha {
		t.Error("declaring a window moved the applied entry's hash; the window rides on the " +
			"size report and the manifest's shape and hash do not move with it")
	}
}

// TestPresetV2RefusesAMalformedMeasurement holds the shape checks the
// declaration's disclosure carries (spc-2609020626048722, "The preset file at
// schema version 2").
func TestPresetV2RefusesAMalformedMeasurement(t *testing.T) {
	for name, entry := range map[string]string{
		"a declaration of zero": v2Entry("widening", `"brief-section"`, "", "", 0),
		"a measured_at that is not a sha": `    "widening": {"object": {"records": [], ` +
			`"paths": []}, "kinds": ["brief-section"], "window": {"tokens_est": 10, ` +
			`"measured_tokens_est": 1, "measured_bytes": 4, "measured_at": "main"}}`,
		"a negative measurement": `    "widening": {"object": {"records": [], "paths": []}, ` +
			`"kinds": ["brief-section"], "window": {"tokens_est": 10, ` +
			`"measured_tokens_est": -1, "measured_bytes": 4, "measured_at": "0abc123"}}`,
		// The comparative-entry refusal is withdrawn (adr-2609021016272867), and
		// TestComparativePresetIsAdmitted holds the withdrawal. What is refused
		// at that position now is an entry naming the candidate kind, because
		// the candidate set is derived from the record and never selected by an
		// entry.
		"the candidate kind at the comparative position": v2Entry("comparative", `"candidate"`, "", "", 10),
		"an unknown kind":             v2Entry("widening", `"warm-ledger"`, "", "", 10),
		"a record id that is not one": v2Entry("widening", `"doc"`, `"nope-1"`, "", 10),
		"a denied path":               v2Entry("widening", `"doc"`, "", `".abcd/.work.local"`, 10),
		"a whole-repository path":     v2Entry("widening", `"doc"`, "", `"."`, 10),
	} {
		t.Run(name, func(t *testing.T) {
			root := fixtureRepo(t)
			writeFile(t, root, PresetConfigPath, v2Preset(entry))
			gitCommitAll(t, root)
			if _, err := LoadPresets(root); err == nil {
				t.Fatalf("%s was accepted", name)
			}
		})
	}
}

// TestDuplicatePositionKeysAreRefused is the version 2 half of the review-evasion
// refusal spc-69 named at version 1: Go's decoder takes the last duplicate
// silently, so a second entry for one position low in the file would replace the
// reviewed one while a reviewer reading top-down sees the first.
func TestDuplicatePositionKeysAreRefused(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath, v2Preset(
		v2Entry("widening", `"brief-section"`, "", "", 10)+",\n"+
			v2Entry("widening", `"doc"`, "", `"docs"`, 10)))
	gitCommitAll(t, root)

	_, err := LoadPresets(root)
	if err == nil {
		t.Fatal("a file naming one position twice loaded; the last would win silently")
	}
	if !strings.Contains(err.Error(), string(PositionWidening)) {
		t.Errorf("the refusal does not name the duplicated position: %v", err)
	}
}

// TestRunRecordCarriesTheStatedBounds is the run record's `bounds` list
// (spc-2609020626048722, "The stated bounds on the run record";
// cond-2609021140329660 and cond-2609021140328523; divergence register 17 and
// 18; ruling M5, that a reading which departs from the one the documents state
// is stated as a bound rather than passed off).
//
// Two statements, both written by the verb from the MANIFEST and never from the
// operator: the glossary bound the readings companion's section 5.6 states in
// advance as three to six terms, and a preset-hash mismatch against a prior
// committed run at the same position, which the framework's section 13 makes a
// bound on the comparison rather than something to repair.
func TestRunRecordCarriesTheStatedBounds(t *testing.T) {
	items := func(glossary int) []ManifestItem {
		out := make([]ManifestItem, 0, glossary)
		for i := range glossary {
			out = append(out, ManifestItem{
				ItemKey: fmt.Sprintf("itm-%04d", i+1),
				Path:    fmt.Sprintf(".abcd/development/brief/glossary/core/term-%d.md", i),
				Kind:    KindGlossaryTerm, Scan: ScanParsed, Bytes: 1,
				SHA256: sha256Hex([]byte("x")),
			})
		}
		return out
	}

	t.Run("seven terms after a prior run under another hash", func(t *testing.T) {
		root := t.TempDir()
		r, err := os.OpenRoot(root)
		if err != nil {
			t.Fatalf("open root: %v", err)
		}
		defer r.Close()
		writeFile(t, root, ReadingsRecordDir+"/rdg-1/run.json",
			`{"_type":"abcd.reading.run/1","position":"widening"}`)
		writeFile(t, root, ReadingsRecordDir+"/rdg-1/manifest.json",
			`{"preset_hash":"aaaa"}`)

		got := statedBounds(r, Manifest{
			RunID: "rdg-2", Position: PositionWidening, PresetHash: "bbbb", Items: items(7),
		})
		if len(got) != 2 {
			t.Fatalf("the run states %d bound(s), want 2: %v", len(got), got)
		}
		joined := strings.Join(got, "\n")
		for _, want := range []string{"glossary", "5.6", "7 glossary-term", "rdg-1", "aaaa", "bbbb"} {
			if !strings.Contains(joined, want) {
				t.Errorf("the stated bounds do not name %q: %v", want, got)
			}
		}
	})

	t.Run("six terms and no prior run", func(t *testing.T) {
		root := t.TempDir()
		r, err := os.OpenRoot(root)
		if err != nil {
			t.Fatalf("open root: %v", err)
		}
		defer r.Close()
		got := statedBounds(r, Manifest{
			RunID: "rdg-2", Position: PositionWidening, PresetHash: "bbbb", Items: items(6),
		})
		if len(got) != 0 {
			t.Fatalf("a run at the stated bound and under no prior entry stated %d bound(s): %v",
				len(got), got)
		}
		if got == nil {
			t.Error("the bounds list is nil; a run with no bound carries an empty list and " +
				"never an absent key, so a reader can tell it from a run written before the " +
				"field existed")
		}
	})
}

// TestRunRecordBoundsIsNeverAbsent holds the shape half of the rule above: the
// key is present on every run record, empty where nothing was stated.
func TestRunRecordBoundsIsNeverAbsent(t *testing.T) {
	raw, err := json.Marshal(RunRecord{
		Type: RunType, SchemaVersion: SchemaVersion, RunID: "rdg-1",
		Position: PositionWidening, Bounds: []string{},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := doc["bounds"]; !ok {
		t.Fatal("a run record stating no bound carries no `bounds` key; an absent key and an " +
			"empty list are different facts, and a reader must be able to tell them apart")
	}
	if string(doc["bounds"]) != "[]" {
		t.Errorf("the bounds key renders as %s, want []", doc["bounds"])
	}
}
