package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/gittest"
)

const (
	shippedDir    = ".abcd/development/intents/shipped/"
	plannedDir    = ".abcd/development/intents/planned/"
	resolvedDir   = ".abcd/work/issues/resolved/"
	openIssuesDir = ".abcd/work/issues/open/"
	specsOpen     = ".abcd/development/specs/open/"
	specsClosed   = ".abcd/development/specs/closed/"
)

// liveSurface is the surface every fixture reports as "current". Its exact
// content is irrelevant to the emit step; what matters is that the same value is
// committed at the tag (the baseline) and at HEAD, which is the only shape in
// which the guardrail can reach a verdict at all.
func liveSurface() surface.Snapshot {
	return surface.NewSnapshot([]surface.Command{{Path: "abcd"}, {Path: "abcd launch"}}, nil)
}

func writeSurface(t *testing.T, r *gittest.Repo, snap surface.Snapshot) {
	t.Helper()
	data, err := surface.Encode(snap)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	r.Write(surface.SnapshotPath, string(data))
}

// releasedRepo is the state every cut is measured from: a tagged release whose
// tree carries the surface baseline and a CHANGELOG heading matching the tag.
func releasedRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	writeSurface(t, r, liveSurface())
	r.Write("CHANGELOG.md", "# Changelog\n\n## [0.4.0] - 2026-07-01\n\n### Added\n\n- the base.\n")
	r.Write(shippedDir+"README.md", "# shipped\n")
	r.Commit("the released state")
	r.Git("tag", "v0.4.0")
	return r
}

// emit runs the step the way the front door does: the caller supplies the
// current surface, the core supplies the judgement.
func emit(t *testing.T, r *gittest.Repo) Cut {
	t.Helper()
	cut, err := Emit(r.Root(), liveSurface())
	if err != nil {
		t.Fatalf("Emit: %v", err)
	}
	return cut
}

func refusalKinds(cut Cut) []string {
	var out []string
	for _, ref := range cut.Refusals {
		out = append(out, string(ref.Kind))
	}
	return out
}

// TestEmitReadyCut is the whole happy path: a tagged base, one user-facing
// record added since it, a guardrail that can compare, and therefore a derived
// version with the source material a composer needs.
func TestEmitReadyCut(t *testing.T) {
	r := releasedRepo(t)
	r.Write(shippedDir+"itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\nthe version is derived from what shipped.\n")
	r.Write(resolvedDir+"iss-24-unreleased-conflicts.md",
		"---\nid: iss-24\nslug: unreleased-conflicts\nimpact: internal\n---\n\nconcurrent PRs conflict on the Unreleased block.\n")
	r.Commit("ship an intent and resolve an internal issue")

	cut := emit(t, r)

	if !cut.Ready {
		t.Fatalf("cut is not ready: %+v", cut.Refusals)
	}
	if cut.BaseTag != "v0.4.0" {
		t.Errorf("BaseTag = %q, want v0.4.0", cut.BaseTag)
	}
	// pre-1.0 additive bumps the patch (changelog.DeriveNext).
	if cut.NextTag != "v0.4.1" || !cut.Bumped {
		t.Errorf("NextTag = %q bumped=%v, want v0.4.1 true", cut.NextTag, cut.Bumped)
	}
	if cut.Impact != changelog.ImpactAdditive {
		t.Errorf("Impact = %q, want additive", cut.Impact)
	}
	if got := strings.Join(cut.DecidedBy, ","); got != "itd-73" {
		t.Errorf("DecidedBy = %v, want [itd-73] — the record that decided the bump", cut.DecidedBy)
	}
	if cut.Guard.Status != changelog.SurfaceGuardPassed {
		t.Errorf("Guard = %q (%s), want passed", cut.Guard.Status, cut.Guard.Reason)
	}
	if len(cut.Added) != 2 || len(cut.Removed) != 0 {
		t.Fatalf("Added/Removed = %d/%d, want 2/0", len(cut.Added), len(cut.Removed))
	}

	byID := map[string]Entry{}
	for _, e := range cut.Added {
		byID[e.ID] = e
	}
	intent := byID["itd-73"]
	if intent.Path != shippedDir+"itd-73-derived-versioning.md" {
		t.Errorf("itd-73 Path = %q", intent.Path)
	}
	if intent.Title != "A Version Is A Fact" || intent.Summary != "the version is derived from what shipped." {
		t.Errorf("itd-73 source material = %q / %q", intent.Title, intent.Summary)
	}
	if !intent.InChangelog {
		t.Error("an additive record must be in the changelog set")
	}
	if issue := byID["iss-24"]; issue.InChangelog {
		t.Error("an internal record must be excluded from the changelog set")
	}
}

