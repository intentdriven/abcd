package changelog

import (
	"strings"
	"testing"
)

// A record that names the release which carried its work is not part of a later
// cut (iss-2608241612087533).
//
// The cut is a set-difference of FOLDER MEMBERSHIP: it answers "did this record
// reach a terminal folder since the anchor?", and every consumer reads that as
// "did this work ship in this release?". The two coincide only while resolution
// lands in the fixing commit, which is what the RS001 gate now requires — so a
// ledger hygiene sweep, which closes records for work released long ago, is the
// legitimate exception.
//
// One ran on 2026-08-24 and closed 33 such records. The v0.6.4 cut then derived
// its version from four of them and tried to announce, as new, fixes first
// released in v0.2.0, v0.4.0, v0.4.1, v0.4.2 and v0.6.2. Four of those records
// were already cited by id under an earlier heading in the same CHANGELOG.
//
// The field is stated, never inferred. An earlier design tested whether
// resolved_by.commit was reachable from the anchor; it could have judged 85 of
// 376 resolved records and would have guessed at the other 291.
func TestShippedInRemovesARecordFromALaterCut(t *testing.T) {
	r := baseRepo(t)
	// Two records reach resolved/ after the tag. One says the work shipped in an
	// earlier release; the other says nothing, which is the ordinary case.
	r.record(resolvedDir+"iss-2-swept.md", "iss-2", "fix")
	r.write(resolvedDir+"iss-2-swept.md",
		"---\nid: iss-2\nimpact: fix\nshipped_in: v0.1.0\n---\n# iss-2\n")
	r.record(resolvedDir+"iss-3-genuine.md", "iss-3", "fix")
	r.commit("a hygiene sweep, plus one real fix")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}

	ids := map[string]bool{}
	for _, rec := range set.Added {
		ids[rec.ID] = true
	}
	if ids["iss-2"] {
		t.Error("a record naming an earlier release is still in the cut; it would " +
			"drive the version and earn a changelog line for work already announced")
	}
	if !ids["iss-3"] {
		t.Error("a record that says nothing about where it shipped must stay in the cut — " +
			"treating silence as already-shipped drops real content from a release record")
	}
}

// The failure DIRECTION is the point, so it is asserted on its own: a value
// nobody can read must not remove a record.
//
// `shipped_in` exists to take a record OUT of a release. That makes a typo
// dangerous in a way a missing field is not — an unreadable value that excluded
// would silently drop genuine content from the release record, which is the
// lie-by-omission this project treats as the worse failure. So an unparseable
// value leaves the record in the cut and reports itself.
func TestAnUnreadableShippedInDoesNotRemoveARecord(t *testing.T) {
	for _, bad := range []string{
		"0.1.0",      // no leading v
		"v0.1",       // not three components
		"v0.1.0-rc1", // a pre-release, which this repository does not tag
		"yesterday",  // prose
		"v0.1.0 eh",  // trailing junk
		"v0.1.0#eh",  // a hash with no whitespace in front of it is part of the value
	} {
		t.Run(bad, func(t *testing.T) {
			r := baseRepo(t)
			r.write(resolvedDir+"iss-2-typo.md",
				"---\nid: iss-2\nimpact: fix\nshipped_in: "+bad+"\n---\n# iss-2\n")
			r.commit("a record with an unreadable shipped_in")

			set, err := ShippedSince(r.root, "v0.1.0")
			if err != nil {
				t.Fatalf("ShippedSince: %v", err)
			}

			var found *Record
			for i := range set.Added {
				if set.Added[i].ID == "iss-2" {
					found = &set.Added[i]
				}
			}
			if found == nil {
				t.Fatalf("shipped_in %q removed the record from the cut; an unreadable "+
					"value must never be able to drop content from a release", bad)
			}
			if found.ShippedInErr == "" {
				t.Errorf("shipped_in %q was neither honoured nor reported — it must say why", bad)
			}
			if !strings.Contains(found.ShippedInErr, bad) {
				t.Errorf("the report must quote the offending value; got %q", found.ShippedInErr)
			}
		})
	}
}

// A trailing comment is not part of the value, and this field is where the old
// reading failed in the more expensive direction: `shipped_in: v0.1.0 # swept`
// read as the unparseable string `v0.1.0 # swept`, so a record that states it
// already shipped stayed in the cut and would be announced a second time. The
// strip lives in the ONE same-line scanner (iss-2608301744268001), so this
// consumer inherited it without a second rule of its own.
func TestShippedInHonoursAValueBehindAComment(t *testing.T) {
	r := baseRepo(t)
	r.write(resolvedDir+"iss-2-swept.md",
		"---\nid: iss-2\nimpact: fix\nshipped_in: v0.1.0 # closed by the sweep\n---\n# iss-2\n")
	r.commit("a swept record whose shipped_in carries a comment")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	for _, rec := range set.Added {
		if rec.ID == "iss-2" {
			t.Errorf("shipped_in behind a comment was not honoured: %+v", rec)
		}
	}
}

