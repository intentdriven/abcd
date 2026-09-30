package reflect

import (
	"errors"
	stdreflect "reflect"
	"strings"
	"testing"
)

func seedIDs(s Seed) []string {
	var ids []string
	for _, it := range s.Intents {
		ids = append(ids, it.ID)
	}
	return ids
}

// Criterion 1, the seed half: given a cut release, the retrospective opens from
// the intents that tag shipped. The window's own arrivals count, a record a
// hygiene sweep stamped with an EARLIER release does not, and a record stamped
// with this release after the tag does.
func TestSeedIsTheIntentsTheTagShipped(t *testing.T) {
	r := releaseRepo(t)
	s, err := BuildSeed(r.Root(), "v0.2.0")
	if err != nil {
		t.Fatalf("BuildSeed: %v", err)
	}
	if got, want := seedIDs(s), []string{"itd-2", "itd-3", "itd-7", "itd-8"}; !stdreflect.DeepEqual(got, want) {
		t.Fatalf("seed intents = %v, want %v (itd-1 shipped in v0.1.0; itd-4 says shipped_in v0.1.0)", got, want)
	}
	if s.Tag != "v0.2.0" || s.Output != ".abcd/development/retrospectives/v0.2.0/README.md" {
		t.Errorf("tag/output = %q/%q", s.Tag, s.Output)
	}
}

// Criterion 1, the audit half: each seeded intent carries its impact and what
// its audit notes say, read as counts, never copied.
func TestSeedCarriesEachIntentsAuditNotesAndImpact(t *testing.T) {
	r := releaseRepo(t)
	s, err := BuildSeed(r.Root(), "v0.2.0")
	if err != nil {
		t.Fatalf("BuildSeed: %v", err)
	}
	it := s.Intents[0]
	if it.ID != "itd-2" || it.Title != "The second promise" || it.Impact != "additive" {
		t.Fatalf("itd-2 seed = %+v", it)
	}
	if it.Path != shippedDir+"itd-2-second.md" {
		t.Errorf("path = %q", it.Path)
	}
	if !it.Audited || it.Receipt != "rcp-0123456789ab" {
		t.Errorf("itd-2 audited=%v receipt=%q, want true and the ingested receipt", it.Audited, it.Receipt)
	}
	if want := (AuditRollup{Met: 3, MetWithConcerns: 1, NotMet: 1}); it.Rollup != want {
		t.Errorf("rollup = %+v, want %+v", it.Rollup, want)
	}
	if want := (GapCounts{Honoured: 2, Diverged: 1}); it.Gaps != want {
		t.Errorf("gaps = %+v, want %+v", it.Gaps, want)
	}
	if s.Intents[1].Impact != "fix" {
		t.Errorf("itd-3 impact = %q, want fix", s.Intents[1].Impact)
	}
}

// Criterion 1: the seed carries the changelog section the cut composed for the
// tag, and the anchor a link to it takes.
func TestSeedCarriesTheReleasesChangelogSection(t *testing.T) {
	r := releaseRepo(t)
	s, err := BuildSeed(r.Root(), "v0.2.0")
	if err != nil {
		t.Fatalf("BuildSeed: %v", err)
	}
	c := s.Changelog
	if !c.Found || c.Heading != "## [0.2.0] - 2026-09-20" || c.Anchor != "020---2026-09-20" {
		t.Fatalf("changelog = %+v", c)
	}
	if !strings.Contains(c.Body, "The second promise (itd-2).") || strings.Contains(c.Body, "The first promise") {
		t.Errorf("changelog body is not exactly the v0.2.0 section:\n%s", c.Body)
	}
}

// Criterion 2: a shipped intent with no audit notes is named, with the audit
// command offered. A placeholder, an owed review and an absent section are all
// "no audit notes"; a sub-heading under the placeholder is not an audit.
func TestSeedNamesEachIntentWithoutAuditNotesAndOffersTheAudit(t *testing.T) {
	r := releaseRepo(t)
	s, err := BuildSeed(r.Root(), "v0.2.0")
	if err != nil {
		t.Fatalf("BuildSeed: %v", err)
	}
	want := []AuditOffer{
		{IntentID: "itd-3", Command: "abcd intent audit itd-3"},
		{IntentID: "itd-7", Command: "abcd intent audit itd-7"},
		{IntentID: "itd-8", Command: "abcd intent audit itd-8"},
	}
	if !stdreflect.DeepEqual(s.Unaudited, want) {
		t.Fatalf("unaudited = %+v, want %+v", s.Unaudited, want)
	}
	for _, it := range s.Intents[1:] {
		if it.Audited {
			t.Errorf("%s reads as audited; its notes are a placeholder, an owed review or absent", it.ID)
		}
	}
}