// TestEmitRefuses walks every fail-closed refusal. Each row builds a repository
// that is ready EXCEPT for the one thing under test, so a refusal can only come
// from that cause.
func TestEmitRefuses(t *testing.T) {
	tests := []struct {
		name      string
		build     func(t *testing.T) *gittest.Repo
		wantKind  RefusalKind
		wantNamed []string
	}{
		{
			name: "a release is in flight — the changelog heading is ahead of the tag",
			build: func(t *testing.T) *gittest.Repo {
				r := releasedRepo(t)
				r.Write("CHANGELOG.md", "# Changelog\n\n## [0.5.0] - 2026-07-20\n\n### Added\n\n- the ship PR merged.\n")
				r.Record(shippedDir+"itd-73-x.md", "itd-73", "additive")
				r.Commit("the ship PR merged; auto-release has not tagged it yet")
				return r
			},
			wantKind:  RefusalReleaseInFlight,
			wantNamed: []string{"v0.5.0", "in flight"},
		},
		{
			name: "a planned intent's spec has already closed",
			build: func(t *testing.T) *gittest.Repo {
				r := releasedRepo(t)
				r.Record(shippedDir+"itd-73-x.md", "itd-73", "additive")
				r.Write(plannedDir+"itd-94-implement-readiness-gate.md",
					"---\nid: itd-94\nkind: standalone\nspec_id: spc-9\n---\n# gate\n")
				r.Write(specsClosed+"spc-9-implement-readiness-gate.md",
					"---\nid: spc-9\nslug: implement-readiness-gate\nintent: itd-94\n---\n# spc-9\n")
				r.Commit("a merged feature whose intent never left planned/")
				return r
			},
			wantKind:  RefusalStaleIntent,
			wantNamed: []string{"itd-94", "spc-9", "shipped/"},
		},
		{
			name: "an added record carries no impact",
			build: func(t *testing.T) *gittest.Repo {
				r := releasedRepo(t)
				r.Write(shippedDir+"itd-73-x.md", "---\nid: itd-73\n---\n# itd-73\n")
				r.Commit("ship an unlabelled record")
				return r
			},
			wantKind:  RefusalUnlabelled,
			wantNamed: []string{"itd-73", "impact"},
		},
		{
			name: "the surface narrowed and nothing declares it",
			build: func(t *testing.T) *gittest.Repo {
				r := gittest.NewRepo(t)
				wider := surface.NewSnapshot(
					append(append([]surface.Command{}, liveSurface().Commands...), surface.Command{Path: "abcd ghost"}), nil)
				writeSurface(t, r, wider)
				r.Write("CHANGELOG.md", "# Changelog\n\n## [0.4.0] - 2026-07-01\n")
				r.Commit("the released state, with one more command")
				r.Git("tag", "v0.4.0")

				writeSurface(t, r, liveSurface())
				r.Record(shippedDir+"itd-73-x.md", "itd-73", "additive")
				r.Commit("remove a command without declaring it")
				return r
			},
			wantKind:  RefusalSurfaceGuard,
			wantNamed: []string{"abcd ghost", "impact: breaking"},
		},
		{
			name: "no anchor tag at all",
			build: func(t *testing.T) *gittest.Repo {
				r := gittest.NewRepo(t)
				r.Record(shippedDir+"itd-73-x.md", "itd-73", "additive")
				r.Commit("nothing has ever been released")
				return r
			},
			wantKind:  RefusalNoReleaseTag,
			wantNamed: []string{"tag"},
		},
		{
			name: "nothing user-facing shipped",
			build: func(t *testing.T) *gittest.Repo {
				r := releasedRepo(t)
				r.Record(resolvedDir+"iss-97-toctou.md", "iss-97", "internal")
				r.Commit("resolve an internal issue only")
				return r
			},
			wantKind:  RefusalEmptyCut,
			wantNamed: []string{"nothing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cut := emit(t, tt.build(t))
			if cut.Ready {
				t.Fatalf("cut is ready; expected a %s refusal", tt.wantKind)
			}
			var found *Refusal
			for i, ref := range cut.Refusals {
				if ref.Kind == tt.wantKind {
					found = &cut.Refusals[i]
				}
			}
			if found == nil {
				t.Fatalf("refusals = %v, want one of kind %q", refusalKinds(cut), tt.wantKind)
			}
			for _, want := range tt.wantNamed {
				if !strings.Contains(found.Reason, want) {
					t.Errorf("refusal %q does not name %q", found.Reason, want)
				}
			}
			if cut.NextTag != "" || cut.Bumped {
				t.Errorf("a refused cut derived %q (bumped=%v); it must derive nothing", cut.NextTag, cut.Bumped)
			}
		})
	}
}

