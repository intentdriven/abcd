package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The paths an uncommitted spec close and issue resolve leave in the working
// tree, which the cut reads at HEAD (iss-2610050259118177).
const (
	uncommittedIntent = ".abcd/development/intents/shipped/itd-80-closed.md"
	uncommittedIssue  = ".abcd/work/issues/resolved/iss-81-resolved.md"
)

// shipWithUncommittedClose is a ready cut (itd-73 committed) with a spec close
// and an issue resolve left uncommitted on top of it: the v0.13.0 shape, where
// the derivation answered rc=0 with a plausible cut missing the closed intent.
func shipWithUncommittedClose(t *testing.T) *gittest.Repo {
	t.Helper()
	r := shipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-x.md", "---\nid: itd-73\nimpact: additive\n---\n# x\n")
	r.Write(".abcd/development/intents/planned/itd-80-closed.md", "---\nid: itd-80\nimpact: additive\n---\n# closed\n")
	r.Commit("ship itd-73; itd-80 is planned")
	r.Write(uncommittedIntent, "---\nid: itd-80\nimpact: additive\n---\n# closed\n")
	r.Remove(".abcd/development/intents/planned/itd-80-closed.md")
	r.Write(uncommittedIssue, "---\nid: iss-81\nimpact: fix\n---\n# resolved\n")
	return r
}

// cutJSON is the slice of the cut's JSON these tests read.
type cutJSON struct {
	Ready    bool   `json:"ready"`
	NextTag  string `json:"next_tag"`
	Refusals []struct {
		Kind   string `json:"kind"`
		Reason string `json:"reason"`
	} `json:"refusals"`
}

func decodeCut(t *testing.T, out []byte) cutJSON {
	t.Helper()
	var got cutJSON
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	return got
}

// TestLaunchShipRefusesUncommittedRecords is the derivation the person reads
// first: a record move left in the working tree is refused, exit 1, naming every
// path, in both renders.
func TestLaunchShipRefusesUncommittedRecords(t *testing.T) {
	r := shipWithUncommittedClose(t)

	out, err := shipIn(t, r, "launch", "ship")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"REFUSED", "refused (uncommitted-records)", uncommittedIntent, uncommittedIssue} {
		if !strings.Contains(string(out), want) {
			t.Errorf("render does not mention %q:\n%s", want, out)
		}
	}

	out, err = shipIn(t, r, "launch", "ship", "--json")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("--json exit = %d, want 1\n%s", code, out)
	}
	got := decodeCut(t, out)
	if got.Ready || got.NextTag != "" || len(got.Refusals) != 1 || got.Refusals[0].Kind != "uncommitted-records" {
		t.Fatalf("ready=%v next_tag=%q refusals=%+v, want the one uncommitted-records refusal", got.Ready, got.NextTag, got.Refusals)
	}
	for _, want := range []string{uncommittedIntent, uncommittedIssue} {
		if !strings.Contains(got.Refusals[0].Reason, want) {
			t.Errorf("JSON reason does not name %q: %s", want, got.Refusals[0].Reason)
		}
	}
}

// TestLaunchShipIgnoresDirtOutsideTheRecords: the emit step renders no payload,
// so dirt the cut does not read is not its refusal — the ingest's pre-flight
// dirty-tree gate still judges the whole tree before anything is written.
func TestLaunchShipIgnoresDirtOutsideTheRecords(t *testing.T) {
	r := shipFixture(t)
	r.Write(".abcd/development/intents/shipped/itd-73-x.md", "---\nid: itd-73\nimpact: additive\n---\n# x\n")
	r.Commit("ship an intent")
	r.Write("notes.txt", "an edit in progress\n")

	out, err := shipIn(t, r, "launch", "ship", "--json")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	if got := decodeCut(t, out); !got.Ready || got.NextTag != "v0.4.1" {
		t.Errorf("ready=%v next_tag=%q refusals=%+v, want a ready v0.4.1", got.Ready, got.NextTag, got.Refusals)
	}
}

// TestChangelogPreviewReportsUncommittedRecords: the preview is a status render
// that always exits 0 and reports a refused cut as information, so it carries
// the same refusal — loudly, as REFUSED with the paths — rather than a cut that
// silently leaves the record out.
func TestChangelogPreviewReportsUncommittedRecords(t *testing.T) {
	r := shipWithUncommittedClose(t)

	out, err := shipIn(t, r, "changelog")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	for _, want := range []string{"REFUSED", "uncommitted-records", uncommittedIntent} {
		if !strings.Contains(string(out), want) {
			t.Errorf("preview does not mention %q:\n%s", want, out)
		}
	}

	out, err = shipIn(t, r, "changelog", "--json")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("--json exit = %d, want 0\n%s", code, out)
	}
	if got := decodeCut(t, out); got.Ready || len(got.Refusals) != 1 || got.Refusals[0].Kind != "uncommitted-records" {
		t.Errorf("ready=%v refusals=%+v, want the uncommitted-records refusal", got.Ready, got.Refusals)
	}
}
