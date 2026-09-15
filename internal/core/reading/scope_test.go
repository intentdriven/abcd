package reading

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// narrowPresets is a preset file that genuinely narrows, so a test can tell an
// entry that works from one that selects everything by accident.
//
// It holds ONE preset, because the invocation names none and a file holding two
// has nothing to choose between them (adr-2609021016286571,
// cond-2609021004074586). A test that wants an unnarrowed baseline assembles
// under the fixture's own all-selecting file BEFORE writing this one over it.
//
// The entailment entry names the fixture's draft and its planned intent because
// those two rows narrow by the entry's record list ALWAYS (ruled 2026-09-02;
// divergence register 1 as corrected), and a version 1 file cannot declare the
// admissibility switch. The item set is what it was before that rule: the
// fixture carries exactly one of each.
func narrowPresets() string {
	return `{
  "schema_version": 1,
  "presets": {
    "default": {
      "positions": {
        "widening": {"kinds": ["brief-section"], "records": [], "paths": []},
        "entailment": {"kinds": ["intent-projection"], "records": ["itd-2", "itd-3"], "paths": []},
        "comparative": {"kinds": ["discipline"], "records": [], "paths": []},
        "detection": {"kinds": ["spec"], "records": [], "paths": []}
      }
    }
  }
}
`
}

// TestThePresetNarrowsNeverWidens is the property the whole mechanism rests on:
// the applied entry intersects what the position already admits and can only
// narrow it.
//
// Proved against the unnarrowed set rather than against an expected list, so an
// entry that somehow reached a row the table denies fails here rather than
// quietly redefining what the test expects.
func TestThePresetNarrowsNeverWidens(t *testing.T) {
	root := fixtureRepo(t)

	wide := map[Position]map[string]bool{}
	for _, p := range AssemblingPositions() {
		set := map[string]bool{}
		for _, m := range assembleFixture(t, root, p).Manifest.Items {
			set[m.Path] = true
		}
		wide[p] = set
	}

	writeFile(t, root, ".abcd/config/reading-presets.json", narrowPresets())
	gitCommitAll(t, root)

	for _, p := range AssemblingPositions() {
		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: p, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			continue // an entry selecting nothing at this position refuses, which is its own test
		}
		got := map[string]bool{}
		for _, m := range res.Manifest.Items {
			got[m.Path] = true
			if !wide[p][m.Path] {
				t.Errorf("position %s admitted %s, which the unnarrowed assembly did not: an entry "+
					"must intersect the table's admission and only narrow it", p, m.Path)
			}
		}
		// The strict half is not asked at the comparative position, and the
		// reason is the position's own shape rather than a weakening: the only
		// entry-selectable material there is the criteria discipline, which is
		// ONE record the assembler has already narrowed the row to, and the
		// candidates are not an entry's to select at all. An entry that narrowed
		// further would select nothing and refuse, which is a different test
		// (adr-2609021016272867). The never-widens half above still runs there.
		if p == PositionComparative {
			continue
		}
		if len(got) >= len(wide[p]) {
			t.Errorf("position %s did not narrow at all (%d of %d paths); the fixture is not "+
				"exercising the filter", p, len(got), len(wide[p]))
		}
	}
}