// TestEmitStaleIntentNamesTheRecord pins the machine-readable half of outcome
// 11: the refusal carries the blocking record's id and path, not just prose, so
// a front door can act on it without parsing English.
func TestEmitStaleIntentNamesTheRecord(t *testing.T) {
	r := releasedRepo(t)
	r.Record(shippedDir+"itd-73-x.md", "itd-73", "additive")
	r.Write(plannedDir+"itd-94-gate.md", "---\nid: itd-94\nkind: standalone\nspec_id: spc-9\n---\n# gate\n")
	r.Write(specsClosed+"spc-9-gate.md", "---\nid: spc-9\nslug: gate\nintent: itd-94\n---\n# spc-9\n")
	// An open spec on a planned intent is the normal, healthy state and must not
	// refuse — otherwise no cut could ever ship while any intent was in flight.
	r.Write(plannedDir+"itd-67-plugin.md", "---\nid: itd-67\nkind: standalone\nspec_id: spc-11\n---\n# plugin\n")
	r.Write(specsOpen+"spc-11-plugin.md", "---\nid: spc-11\nslug: plugin\nintent: itd-67\n---\n# spc-11\n")
	r.Commit("one stale intent, one healthy one")

	cut := emit(t, r)
	var stale *Refusal
	for i, ref := range cut.Refusals {
		if ref.Kind == RefusalStaleIntent {
			stale = &cut.Refusals[i]
		}
	}
	if stale == nil {
		t.Fatalf("refusals = %v, want a stale-intent refusal", refusalKinds(cut))
	}
	if len(stale.Records) != 1 || stale.Records[0] != "itd-94" {
		t.Errorf("Records = %v, want exactly [itd-94] — itd-67's spec is still open", stale.Records)
	}
	if strings.Contains(stale.Reason, "itd-67") {
		t.Errorf("refusal %q names an intent whose spec is still open", stale.Reason)
	}
}

// TestEmitWritesNothing is the zero-write contract. The emit step is the input to
// a review, not a mutation: everything it touches is read out of git or off disk,
// and the CHANGELOG is written only by the ingest step that follows the composer.
func TestEmitWritesNothing(t *testing.T) {
	r := releasedRepo(t)
	r.Record(shippedDir+"itd-73-x.md", "itd-73", "additive")
	r.Commit("ship something")

	before := treeDigest(t, r.Root())
	if _, err := Emit(r.Root(), liveSurface()); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if after := treeDigest(t, r.Root()); after != before {
		t.Errorf("the working tree changed:\nbefore %s\nafter  %s", before, after)
	}
}

// TestCutJSONShape pins the wire contract the next two stages consume: the
// composer reads this JSON, and the ingest step is written against the same keys.
func TestCutJSONShape(t *testing.T) {
	r := releasedRepo(t)
	r.Write(shippedDir+"itd-73-x.md", "---\nid: itd-73\nimpact: additive\n---\n\n# Title\n\nsummary line.\n")
	r.Commit("ship something")

	data, err := json.Marshal(emit(t, r))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, key := range []string{"ready", "base_tag", "next_tag", "bumped", "impact", "decided_by", "added", "removed", "guard"} {
		if _, ok := generic[key]; !ok {
			t.Errorf("cut JSON has no %q key: %s", key, data)
		}
	}
	added, _ := generic["added"].([]any)
	if len(added) != 1 {
		t.Fatalf("added = %v", generic["added"])
	}
	entry, _ := added[0].(map[string]any)
	for _, key := range []string{"id", "path", "impact", "title", "summary", "in_changelog"} {
		if _, ok := entry[key]; !ok {
			t.Errorf("entry JSON has no %q key: %v", key, entry)
		}
	}
	// The guard verdict is nested in the same document, so it must speak the
	// same wire dialect: snake_case keys, not Go field names.
	guard, _ := generic["guard"].(map[string]any)
	for _, key := range []string{"base_tag", "status", "breaking_declared"} {
		if _, ok := guard[key]; !ok {
			t.Errorf("guard JSON has no %q key: %v", key, guard)
		}
	}
}

