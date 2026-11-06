package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// TestImplementResumeListsNoneInAnEmptyCheckout: the verb is wired, writes nothing and
// says there is no unfinished run where there is none.
func TestImplementResumeListsNoneInAnEmptyCheckout(t *testing.T) {
	root := checklistCheckout(t)
	t.Chdir(root)
	var out, errb bytes.Buffer
	if code := Run([]string{"implement", "resume"}, &out, &errb); code != 0 {
		t.Fatalf("resume: %d %q", code, errb.String())
	}
	if !strings.Contains(out.String(), "no unfinished run") {
		t.Fatalf("the listing: %q", out.String())
	}
	out.Reset()
	if code := Run([]string{"--json", "implement", "resume"}, &out, &errb); code != 0 {
		t.Fatalf("resume --json: %d %q", code, errb.String())
	}
	var got loop.Listing
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got.Runs == nil {
		t.Fatalf("the JSON: %q %v", out.String(), err)
	}
}

// TestImplementResumeTakeRefusesAnUnknownRun: a take names a run this checkout
// holds.
func TestImplementResumeTakeRefusesAnUnknownRun(t *testing.T) {
	root := checklistCheckout(t)
	t.Chdir(root)
	var out, errb bytes.Buffer
	if code := Run([]string{"implement", "resume", "--take", "run-2611040000000001"}, &out, &errb); code != 2 {
		t.Fatalf("resume --take over nothing: %d %q %q", code, out.String(), errb.String())
	}
}

// TestTheListingSaysAFailedReleasesTipHolds: the run let go and the tip stands, so "not held"
// would not be true.
func TestTheListingSaysAFailedReleasesTipHolds(t *testing.T) {
	var b bytes.Buffer
	renderListedRun(&b, loop.ListedRun{ReleaseFailed: true})
	if !strings.Contains(b.String(), "did not reach origin") || strings.Contains(b.String(), "not held") {
		t.Fatalf("the listing says what is true:\n%s", b.String())
	}
}

// TestTheListingNamesWhatAStopCannotHold: a pull request armed before it, and a page it could
// not write.
func TestTheListingNamesWhatAStopCannotHold(t *testing.T) {
	var b bytes.Buffer
	renderListedRun(&b, loop.ListedRun{Stop: &loop.RunStop{SetBy: "someone", PageFailed: "the day's page could not be written: no room",
		Left: []loop.StopLeft{{Lane: "lane-1", Stage: "land", Settled: true, ArmedPR: 7}}}})
	for _, want := range []string{"gh pr merge 7 --disable-auto", "the day's page could not be written: no room"} {
		if !strings.Contains(b.String(), want) {
			t.Fatalf("the listing lacks %q:\n%s", want, b.String())
		}
	}
}

// TestAListedRunAndATakeOverSayTheProfile: a session taking up what a drive left is told
// which subscription the run started on, so going on elsewhere is never silent.
func TestAListedRunAndATakeOverSayTheProfile(t *testing.T) {
	var listed strings.Builder
	renderListedRun(&listed, loop.ListedRun{RunID: "run-1", Key: "iss-1", Profile: "alpha"})
	if !strings.Contains(listed.String(), "profile: alpha") {
		t.Fatalf("the listing says:\n%s", listed.String())
	}
	var none strings.Builder
	renderListedRun(&none, loop.ListedRun{RunID: "run-1", Key: "iss-1"})
	if strings.Contains(none.String(), "profile:") {
		t.Fatalf("no profile, no line:\n%s", none.String())
	}
	var took strings.Builder
	renderTake(&took, loop.TakeResult{RunID: "run-1", Profile: "alpha", Next: "step"})
	for _, want := range []string{"alpha", "this session's own login"} {
		if !strings.Contains(took.String(), want) {
			t.Errorf("the take-over does not say %q:\n%s", want, took.String())
		}
	}
}
