package reading

// objectset_test.go holds the maintainer's ruling of 2026-09-02, given at the
// Phase A review: the entailment reading is handed the OBJECT SET's drafts and
// planned intents, and none by default.
//
// The design framework v4's section 13 fixes the object set as "Iteration 1's
// shipped state and claim record" — the fifteen workstream intents, which the
// record extends by the ten Iteration 2 intents — and the readings companion
// v4's section 6.2 makes drafts and planned intents ADMISSIBLE at this position.
// Admissible is a permission; the object set is the scope. So the drafts and
// planned rows narrow by the entry's record list exactly as the shipped row
// does, a draft or a planned intent travels only when the committed entry names
// it, and the companion's admissibility becomes a switch on the entailment
// entry — yes or no for drafts and planned intents beyond the object set,
// default no. Divergence register entry 1 is corrected to say so, and the
// decision log's 2026-09-02 entry records the ruling.

import (
	"fmt"
	"strings"
	"testing"
)

// v2EntailmentEntry renders an entailment entry naming the intent projection,
// so a test states the records it is about and nothing else. `extra` carries
// the switch where a test declares one.
func v2EntailmentEntry(records, extra string) string {
	return fmt.Sprintf(`    "entailment": {"object": {"records": [%s], "paths": []}, `+
		`"kinds": ["intent-projection"]%s, "window": {"tokens_est": 1000000, `+
		`"measured_tokens_est": 1, "measured_bytes": 4, "measured_at": "0abc123"}}`,
		records, extra)
}

// entailmentPaths assembles the fixture at entailment under the given entry and
// returns the manifest's item paths.
func entailmentPaths(t *testing.T, root string) []string {
	t.Helper()
	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionEntailment, Target: "HEAD", DryRun: true,
	})
	if err != nil {
		t.Fatalf("assemble at entailment: %v", err)
	}
	out := make([]string, 0, len(res.Manifest.Items))
	for _, it := range res.Manifest.Items {
		out = append(out, it.Path)
	}
	return out
}

// containsPath reports whether any item path names the record id.
func containsPath(paths []string, id string) bool {
	for _, p := range paths {
		if pathNamesRecord(p, id) {
			return true
		}
	}
	return false
}

// The fixture's own candidate set: a draft and a planned intent the entailment
// position's rows reach, and the shipped intent an entry names to keep the
// assembly non-empty.
const (
	fixtureShipped = "itd-1"
	fixtureDraft   = "itd-2"
	fixturePlanned = "itd-3"
)

// TestADraftTheEntailmentEntryDoesNotNameDoesNotTravel is the ruling's default
// (framework 13, the object set; companion 6.2, admissibility as a permission;
// divergence register 1 as corrected).
//
// The defect this closes handed every draft and planned intent in the
// repository — 147 projected intents on the tree the ruling was given over —
// because a record row was admitted whole when the object set named none of its
// records. The drafts and planned rows do not work that way: they narrow to the
// records the entry names, and an entry naming none of them hands none.
func TestADraftTheEntailmentEntryDoesNotNameDoesNotTravel(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath,
		v2Preset(v2EntailmentEntry(`"`+fixtureShipped+`"`, "")))
	gitCommitAll(t, root)

	paths := entailmentPaths(t, root)
	if !containsPath(paths, fixtureShipped) {
		t.Fatalf("the entry names %s and it did not travel, so this test asserts an absence "+
			"over an assembly that carried nothing: %v", fixtureShipped, paths)
	}
	if containsPath(paths, fixtureDraft) {
		t.Errorf("the draft %s travelled at entailment under an entry that does not name it; "+
			"the object set is the scope and the companion's admissibility is a permission "+
			"(framework 13, companion 6.2, ruled 2026-09-02)", fixtureDraft)
	}
	if containsPath(paths, fixturePlanned) {
		t.Errorf("the planned intent %s travelled at entailment under an entry that does not "+
			"name it; the planned row narrows by the entry's record list exactly as the "+
			"shipped row does", fixturePlanned)
	}
}

// TestADraftTheEntailmentEntryNamesTravels is the other half of the same rule:
// the narrowing is by the entry's record list, so naming a draft is how it
// reaches the reading (framework 13; the decision entry of 2026-09-02).
func TestADraftTheEntailmentEntryNamesTravels(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath,
		v2Preset(v2EntailmentEntry(`"`+fixtureShipped+`", "`+fixtureDraft+`"`, "")))
	gitCommitAll(t, root)

	paths := entailmentPaths(t, root)
	if !containsPath(paths, fixtureDraft) {
		t.Errorf("the entry names the draft %s and it did not travel; a draft travels when "+
			"the committed entry names it: %v", fixtureDraft, paths)
	}
	if containsPath(paths, fixturePlanned) {
		t.Errorf("the planned intent %s travelled beside the named draft; naming one record "+
			"does not admit the row whole", fixturePlanned)
	}
}