// TestASecondNamedPresetIsRefused holds scope condition cond-2609021004074586:
// a second named preset is not a concept this instrument keeps.
//
// Nothing at the invocation names a preset, so a file holding two has nothing
// to choose between them, and whichever the loader picked would be a resolution
// ORDER deciding silently — the failure this package refuses everywhere else it
// can arise. A repository with two calibrations commits one and keeps the other
// in the file's history.
func TestASecondNamedPresetIsRefused(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/config/reading-presets.json", `{
  "schema_version": 1,
  "presets": {
    "cold": {"positions": {"widening": {"kinds": ["brief-section"], "records": [], "paths": []}}},
    "warm": {"positions": {"widening": {"kinds": ["doc"], "records": [], "paths": []}}}
  }
}`)
	_, err := LoadPresets(root)
	if err == nil {
		t.Fatal("a preset file naming two presets was accepted; the invocation names none, so " +
			"there is nothing to choose between them")
	}
	for _, want := range []string{"cold", "warm"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
}

// TestPresetForRefusesAMissingPosition is spc-2609021004075744's rule for the
// resolver: PresetFor returns the one committed entry for the position and
// refuses when the file holds no entry for it.
//
// Refusing beats defaulting. A position served its whole corpus because its
// entry was forgotten is exactly the silent widening the presets exist to
// close, and it would arrive as a reading that is about the wrong thing with
// every gate green.
func TestPresetForRefusesAMissingPosition(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/config/reading-presets.json", `{
  "schema_version": 1,
  "presets": {
    "default": {"positions": {"widening": {"kinds": ["brief-section"], "records": [], "paths": []}}}
  }
}`)
	pf, err := LoadPresets(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, err := PresetFor(pf, PositionWidening); err != nil {
		t.Fatalf("the entry the file names was refused: %v", err)
	}
	_, err = PresetFor(pf, PositionDetection)
	if err == nil {
		t.Fatal("a position the file names no entry for resolved; a run there would assemble " +
			"nothing, or everything, and neither is what the position is about")
	}
	if !strings.Contains(err.Error(), string(PositionDetection)) {
		t.Errorf("the refusal does not name the position: %v", err)
	}
	if !strings.Contains(err.Error(), PresetConfigPath) {
		t.Errorf("the refusal does not name the file to commit the entry in: %v", err)
	}
}

// TestComparativePresetIsAdmitted replaces itd-199's ac-10 refusal, which
// adr-2609021016272867 withdraws.
//
// itd-199 refused a comparative preset entry because the position did not
// assemble: its object was the widening reading's pre-admission output and no
// channel supplied it, so an entry would have described a run that could not
// happen. The channel exists, and what an entry names at that position is the
// repository material passed BESIDE the candidates — the criteria discipline and
// nothing else, because every other row withdraws.
func TestComparativePresetIsAdmitted(t *testing.T) {
	root := fixtureRepo(t)
	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionComparative, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("the comparative position did not assemble: %v", err)
	}
	if res.CandidateRun != fixtureCandidateRun {
		t.Errorf("the assembly derived %q, want %q", res.CandidateRun, fixtureCandidateRun)
	}
	// The entry is loaded and applied like any other, so a file naming one is
	// no longer refused.
	presets, err := LoadPresets(root)
	if err != nil {
		t.Fatalf("LoadPresets over a file naming a comparative entry: %v", err)
	}
	if _, err := PresetFor(presets, PositionComparative); err != nil {
		t.Fatalf("PresetFor(comparative): %v", err)
	}
}

// TestCandidateIsNotAPresetKind is the limit on the withdrawal above: an entry
// selects repository material, and the candidate set is DERIVED. A file naming
// the kind is refused by name rather than accepted and quietly ignored, because
// an entry naming it would read as choosing the candidate set — which nothing at
// the invocation and nothing in a preset may do (adr-2609021016272867).
func TestCandidateIsNotAPresetKind(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath, `{
  "schema_version": 2,
  "positions": {
    "comparative": {"object": {"records": ["itd-191"], "paths": []},
      "kinds": ["discipline", "candidate"],
      "window": {"tokens_est": 1000, "measured_tokens_est": 0, "measured_bytes": 0, "measured_at": "0000000"}}
  }
}
`)
	gitCommitAll(t, root)
	_, err := LoadPresets(root)
	if err == nil {
		t.Fatal("an entry naming the candidate kind was accepted; a candidate is selected by the " +
			"derived widening run, never by an entry")
	}
	for _, want := range []string{string(KindCandidate), "DERIVED"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
}

// TestThreePositionsCarryDistinctItemSets is itd-199 ac-9, and it is the
// measured finding this intent exists to fix: three of the four positions used
// to receive a byte-identical item set.
func TestThreePositionsCarryDistinctItemSets(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/config/reading-presets.json", narrowPresets())
	gitCommitAll(t, root)

	seen := map[string]Position{}
	for _, p := range AssemblingPositions() {
		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: p, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			t.Fatalf("position %s: %v", p, err)
		}
		key := strings.Join(itemPaths(res.Manifest), "\n")
		if other, dup := seen[key]; dup {
			t.Errorf("positions %s and %s carry the same item set; a table that cannot say what a "+
				"reading is about cannot distinguish readings that are about different things", other, p)
		}
		seen[key] = p
	}
}

// TestThePresetCannotReachTheLedgerTier is brief invariant 14, held against the
// applied entry rather than only against the table.
//
// The entries are the widest a committed file could name — every kind at every
// assembling position — because an entry that narrowed would prove nothing
// about the tier it never asked for.
func TestThePresetCannotReachTheLedgerTier(t *testing.T) {
	root := fixtureRepo(t)
	for _, p := range AssemblingPositions() {
		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: p, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			continue
		}
		for _, m := range res.Manifest.Items {
			// The ONE exception, and it is not an entry's doing: at the
			// comparative position the derived widening run's items are the
			// reading's object, admitted by the table and selected by the record
			// (adr-2609021016272867). No entry reaches them — `candidate` is
			// refused as a preset kind — so the property this test holds is
			// untouched: an ENTRY still cannot reach the ledger tier.
			if m.Kind == KindCandidate && p == PositionComparative {
				continue
			}
			if strings.Contains(m.Path, ".work.local") || strings.HasPrefix(m.Path, ".abcd/work/") {
				t.Errorf("the entry at %s admitted %s from the ledger tier", p, m.Path)
			}
		}
	}
}

// TestDraftsStayDeniedAtWidening is itd-199 ac-4: an entry intersects what the
// table admits at the position and never widens it, so the deliberate drafts
// asymmetry survives an entry that names the intent kind.
func TestDraftsStayDeniedAtWidening(t *testing.T) {
	root := fixtureRepo(t)
	// The entry names the intent kind AND a kind the widening position still
	// admits, because since itd-194 no intent-projection row is admitted at
	// widening at all — the drafts and planned rows never were, and the shipped
	// row withdrew — so an entry naming that kind alone selects nothing and the
	// run refuses before it can demonstrate anything about narrowing.
	writeFile(t, root, ".abcd/config/reading-presets.json", `{
  "schema_version": 1,
  "presets": {
    "default": {"positions": {"widening":
      {"kinds": ["intent-projection", "brief-section"], "records": [], "paths": []}}}
  }
}`)
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	for _, m := range res.Manifest.Items {
		if strings.Contains(m.Path, "/drafts/") || strings.Contains(m.Path, "/planned/") ||
			strings.Contains(m.Path, "/shipped/") {
			t.Errorf("an entry naming the intent kind at widening admitted %s; an entry narrows the "+
				"table's admission and never widens it", m.Path)
		}
	}
	if text := bundleText(res.Bundle); strings.Contains(text, sentinelDraftBody) {
		t.Error("the draft body reached a narrowed widening assembly")
	}
}

// TestBundleCarriesThePresetAndManifestCarriesItsHash is itd-199 ac-5 and ac-6,
// carried forward onto the preset path: the bundle states what THIS run was
// handed, and the manifest carries the applied entry's hash.
func TestBundleCarriesThePresetAndManifestCarriesItsHash(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)

	if len(res.Bundle.Preset.Kinds) == 0 && len(res.Bundle.Preset.Records) == 0 &&
		res.Bundle.Preset.LocationNarrowings == 0 {
		t.Error("the bundle states nothing about what it was handed; a reader told its object is " +
			"the shipped tree and handed a subset reports the rest as a finding")
	}
	if res.Manifest.PresetHash == "" {
		t.Error("the manifest carries no preset hash")
	}
	want, err := res.Manifest.Preset.Hash()
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if res.Manifest.PresetHash != want {
		t.Errorf("the manifest's preset hash is %q, want %q", res.Manifest.PresetHash, want)
	}
	// The bundle carries what the reading was given and NOT the auditor's
	// account of it. There is no override stamp anywhere now, and the bundle is
	// the artefact that must never have carried one.
	if strings.Contains(string(mustEncodeBundle(t, res.Bundle)), "overridden") {
		t.Error("the bundle carries an override stamp; nothing can be overridden")
	}
}

func mustEncodeBundle(t *testing.T, b Bundle) []byte {
	t.Helper()
	raw, err := EncodeBundle(b)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return raw
}

// TestPresetCollisionsAreRefused holds the decision that a colliding name is a
// configuration error rather than something a resolution ORDER decides
// silently.
func TestPresetCollisionsAreRefused(t *testing.T) {
	root := fixtureRepo(t)
	for name, body := range map[string]string{
		"a preset named for a kind": `{"schema_version": 1, "presets": {"source": {"positions":
			{"widening": {"kinds": ["doc"], "records": [], "paths": []}}}}}`,
		"a preset naming an unknown kind": `{"schema_version": 1, "presets": {"p": {"positions":
			{"widening": {"kinds": ["warm-ledger"], "records": [], "paths": []}}}}}`,
		"a preset naming a denied path": `{"schema_version": 1, "presets": {"p": {"positions":
			{"widening": {"kinds": [], "records": [], "paths": [".abcd/.work.local"]}}}}}`,
		"a preset naming an absolute path": `{"schema_version": 1, "presets": {"p": {"positions":
			{"widening": {"kinds": [], "records": [], "paths": ["/etc"]}}}}}`,
		"a preset escaping the repository": `{"schema_version": 1, "presets": {"p": {"positions":
			{"widening": {"kinds": [], "records": [], "paths": ["../elsewhere"]}}}}}`,
		// The comparative-entry refusal is withdrawn (adr-2609021016272867), and
		// TestComparativePresetIsAdmitted holds the withdrawal. What is refused
		// there now is the CANDIDATE KIND, because a candidate is selected by the
		// derived widening run and never by an entry.
		"a preset naming the candidate kind": `{"schema_version": 1, "presets": {"p": {"positions":
			{"comparative": {"kinds": ["candidate"], "records": [], "paths": []}}}}}`,
		// `extends` retired with the second preset name, so a file still
		// carrying it is an unknown field rather than a chain to walk.
		"a preset naming extends": `{"schema_version": 1, "presets": {
			"a": {"extends": "b", "positions": {"widening":
				{"kinds": ["doc"], "records": [], "paths": []}}}}}`,
		"a wrong schema version": `{"schema_version": 99, "presets": {"p": {"positions": {}}}}`,
		"an unknown field":       `{"schema_version": 1, "presets": {}, "extra": true}`,
	} {
		writeFile(t, root, ".abcd/config/reading-presets.json", body)
		if _, err := LoadPresets(root); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// TestUncommittedPresetRefuses is itd-199 ac-8: the preset configuration
// reshapes an assembly, so an uncommitted edit to it refuses exactly as an
// uncommitted edit to the record configuration does.
func TestUncommittedPresetRefuses(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/config/reading-presets.json", narrowPresets())
	// deliberately NOT committed
	_, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD",
	})
	if err == nil {
		t.Fatal("an assembly ran against an uncommitted preset configuration")
	}
	if !strings.Contains(err.Error(), PresetConfigPath) {
		t.Errorf("the refusal does not name the preset configuration: %v", err)
	}
}

// TestAPresetSelectingNothingRefuses holds the choice of a refusal over an
// empty bundle. A reader handed an empty assembly has no way to tell "nothing
// matched" from "this object is empty", and would report the second.
func TestAPresetSelectingNothingRefuses(t *testing.T) {
	root := fixtureRepo(t)
	// A well-formed id of an admitted family that names no record in the
	// fixture. It must be a selector the loader ACCEPTS, or this tests a
	// validation refusal instead of the selects-nothing one. The entry names
	// the intent-projection kind because the kinds ADMIT
	// (spc-2609020626048722): an entry naming no kind is refused by the
	// resolver before an assembly is attempted, which is a different refusal
	// from the one this test is about.
	writeFile(t, root, ".abcd/config/reading-presets.json", `{
  "schema_version": 1,
  "presets": {
    "default": {"positions": {"widening":
      {"kinds": ["intent-projection"], "records": ["itd-9999"], "paths": []}}}
  }
}`)
	gitCommitAll(t, root)

	_, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err == nil {
		t.Fatal("an entry selecting no item assembled an empty bundle")
	}
	if !strings.Contains(err.Error(), "nothing to assemble") {
		t.Errorf("the refusal does not say the entry selected nothing: %v", err)
	}
}

// TestRecordSelectorDoesNotMatchByPrefix guards the obvious way to write the
// record matcher wrong: itd-19 must not select itd-198.
func TestRecordSelectorDoesNotMatchByPrefix(t *testing.T) {
	if pathNamesRecord(".abcd/development/intents/planned/itd-198-a-thing.md", "itd-19") {
		t.Error("itd-19 selected itd-198's record; a record selector matches the id, not a prefix of it")
	}
	if !pathNamesRecord(".abcd/development/intents/planned/itd-198-a-thing.md", "itd-198") {
		t.Error("itd-198 did not select its own record")
	}
	if !pathNamesRecord(".abcd/development/decisions/adrs/adr-58.md", "adr-58") {
		t.Error("an id-only basename was not selected")
	}
}

// TestThePresetHashIsOrderIndependent holds the canonical ordering: two entries
// naming the same thing in a different order are one entry, so a manifest's
// preset hash identifies WHAT was selected rather than how the file spelled it.
func TestThePresetHashIsOrderIndependent(t *testing.T) {
	a := AppliedPreset{Selectors: canonicalise([]Selector{{Kind: KindDoc}, {Kind: KindSpec}})}
	b := AppliedPreset{Selectors: canonicalise([]Selector{{Kind: KindSpec}, {Kind: KindDoc}, {Kind: KindDoc}})}
	ha, err := a.Hash()
	if err != nil {
		t.Fatal(err)
	}
	hb, err := b.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if ha != hb {
		t.Errorf("two entries naming the same selectors hashed differently: %s vs %s", ha, hb)
	}
}

// TestBundlePresetCarriesNoRepositoryPath is brief invariant 15 held against
// the applied entry, and it exists because the obvious implementation broke it:
// writing one type into both artefacts put a preset's path selectors — which
// are repository paths — into the reading's own working set
// (iss-2608312058244357).
//
// The manifest MAY carry paths; that is its job, and it is why the bundle can
// be pathless and still checkable. The bundle may not, under any entry.
func TestBundlePresetCarriesNoRepositoryPath(t *testing.T) {
	root := fixtureRepo(t)
	const secret = "internal/core/lint"
	// Two paths: the one whose spelling must not reach the bundle, and `docs`,
	// which the fixture carries doc material under. The object set NARROWS the
	// tree rows (spc-2609020626048722), so an entry naming only a path the
	// fixture does not hold would select nothing and refuse before the bundle
	// this test reads existed.
	writeFile(t, root, ".abcd/config/reading-presets.json", `{
  "schema_version": 1,
  "presets": {"default": {"positions": {"widening":
    {"kinds": ["doc"], "records": [], "paths": ["`+secret+`", "docs"]}}}}
}`)
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	raw, err := EncodeBundle(res.Bundle)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// Checked against the PRESET BLOCK and the structural keys, not against the
	// whole document. Item text legitimately contains path-like strings — a
	// record's prose mentions packages, and Go source carries import paths —
	// and that is pre-existing and inherent to carrying content at all. What
	// invariant 15 forbids is the bundle's own STRUCTURE mapping items to
	// locations: item keys are ordinals and the path mapping lives in the
	// manifest alone. A whole-document scan would pass here only for as long as
	// no fixture happened to mention the path, which is not a property.
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if strings.Contains(string(doc["preset"]), secret) {
		t.Errorf("the bundle's preset block carries the repository path %q; the assembled input is "+
			"the reading's whole working set and no repository path may enter its structure", secret)
	}
	for key, val := range doc {
		if key == "items" {
			continue
		}
		if strings.Contains(string(val), secret) {
			t.Errorf("the bundle's %q carries the repository path %q", key, secret)
		}
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(doc["items"], &items); err != nil {
		t.Fatalf("decode items: %v", err)
	}
	for i, item := range items {
		if _, has := item["path"]; has {
			t.Errorf("bundle item %d carries a path key; the mapping is the manifest's alone", i)
		}
	}
	// The reading must still be TOLD it was narrowed, or it reports the absent
	// material as a finding against the record.
	if res.Bundle.Preset.LocationNarrowings == 0 {
		t.Error("the bundle does not record that a narrowing by location applied, so the reading " +
			"cannot tell it was handed a subset")
	}
	// And the manifest, which is the auditor's, must still carry the path.
	if !strings.Contains(mustEncodeManifest(t, res.Manifest), secret) {
		t.Errorf("the manifest does not carry %q; the auditor's account must stay complete", secret)
	}
}

func mustEncodeManifest(t *testing.T, m Manifest) string {
	t.Helper()
	raw, err := EncodeManifest(m)
	if err != nil {
		t.Fatalf("encode manifest: %v", err)
	}
	return string(raw)
}

// TestNoBundleFieldIsAPresetSelector guards the shape rather than one path: a
// future field added to Selector would flow into the bundle again unless the
// projection is explicit, so the bundle's preset block is pinned to the three
// keys it is allowed to have.
func TestNoBundleFieldIsAPresetSelector(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)
	raw, err := EncodeBundle(res.Bundle)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var doc struct {
		Preset map[string]any `json:"preset"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	allowed := map[string]bool{"kinds": true, "records": true, "location_narrowings": true}
	for key := range doc.Preset {
		if !allowed[key] {
			t.Errorf("the bundle's preset block carries the key %q; only a pathless projection may "+
				"reach a reading", key)
		}
	}
	if _, leaked := doc.Preset["selectors"]; leaked {
		t.Error("the bundle carries raw selectors, which is the shape that leaked a path")
	}
}