// Criterion 3: a tag whose release shipped no intent refuses, in the
// criterion's words.
func TestSeedRefusesAReleaseThatShippedNoIntent(t *testing.T) {
	r := releaseRepo(t)
	_, err := BuildSeed(r.Root(), "v0.3.0")
	if !errors.Is(err, ErrNothingShipped) {
		t.Fatalf("err = %v, want ErrNothingShipped", err)
	}
	if got, want := err.Error(), "no intent shipped in `v0.3.0` — nothing shipped to reflect on"; got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
}

// Criterion 7: intents whose target_release names the release and which are
// still unshipped are listed; one targeted at another release is not.
func TestSeedListsIntentsTargetedAtTheReleaseStillUnshipped(t *testing.T) {
	r := releaseRepo(t)
	s, err := BuildSeed(r.Root(), "v0.2.0")
	if err != nil {
		t.Fatalf("BuildSeed: %v", err)
	}
	want := []TargetedIntent{{ID: "itd-5", Path: plannedDir + "itd-5-late.md"}}
	if !stdreflect.DeepEqual(s.Unshipped, want) {
		t.Fatalf("unshipped = %+v, want %+v", s.Unshipped, want)
	}
}

// Spec scope 2: the metrics are computed from the seed, not asked.
func TestSeedComputesTheMetrics(t *testing.T) {
	r := releaseRepo(t)
	s, err := BuildSeed(r.Root(), "v0.2.0")
	if err != nil {
		t.Fatalf("BuildSeed: %v", err)
	}
	m := s.Metrics
	if m.IntentsShipped != 4 || m.Audited != 1 || m.Unaudited != 3 {
		t.Errorf("counts = %+v", m)
	}
	if want := (AuditRollup{Met: 3, MetWithConcerns: 1, NotMet: 1}); m.Rollup != want {
		t.Errorf("rollup = %+v, want %+v", m.Rollup, want)
	}
	if want := (GapCounts{Honoured: 2, Diverged: 1}); m.Gaps != want {
		t.Errorf("gaps = %+v, want %+v", m.Gaps, want)
	}
	if want := r.Git("log", "-1", "--format=%cs", "v0.2.0"); m.TagDate != want {
		t.Errorf("tag date = %q, want %q", m.TagDate, want)
	}
	if m.PreviousTag != "v0.1.0" || m.PreviousTagDate != r.Git("log", "-1", "--format=%cs", "v0.1.0") {
		t.Errorf("previous = %q on %q", m.PreviousTag, m.PreviousTagDate)
	}
}

// The first release has no previous tag: everything in shipped/ at the tag is
// its own.
func TestSeedOfTheFirstReleaseHasNoPreviousTag(t *testing.T) {
	r := releaseRepo(t)
	s, err := BuildSeed(r.Root(), "v0.1.0")
	if err != nil {
		t.Fatalf("BuildSeed: %v", err)
	}
	if got, want := seedIDs(s), []string{"itd-1", "itd-4"}; !stdreflect.DeepEqual(got, want) {
		t.Errorf("seed intents = %v, want %v (itd-4 says shipped_in v0.1.0)", got, want)
	}
	if s.Metrics.PreviousTag != "" {
		t.Errorf("previous tag = %q, want none", s.Metrics.PreviousTag)
	}
}

// A malformed tag never reaches a path, and a well-formed tag the repository
// does not hold is refused rather than read as an empty release.
func TestSeedRefusesAMalformedOrUnknownTag(t *testing.T) {
	r := releaseRepo(t)
	for _, tag := range []string{"0.2.0", "v0.2", "v0.2.0-rc.1", "../v0.2.0", ""} {
		if _, err := BuildSeed(r.Root(), tag); err == nil || errors.Is(err, ErrNothingShipped) {
			t.Errorf("BuildSeed(%q) err = %v, want a shape refusal", tag, err)
		}
	}
	_, err := BuildSeed(r.Root(), "v9.9.9")
	if err == nil || errors.Is(err, ErrNothingShipped) || !strings.Contains(err.Error(), "v9.9.9") {
		t.Errorf("unknown tag err = %v, want a refusal naming the tag", err)
	}
}

// Out of scope, stated: a second run on the same tag refuses naming the
// existing file, before any interview is run.
func TestSeedRefusesWhenTheRetrospectiveExists(t *testing.T) {
	r := releaseRepo(t)
	r.Write(outputRel("v0.2.0"), "---\nrelease: v0.2.0\n---\n")
	_, err := BuildSeed(r.Root(), "v0.2.0")
	if !errors.Is(err, ErrExists) || !strings.Contains(err.Error(), outputRel("v0.2.0")) {
		t.Fatalf("err = %v, want ErrExists naming the file", err)
	}
}
