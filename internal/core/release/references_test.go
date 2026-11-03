package release

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The release cut is where a pending reference must have landed
// (iss-2611031731385043): record-lint accepts an id that names a record still
// in review, so the cut refuses while any reference names a record the tree
// being cut does not hold, and reports each forward-looking marker whose
// record has arrived, for removal.

const decisionsDir = ".abcd/work/decisions/"

// proseLintConfig arms prose_citation_resolves over the stores a fixture
// writes, as the shipped .abcd/record-lint.json does.
const proseLintConfig = `{
  "rules": {
    "prose_citation_resolves": {
      "enabled": true,
      "severity": "blocker",
      "record_stores": {
        "itd": ".abcd/development/intents",
        "spc": ".abcd/development/specs",
        "iss": ".abcd/work/issues",
        "dcn": ".abcd/work/decisions"
      }
    }
  }
}
`

// citingRepo is a released repository with a cut that is ready but for the
// decision at rel, whose body is body.
func citingRepo(t *testing.T, rel, body string) *gittest.Repo {
	t.Helper()
	r := releasedRepo(t)
	r.Write(".abcd/record-lint.json", proseLintConfig)
	r.Write(shippedDir+"itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\nthe version is derived from what shipped.\n")
	r.Write(rel, body)
	r.Commit("ship an intent beside a decision")
	return r
}

// TestEmitRefusesACutCitingARecordNotOnMain: the decision cites a ruling whose
// record reached no tree the cut reads. Marked forward-looking or not, it is a
// citation of a record that has not landed, and the cut refuses until it does
// or the reference is corrected.
func TestEmitRefusesACutCitingARecordNotOnMain(t *testing.T) {
	for name, line := range map[string]string{
		"marked forward-looking":                       "The keys stay a fallback, per dcn-2611031615094092. <!-- record-lint: forward-looking -->",
		"unmarked, accepted as pending by record-lint": "The keys stay a fallback, per dcn-2611031615094092.",
	} {
		t.Run(name, func(t *testing.T) {
			rel := decisionsDir + "dcn-2611031720012399-with-both-api-keys-down.md"
			r := citingRepo(t, rel, "---\nid: dcn-2611031720012399\n---\n"+line+"\n")

			cut := emit(t, r)

			if cut.Ready {
				t.Fatalf("a cut citing a record that never landed is ready: %+v", cut)
			}
			var ref *Refusal
			for i := range cut.Refusals {
				if cut.Refusals[i].Kind == RefusalUnlandedReference {
					ref = &cut.Refusals[i]
				}
			}
			if ref == nil {
				t.Fatalf("refusals = %v, want %s", refusalKinds(cut), RefusalUnlandedReference)
			}
			if strings.Join(ref.Records, ",") != "dcn-2611031615094092" {
				t.Errorf("Records = %v, want the cited id", ref.Records)
			}
			for _, want := range []string{rel + ":4", "dcn-2611031615094092", "land"} {
				if !strings.Contains(ref.Reason, want) {
					t.Errorf("the refusal does not say %q:\n%s", want, ref.Reason)
				}
			}
			if len(cut.References.Unlanded) != 1 || cut.References.Unlanded[0].Line != 4 {
				t.Errorf("References.Unlanded = %+v, want the one citing line", cut.References.Unlanded)
			}
		})
	}
}

// TestEmitReportsAForwardLookingMarkerWhoseRecordReachedMain: the ruling the
// marker waited for is now in the tree, so the marker says nothing true any
// more. The cut reports it for removal and refuses nothing over it.
func TestEmitReportsAForwardLookingMarkerWhoseRecordReachedMain(t *testing.T) {
	rel := decisionsDir + "dcn-2611031720012399-with-both-api-keys-down.md"
	r := releasedRepo(t)
	r.Write(".abcd/record-lint.json", proseLintConfig)
	r.Write(shippedDir+"itd-73-derived-versioning.md",
		"---\nid: itd-73\nimpact: additive\n---\n\n# A Version Is A Fact\n\nthe version is derived from what shipped.\n")
	r.Write(decisionsDir+"dcn-2611031615094092-the-run-builds-on-subscriptions.md",
		"---\nid: dcn-2611031615094092\n---\nthe ruling, landed.\n")
	r.Write(rel, "---\nid: dcn-2611031720012399\n---\n"+
		"The keys stay a fallback, per dcn-2611031615094092. <!-- record-lint: forward-looking -->\n")
	r.Commit("the ruling lands beside the decision that waited for it")

	cut := emit(t, r)

	if !cut.Ready {
		t.Fatalf("a marker whose record landed refused the cut: %+v", cut.Refusals)
	}
	if len(cut.References.StaleMarkers) != 1 {
		t.Fatalf("StaleMarkers = %+v, want the one marker", cut.References.StaleMarkers)
	}
	m := cut.References.StaleMarkers[0]
	if m.File != rel || m.Line != 4 || strings.Join(m.IDs, ",") != "dcn-2611031615094092" {
		t.Errorf("stale marker = %+v, want %s:4 citing dcn-2611031615094092", m, rel)
	}
}
