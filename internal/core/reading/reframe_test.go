package reading

import (
	"strings"
	"testing"
)

// The reframe record (spc-2609020626048705) is warm: it reaches no reading at
// any position, and every manifest asserts its exclusion.

const sentinelReframe = "SENTINEL-REFRAME-RECORD"

// ac-4. TestReframeRecordsNeverReachTheBundle plants a reframe record in the
// ledger and assembles at all four positions: the record's text never reaches
// the bundle, its path never reaches the manifest, and the manifest asserts
// the family's exclusion — by the container row at the three cold positions,
// by the derived per-family row at comparative, and by the floor's own
// reframe row at every one.
func TestReframeRecordsNeverReachTheBundle(t *testing.T) {
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/work/issues/reframes/rfm-1.md",
		"---\nschema_version: 1\nid: rfm-1\noccasioned_by: rdi-1\ngrounds: \""+sentinelReframe+"\"\n---\n\n"+sentinelReframe+"\n")
	gitCommitAll(t, root)

	for _, p := range AssemblingPositions() {
		var res AssembleResult
		if p == PositionComparative {
			var err error
			if res, err = assembleComparative(t, root); err != nil {
				t.Fatalf("assemble at comparative: %v", err)
			}
		} else {
			res = assembleFixture(t, root, p)
		}
		if strings.Contains(bundleText(res.Bundle), sentinelReframe) {
			t.Errorf("position %s passed a reframe record into the bundle", p)
		}
		for _, path := range itemPaths(res.Manifest) {
			if strings.HasPrefix(path, ".abcd/work/issues/reframes/") {
				t.Errorf("position %s named %s in the manifest", p, path)
			}
		}
		asserted := map[string]bool{}
		for _, e := range res.Manifest.Exclusions {
			asserted[e.Detail] = true
		}
		if !asserted["the reframe record"] {
			t.Errorf("position %s: the manifest does not assert the reframe record's exclusion", p)
		}
		container := ".abcd/work/issues"
		if p == PositionComparative {
			container = ".abcd/work/issues/reframes"
		}
		if !asserted[container] {
			t.Errorf("position %s: the manifest does not assert the exclusion of %s", p, container)
		}
	}
}

// TestExclusionFloorNamesTheReframeRecord: the floor carries the reframe row
// at every position, beside the lapse log's, on the same rule and signal.
func TestExclusionFloorNamesTheReframeRecord(t *testing.T) {
	for _, p := range Positions() {
		found := false
		for _, e := range ExclusionsFor(p) {
			if e.Detail == "the reframe record" {
				found = true
				if e.Rule != "absent from the positive walk" || e.Signal != "record type in a denied path" {
					t.Errorf("the reframe row reads %+v", e)
				}
			}
		}
		if !found {
			t.Errorf("position %s: the exclusion floor does not name the reframe record", p)
		}
	}
	if Admits(PositionComparative, ".abcd/work/issues/reframes/rfm-1.md") {
		t.Error("the comparative position admits a reframe record")
	}
}
