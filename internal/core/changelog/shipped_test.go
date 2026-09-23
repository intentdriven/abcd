package changelog

import (
	"reflect"
	"strings"
	"testing"
)

const (
	shippedDir  = ".abcd/development/intents/shipped/"
	resolvedDir = ".abcd/work/issues/resolved/"
	plannedDir  = ".abcd/development/intents/planned/"
)

// ids renders a record slice as its ids, for readable assertions.
func ids(records []Record) []string {
	out := make([]string, 0, len(records))
	for _, r := range records {
		out = append(out, r.ID)
	}
	return out
}

// baseRepo builds a repo with one shipped intent and one resolved issue at
// v0.1.0 — the state every caveat test diverges from.
func baseRepo(t *testing.T) *fixtureRepo {
	t.Helper()
	r := newFixtureRepo(t)
	r.write(shippedDir+"README.md", "# shipped\n")
	r.record(shippedDir+"itd-1-first.md", "itd-1", "additive")
	r.record(resolvedDir+"iss-1-first.md", "iss-1", "fix")
	r.commit("the released state")
	r.git("tag", "v0.1.0")
	return r
}

// TestShippedSinceAgreesAcrossMergeAndSquash is the PHASE 1 STOP-CONDITION test.
// The set is computed as a difference of END STATES precisely so that HOW a
// branch landed cannot change it: a squash merge collapses the record's move
// into a single new commit with no move to walk, and a log walk would report a
// different set for the same tree. If these two histories ever disagree, the
// anchor model is unsound and the phase stops.
func TestShippedSinceAgreesAcrossMergeAndSquash(t *testing.T) {
	// Each lands the SAME two records on main by a different route. An
	// intervening commit on main makes the squash and rebase routes genuinely
	// rewrite history rather than fast-forward.
	land := map[string]func(r *fixtureRepo){
		"merge-commit": func(r *fixtureRepo) {
			r.git("merge", "--no-ff", "-m", "merge feature", "feature")
		},
		"squash-merge": func(r *fixtureRepo) {
			r.git("merge", "--squash", "feature")
			r.git("commit", "-m", "squashed feature")
		},
		"rebase-merge": func(r *fixtureRepo) {
			r.git("rebase", "main", "feature")
			r.git("checkout", "main")
			r.git("merge", "--ff-only", "feature")
		},
	}

	got := map[string][]string{}
	for name, landFn := range land {
		t.Run(name, func(t *testing.T) {
			r := baseRepo(t)
			r.git("checkout", "-b", "feature")
			r.record(shippedDir+"itd-2-second.md", "itd-2", "breaking")
			r.record(resolvedDir+"iss-2-second.md", "iss-2", "fix")
			r.commit("ship itd-2 and resolve iss-2")
			r.git("checkout", "main")
			r.write("README.md", "# moved on\n")
			r.commit("unrelated work on main")
			landFn(r)

			set, err := ShippedSince(r.root, "v0.1.0")
			if err != nil {
				t.Fatalf("ShippedSince: %v", err)
			}
			want := []string{"itd-2", "iss-2"}
			if added := ids(set.Added); !reflect.DeepEqual(added, want) {
				t.Errorf("STOP: %s history added = %v, want %v", name, added, want)
			}
			if len(set.Removed) != 0 {
				t.Errorf("%s history removed = %v, want none", name, ids(set.Removed))
			}
			got[name] = ids(set.Added)
		})
	}

	for name, added := range got {
		if !reflect.DeepEqual(added, got["merge-commit"]) {
			t.Fatalf("STOP: the shipped set is history-dependent — %s=%v merge-commit=%v",
				name, added, got["merge-commit"])
		}
	}
}

// TestShippedSinceRenameReadsAsDeleteAndAdd pins a KNOWN LIMIT rather than a
// desired behaviour: the set-difference compares paths, so re-slugging a file
// inside shipped/ surfaces as a removal plus an addition. It is pinned so a
// future change to it (a rename-detecting diff, say) is a loud test failure and
// a deliberate decision, not a silent shift in what a release reports.
func TestShippedSinceRenameReadsAsDeleteAndAdd(t *testing.T) {
	r := baseRepo(t)
	r.git("mv", shippedDir+"itd-1-first.md", shippedDir+"itd-1-renamed-slug.md")
	r.commit("re-slug itd-1")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	if got := ids(set.Added); !reflect.DeepEqual(got, []string{"itd-1"}) {
		t.Errorf("added = %v, want the renamed path to read as an addition of itd-1", got)
	}
	if got := ids(set.Removed); !reflect.DeepEqual(got, []string{"itd-1"}) {
		t.Errorf("removed = %v, want the old path to read as a removal of itd-1", got)
	}
}

