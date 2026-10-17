package changelog

import (
	"reflect"
	"strings"
	"testing"
)

// changelogWith renders a CHANGELOG.md whose newest dated heading is version.
func changelogWith(version string) string {
	return strings.Join([]string{
		"# Changelog",
		"",
		"## [Unreleased]",
		"",
		"## [" + version + "] - 2026-07-21",
		"",
		"### Added",
		"",
		"- the release " + version,
		"",
	}, "\n")
}

// releasedRepo is a repo at v0.1.0 whose CHANGELOG heading agrees with the tag —
// the steady state a derivation runs against.
func releasedRepo(t *testing.T) *fixtureRepo {
	t.Helper()
	r := newFixtureRepo(t)
	r.write("CHANGELOG.md", changelogWith("0.1.0"))
	r.record(shippedDir+"itd-1-first.md", "itd-1", "additive")
	r.commit("release 0.1.0")
	r.git("tag", "v0.1.0")
	return r
}

// TestDeriveHappyPath walks the whole deterministic pipeline: anchor on the tag,
// diff the record trees, take the strongest impact, apply the pre-1.0 table.
func TestDeriveHappyPath(t *testing.T) {
	r := releasedRepo(t)
	r.record(shippedDir+"itd-2-second.md", "itd-2", "additive")
	r.record(resolvedDir+"iss-9-nine.md", "iss-9", "internal")
	r.commit("ship itd-2, resolve iss-9")

	d, err := Derive(r.root)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if d.Refused {
		t.Fatalf("unexpected refusal: %s", d.RefusalReason)
	}
	if !d.Bumped {
		t.Fatal("a cut with an additive record must bump")
	}
	if d.BaseTag != "v0.1.0" || d.Base.String() != "0.1.0" {
		t.Errorf("base = %s (%s), want v0.1.0", d.Base, d.BaseTag)
	}
	if d.Bump != ImpactAdditive {
		t.Errorf("bump = %q, want additive", d.Bump)
	}
	if d.Next.String() != "0.1.1" || d.NextTag != "v0.1.1" {
		t.Errorf("next = %s (%s), want 0.1.1 (v0.1.1)", d.Next, d.NextTag)
	}
	if got := ids(d.Records.Added); len(got) != 2 {
		t.Errorf("added = %v, want both records in the cut", got)
	}
	if got := ids(d.Records.ChangelogRequired()); len(got) != 1 || got[0] != "itd-2" {
		t.Errorf("changelog-required = %v, want [itd-2] (internal excluded)", got)
	}
}

// TestDeriveRefusesReleaseInFlight is the post-merge/pre-tag guard: once the
// ship PR has merged, the newest CHANGELOG heading names a release
// auto-release.yml has not tagged yet. Deriving then would compute the cut
// against a base that does not exist, so the derivation refuses by name instead.
func TestDeriveRefusesReleaseInFlight(t *testing.T) {
	r := releasedRepo(t)
	r.write("CHANGELOG.md", changelogWith("0.2.0"))
	r.record(shippedDir+"itd-2-second.md", "itd-2", "breaking")
	r.commit("the ship PR merged; the tag has not landed yet")

	d, err := Derive(r.root)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !d.Refused {
		t.Fatalf("expected a refusal; got next=%s bumped=%v", d.Next, d.Bumped)
	}
	if d.Bumped {
		t.Error("a refusal must derive no version")
	}
	for _, want := range []string{"0.2.0", "in flight", "tag pending"} {
		if !strings.Contains(d.RefusalReason, want) {
			t.Errorf("refusal %q does not mention %q", d.RefusalReason, want)
		}
	}
}

// TestDeriveEmptySetDoesNotBump pins "nothing to release": a cut with no records
// is not a refusal (nothing is wrong) and not a version (nothing changed) — the
// caller must be able to tell the two apart and write no heading.
func TestDeriveEmptySetDoesNotBump(t *testing.T) {
	r := releasedRepo(t)
	r.write("docs/README.md", "# docs\n")
	r.commit("documentation only")

	d, err := Derive(r.root)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if d.Refused {
		t.Fatalf("an empty cut is not a refusal: %s", d.RefusalReason)
	}
	if d.Bumped {
		t.Errorf("an empty cut must not bump; got %s", d.Next)
	}
	if d.Bump != ImpactInternal {
		t.Errorf("bump = %q, want internal", d.Bump)
	}
	if d.NextTag != "" {
		t.Errorf("NextTag = %q, want empty when nothing is released", d.NextTag)
	}
}

// TestDeriveRefusesUnlabelledRecord pins fail-closed labelling: a record ADDED
// to the cut with no valid impact ranks below every real one, so deriving over
// it would under-bump a release that may contain a break. The refusal names the
// record AND the file to edit, so the remedy is followable without a search.
func TestDeriveRefusesUnlabelledRecord(t *testing.T) {
	r := releasedRepo(t)
	r.write(shippedDir+"itd-7-unlabelled.md", "---\nid: itd-7\n---\n# itd-7\n")
	r.commit("ship an unlabelled intent")

	d, err := Derive(r.root)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !d.Refused {
		t.Fatalf("expected a refusal; got next=%s bumped=%v", d.Next, d.Bumped)
	}
	for _, want := range []string{"itd-7", shippedDir + "itd-7-unlabelled.md"} {
		if !strings.Contains(d.RefusalReason, want) {
			t.Errorf("refusal %q does not name %q", d.RefusalReason, want)
		}
	}
}