// A null or absent field is the ordinary case and is neither honoured nor
// reported. Without this, the error path above would fire on the 286 resolved
// records that carry no such field and every cut would be noise.
func TestAbsentOrNullShippedInIsSilent(t *testing.T) {
	for name, line := range map[string]string{
		"absent":     "",
		"null":       "shipped_in: null\n",
		"upper null": "shipped_in: NULL\n",
		"tilde":      "shipped_in: ~\n",
		"empty":      "shipped_in:\n",
	} {
		t.Run(name, func(t *testing.T) {
			r := baseRepo(t)
			r.write(resolvedDir+"iss-2-quiet.md",
				"---\nid: iss-2\nimpact: fix\n"+line+"---\n# iss-2\n")
			r.commit("a record with no shipped_in to speak of")

			set, err := ShippedSince(r.root, "v0.1.0")
			if err != nil {
				t.Fatalf("ShippedSince: %v", err)
			}
			for _, rec := range set.Added {
				if rec.ID != "iss-2" {
					continue
				}
				if rec.ShippedInErr != "" {
					t.Errorf("a %s shipped_in reported an error: %q", name, rec.ShippedInErr)
				}
				if rec.ShippedIn != "" {
					t.Errorf("a %s shipped_in yielded a value: %q", name, rec.ShippedIn)
				}
				return
			}
			t.Fatal("the record left the cut; a record with no shipped_in belongs to it")
		})
	}
}

// Shape is not correctness. A value may match the pattern perfectly and still be
// wrong, and a wrong-but-well-formed value is the last path by which shipped_in
// could silently drop content from a release record.
//
// An adversarial review demonstrated it: `shipped_in: v9.9.9` on a breaking
// record removed it from the cut with no error, taking its impact out of the
// version arithmetic and leaving the cut empty. The realistic version is a typo —
// v0.64.0 for v0.6.4 — which no shape check can catch.
//
// So the value must name a tag that EXISTS and that the anchor can REACH: a
// release already published at or before the one being measured from.
func TestShippedInMustNameARealAlreadyReleasedTag(t *testing.T) {
	for name, tag := range map[string]string{
		"a tag that never existed": "v9.9.9",
		"a plausible typo":         "v0.10.0",
		"a leading-zero non-tag":   "v01.1.0",
	} {
		t.Run(name, func(t *testing.T) {
			r := baseRepo(t)
			r.write(resolvedDir+"iss-2-wrong.md",
				"---\nid: iss-2\nimpact: breaking\nshipped_in: "+tag+"\n---\n# iss-2\n")
			r.commit("a record naming a release that did not carry it")

			set, err := ShippedSince(r.root, "v0.1.0")
			if err != nil {
				t.Fatalf("ShippedSince: %v", err)
			}
			var found *Record
			for i := range set.Added {
				if set.Added[i].ID == "iss-2" {
					found = &set.Added[i]
				}
			}
			if found == nil {
				t.Fatalf("%s removed the record from the cut; a shape check is not a "+
					"truth check, and this one takes a breaking impact out of the version", tag)
			}
			if found.ShippedInErr == "" {
				t.Errorf("%s was neither honoured nor reported", tag)
			}
		})
	}
}

// The REMOVED side keeps everything, and this is not an oversight.
//
// `shipped_in` answers "when did this WORK ship". A record leaving a terminal
// folder is a different event: the withdrawal happened in THIS window whenever
// the work first shipped. Filtering it dropped a supersession's impact from the
// version arithmetic, and — alone in a window — emptied the cut and refused the
// release outright. ingest.go states the rule plainly: letting a removal go
// uncited "would be the same omission as dropping an addition".
//
// The branch had no test at all until this one. A reviewer deleted the guard
// wholesale and the entire suite stayed green.
func TestARemovedRecordSurvivesItsShippedIn(t *testing.T) {
	r := baseRepo(t)
	// The record exists at the anchor carrying shipped_in, then leaves.
	r.write(resolvedDir+"iss-2-withdrawn.md",
		"---\nid: iss-2\nimpact: breaking\nshipped_in: v0.1.0\n---\n# iss-2\n")
	r.commit("a swept record, present at the anchor")
	r.git("tag", "-f", "v0.1.0")
	r.remove(resolvedDir + "iss-2-withdrawn.md")
	r.commit("withdrawn")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	for _, rec := range set.Removed {
		if rec.ID == "iss-2" {
			return
		}
	}
	t.Fatal("a withdrawn record was filtered out by its shipped_in; the withdrawal " +
		"is this cut's event whatever release first carried the work, and dropping " +
		"it takes its impact out of the version and its line out of the changelog")
}