// TestShippedSinceSurfacesSupersession pins that an intent moved OUT of shipped/
// is reported, not silently dropped: a supersession is a user-visible change
// (a Removed/Changed line), and a set that only ever grew would hide it.
func TestShippedSinceSurfacesSupersession(t *testing.T) {
	r := baseRepo(t)
	r.write(plannedDir+"itd-1-first.md", "---\nid: itd-1\nimpact: breaking\n---\n# itd-1\n")
	r.remove(shippedDir + "itd-1-first.md")
	r.commit("supersede itd-1 back to planned")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	if len(set.Added) != 0 {
		t.Errorf("added = %v, want none (planned/ is not in the release set)", ids(set.Added))
	}
	if got := ids(set.Removed); !reflect.DeepEqual(got, []string{"itd-1"}) {
		t.Fatalf("removed = %v, want [itd-1]", got)
	}
	// The removed record's impact is read from the TAG's tree — the only place
	// the record still exists — so a supersession can still drive the bump.
	if set.Removed[0].Impact != ImpactAdditive {
		t.Errorf("removed impact = %q, want the impact recorded at the tag (additive)", set.Removed[0].Impact)
	}
	if got := set.Impact(); got != ImpactAdditive {
		t.Errorf("set impact = %q, want additive", got)
	}
}

// TestShippedSinceEmptyWhenNothingMoved pins the "nothing to release" input: a
// commit that touches no record leaves both sets empty and the impact at
// internal, which drives no bump.
func TestShippedSinceEmptyWhenNothingMoved(t *testing.T) {
	r := baseRepo(t)
	r.write("README.md", "# unrelated\n")
	r.commit("docs only")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	if len(set.Added) != 0 || len(set.Removed) != 0 {
		t.Errorf("expected an empty set, got added=%v removed=%v", ids(set.Added), ids(set.Removed))
	}
	if got := set.Impact(); got != ImpactInternal {
		t.Errorf("impact = %q, want internal (nothing to release)", got)
	}
	if _, bumped := DeriveNext(mustSemver(t, "0.1.0"), set.Impact()); bumped {
		t.Error("an empty set must not bump")
	}
}

// TestShippedSinceExcludesInternalFromTheChangelog pins outcome 8: an internal
// record is part of the cut (so the bijection can account for it) but earns no
// changelog line and drives no bump.
func TestShippedSinceExcludesInternalFromTheChangelog(t *testing.T) {
	r := baseRepo(t)
	r.record(resolvedDir+"iss-2-plumbing.md", "iss-2", "internal")
	r.record(resolvedDir+"iss-3-visible.md", "iss-3", "fix")
	r.commit("resolve two issues")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	if got := ids(set.All()); !reflect.DeepEqual(got, []string{"iss-2", "iss-3"}) {
		t.Errorf("all = %v, want [iss-2 iss-3]", got)
	}
	if got := ids(set.ChangelogRequired()); !reflect.DeepEqual(got, []string{"iss-3"}) {
		t.Errorf("changelog-required = %v, want [iss-3] (internal excluded)", got)
	}
	if got := set.Impact(); got != ImpactFix {
		t.Errorf("impact = %q, want fix (internal drives no bump)", got)
	}
}

// TestChangelogRequiredKeepsAnUnlabelledRemovedRecord pins the other half of
// outcome 8. UnlabelledAdded's contract promises that a record with no valid
// impact on the REMOVED side "still travels in Removed, so the release still
// reports it rather than dropping it silently" — which only holds if the
// required set treats unknown as reportable rather than folding it into
// internal. Its blob is read from the anchor tag's immutable tree, so no
// operator can ever label it; dropping it would be a permanent silent omission.
func TestChangelogRequiredKeepsAnUnlabelledRemovedRecord(t *testing.T) {
	r := newFixtureRepo(t)
	r.write(shippedDir+"README.md", "# shipped\n")
	// No `impact:` line — the shape of every record predating the impact field.
	r.write(shippedDir+"itd-1-first.md", "---\nid: itd-1\n---\n# itd-1\n")
	r.commit("the released state")
	r.git("tag", "v0.1.0")

	r.remove(shippedDir + "itd-1-first.md")
	r.record(resolvedDir+"iss-2-plumbing.md", "iss-2", "internal")
	r.commit("supersede the unlabelled intent, resolve an internal issue")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	if got := ids(set.Removed); !reflect.DeepEqual(got, []string{"itd-1"}) {
		t.Fatalf("removed = %v, want [itd-1]", got)
	}
	if len(set.UnlabelledAdded()) != 0 {
		t.Errorf("UnlabelledAdded = %v, want none — the unlabelled record is on the removed side",
			ids(set.UnlabelledAdded()))
	}
	if got := ids(set.ChangelogRequired()); !reflect.DeepEqual(got, []string{"itd-1"}) {
		t.Errorf("changelog-required = %v, want [itd-1] — unknown is not internal", got)
	}
}