// TestGatesSeeTheUnfilteredWalk pins the pipeline's ORDER, which was a claim in
// a comment and nothing more until this test existed.
//
// The preset filter must run AFTER the dirty gate and the exclusion assertion,
// so a narrow entry cannot shrink the set those gates examine. Moving the
// filter earlier passed the entire package before this test: the dirty gate's
// predicate is a pure function of the position and so refuses under either
// order, and the exclusion floor's paths are structurally denied so no
// candidate can breach one. The property was real, load-bearing and
// unfalsifiable — which is exactly the shape itd-195 says to make executable.
//
// The two runs are two COMMITTED files, because the entry is no longer
// something an invocation can vary: the wide run goes first against the
// fixture's all-selecting file, then the narrow file is committed over it.
func TestGatesSeeTheUnfilteredWalk(t *testing.T) {
	root := fixtureRepo(t)

	var sawScoped, sawWide int
	restore := assertExclusionsHook
	t.Cleanup(func() { assertExclusionsHook = restore })

	assertExclusionsHook = func(cands []candidate, ex []Exclusion) error {
		sawWide = len(cands)
		return restore(cands, ex)
	}
	wide, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionDetection, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("wide assembly: %v", err)
	}

	writeFile(t, root, ".abcd/config/reading-presets.json", narrowPresets())
	gitCommitAll(t, root)

	assertExclusionsHook = func(cands []candidate, ex []Exclusion) error {
		sawScoped = len(cands)
		return restore(cands, ex)
	}
	narrow, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionDetection, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("narrow assembly: %v", err)
	}

	if sawScoped != sawWide {
		t.Errorf("the exclusion assertion saw %d candidates under a narrow entry and %d under a "+
			"wide one; the gates must run over the unfiltered walk, or a narrow entry can quiet "+
			"a breach a wide one would have caught", sawScoped, sawWide)
	}
	// And the guard must not be vacuous: the entry has to actually narrow, or
	// the two counts would match for an uninteresting reason.
	if len(narrow.Manifest.Items) >= len(wide.Manifest.Items) {
		t.Fatalf("the narrow entry emitted %d items and the wide one %d; this test proves nothing "+
			"unless the entry narrows", len(narrow.Manifest.Items), len(wide.Manifest.Items))
	}
	if sawScoped <= len(narrow.Manifest.Items) {
		t.Errorf("the exclusion assertion saw %d candidates but the run emitted %d; the gate was "+
			"handed the filtered set, not the walk", sawScoped, len(narrow.Manifest.Items))
	}
}