// TestDeriveCarriesUnlabelledRemovedRecord pins that ONLY the added side of a
// cut can refuse it. A removed record is read from the anchor tag's immutable
// tree, which may predate the impact back-fill, so refusing on it would name a
// record the operator cannot label — at HEAD the file is gone — and would block
// every release until the move was reverted. A re-slug inside resolved/ is no
// longer a removal at all (iss-2610121215586434): the cut is keyed on the id, so
// relabelling a record under a new slug neither adds nor removes it.
func TestDeriveCarriesUnlabelledRemovedRecord(t *testing.T) {
	cases := map[string]struct {
		mutate      func(r *fixtureRepo)
		wantAdded   []string
		wantRemoved []string
		wantBumped  bool
	}{
		"supersession": {
			mutate: func(r *fixtureRepo) {
				r.remove(resolvedDir + "iss-4-unlabelled.md")
				r.commit("supersede iss-4 out of resolved/")
			},
			wantAdded:   []string{},
			wantRemoved: []string{"iss-4"},
			wantBumped:  false,
		},
		"re-slug": {
			mutate: func(r *fixtureRepo) {
				r.remove(resolvedDir + "iss-4-unlabelled.md")
				r.record(resolvedDir+"iss-4-renamed.md", "iss-4", "fix")
				r.commit("re-slug iss-4 and label it at HEAD")
			},
			wantAdded:   []string{},
			wantRemoved: []string{},
			wantBumped:  false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := newFixtureRepo(t)
			r.write("CHANGELOG.md", changelogWith("0.1.0"))
			r.record(shippedDir+"itd-1-first.md", "itd-1", "additive")
			// Written before the tag with no impact: the state every record
			// resolved before the back-fill is frozen in at v0.3.0.
			r.write(resolvedDir+"iss-4-unlabelled.md", "---\nid: iss-4\n---\n# iss-4\n")
			r.commit("release 0.1.0")
			r.git("tag", "v0.1.0")
			tc.mutate(r)

			d, err := Derive(r.root)
			if err != nil {
				t.Fatalf("Derive: %v", err)
			}
			if d.Refused {
				t.Fatalf("refused a cut whose only unlabelled record is on the removed side: %s", d.RefusalReason)
			}
			if got := ids(d.Records.Added); !reflect.DeepEqual(got, tc.wantAdded) {
				t.Errorf("added = %v, want %v", got, tc.wantAdded)
			}
			if got := ids(d.Records.Removed); !reflect.DeepEqual(got, tc.wantRemoved) {
				t.Fatalf("removed = %v, want %v", got, tc.wantRemoved)
			}
			if len(d.Records.Removed) > 0 && d.Records.Removed[0].ImpactErr == "" {
				t.Error("the removed record must keep the reason it is unlabelled")
			}
			if d.Bumped != tc.wantBumped {
				t.Errorf("bumped = %v, want %v", d.Bumped, tc.wantBumped)
			}
		})
	}
}

// TestDeriveRefusesWithoutAnchor pins the fail-closed no-tag case: with no
// release tag there is no immutable base, and inventing one (0.0.0, say) would
// report every record ever written as the next release's contents.
func TestDeriveRefusesWithoutAnchor(t *testing.T) {
	r := newFixtureRepo(t)
	r.record(shippedDir+"itd-1-first.md", "itd-1", "additive")
	r.commit("no release has ever been tagged")

	d, err := Derive(r.root)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !d.Refused || d.Bumped {
		t.Errorf("expected a refusal with no derived version, got %+v", d)
	}
}

// TestDeriveOnThisRepo exercises the derivation against abcd's own history — the
// only fixture that proves the anchor works over a real tag (v0.1.0/v0.2.0/
// v0.3.0) and hundreds of real records, not just a synthetic two-commit repo.
// It asserts the shape of the outcome, never a specific version, so the test
// does not have to be edited every release.
//
// It is an OPPORTUNISTIC test: it needs a checkout that actually carries the
// tags. CI's `check` job checks out at the default depth with `fetch-tags:
// false`, so the tags are simply absent there and Derive correctly refuses with
// RefusalNoReleaseTag. That is an environment property, not a defect, so the
// test SKIPS rather than fails — asserting "this repo has tags" would be
// asserting something about the checkout, not about the code. The no-tag
// fail-closed refusal itself is pinned deterministically by
// TestDeriveRefusalKind/no_anchor_tag over a fixture repo, so skipping here
// loses no coverage of the behaviour.
func TestDeriveOnThisRepo(t *testing.T) {
	d, err := Derive("../../..")
	if err != nil {
		t.Skipf("not runnable outside a git checkout: %v", err)
	}
	if d.BaseTag == "" {
		t.Skipf("this checkout carries no release tags (a shallow clone fetched without them); "+
			"Derive correctly refuses: %s", d.RefusalReason)
	}
	if d.Refused {
		t.Logf("derivation refuses: %s", d.RefusalReason)
		return
	}
	for _, rec := range d.Records.All() {
		if rec.ID == "" || rec.Path == "" {
			t.Errorf("malformed record in the cut: %+v", rec)
		}
	}
	t.Logf("base=%s bump=%s next=%s added=%d removed=%d",
		d.BaseTag, d.Bump, d.NextTag, len(d.Records.Added), len(d.Records.Removed))
}
