package lint

import (
	"path/filepath"
	"testing"
)

// More than one spec naming one intent is the normal state, not drift: the
// bidirectional check asks whether the intent's spec_id names a spec that
// realises it, not whether it names THIS spec.
func TestSpecLifecycleAcceptsASecondSpecOnOneIntent(t *testing.T) {
	root := t.TempDir()
	base := "rec/specs"
	ibase := "rec/intents"

	writeFile(t, root, ibase+"/planned/itd-10-alpha.md", "---\nid: itd-10\nkind: standalone\nspec_id: spc-1\n---\n# ok\n")
	writeFile(t, root, base+"/closed/spc-1-alpha.md", "---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# ok\n")
	writeFile(t, root, base+"/open/spc-2-rest.md", "---\nid: spc-2\nslug: rest\nintent: itd-10\n---\n# ok\n")

	cfg := Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			"spec_lifecycle": {Enabled: true, Severity: "blocker", SpecsDir: "specs", IntentsDir: "intents"},
		},
	}
	fs, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if f.RuleID == "spec_lifecycle" && filepath.Base(f.File) == "spc-2-rest.md" {
			t.Fatalf("a remainder spec on the same intent is not drift: %+v", f)
		}
	}
}

// F2: the index's intent match is the shared canonical primitive, so a value
// that is not a record id at all groups nothing. Two specs carrying `intent:
// null` do not realise one pseudo-intent named "null".
func TestSpecsForIntentGroupsOnlyRealRecordIDs(t *testing.T) {
	idx := SpecLinkIndex{Specs: []SpecLink{
		{ID: "spc-1", IntentID: "null"},
		{ID: "spc-2", IntentID: "null"},
		{ID: "spc-3", IntentID: "itd-007"},
	}}
	if got := idx.SpecsForIntent("null"); len(got) != 0 {
		t.Fatalf("a non-id must group nothing, got %+v", got)
	}
	if got := idx.SpecsForIntent("itd-7"); len(got) != 1 || got[0].ID != "spc-3" {
		t.Fatalf("SpecsForIntent(itd-7) = %+v, want spc-3", got)
	}
}