// treeDigest hashes every path and byte under root except .git, whose internals
// git itself rewrites (index mtimes, gc) for reasons that are not the code under
// test.
func treeDigest(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		lines = append(lines, filepath.ToSlash(rel)+" "+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

// The unfixed-findings guardrail reaches the cut as a refusal that names its
// records, and clears once the finding is answered.
//
// This is the composition half of the gate: core/changelog owns the judgement
// and is exercised there; what has to hold HERE is that the verdict actually
// stops a release rather than being computed and rendered beside a ready cut.
// A gate whose verdict a caller can ignore is the phantom gate
// enforcement-claims-are-facts refuses.
func TestEmitRefusesACutStandingOverItsOwnFindings(t *testing.T) {
	r := releasedRepo(t)
	r.Write(shippedDir+"itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\nderived.\n")
	r.Write(openIssuesDir+"iss-90-found-while-shipping.md",
		"---\nid: \"iss-90\"\nseverity: \"major\"\n---\n\nfound while shipping the intent above.\n")
	r.Commit("ship an intent and capture what shipping it turned up")

	cut := emit(t, r)

	if cut.Ready {
		t.Fatalf("the cut is ready while standing over a major finding it captured itself")
	}
	if !contains(refusalKinds(cut), string(RefusalUnfixedFinding)) {
		t.Fatalf("refusals = %v, want one of kind %q", refusalKinds(cut), RefusalUnfixedFinding)
	}
	for _, ref := range cut.Refusals {
		if ref.Kind != RefusalUnfixedFinding {
			continue
		}
		if !contains(ref.Records, "iss-90") {
			t.Errorf("the refusal does not name iss-90 in Records (%v), so a front door has to "+
				"parse its prose to act on it", ref.Records)
		}
	}
	if cut.Findings.Status != changelog.FindingGuardFailed {
		t.Errorf("Findings.Status = %q, want failed", cut.Findings.Status)
	}
	// A refused cut carries no derived version, exactly as the other refusals.
	if cut.NextTag != "" || cut.Bumped {
		t.Errorf("NextTag = %q bumped=%v on a refused cut", cut.NextTag, cut.Bumped)
	}

	// Answering it clears the gate, and the release proceeds.
	r.Remove(openIssuesDir + "iss-90-found-while-shipping.md")
	r.Write(resolvedDir+"iss-90-found-while-shipping.md",
		"---\nid: \"iss-90\"\nseverity: \"major\"\nimpact: fix\n---\n\nfixed in this cut.\n")
	r.Commit("fix it in the release it was found in")

	cut = emit(t, r)

	if !cut.Ready {
		t.Fatalf("the cut is still refused after the finding was resolved: %+v", cut.Refusals)
	}
	if cut.Findings.Status != changelog.FindingGuardPassed {
		t.Errorf("Findings.Status = %q, want passed", cut.Findings.Status)
	}
}

// A finding deferred out loud lets the release through AND is carried on the
// cut, so the release report says what was held over and why.
func TestEmitCarriesADeferredFindingOntoAReadyCut(t *testing.T) {
	r := releasedRepo(t)
	r.Write(shippedDir+"itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\nderived.\n")
	r.Write(openIssuesDir+"iss-90-found-while-shipping.md",
		"---\nid: \"iss-90\"\nseverity: \"major\"\ndeferred_after: \"v0.4.0\"\n"+
			"deferral_reason: \"the fix needs a schema migration\"\n---\n\nheld over.\n")
	r.Commit("ship an intent and defer what shipping it turned up")

	cut := emit(t, r)

	if !cut.Ready {
		t.Fatalf("a recorded deferral did not let the cut through: %+v", cut.Refusals)
	}
	if len(cut.Findings.Waived) != 1 || cut.Findings.Waived[0].ID != "iss-90" {
		t.Fatalf("Findings.Waived = %+v, want the deferred iss-90 — a deferral the release "+
			"report does not carry is indistinguishable from a finding nobody looked at",
			cut.Findings.Waived)
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// A cut that DELETES a blocking record instead of answering it is refused under
// its own kind (iss-2609091143455568).
//
// The composition half again: core/changelog owns the judgement and is exercised
// there, and what has to hold here is that the verdict reaches the cut as a
// refusal of its own kind, naming the record — because the remedy differs from
// the unfixed case. The record has to come back before it can be resolved,
// waived or wontfixed, and a front door handed `unfixed-finding` would send its
// reader to open a file that is no longer in the tree.
func TestEmitRefusesACutThatDeletesItsFindingsRecord(t *testing.T) {
	r := releasedRepo(t)
	r.Write(openIssuesDir+"iss-91-found-last-cycle.md",
		"---\nid: \"iss-91\"\nseverity: \"major\"\n---\n\nfound before the anchor moved.\n")
	r.Commit("capture a finding")
	r.Git("tag", "v0.5.0")
	r.Write("CHANGELOG.md", "# Changelog\n\n## [0.5.0] - 2026-07-02\n\n### Added\n\n- the base.\n")
	r.Write(shippedDir+"itd-74-something-shipped.md",
		"---\nid: itd-74\nimpact: additive\n---\n\n# Something Shipped\n\nderived.\n")
	r.Remove(openIssuesDir + "iss-91-found-last-cycle.md")
	r.Commit("ship an intent and delete the standing finding")

	cut := emit(t, r)

	if cut.Ready {
		t.Fatalf("the cut is ready having removed a major finding from the ledger: %+v", cut.Findings)
	}
	if !contains(refusalKinds(cut), string(RefusalDeletedFinding)) {
		t.Fatalf("refusals = %v, want one of kind %q", refusalKinds(cut), RefusalDeletedFinding)
	}
	for _, ref := range cut.Refusals {
		if ref.Kind != RefusalDeletedFinding {
			continue
		}
		if !contains(ref.Records, "iss-91") {
			t.Errorf("the refusal does not name iss-91 in Records (%v), so a front door has to "+
				"parse its prose to act on it", ref.Records)
		}
		if !strings.Contains(ref.Reason, "no status directory") {
			t.Errorf("the refusal does not say the record left the ledger: %q", ref.Reason)
		}
		// The unfixed half's remedy must not ride along on this refusal: there is
		// no open record to resolve until the deleted one is restored.
		if strings.Contains(ref.Reason, "has not answered") {
			t.Errorf("the deletion refusal carries the unfixed half's prose:\n%s", ref.Reason)
		}
	}
	if cut.NextTag != "" || cut.Bumped {
		t.Errorf("NextTag = %q bumped=%v on a refused cut", cut.NextTag, cut.Bumped)
	}

	// Restoring the record and resolving it clears the gate: the ledger can say
	// what happened to the finding again.
	r.Write(resolvedDir+"iss-91-found-last-cycle.md",
		"---\nid: \"iss-91\"\nseverity: \"major\"\nimpact: fix\n---\n\nfixed rather than removed.\n")
	r.Commit("restore the record into resolved/")

	cut = emit(t, r)

	if !cut.Ready {
		t.Fatalf("the cut is still refused after the record was restored and resolved: %+v", cut.Refusals)
	}
}

// TestEmitMarksInPressRelease: each entry says whether the release page must
// cite it, so the host never re-derives the rule. Only added, user-facing
// intents are marked; an issue, an internal intent and a removed intent are not.
func TestEmitMarksInPressRelease(t *testing.T) {
	cut := emit(t, pageRepo(t))
	got := map[string]bool{}
	for _, e := range append(append([]Entry{}, cut.Added...), cut.Removed...) {
		got[e.ID] = e.InPressRelease
	}
	want := map[string]bool{"itd-73": true, "itd-74": true, "itd-97": false, "iss-51": false, "itd-40": false}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s in_press_release = %v, want %v", id, got[id], w)
		}
	}
	data, err := json.Marshal(cut)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"in_press_release":true`) {
		t.Errorf("the cut JSON does not carry in_press_release: %s", data)
	}
	if strings.Contains(string(data), "stopped typing") {
		t.Error("the press-release source text leaked into the cut JSON")
	}
}