// TestAnUntrackedPresetRefuses is the committed-and-reviewed half of adr-58,
// which brief invariant 15 now asserts in prose and nothing checked.
//
// The dirty gate cannot supply it: git reports nothing for a file it ignores,
// so a repository gitignoring `.abcd/` — which brief invariant 5 does for
// public visibility — ran against an untracked preset and stamped the run
// `overridden: false`, asserting "ran as reviewed" on an examination that
// established only "git reported no modification".
func TestAnUntrackedPresetRefuses(t *testing.T) {
	root := fixtureRepo(t)
	// The preset must never have been committed, or the dirty gate catches it
	// as a MODIFIED tracked file and this test passes for the wrong reason.
	// So: ignore the directory, remove the fixture's committed preset, commit
	// that removal, and only then write the forged one.
	// The readings line is the fixture's own and is carried forward: the durable
	// run family is ignored so the fixture's run record can follow HEAD without
	// leaving the tree dirty, and the clean-tree precondition below depends on it.
	writeFile(t, root, ".gitignore", ".abcd/config/\n.abcd/development/readings/\n")
	if err := os.Remove(filepath.Join(root, ".abcd", "config", "reading-presets.json")); err != nil {
		t.Fatal(err)
	}
	gitCommitAll(t, root)
	writeFile(t, root, ".abcd/config/reading-presets.json", narrowPresets())

	// Precondition: git must see a clean tree, or the dirty gate is what
	// refuses below and this test proves nothing about trackedness.
	if out := gitStatusPorcelain(t, root); out != "" {
		t.Fatalf("the forged preset is visible to git, so this test cannot isolate the "+
			"tracked check:\n%s", out)
	}

	_, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionDetection, Target: "HEAD", DryRun: true,
	})
	if err == nil {
		t.Fatal("an assembly ran against an untracked, ignored preset; a preset is admitted at " +
			"the invocation because it is committed and reviewed, and this one was neither")
	}
	if !strings.Contains(err.Error(), "not tracked") {
		t.Errorf("the refusal does not say the preset is untracked: %v", err)
	}
}