// TestShippedSinceIgnoresNonRecords pins that only itd-N/iss-N markdown counts:
// a directory README living beside the records is not a release line.
func TestShippedSinceIgnoresNonRecords(t *testing.T) {
	r := baseRepo(t)
	r.write(shippedDir+"NOTES.md", "# notes\n")
	r.write(shippedDir+"itd-9-draft.txt", "not markdown\n")
	r.record(shippedDir+"itd-9-ninth.md", "itd-9", "fix")
	r.commit("add noise and one record")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	if got := ids(set.Added); !reflect.DeepEqual(got, []string{"itd-9"}) {
		t.Errorf("added = %v, want [itd-9]", got)
	}
}

// TestShippedSinceNamesUnlabelledRecords pins that a record with no (or an
// invalid) impact is surfaced by name rather than silently ranked at the bottom:
// treating a missing judgement as "internal" would under-bump the release, which
// is the exact failure the no-silent-default rule exists to prevent.
func TestShippedSinceNamesUnlabelledRecords(t *testing.T) {
	r := baseRepo(t)
	r.write(shippedDir+"itd-4-unlabelled.md", "---\nid: itd-4\n---\n# itd-4\n")
	r.record(shippedDir+"itd-5-misspelled.md", "itd-5", "Additive")
	r.commit("ship two badly-labelled intents")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	if got := ids(set.Unlabelled()); !reflect.DeepEqual(got, []string{"itd-4", "itd-5"}) {
		t.Errorf("unlabelled = %v, want [itd-4 itd-5]", got)
	}
	for _, rec := range set.Unlabelled() {
		if rec.ImpactErr == "" {
			t.Errorf("%s: unlabelled record carries no reason", rec.ID)
		}
	}
}

// TestShippedSinceDiagnosesANullImpactAsMissing pins MESSAGE PARITY with
// record-lint (iss-286). A YAML null is an ABSENT impact, not a malformed one,
// and record-lint says so — "impact must be set explicitly". Ungated, the
// release cut handed the raw scalar to ParseImpact, which knows only the empty
// string as absent, so `impact: NULL` came back as "invalid impact" and one line
// carried two diagnoses: the operator is told to fix a typo in a field that in
// fact has no value at all. The verdicts never diverged — both paths reject —
// so what is pinned here is the wording an operator has to act on.
//
// Every null spelling in the YAML 1.2 core schema must therefore reach the same
// missing-class diagnosis as the absent field, which is the baseline this table
// measures against.
func TestShippedSinceDiagnosesANullImpactAsMissing(t *testing.T) {
	// The absent-field diagnosis, taken from the boundary validator rather than
	// copied, so the two cannot drift apart.
	_, absentErr := ParseImpact("")
	if absentErr == nil {
		t.Fatalf("ParseImpact(%q) unexpectedly succeeded; the missing-impact baseline is gone", "")
	}
	want := absentErr.Error()

	// The YAML 1.2 core schema's null set, exactly as frontmatter.IsNull holds
	// it: the empty value, "~", and the three spellings of null.
	for _, spelling := range []string{"", "~", "null", "Null", "NULL"} {
		t.Run("impact-"+spelling, func(t *testing.T) {
			r := baseRepo(t)
			r.record(shippedDir+"itd-4-null-impact.md", "itd-4", spelling)
			r.commit("ship an intent whose impact is a YAML null")

			set, err := ShippedSince(r.root, "v0.1.0")
			if err != nil {
				t.Fatalf("ShippedSince: %v", err)
			}
			unlabelled := set.UnlabelledAdded()
			if got := ids(unlabelled); !reflect.DeepEqual(got, []string{"itd-4"}) {
				t.Fatalf("unlabelled-added = %v, want [itd-4]", got)
			}
			got := unlabelled[0].ImpactErr
			if got != want {
				t.Errorf("impact: %q diagnosed as\n  %q\nwant the missing-impact diagnosis record-lint gives\n  %q",
					spelling, got, want)
			}
			if strings.Contains(got, "invalid impact") {
				t.Errorf("impact: %q diagnosed as malformed (%q); a YAML null is an absent value, not a bad one",
					spelling, got)
			}
		})
	}
}

