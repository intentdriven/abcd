package reading

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// TestManifestCoversEveryBundleItem is itd-183's fourth criterion: every item
// passed appears in the manifest with its path, its field where projection
// occurred, and a hash.
func TestManifestCoversEveryBundleItem(t *testing.T) {
	root := fixtureRepo(t)
	for _, p := range AssemblingPositions() {
		res := assembleFixture(t, root, p)
		if len(res.Bundle.Items) == 0 {
			t.Fatalf("position %s assembled nothing", p)
		}
		if len(res.Manifest.Items) != len(res.Bundle.Items) {
			t.Fatalf("position %s: %d bundle items, %d manifest items",
				p, len(res.Bundle.Items), len(res.Manifest.Items))
		}
		for i, item := range res.Bundle.Items {
			m := res.Manifest.Items[i]
			if m.ItemKey != item.ItemKey {
				t.Errorf("position %s: manifest row %d keys %s, bundle keys %s", p, i, m.ItemKey, item.ItemKey)
			}
			if m.Path == "" {
				t.Errorf("position %s: %s has no path", p, m.ItemKey)
			}
			if m.SHA256 != sha256Hex([]byte(item.Text)) {
				t.Errorf("position %s: %s hashes text it does not carry", p, m.ItemKey)
			}
		}
	}
}

// TestManifestNamesTheFieldWhereProjectionOccurred holds the other half of the
// same criterion: a projected item says which field it is.
//
// At detection, because itd-194 withdrew the shipped row from widening and the
// shipped intent is the fixture's projected record.
func TestManifestNamesTheFieldWhereProjectionOccurred(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionDetection)
	projected, whole := 0, 0
	for _, m := range res.Manifest.Items {
		if strings.HasPrefix(m.Path, ".abcd/development/intents/shipped/") {
			if m.Field == "" {
				t.Errorf("%s is a projection of %s and names no field", m.ItemKey, m.Path)
			}
			projected++
			continue
		}
		if m.Field == "" {
			whole++
		}
	}
	if projected == 0 {
		t.Error("no projected item reached the manifest")
	}
	if whole == 0 {
		t.Error("no whole-file item reached the manifest")
	}
}

// TestManifestAssertsNamedExclusions holds the manifest's other job: a reader
// can determine that a named excluded field was not passed, rather than taking
// a disclosure on trust.
func TestManifestAssertsNamedExclusions(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)
	if len(res.Manifest.Exclusions) == 0 {
		t.Fatal("the manifest asserts no exclusions")
	}
	want := []string{"origin", "production_mode", "Audit Notes", ".abcd/work/issues",
		".abcd/development/decisions", ".abcd/development/readings"}
	for _, w := range want {
		found := false
		for _, e := range res.Manifest.Exclusions {
			if e.Detail == w {
				found = true
				if e.Signal == "" || e.Rule == "" {
					t.Errorf("exclusion %q carries no signal or no rule", w)
				}
			}
		}
		if !found {
			t.Errorf("the manifest does not assert the exclusion of %q", w)
		}
	}
}

// TestManifestExclusionsAreAsserted holds the fail-closed half: the assembler
// refuses to emit an item under a path the manifest declares refused, so the
// declaration cannot part company with the run.
func TestManifestExclusionsAreAsserted(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)
	for _, m := range res.Manifest.Items {
		for _, e := range res.Manifest.Exclusions {
			if e.Signal != "directory" && e.Signal != "file" {
				continue
			}
			if m.Path == e.Detail || strings.HasPrefix(m.Path, e.Detail+"/") {
				t.Errorf("%s lies under the excluded %s", m.Path, e.Detail)
			}
		}
	}
}

// timestampRe is a date or a clock time in any of the shapes a serialiser
// reaches for.
var timestampRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}|\d{2}:\d{2}:\d{2}|\bT\d{2}:\d{2}`)

// TestManifestCarriesNoTimestamp holds the property that makes one repository
// state produce one manifest.
func TestManifestCarriesNoTimestamp(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)
	raw, err := EncodeManifest(res.Manifest)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// The run identifier carries adr-45's stamp by construction; every other
	// scalar must be timestamp-free.
	scrubbed := strings.ReplaceAll(string(raw), res.RunID, "<run-id>")
	if loc := timestampRe.FindString(scrubbed); loc != "" {
		t.Errorf("the manifest carries the timestamp-shaped token %q", loc)
	}
	for _, key := range []string{"\"date\"", "\"time\"", "\"timestamp\"", "\"generated_at\"", "\"assembled_at\""} {
		if strings.Contains(scrubbed, key) {
			t.Errorf("the manifest carries the key %s", key)
		}
	}
}

// TestManifestHashIsStableAcrossRuns holds the determinism the hash rests on:
// one repository state assembled twice under one run id produces one manifest,
// byte for byte, and two runs differ in the run identifier alone.
func TestManifestHashIsStableAcrossRuns(t *testing.T) {
	root := fixtureRepo(t)

	setMinter(t, fixedMinter("2608301200", 789))
	first := assembleFixture(t, root, PositionWidening)
	second := assembleFixture(t, root, PositionWidening)
	if first.ManifestHash != second.ManifestHash {
		t.Errorf("one state hashed to %s then %s", first.ManifestHash, second.ManifestHash)
	}

	setMinter(t, fixedMinter("2608301300", 42))
	third := assembleFixture(t, root, PositionWidening)
	if third.RunID == first.RunID {
		t.Fatal("two runs minted one identifier")
	}
	if third.ManifestHash == first.ManifestHash {
		t.Error("two runs hashed identically; the run identifier is part of the manifest")
	}
	a := first.Manifest
	b := third.Manifest
	a.RunID, b.RunID = "", ""
	rawA, _ := EncodeManifest(a)
	rawB, _ := EncodeManifest(b)
	if string(rawA) != string(rawB) {
		t.Error("two runs over one state differ in more than the run identifier")
	}
}

// TestDecodeManifestIsStrict holds the read side fail-closed in all three
// directions the evidence depends on.
func TestDecodeManifestIsStrict(t *testing.T) {
	root := fixtureRepo(t)
	res := assembleFixture(t, root, PositionWidening)
	raw, err := EncodeManifest(res.Manifest)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if _, err := DecodeManifest(raw); err != nil {
		t.Fatalf("a manifest this package wrote does not decode: %v", err)
	}
	for name, bad := range map[string]string{
		"unknown field":    strings.Replace(string(raw), "\"run_id\"", "\"run_when\"", 1),
		"trailing content": string(raw) + "{}\n",
		"wrong type":       strings.Replace(string(raw), ManifestType, "abcd.reading.bundle", 1),
		// Derived from SchemaVersion, not written as a literal: pinned to "1"
		// this replacement silently became a no-op the moment the schema moved,
		// and a no-op replacement produces a VALID manifest, so the case went on
		// asserting strictness while testing nothing (spc-68).
		"wrong schema": strings.Replace(string(raw),
			fmt.Sprintf("\"schema_version\": %d", SchemaVersion), "\"schema_version\": 99", 1),
	} {
		if _, err := DecodeManifest([]byte(bad)); err == nil {
			t.Errorf("a manifest with a %s decoded", name)
		}
	}
}