// TestASymlinkedPresetRefuses closes the other half. A committed symlink IS
// tracked, and git reports nothing when its target changes, so the effective
// configuration would be permanently unreviewed and freely mutable with every
// gate green.
func TestASymlinkedPresetRefuses(t *testing.T) {
	root := fixtureRepo(t)
	outside := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(outside, []byte(narrowPresets()), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, ".abcd", "config", "reading-presets.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, target); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	gitCommitAll(t, root)

	_, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionDetection, Target: "HEAD", DryRun: true,
	})
	if err == nil {
		t.Fatal("an assembly ran from a preset symlinked out of the tree; what it resolves to " +
			"is not what the commit recorded")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("the refusal does not name the symlink: %v", err)
	}
}

// TestDuplicatePresetKeysAreRefused holds F2: Go's decoder takes the last
// duplicate silently, so a second block low in the file would replace the
// reviewed one while a reviewer reading top-down sees the first.
func TestDuplicatePresetKeysAreRefused(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/config/reading-presets.json", `{
  "schema_version": 1,
  "presets": {
    "cold": {"positions": {"widening": {"kinds": ["brief-section"], "records": [], "paths": []}}},
    "cold": {"positions": {"widening": {"kinds": ["source", "test", "doc"], "records": [], "paths": []}}}
  }
}`)
	if _, err := LoadPresets(root); err == nil {
		t.Error("a preset file naming one preset twice was accepted; the last would win silently")
	} else if !strings.Contains(err.Error(), "more than once") {
		t.Errorf("the refusal does not name the duplication: %v", err)
	}
}

// TestPresetPathsRefuseTerminalAndWholeRepoForms holds F4 and the "." case: a
// control character in a path reaches a rendered line, and "." is a scope that
// narrows nothing.
func TestPresetPathsRefuseTerminalAndWholeRepoForms(t *testing.T) {
	for name, bad := range map[string]string{
		"an escape sequence":   "internal/core/\x1b]0;x\alint",
		"a NUL":                "internal/\x00core",
		"a newline":            "internal/core\n",
		"surrounding space":    " internal/core ",
		"backslash separator":  `internal\core\reading`,
		"the whole repo":       ".",
		"the whole repo slash": "./",
		"a case-folded deny":   "Internal/Core/Reading",
	} {
		if err := validPresetPath(bad); err == nil {
			t.Errorf("%s (%q) was accepted as a preset path", name, bad)
		}
	}
	if err := validPresetPath("internal/core/lint"); err != nil {
		t.Errorf("an ordinary repo-relative path was refused: %v", err)
	}
}

// gitStatusPorcelain is the tree's own account of itself, used to prove a
// precondition rather than to assert a result.
func gitStatusPorcelain(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", root, "status", "--porcelain", "-uall")
	// Hermetic, like every other git call in the tests: the ambient environment
	// must not decide what this probe sees (iss-28).
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// TestTheShippedPresetFileIsValid loads the COMMITTED preset configuration, not
// a fixture.
//
// This is the gap fidelity review called the largest in the change, and it was
// exactly right: every other test in this file overwrites
// .abcd/config/reading-presets.json with its own, so the file the whole
// "committed, reviewed, shape-validated" argument rests on — the file the
// assembler now applies with no operand at all — was checked by nothing. It
// could have shipped malformed, or empty at a position, and `make preflight`
// would have stayed green.
//
// One preset, and an entry for every position that assembles
// (adr-2609021016286571). The entry CONTENTS are the preset-windows spec's, not
// this test's; what is held here is the shape the invocation depends on.
func TestTheShippedPresetFileIsValid(t *testing.T) {
	root := repoRoot(t)
	pf, err := LoadPresets(root)
	if err != nil {
		t.Fatalf("the committed preset file does not load: %v", err)
	}
	if len(pf.Positions) != len(AssemblingPositions()) {
		t.Fatalf("the committed file holds %d position entries; one entry per assembling position "+
			"means %d", len(pf.Positions), len(AssemblingPositions()))
	}
	for _, p := range AssemblingPositions() {
		if _, err := PresetFor(pf, p); err != nil {
			t.Errorf("the committed file names no usable entry for %s, so a run there refuses: %v",
				p, err)
		}
	}
}

// TestTheShippedPresetScopesEveryAssemblingPositionDistinctly holds itd-199
// ac-9 against the committed file. The measured finding that intent exists to
// fix is that three positions received a byte-identical item set; an entry set
// that scoped them alike would reproduce it exactly, and only the shipped file
// can say.
func TestTheShippedPresetScopesEveryAssemblingPositionDistinctly(t *testing.T) {
	pf, err := LoadPresets(repoRoot(t))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	seen := map[string]Position{}
	for _, p := range AssemblingPositions() {
		applied, err := PresetFor(pf, p)
		if err != nil {
			t.Errorf("the committed file names no entry at %s, so that position cannot "+
				"assemble at all: %v", p, err)
			continue
		}
		key := fmt.Sprintf("%v", applied.Applied().Selectors)
		if other, dup := seen[key]; dup {
			t.Errorf("the committed file gives %s and %s the same entry; a table that cannot "+
				"say what a reading is about cannot distinguish readings about different things",
				other, p)
		}
		seen[key] = p
	}
}

// TestARecordSelectorAssemblesThatRecordsMaterial is itd-199 ac-2's POSITIVE
// half, which nothing exercised.
//
// Every preset — the generated fixture and the committed file alike — carries an
// empty `records` list, and the one record-shaped assembly in the suite used an
// id that selects nothing and asserted the refusal. So the selecting path was
// held by reading the matcher, not by running it: `pathNamesRecord` was unit
// tested, and no assembly ever passed a record's material through it.
func TestARecordSelectorAssemblesThatRecordsMaterial(t *testing.T) {
	root := fixtureRepo(t)
	wide := assembleFixture(t, root, PositionEntailment)

	// The entry names the projection kind beside the record, because the kinds
	// ADMIT and the object set NARROWS (spc-2609020626048722): an entry naming
	// a record and no kind hands nothing, which is a refusal rather than the
	// selecting path this test exists to run.
	writeFile(t, root, ".abcd/config/reading-presets.json", `{
  "schema_version": 1,
  "presets": {
    "default": {"positions": {"entailment":
      {"kinds": ["intent-projection"], "records": ["itd-1"], "paths": []}}}
  }
}`)
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionEntailment, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("a record selector naming a record the fixture holds refused: %v", err)
	}
	if len(res.Manifest.Items) == 0 {
		t.Fatal("a record selector assembled nothing")
	}
	// The narrowing is per ROW, which is spc-2609020626048722's rule and not a
	// weakening of this one: a record row is narrowed to the object set's
	// records when the object set names any record under that row's source, and
	// admitted whole when it names none. itd-1 is the fixture's shipped intent,
	// so the SHIPPED row narrows to it; the drafts and planned rows, which the
	// object set names no record under, are handed whole at this position, as
	// the readings companion's section 6.2 asks.
	const shipped = ".abcd/development/intents/shipped/"
	narrowed := 0
	for _, m := range res.Manifest.Items {
		if !strings.HasPrefix(m.Path, shipped) {
			continue
		}
		narrowed++
		if !pathNamesRecord(m.Path, "itd-1") {
			t.Errorf("the assembly passed %s, which the selector itd-1 does not name; a record "+
				"selector admits its own record's material and no other", m.Path)
		}
	}
	if narrowed == 0 {
		t.Fatal("the assembly passed no item from the shipped row, so the record selector " +
			"carried nothing and this test proves nothing")
	}
	// And it must be a genuine narrowing, or the assertion above is trivially
	// satisfied by an entry that happened to select everything.
	if len(res.Manifest.Items) >= len(wide.Manifest.Items) {
		t.Errorf("the record selector passed %d items and the unnarrowed assembly %d; this test "+
			"proves nothing unless the entry narrows",
			len(res.Manifest.Items), len(wide.Manifest.Items))
	}
	// The bundle must tell the reading which record it was given.
	if len(res.Bundle.Preset.Records) != 1 || res.Bundle.Preset.Records[0] != "itd-1" {
		t.Errorf("the bundle's preset records are %v, want [itd-1]", res.Bundle.Preset.Records)
	}
}

// decodedManifest decodes one run's manifest as a bare document, so a test can
// assert about the KEYS a reader would find rather than about the struct the
// writer happened to have.
func decodedManifest(t *testing.T, m Manifest) map[string]json.RawMessage {
	t.Helper()
	raw, err := EncodeManifest(m)
	if err != nil {
		t.Fatalf("encode manifest: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	return doc
}

// TestAssemblyAppliesTheCommittedPresetForThePosition is itd-2609021003095168
// ac-2 (framework v4 section 8.2 and ruling M8; adr-2609021016286571).
//
// No operand names what a run is handed. The assembler applies the committed
// entry for the position it was invoked at, and the bundle carries the
// intersection of that entry with what the include table already admits there —
// nothing outside the entry, and nothing the table denies. The manifest records
// the entry applied and its hash, so a reader can say which commit's statement
// of the object the run used.
func TestAssemblyAppliesTheCommittedPresetForThePosition(t *testing.T) {
	root := fixtureRepo(t)

	// The whole admission at each position, before the entry narrows it. The
	// fixture's own preset names every kind, so this IS the table's admission,
	// and the intersection below is computed against it rather than declared.
	whole := map[Position][]ManifestItem{}
	for _, p := range AssemblingPositions() {
		whole[p] = assembleFixture(t, root, p).Manifest.Items
	}

	writeFile(t, root, ".abcd/config/reading-presets.json", narrowPresets())
	gitCommitAll(t, root)
	pf, err := LoadPresets(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	entry := pf

	for _, p := range AssemblingPositions() {
		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: p, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			t.Fatalf("assemble at %s: %v", p, err)
		}
		kinds := map[Kind]bool{}
		for _, k := range entry.Positions[string(p)].Kinds {
			kinds[k] = true
		}
		want := map[string]bool{}
		for _, m := range whole[p] {
			if kinds[m.Kind] {
				want[m.Path] = true
			}
		}
		if len(want) == 0 {
			t.Fatalf("the entry for %s selects nothing out of the table's admission, so this "+
				"test proves nothing about the intersection", p)
		}
		if len(want) >= len(whole[p]) {
			t.Fatalf("the entry for %s does not narrow the table's admission (%d of %d), so this "+
				"test proves nothing", p, len(want), len(whole[p]))
		}
		got := map[string]bool{}
		for _, m := range res.Manifest.Items {
			// A CANDIDATE is not selected by an entry and no entry may name the
			// kind: at the comparative position the candidate set is derived from
			// the record, so it is outside the intersection this test is about
			// (adr-2609021016272867).
			if m.Kind == KindCandidate {
				continue
			}
			got[m.Path] = true
			if !kinds[m.Kind] {
				t.Errorf("the assembly at %s passed %s of kind %s, which the committed entry does "+
					"not name; nothing outside the entry travels", p, m.Path, m.Kind)
			}
		}
		for path := range want {
			if !got[path] {
				t.Errorf("the assembly at %s omitted %s, which the entry names and the table "+
					"admits; the bundle is the intersection of the two", p, path)
			}
		}

		// And the manifest names the entry applied, with its hash.
		doc := decodedManifest(t, res.Manifest)
		var applied struct {
			Selectors []Selector `json:"selectors"`
		}
		if err := json.Unmarshal(doc["preset"], &applied); err != nil {
			t.Fatalf("the manifest at %s carries no preset block: %v", p, err)
		}
		wantSels := entry.Positions[string(p)].Applied().Selectors
		if fmt.Sprintf("%v", applied.Selectors) != fmt.Sprintf("%v", wantSels) {
			t.Errorf("the manifest at %s records the preset %v, want %v",
				p, applied.Selectors, wantSels)
		}
		var hash string
		if err := json.Unmarshal(doc["preset_hash"], &hash); err != nil || hash == "" {
			t.Errorf("the manifest at %s carries no preset hash, so a reader cannot tell two runs "+
				"apart by the entry they applied", p)
		}
	}
}

// TestManifestCarriesNoOverride is itd-2609021003095168 ac-4
// (adr-2609021016286571: there is no override at invocation and nothing to
// stamp).
//
// The stamp counted departures from the committed presets, which was worth
// counting while an operand could depart from them. Nothing can now, so a
// manifest carrying the stamp would assert a distinction that no longer exists.
func TestManifestCarriesNoOverride(t *testing.T) {
	root := fixtureRepo(t)
	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	doc := decodedManifest(t, res.Manifest)
	for _, gone := range []string{"scope", "scope_hash", "scope_overridden"} {
		if _, has := doc[gone]; has {
			t.Errorf("the manifest carries the key %q; the scope operand and its override stamp "+
				"are withdrawn, and a run is reproducible from the commit and the preset entry "+
				"the manifest names", gone)
		}
	}
	var applied map[string]json.RawMessage
	if err := json.Unmarshal(doc["preset"], &applied); err != nil {
		t.Fatalf("the manifest carries no preset block: %v", err)
	}
	for _, gone := range []string{"source", "overridden"} {
		if _, has := applied[gone]; has {
			t.Errorf("the manifest's preset block carries %q; there is no token an operator gave "+
				"and no departure to stamp", gone)
		}
	}
}

// TestRunIsReproducibleFromCommitAndPreset is itd-2609021003095168 ac-4's
// second half: the invocation carries nothing that could make two runs of one
// commit differ, so two assemblies at one commit produce byte-identical
// bundles. This is the amnesia eval's property, exercised here on the preset
// path.
func TestRunIsReproducibleFromCommitAndPreset(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/config/reading-presets.json", narrowPresets())
	gitCommitAll(t, root)

	for _, p := range AssemblingPositions() {
		first, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: p, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			t.Fatalf("first assembly at %s: %v", p, err)
		}
		second, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: p, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			t.Fatalf("second assembly at %s: %v", p, err)
		}
		a, b := mustEncodeBundle(t, first.Bundle), mustEncodeBundle(t, second.Bundle)
		if string(a) != string(b) {
			t.Errorf("two assemblies at %s of one commit produced different bundles; the "+
				"invocation carries nothing a re-run could differ on", p)
		}
		// The manifests differ in the run id and in nothing else, so the entry
		// applied and its hash are the same in both. Read as document keys
		// rather than as struct fields, because what a reader reproduces the
		// run from is the document.
		h := decodedManifest(t, first.Manifest)["preset_hash"]
		if len(h) == 0 {
			t.Errorf("the manifest at %s records no preset hash, so a reader cannot say which "+
				"commit's statement of the object the run applied", p)
		}
		if string(h) != string(decodedManifest(t, second.Manifest)["preset_hash"]) {
			t.Errorf("two assemblies at %s recorded different preset hashes", p)
		}
	}
}

// TestOnlyTheTreePositionsNameSourceOrTest is itd-194 ac-5 and its third scope
// condition (cond-2609021003130127), which is the object-set ruling of
// 2026-09-02: the committed detection and widening entries name the `source`
// and `test` kinds, because both design documents name code and tests as the
// shipped tree a reading may see, and the entailment entry names neither.
//
// An entry that names either kind ACCEPTS the unscanned items it selects, and
// the acceptance is paid for by disclosure — so the second half of this test
// assembles under those entries and holds every source and test item to
// carrying the mark. The live fixture leak at itm-0736 is absent from every
// committed entry, whose paths reach nothing under internal/core/site, and that
// absence is asserted here rather than left to the eval alone
// (iss-2608301450065320).
//
// The assembly runs over the FIXTURE tree carrying the committed entries rather
// than over this repository: a unit test cannot assume the working tree is
// clean, and the assembler rightly refuses a dirty one. What is read from the
// committed file is the thing under test — which kinds the entries name — and
// the tree the entries are applied to is incidental to that claim. The eval's
// TestTheFixtureLeakIsAbsentUnderEveryCommittedPreset takes the other half, over
// a clone of HEAD.
func TestOnlyTheTreePositionsNameSourceOrTest(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(PresetConfigPath)))
	if err != nil {
		t.Fatalf("read %s: %v", PresetConfigPath, err)
	}
	// Read as raw JSON rather than through the loader, so what is checked is the
	// file a reviewer reads rather than what the code under test made of it.
	var preset struct {
		Positions map[string]struct {
			Kinds  []string `json:"kinds"`
			Object struct {
				Paths []string `json:"paths"`
			} `json:"object"`
		} `json:"positions"`
	}
	if err := json.Unmarshal(raw, &preset); err != nil {
		t.Fatalf("decode %s: %v", PresetConfigPath, err)
	}
	if len(preset.Positions) != len(AssemblingPositions()) {
		t.Fatalf("%s holds %d position entries; one entry per assembling position means %d",
			PresetConfigPath, len(preset.Positions), len(AssemblingPositions()))
	}

	names := func(position string, kind Kind) bool {
		for _, k := range preset.Positions[position].Kinds {
			if k == string(kind) {
				return true
			}
		}
		return false
	}
	for _, position := range []Position{PositionDetection, PositionWidening} {
		for _, kind := range []Kind{KindSource, KindTest} {
			if !names(string(position), kind) {
				t.Errorf("the committed %s entry does not name %q; both design documents name "+
					"code and tests as the shipped tree a reading may see, and the ruling of "+
					"2026-09-02 keeps them in", position, kind)
			}
		}
	}
	for _, kind := range []Kind{KindSource, KindTest} {
		if names(string(PositionEntailment), kind) {
			t.Errorf("the committed entailment entry names %q; the ruling of 2026-09-02 gives "+
				"the tree kinds to detection and widening alone", kind)
		}
	}
	for position, entry := range preset.Positions {
		for _, p := range entry.Object.Paths {
			if strings.HasPrefix(path.Clean(p), "internal/core/site") {
				t.Errorf("the committed %s entry names the path %q, which reaches the Go fixture "+
					"the itd-183 audit found leaking; no committed entry reaches it", position, p)
			}
		}
	}

	// The disclosure half. The fixture carries the committed entries verbatim,
	// so what is applied here is what a run of this repository applies — and the
	// two Go files are written UNDER one of the entries' own object-set paths,
	// because a tree row is narrowed to those paths and a file outside them
	// would be handed to nothing (spc-2609020626048722).
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath, string(raw))
	writeFile(t, root, "internal/core/lint/lint.go",
		"package lint\n\n// the fixture's own source file is corpus, never built\n")
	writeFile(t, root, "internal/core/lint/lint_test.go",
		"package lint\n\n// the fixture's own test file is corpus, never built\n")
	gitCommitAll(t, root)

	for _, position := range []Position{PositionDetection, PositionWidening} {
		res, err := Assemble(AssembleRequest{
			RepoRoot: root, Position: position, Target: "HEAD", DryRun: true,
		})
		if err != nil {
			t.Fatalf("assemble at %s under the committed entry: %v", position, err)
		}
		marked := 0
		for _, m := range res.Manifest.Items {
			if m.Kind != KindSource && m.Kind != KindTest {
				continue
			}
			marked++
			if m.Scan != ScanUnscanned {
				t.Errorf("the committed %s entry passed %s (%s) marked %q; an entry that names "+
					"either tree kind accepts the unscanned items it selects, disclosed by the mark",
					position, m.Path, m.Kind, m.Scan)
			}
		}
		if marked == 0 {
			t.Errorf("the committed %s entry passed no source or test item, so the mark was "+
				"asserted over nothing", position)
		}
	}
}