// TestShippedSinceSplitsUnlabelledBySide pins the two accessors apart: the whole
// cut is what a preview shows, but only the added side is refusable. A record
// unlabelled at the tag cannot be labelled anywhere — its blob lives in an
// immutable tree — so counting it as refusable would block the release with a
// remedy nobody can perform.
func TestShippedSinceSplitsUnlabelledBySide(t *testing.T) {
	r := newFixtureRepo(t)
	r.write(shippedDir+"itd-1-first.md", "---\nid: itd-1\n---\n# itd-1\n")
	r.commit("a release that predates the impact field")
	r.git("tag", "v0.1.0")
	r.remove(shippedDir + "itd-1-first.md")
	r.write(shippedDir+"itd-2-unlabelled.md", "---\nid: itd-2\n---\n# itd-2\n")
	r.commit("supersede itd-1, ship an unlabelled itd-2")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	if got := ids(set.Unlabelled()); !reflect.DeepEqual(got, []string{"itd-2", "itd-1"}) {
		t.Errorf("unlabelled = %v, want both sides [itd-2 itd-1]", got)
	}
	if got := ids(set.UnlabelledAdded()); !reflect.DeepEqual(got, []string{"itd-2"}) {
		t.Errorf("unlabelled-added = %v, want [itd-2] (the removed side is unfixable)", got)
	}
}

// TestShippedSinceIsSorted pins deterministic ordering: the set feeds a rendered
// preview and a changelog bijection, both of which must be reproducible.
func TestShippedSinceIsSorted(t *testing.T) {
	r := baseRepo(t)
	for _, id := range []string{"itd-30", "itd-4", "itd-100"} {
		r.record(shippedDir+id+"-x.md", id, "fix")
	}
	r.record(resolvedDir+"iss-7-x.md", "iss-7", "fix")
	r.commit("ship several")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	want := []string{"itd-100", "itd-30", "itd-4", "iss-7"}
	if got := ids(set.Added); !reflect.DeepEqual(got, want) {
		t.Errorf("added = %v, want %v (sorted by path)", got, want)
	}
}

// TestShippedSinceUnknownRef pins that a bad anchor is an error, never an empty
// set: silently reporting "nothing shipped" against a tag that does not exist
// would let a release cut claim there is nothing to release.
func TestShippedSinceUnknownRef(t *testing.T) {
	r := baseRepo(t)
	if _, err := ShippedSince(r.root, "v9.9.9"); err == nil {
		t.Error("expected an error for an unknown ref")
	}
}

// TestPressReleaseRequiredIsAddedUserFacingIntents pins the one definition of the
// release page's set: the ADDED intents whose impact earns a changelog line.
// Issues, internal intents, removed intents and intents carrying a valid
// `shipped_in:` fall out by construction.
func TestPressReleaseRequiredIsAddedUserFacingIntents(t *testing.T) {
	r := baseRepo(t)
	r.remove(shippedDir + "itd-1-first.md")
	r.record(shippedDir+"itd-2-feature.md", "itd-2", "additive")
	r.record(shippedDir+"itd-3-break.md", "itd-3", "breaking")
	r.record(shippedDir+"itd-4-plumbing.md", "itd-4", "internal")
	r.record(resolvedDir+"iss-2-fix.md", "iss-2", "fix")
	r.write(shippedDir+"itd-5-old.md", "---\nid: itd-5\nimpact: additive\nshipped_in: v0.1.0\n---\n# old\n")
	r.commit("ship a mixed cut")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("ShippedSince: %v", err)
	}
	if got := ids(set.PressReleaseRequired()); !reflect.DeepEqual(got, []string{"itd-2", "itd-3"}) {
		t.Errorf("press-release set = %v, want [itd-2 itd-3]", got)
	}
}