// TestTheAdmitSwitchHandsEveryDraftAndPlannedIntent is the companion's
// admissibility (section 6.2) as the switch the ruling makes of it: yes or no
// for drafts and planned intents beyond the object set, default no.
func TestTheAdmitSwitchHandsEveryDraftAndPlannedIntent(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, PresetConfigPath,
		v2Preset(v2EntailmentEntry(`"`+fixtureShipped+`"`, `, "admit_drafts_and_planned": true`)))
	gitCommitAll(t, root)

	paths := entailmentPaths(t, root)
	for _, id := range []string{fixtureDraft, fixturePlanned} {
		if !containsPath(paths, id) {
			t.Errorf("the entry declares admit_drafts_and_planned and %s did not travel; the "+
				"switch on hands every draft and planned intent the position admits: %v",
				id, paths)
		}
	}
}

// TestTheAdmitSwitchIsRefusedAtEveryOtherPosition holds the switch to the one
// position it means anything at.
//
// The drafts and planned rows are admitted at entailment and nowhere else
// (companion 6.2; the include table's two rows), so the key at any other
// position is a permission over material that position never sees — and a key
// that reads as though it does something is worse than an absent one, which is
// the argument the strict decoder already makes for every other field here.
func TestTheAdmitSwitchIsRefusedAtEveryOtherPosition(t *testing.T) {
	for _, pos := range Positions() {
		if pos == PositionEntailment {
			continue
		}
		t.Run(string(pos), func(t *testing.T) {
			root := fixtureRepo(t)
			entry := fmt.Sprintf(`    %q: {"object": {"records": [], "paths": []}, `+
				`"kinds": ["brief-section"], "admit_drafts_and_planned": true, `+
				`"window": {"tokens_est": 1000000, "measured_tokens_est": 1, `+
				`"measured_bytes": 4, "measured_at": "0abc123"}}`, string(pos))
			writeFile(t, root, PresetConfigPath, v2Preset(entry))
			gitCommitAll(t, root)

			_, err := LoadPresets(root)
			if err == nil {
				t.Fatalf("the entry for %s declared admit_drafts_and_planned and loaded; the "+
					"drafts and planned rows are admitted at entailment alone, so the key "+
					"means nothing here", pos)
			}
			for _, want := range []string{string(pos), "admit_drafts_and_planned"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal does not name %q: %v", want, err)
				}
			}
		})
	}
}

// iteration1Records are the fifteen workstream intents the object set names,
// which framework 13 calls Iteration 1's shipped state and claim record.
var iteration1Records = []string{
	"itd-177", "itd-178", "itd-179", "itd-180", "itd-181", "itd-182", "itd-183",
	"itd-184", "itd-185", "itd-186", "itd-187", "itd-188", "itd-189", "itd-198",
	"itd-199",
}

// iteration2Records are the ten Iteration 2 intents the record extends the
// object set by, and which the entailment entry names so that the drafts and
// planned rows reach them (divergence register 1 as corrected; the decision
// entry of 2026-09-02). Several are shipped by the time Phase A completes and
// travel through the shipped row, which is the same object set either way.
var iteration2Records = []string{
	"itd-194", "itd-2609021003095168", "itd-2609020625400169", "itd-2609020625400194",
	"itd-2609020625400445", "itd-2609020625402518", "itd-2609020625402599",
	"itd-2609020625405170", "itd-2609020625405251", "itd-2609020625407419",
}

// TestTheCommittedEntailmentEntryNamesTheObjectSet is the ruling applied to the
// file this repository actually runs: the entailment entry names the ten
// Iteration 2 intents beside the fifteen workstream intents, and declares no
// switch (framework 13; companion 6.2; divergence register 1; the decision
// entry of 2026-09-02).
func TestTheCommittedEntailmentEntryNamesTheObjectSet(t *testing.T) {
	pf, err := LoadPresets(repoRoot(t))
	if err != nil {
		t.Fatalf("the committed preset file does not load: %v", err)
	}
	entry, err := PresetFor(pf, PositionEntailment)
	if err != nil {
		t.Fatalf("the committed file names no entailment entry: %v", err)
	}
	if entry.AdmitDraftsAndPlanned != nil {
		t.Errorf("the committed entailment entry declares admit_drafts_and_planned; the " +
			"default is off and this repository's reading is handed the object set alone " +
			"(ruled 2026-09-02)")
	}
	named := make(map[string]bool, len(entry.Object.Records))
	for _, r := range entry.Object.Records {
		named[r] = true
	}
	for _, id := range iteration1Records {
		if !named[id] {
			t.Errorf("the committed entailment entry does not name %s, one of the fifteen "+
				"workstream intents framework 13 fixes as the object set", id)
		}
	}
	for _, id := range iteration2Records {
		if !named[id] {
			t.Errorf("the committed entailment entry does not name %s, one of the ten "+
				"Iteration 2 intents the record extends the object set by; a draft or a "+
				"planned intent travels only when the entry names it", id)
		}
	}
	for _, r := range entry.Object.Records {
		if strings.HasPrefix(r, "spc-") {
			continue
		}
		if !named[r] {
			continue
		}
		found := false
		for _, id := range append(append([]string{}, iteration1Records...), iteration2Records...) {
			if id == r {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("the committed entailment entry names the intent %s, which is neither "+
				"one of the fifteen nor one of the ten; the object set is what the entry "+
				"names and nothing else", r)
		}
	}
}
